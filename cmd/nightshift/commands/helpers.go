package commands

import (
	"fmt"
	"strings"

	"github.com/marcus/nightshift/internal/agents"
	"github.com/marcus/nightshift/internal/config"
)

// agentByName creates an agent for the given provider name.
// Returns an error if the provider is unknown or its CLI is not in PATH.
func agentByName(cfg *config.Config, provider string) (agents.Agent, error) {
	switch strings.ToLower(provider) {
	case "claude":
		a := newClaudeAgentFromConfig(cfg)
		if !a.Available() {
			return nil, fmt.Errorf("claude CLI not found in PATH")
		}
		return a, nil
	case "codex":
		a := newCodexAgentFromConfig(cfg)
		if !a.Available() {
			return nil, fmt.Errorf("codex CLI not found in PATH")
		}
		return a, nil
	case "gemini":
		a := newGeminiAgentFromConfig(cfg)
		if !a.Available() {
			return nil, fmt.Errorf("gemini CLI not found in PATH")
		}
		return a, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: claude, codex, gemini)", provider)
	}
}

func newClaudeAgentFromConfig(cfg *config.Config) *agents.ClaudeAgent {
	if cfg == nil {
		return agents.NewClaudeAgent()
	}
	return agents.NewClaudeAgent(
		agents.WithDangerouslySkipPermissions(cfg.Providers.Claude.DangerouslySkipPermissions),
	)
}

func newCodexAgentFromConfig(cfg *config.Config) *agents.CodexAgent {
	if cfg == nil {
		return agents.NewCodexAgent()
	}
	return agents.NewCodexAgent(
		agents.WithDangerouslyBypassApprovalsAndSandbox(cfg.Providers.Codex.DangerouslyBypassApprovalsAndSandbox),
	)
}

func newGeminiAgentFromConfig(cfg *config.Config) *agents.GeminiAgent {
	if cfg == nil {
		return agents.NewGeminiAgent()
	}
	opts := []agents.GeminiOption{
		agents.WithGeminiAutoApprove(cfg.Providers.Gemini.DangerouslySkipPermissions),
	}
	if cfg.Providers.Gemini.Model != "" {
		opts = append(opts, agents.WithGeminiModel(cfg.Providers.Gemini.Model))
	}
	return agents.NewGeminiAgent(opts...)
}
