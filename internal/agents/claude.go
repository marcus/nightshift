// claude.go implements the Agent interface for Claude Code CLI.
package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandRunner executes shell commands. Allows mocking in tests.
type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, dir string, stdin string) (stdout, stderr string, exitCode int, err error)
}

// ExecRunner is the default CommandRunner using os/exec.
type ExecRunner struct{}

// Run executes a command and returns output.
func (r *ExecRunner) Run(ctx context.Context, name string, args []string, dir string, stdin string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	err := cmd.Run()

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode, err
}

// ClaudeAgent spawns Claude Code CLI for task execution.
type ClaudeAgent struct {
	binaryPath string        // Path to claude binary (default: "claude")
	timeout    time.Duration // Default timeout
	runner     CommandRunner // Command executor (for testing)
}

// ClaudeOption configures a ClaudeAgent.
type ClaudeOption func(*ClaudeAgent)

// WithBinaryPath sets a custom path to the claude binary.
func WithBinaryPath(path string) ClaudeOption {
	return func(a *ClaudeAgent) {
		a.binaryPath = path
	}
}

// WithDefaultTimeout sets the default execution timeout.
func WithDefaultTimeout(d time.Duration) ClaudeOption {
	return func(a *ClaudeAgent) {
		a.timeout = d
	}
}

// WithRunner sets a custom command runner (for testing).
func WithRunner(r CommandRunner) ClaudeOption {
	return func(a *ClaudeAgent) {
		a.runner = r
	}
}

// NewClaudeAgent creates a Claude Code agent.
func NewClaudeAgent(opts ...ClaudeOption) *ClaudeAgent {
	a := &ClaudeAgent{
		binaryPath: "claude",
		timeout:    DefaultTimeout,
		runner:     &ExecRunner{},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns "claude".
func (a *ClaudeAgent) Name() string {
	return "claude"
}

// Execute runs claude --print with the given prompt.
func (a *ClaudeAgent) Execute(ctx context.Context, opts ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Determine timeout
	timeout := a.timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command args
	args := []string{"--print"}

	// Add prompt directly as argument
	if opts.Prompt != "" {
		args = append(args, opts.Prompt)
	}

	// Build stdin content from files if provided
	var stdinContent string
	if len(opts.Files) > 0 {
		var err error
		stdinContent, err = a.buildFileContext(opts.Files)
		if err != nil {
			return &ExecuteResult{
				Error:    fmt.Sprintf("building file context: %v", err),
				Duration: time.Since(start),
			}, err
		}
	}

	// Run command
	stdout, stderr, exitCode, err := a.runner.Run(ctx, a.binaryPath, args, opts.WorkDir, stdinContent)

	result := &ExecuteResult{
		Output:   stdout,
		ExitCode: exitCode,
		Duration: time.Since(start),
	}

	// Check for context timeout
	if ctx.Err() == context.DeadlineExceeded {
		result.Error = fmt.Sprintf("timeout after %v", timeout)
		result.ExitCode = -1
		return result, ctx.Err()
	}

	// Check for other errors
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = stderr
		} else {
			result.Error = err.Error()
		}
		return result, err
	}

	// Try to parse JSON output
	result.JSON = a.extractJSON([]byte(stdout))

	return result, nil
}

// ExecuteWithFiles runs claude with file context included.
func (a *ClaudeAgent) ExecuteWithFiles(ctx context.Context, prompt string, files []string, workDir string) (*ExecuteResult, error) {
	return a.Execute(ctx, ExecuteOptions{
		Prompt:  prompt,
		Files:   files,
		WorkDir: workDir,
	})
}

// buildFileContext reads files and formats them as context.
func (a *ClaudeAgent) buildFileContext(files []string) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Context Files\n\n")

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", path, err)
		}

		// Use relative path if possible for cleaner output
		displayPath := path
		if abs, err := filepath.Abs(path); err == nil {
			displayPath = abs
		}

		sb.WriteString(fmt.Sprintf("## File: %s\n\n```\n%s\n```\n\n", displayPath, string(content)))
	}

	return sb.String(), nil
}

// extractJSON attempts to find and parse JSON from the output.
// Returns nil if no valid JSON found.
func (a *ClaudeAgent) extractJSON(output []byte) []byte {
	// Try to parse the entire output as JSON
	if json.Valid(output) {
		return output
	}

	// Look for JSON object or array in output
	// Find first { or [ and matching closer
	start := -1
	var opener, closer byte

	for i, b := range output {
		if b == '{' || b == '[' {
			start = i
			opener = b
			if b == '{' {
				closer = '}'
			} else {
				closer = ']'
			}
			break
		}
	}

	if start == -1 {
		return nil
	}

	// Find matching closer by counting nesting
	depth := 0
	for i := start; i < len(output); i++ {
		if output[i] == opener {
			depth++
		} else if output[i] == closer {
			depth--
			if depth == 0 {
				candidate := output[start : i+1]
				if json.Valid(candidate) {
					return candidate
				}
				break
			}
		}
	}

	return nil
}

// Available checks if the claude binary is available in PATH.
func (a *ClaudeAgent) Available() bool {
	_, err := exec.LookPath(a.binaryPath)
	return err == nil
}

// Version returns the claude CLI version.
func (a *ClaudeAgent) Version() (string, error) {
	cmd := exec.Command(a.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
