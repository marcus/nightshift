// codex.go implements the Provider interface for OpenAI Codex CLI.
package providers

import "context"

// Codex wraps the Codex CLI as a provider.
type Codex struct {
	// TODO: Add fields (model, API key, etc.)
}

// NewCodex creates a Codex provider.
func NewCodex() *Codex {
	// TODO: Implement
	return &Codex{}
}

// Name returns "codex".
func (c *Codex) Name() string {
	return "codex"
}

// Execute runs a task via Codex CLI.
func (c *Codex) Execute(ctx context.Context, task Task) (Result, error) {
	// TODO: Implement - spawn codex CLI process
	return Result{}, nil
}

// Cost returns Codex's token pricing.
func (c *Codex) Cost() (inputCents, outputCents int64) {
	// TODO: Return actual pricing
	return 0, 0
}
