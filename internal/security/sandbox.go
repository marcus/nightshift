// Package security provides sandboxed execution for nightshift agents.
// Agents run in isolated environments with minimal permissions.
package security

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SandboxConfig configures the sandbox environment.
type SandboxConfig struct {
	// WorkDir is the working directory for the sandboxed process.
	WorkDir string
	// TempDir is the temporary directory for working files.
	TempDir string
	// AllowNetwork enables network access (default false).
	AllowNetwork bool
	// AllowedPaths are paths the process can access.
	AllowedPaths []string
	// DeniedPaths are paths explicitly blocked.
	DeniedPaths []string
	// MaxDuration is the maximum execution time.
	MaxDuration time.Duration
	// MaxMemoryMB is the max memory in megabytes (0 = unlimited).
	MaxMemoryMB int
	// Environment variables to pass through.
	Environment map[string]string
	// Cleanup removes temp files after execution (default true).
	Cleanup bool
}

// DefaultSandboxConfig returns a secure default configuration.
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		AllowNetwork: false,
		MaxDuration:  30 * time.Minute,
		MaxMemoryMB:  0, // No limit by default
		Environment:  make(map[string]string),
		Cleanup:      true,
	}
}

// Sandbox provides an isolated execution environment.
type Sandbox struct {
	config  SandboxConfig
	tempDir string
	mu      sync.Mutex
	active  bool
}

// NewSandbox creates a new sandbox with the given configuration.
func NewSandbox(cfg SandboxConfig) (*Sandbox, error) {
	// Create temp directory if not specified
	tempDir := cfg.TempDir
	if tempDir == "" {
		var err error
		tempDir, err = os.MkdirTemp("", "nightshift-sandbox-*")
		if err != nil {
			return nil, fmt.Errorf("creating sandbox temp dir: %w", err)
		}
	}

	return &Sandbox{
		config:  cfg,
		tempDir: tempDir,
	}, nil
}

// TempDir returns the sandbox temporary directory.
func (s *Sandbox) TempDir() string {
	return s.tempDir
}

// Execute runs a command within the sandbox.
func (s *Sandbox) Execute(ctx context.Context, name string, args ...string) (*ExecResult, error) {
	s.mu.Lock()
	s.active = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.active = false
		s.mu.Unlock()
	}()

	// Apply timeout if configured
	if s.config.MaxDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.MaxDuration)
		defer cancel()
	}

	// Validate command path
	if err := s.validateCommand(name); err != nil {
		return nil, err
	}

	// Build command
	cmd := exec.CommandContext(ctx, name, args...)

	// Set working directory
	if s.config.WorkDir != "" {
		cmd.Dir = s.config.WorkDir
	} else {
		cmd.Dir = s.tempDir
	}

	// Configure environment
	cmd.Env = s.buildEnvironment()

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err.Error()
			result.ExitCode = -1
		}
	}

	return result, nil
}

// ExecResult holds the result of a sandboxed execution.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    string
}

// Success returns true if the command completed successfully.
func (r *ExecResult) Success() bool {
	return r.ExitCode == 0 && r.Error == ""
}

// validateCommand checks if a command is allowed to run.
func (s *Sandbox) validateCommand(name string) error {
	// Resolve full path
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("command not found: %s", name)
	}

	// Check against denied paths
	for _, denied := range s.config.DeniedPaths {
		if strings.HasPrefix(path, denied) {
			return fmt.Errorf("command path denied: %s", path)
		}
	}

	return nil
}

// buildEnvironment constructs the environment for sandboxed execution.
func (s *Sandbox) buildEnvironment() []string {
	env := make([]string, 0)

	// Minimal safe environment
	safeVars := []string{"PATH", "HOME", "USER", "SHELL", "TERM", "LANG", "LC_ALL"}
	for _, v := range safeVars {
		if val := os.Getenv(v); val != "" {
			env = append(env, v+"="+val)
		}
	}

	// API keys (needed for agent execution)
	apiKeyVars := []string{EnvAnthropicKey, EnvOpenAIKey}
	for _, v := range apiKeyVars {
		if val := os.Getenv(v); val != "" {
			env = append(env, v+"="+val)
		}
	}

	// Add configured environment variables
	for k, v := range s.config.Environment {
		env = append(env, k+"="+v)
	}

	// Set sandbox-specific vars
	env = append(env, "NIGHTSHIFT_SANDBOX=1")
	env = append(env, "TMPDIR="+s.tempDir)

	// Restrict network if not allowed
	if !s.config.AllowNetwork {
		// This is a hint - actual network restriction requires OS-level controls
		env = append(env, "NIGHTSHIFT_NO_NETWORK=1")
	}

	return env
}

// ValidatePath checks if a path is accessible within the sandbox.
func (s *Sandbox) ValidatePath(path string) error {
	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Check denied paths first
	for _, denied := range s.config.DeniedPaths {
		if strings.HasPrefix(absPath, denied) {
			return fmt.Errorf("path access denied: %s", path)
		}
	}

	// If allowed paths are specified, path must be within one
	if len(s.config.AllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range s.config.AllowedPaths {
			if strings.HasPrefix(absPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path not in allowed list: %s", path)
		}
	}

	return nil
}

// CreateTempFile creates a temporary file within the sandbox.
func (s *Sandbox) CreateTempFile(pattern string) (*os.File, error) {
	return os.CreateTemp(s.tempDir, pattern)
}

// CreateTempDir creates a temporary directory within the sandbox.
func (s *Sandbox) CreateTempDir(pattern string) (string, error) {
	return os.MkdirTemp(s.tempDir, pattern)
}

// Cleanup removes all temporary files created by the sandbox.
func (s *Sandbox) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.config.Cleanup {
		return nil
	}

	if s.active {
		return fmt.Errorf("cannot cleanup while sandbox is active")
	}

	if s.tempDir != "" && s.tempDir != "/" && s.tempDir != os.TempDir() {
		return os.RemoveAll(s.tempDir)
	}

	return nil
}

// IsActive returns true if a command is currently executing.
func (s *Sandbox) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// SandboxedAgent wraps an agent to run in a sandbox.
type SandboxedAgent struct {
	sandbox *Sandbox
	workDir string
}

// NewSandboxedAgent creates a new sandboxed agent wrapper.
func NewSandboxedAgent(cfg SandboxConfig) (*SandboxedAgent, error) {
	sandbox, err := NewSandbox(cfg)
	if err != nil {
		return nil, err
	}

	return &SandboxedAgent{
		sandbox: sandbox,
		workDir: cfg.WorkDir,
	}, nil
}

// Sandbox returns the underlying sandbox.
func (a *SandboxedAgent) Sandbox() *Sandbox {
	return a.sandbox
}

// Close cleans up sandbox resources.
func (a *SandboxedAgent) Close() error {
	return a.sandbox.Cleanup()
}
