// claude.go implements the Provider interface for Claude Code CLI.
package providers

import "context"

// Claude wraps the Claude Code CLI as a provider.
type Claude struct {
	// TODO: Add fields (model, config path, etc.)
}

// NewClaude creates a Claude Code provider.
func NewClaude() *Claude {
	// TODO: Implement
	return &Claude{}
}

// Name returns "claude".
func (c *Claude) Name() string {
	return "claude"
}

// Execute runs a task via Claude Code CLI.
func (c *Claude) Execute(ctx context.Context, task Task) (Result, error) {
	// TODO: Implement - spawn claude CLI process
	return Result{}, nil
}

// Cost returns Claude's token pricing.
func (c *Claude) Cost() (inputCents, outputCents int64) {
	// TODO: Return actual pricing
	return 0, 0
}
