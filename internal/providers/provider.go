// Package providers defines interfaces and implementations for AI coding agents.
// Supports multiple backends: Claude Code, Codex CLI, etc.
package providers

import "context"

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
	// TODO: Add task fields (prompt, files, etc.)
}

// Result holds the outcome of a provider execution.
type Result struct {
	// TODO: Add result fields (output, tokens used, etc.)
}
