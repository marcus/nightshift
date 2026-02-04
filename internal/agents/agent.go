// Package agents provides interfaces and implementations for spawning AI agents.
// Unlike providers (which track usage), agents execute tasks autonomously.
package agents

import (
	"context"
	"time"
)

// DefaultTimeout is the default agent execution timeout (30 minutes).
const DefaultTimeout = 30 * time.Minute

// Agent is the interface for AI agent execution.
type Agent interface {
	// Name returns the agent identifier.
	Name() string

	// Execute runs a prompt and returns the output.
	Execute(ctx context.Context, opts ExecuteOptions) (*ExecuteResult, error)
}

// ExecuteOptions configures an agent execution.
type ExecuteOptions struct {
	Prompt  string        // The prompt/task for the agent
	WorkDir string        // Working directory for execution
	Files   []string      // Optional file paths to include as context
	Timeout time.Duration // Execution timeout (0 = default)
}

// ExecuteResult holds the outcome of an agent execution.
type ExecuteResult struct {
	Output   string        // Agent's text output
	JSON     []byte        // Structured JSON output if available
	ExitCode int           // Process exit code
	Duration time.Duration // Execution duration
	Error    string        // Error message if failed
}

// IsSuccess returns true if the execution succeeded.
func (r *ExecuteResult) IsSuccess() bool {
	return r.ExitCode == 0 && r.Error == ""
}
