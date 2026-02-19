package commands

import (
	"fmt"
	"os/exec"
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
	case "copilot":
		a := newCopilotAgentFromConfig(cfg)
		if !a.Available() {
			return nil, fmt.Errorf("copilot CLI not found in PATH (install via 'gh' or standalone)")
		}
		return a, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: claude, codex, copilot)", provider)
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

func newCopilotAgentFromConfig(cfg *config.Config) *agents.CopilotAgent {
	if cfg == nil {
		return agents.NewCopilotAgent()
	}

	// Auto-detect: prefer standalone copilot, fallback to gh
	binaryPath := "gh"
	if _, err := exec.LookPath("copilot"); err == nil {
		binaryPath = "copilot"
	}

	// Copilot uses DangerouslySkipPermissions for --allow-all-tools flag
	// Note: The agent already uses --no-ask-user for autonomous mode
	opts := []agents.CopilotOption{
		agents.WithCopilotBinaryPath(binaryPath),
	}
	if cfg.Providers.Copilot.DangerouslySkipPermissions {
		// When enabled, this should pass --allow-all-tools
		// Currently handled via config, future: add agent option
	}
	return agents.NewCopilotAgent(opts...)
}
