// codex.go implements the Agent interface for OpenAI Codex CLI.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CodexAgent spawns Codex CLI for task execution.
type CodexAgent struct {
	binaryPath string        // Path to codex binary (default: "codex")
	timeout    time.Duration // Default timeout
	runner     CommandRunner // Command executor (for testing)
	bypassPerm bool          // Pass --dangerously-bypass-approvals-and-sandbox
}

// CodexOption configures a CodexAgent.
type CodexOption func(*CodexAgent)

// WithCodexBinaryPath sets a custom path to the codex binary.
func WithCodexBinaryPath(path string) CodexOption {
	return func(a *CodexAgent) {
		a.binaryPath = path
	}
}

// WithCodexDefaultTimeout sets the default execution timeout.
func WithCodexDefaultTimeout(d time.Duration) CodexOption {
	return func(a *CodexAgent) {
		a.timeout = d
	}
}

// WithDangerouslyBypassApprovalsAndSandbox sets whether to pass --dangerously-bypass-approvals-and-sandbox.
func WithDangerouslyBypassApprovalsAndSandbox(enabled bool) CodexOption {
	return func(a *CodexAgent) {
		a.bypassPerm = enabled
	}
}

// WithCodexRunner sets a custom command runner (for testing).
func WithCodexRunner(r CommandRunner) CodexOption {
	return func(a *CodexAgent) {
		a.runner = r
	}
}

// NewCodexAgent creates a Codex CLI agent.
func NewCodexAgent(opts ...CodexOption) *CodexAgent {
	a := &CodexAgent{
		binaryPath: "codex",
		timeout:    DefaultTimeout,
		runner:     &ExecRunner{},
		bypassPerm: true,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns "codex".
func (a *CodexAgent) Name() string {
	return "codex"
}

// Execute runs codex with the given prompt in non-interactive mode.
func (a *CodexAgent) Execute(ctx context.Context, opts ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Determine timeout
	timeout := a.timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command args for headless/non-interactive execution
	// Codex CLI uses --quiet for non-interactive mode and accepts prompt directly
	args := []string{"--quiet"}
	if a.bypassPerm {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}

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

// ExecuteWithFiles runs codex with file context included.
func (a *CodexAgent) ExecuteWithFiles(ctx context.Context, prompt string, files []string, workDir string) (*ExecuteResult, error) {
	return a.Execute(ctx, ExecuteOptions{
		Prompt:  prompt,
		Files:   files,
		WorkDir: workDir,
	})
}

// buildFileContext reads files and formats them as context.
func (a *CodexAgent) buildFileContext(files []string) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Context Files\n\n")

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", path, err)
		}

		// Use absolute path for cleaner output
		displayPath := path
		if abs, err := filepath.Abs(path); err == nil {
			displayPath = abs
		}

		fmt.Fprintf(&sb, "## File: %s\n\n```\n%s\n```\n\n", displayPath, string(content))
	}

	return sb.String(), nil
}

// extractJSON attempts to find and parse JSON from the output.
// Returns nil if no valid JSON found.
func (a *CodexAgent) extractJSON(output []byte) []byte {
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

// Available checks if the codex binary is available in PATH.
func (a *CodexAgent) Available() bool {
	_, err := exec.LookPath(a.binaryPath)
	return err == nil
}

// Version returns the codex CLI version.
func (a *CodexAgent) Version() (string, error) {
	cmd := exec.Command(a.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
