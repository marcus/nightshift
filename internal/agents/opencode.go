// Package agents provides interfaces and implementations for spawning AI agents.
// Unlike providers (which track usage), agents execute tasks autonomously.
package agents

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// OpencodeAgent implements the Agent interface for the opencode CLI.
type OpencodeAgent struct {
	binaryPath string
}

// NewOpencodeAgent creates a new Opencode agent.
func NewOpencodeAgent() *OpencodeAgent {
	return &OpencodeAgent{
		binaryPath: "opencode",
	}
}

// Name returns "opencode".
func (a *OpencodeAgent) Name() string {
	return "opencode"
}

// Execute runs a prompt using the opencode CLI.
func (a *OpencodeAgent) Execute(ctx context.Context, opts ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Check if opencode CLI is available
	cmd := exec.CommandContext(ctx, a.binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ExecuteResult{
			Duration: time.Since(start),
			Error:    fmt.Sprintf("opencode CLI not found: %v", err),
		}, nil
	}

	// Construct command: opencode run [options] [prompt]
	args := []string{"run"}

	// Add prompt as positional argument
	if opts.Prompt != "" {
		args = append(args, opts.Prompt)
	}

	// Add files as --file arguments
	for _, file := range opts.Files {
		args = append(args, "--file", file)
	}

	// Execute opencode command
	cmd = exec.CommandContext(ctx, a.binaryPath, args...)
	cmd.Dir = opts.WorkDir

	output, err = cmd.CombinedOutput()

	duration := time.Since(start)

	if err != nil {
		return &ExecuteResult{
			Duration: duration,
			Error:    err.Error(),
			Output:   string(output),
		}, nil
	}

	// Handle exit code properly
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	return &ExecuteResult{
		Output:   string(output),
		Duration: duration,
		ExitCode: exitCode,
	}, nil
}

// Available checks if the opencode CLI is available in PATH.
func (a *OpencodeAgent) Available() bool {
	cmd := exec.Command(a.binaryPath, "--version")
	err := cmd.Run()
	return err == nil
}
