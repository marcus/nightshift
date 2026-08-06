// copilot.go implements the Agent interface for GitHub Copilot CLI.
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

// CopilotAgent spawns GitHub Copilot CLI for task execution.
//
// GitHub Copilot CLI implementation notes:
// - Supports both 'gh copilot' (passthrough to copilot binary) and standalone 'copilot' binary
// - Non-interactive mode: use -p/--prompt flag (exits after completion)
// - Uses --no-ask-user to disable the ask_user tool (fully autonomous)
// - Uses --silent to output only the agent response (no stats)
//
// Install options:
// - Via gh: gh copilot (downloads copilot binary automatically if not in PATH)
// - Standalone: npm install -g @github/copilot or curl script
// - Usage: copilot -p "<prompt>" --no-ask-user --silent
type CopilotAgent struct {
	binaryPath           string        // Path to binary: "gh" or "copilot" (default: "gh")
	dangerouslySkipPerms bool          // Pass --allow-all-tools --allow-all-urls
	model                string        // Default model to use
	timeout              time.Duration // Default timeout
	runner               CommandRunner // Command executor (for testing)
}

// CopilotOption configures a CopilotAgent.
type CopilotOption func(*CopilotAgent)

// WithCopilotBinaryPath sets a custom path to the copilot binary ("gh" or "copilot").
func WithCopilotBinaryPath(path string) CopilotOption {
	return func(a *CopilotAgent) {
		a.binaryPath = path
	}
}

// WithCopilotDangerouslySkipPermissions sets whether to pass --allow-all-tools and --allow-all-urls.
func WithCopilotDangerouslySkipPermissions(enabled bool) CopilotOption {
	return func(a *CopilotAgent) {
		a.dangerouslySkipPerms = enabled
	}
}

// WithCopilotDefaultTimeout sets the default execution timeout.
func WithCopilotDefaultTimeout(d time.Duration) CopilotOption {
	return func(a *CopilotAgent) {
		a.timeout = d
	}
}

// WithCopilotModel sets the default model to use.
func WithCopilotModel(model string) CopilotOption {
	return func(a *CopilotAgent) {
		a.model = model
	}
}

// WithCopilotRunner sets a custom command runner (for testing).
func WithCopilotRunner(r CommandRunner) CopilotOption {
	return func(a *CopilotAgent) {
		a.runner = r
	}
}

// NewCopilotAgent creates a GitHub Copilot CLI agent.
func NewCopilotAgent(opts ...CopilotOption) *CopilotAgent {
	a := &CopilotAgent{
		binaryPath: "gh",
		timeout:    DefaultTimeout,
		runner:     &ExecRunner{},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns "copilot".
func (a *CopilotAgent) Name() string {
	return "copilot"
}

// Execute runs the Copilot CLI with the given prompt in non-interactive mode.
//
// Both 'gh copilot' and standalone 'copilot' use the same -p flag interface.
// For 'gh copilot', '--' is used to pass flags through to the copilot binary.
func (a *CopilotAgent) Execute(ctx context.Context, opts ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Determine timeout
	timeout := a.timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command args. Both modes use -p for non-interactive prompt.
	// gh mode uses '--' to pass flags through to the copilot binary.
	// 1. gh copilot: gh copilot -- -p <prompt> --no-ask-user [--allow-all-tools --allow-all-urls] --silent
	// 2. standalone: copilot -p <prompt> --no-ask-user [--allow-all-tools --allow-all-urls] --silent
	var args []string

	// Determine model
	model := opts.Model
	if model == "" {
		model = a.model
	}

	if a.binaryPath == "gh" {
		args = []string{"copilot", "--", "-p", opts.Prompt, "--no-ask-user", "--silent"}
	} else {
		args = []string{"-p", opts.Prompt, "--no-ask-user", "--silent"}
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	if a.dangerouslySkipPerms {
		args = append(args, "--allow-all-tools", "--allow-all-urls")
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

// ExecuteWithFiles runs gh copilot with file context included.
func (a *CopilotAgent) ExecuteWithFiles(ctx context.Context, prompt string, files []string, workDir string) (*ExecuteResult, error) {
	return a.Execute(ctx, ExecuteOptions{
		Prompt:  prompt,
		Files:   files,
		WorkDir: workDir,
	})
}

// buildFileContext reads files and formats them as context.
func (a *CopilotAgent) buildFileContext(files []string) (string, error) {
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
func (a *CopilotAgent) extractJSON(output []byte) []byte {
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

// Available checks if the gh binary is available in PATH and copilot extension is installed.
func (a *CopilotAgent) Available() bool {
	// Check if binary is available
	if _, err := exec.LookPath(a.binaryPath); err != nil {
		return false
	}

	// If using standalone copilot binary, it's available
	if a.binaryPath == "copilot" {
		return true
	}

	// If using gh, check if copilot extension is installed
	// Run: gh extension list | grep copilot
	cmd := exec.Command(a.binaryPath, "extension", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Look for github/gh-copilot in the extension list
	return strings.Contains(string(output), "github/gh-copilot") ||
		strings.Contains(string(output), "gh-copilot")
}

// Version returns the copilot CLI version.
func (a *CopilotAgent) Version() (string, error) {
	var cmd *exec.Cmd
	if a.binaryPath == "gh" {
		cmd = exec.Command("gh", "copilot", "--", "--version")
	} else {
		cmd = exec.Command(a.binaryPath, "--version")
	}
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
