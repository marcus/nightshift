// Package providers defines interfaces and implementations for AI coding agents.
// Supports multiple backends: Claude Code, Codex CLI, Opencode, etc.
package providers

import (
	"context"
	"time"
)

// Provider is the interface all AI coding agents must implement.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// Execute runs a task and returns the result.
	Execute(ctx context.Context, task Task) (Result, error)

	// Cost returns estimated cost per 1K tokens (input, output).
	Cost() (inputCents, outputCents int64)
}

// Task represents work to be done by a provider.
type Task struct {
	// Prompt is the instruction for the AI agent
	Prompt string

	// Files contains file paths to be processed
	Files []string

	// Context provides additional context for the task
	Context map[string]interface{}
}

// Result holds the outcome of a provider execution.
type Result struct {
	// Output is the generated content from the AI agent
	Output string

	// TokensUsed tracks the number of tokens consumed
	TokensUsed int64

	// Duration is how long the execution took
	Duration time.Duration

	// Error contains any error that occurred during execution
	Error error
}
