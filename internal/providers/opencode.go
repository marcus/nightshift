// Package providers defines interfaces and implementations for AI coding agents.
// Supports multiple backends: Claude Code, Codex CLI, Opencode, etc.
package providers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Opencode represents the Opencode provider.
type Opencode struct {
	// Add any necessary fields for opencode integration
}

// NewOpencode creates a new Opencode provider.
func NewOpencode() *Opencode {
	return &Opencode{}
}

// Name returns "opencode".
func (o *Opencode) Name() string {
	return "opencode"
}

// Execute runs a task via Opencode CLI.
func (o *Opencode) Execute(ctx context.Context, task Task) (Result, error) {
	start := time.Now()

	// Check if opencode CLI is available
	cmd := exec.CommandContext(ctx, "opencode", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Result{
			Duration: time.Since(start),
			Error:    fmt.Errorf("opencode CLI not available: %w", err),
		}, nil
	}

	// Prepare the opencode command with the task parameters
	args := []string{"run"}

	// Add prompt as argument
	if task.Prompt != "" {
		args = append(args, task.Prompt)
	}

	// Add files as --file arguments
	for _, file := range task.Files {
		args = append(args, "--file", file)
	}

	// Add context information if present
	if len(task.Context) > 0 {
		// Convert context to string for opencode command
		contextStr := ""
		for key, value := range task.Context {
			contextStr += fmt.Sprintf("%s=%v ", key, value)
		}
		args = append(args, "--context", strings.TrimSpace(contextStr))
	}

	// Execute opencode command
	cmd = exec.CommandContext(ctx, "opencode", args...)
	output, err = cmd.CombinedOutput()

	duration := time.Since(start)

	if err != nil {
		return Result{
			Duration: duration,
			Error:    fmt.Errorf("opencode execution failed: %w", err),
			Output:   string(output),
		}, nil
	}

	return Result{
		Output:     string(output),
		Duration:   duration,
		TokensUsed: 0, // TODO: Parse tokens from opencode response if available
	}, nil
}

// Cost returns estimated cost per 1K tokens (input, output).
func (o *Opencode) Cost() (inputCents, outputCents int64) {
	// Placeholder values - adjust based on actual opencode pricing
	// Assuming similar pricing to other local models
	return 0, 0
}
