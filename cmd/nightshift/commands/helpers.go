package commands

import (
	"fmt"
	"strings"

	"github.com/marcus/nightshift/internal/agents"
)

// agentByName creates an agent for the given provider name.
// Returns an error if the provider is unknown or its CLI is not in PATH.
func agentByName(provider string) (agents.Agent, error) {
	switch strings.ToLower(provider) {
	case "claude":
		a := agents.NewClaudeAgent()
		if !a.Available() {
			return nil, fmt.Errorf("claude CLI not found in PATH")
		}
		return a, nil
	case "codex":
		a := agents.NewCodexAgent()
		if !a.Available() {
			return nil, fmt.Errorf("codex CLI not found in PATH")
		}
		return a, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: claude, codex)", provider)
	}
}
