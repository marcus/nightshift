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
	// The --dangerously-bypass-approvals-and-sandbox flag is required for
	// non-interactive (headless) Codex execution. The agent defaults to true.
	// We only pass the config value when it is explicitly enabled; when the
	// field is false (Go zero value / unconfigured) we preserve the agent's
	// default so that Codex continues to work as a fallback provider even
	// when the user has only configured Claude in their nightshift.yaml.
	//
	// NOTE: Both bool fields on ProviderConfig are zero-valued false, so we
	// cannot distinguish "not configured" from "explicitly false". Erring on
	// the side of enabling the flag for headless operation is the safe choice;
	// users who want Codex to prompt for approvals should disable the provider
	// entirely rather than toggling this flag.
	opts := []agents.CodexOption{}
	if cfg.Providers.Codex.DangerouslyBypassApprovalsAndSandbox {
		opts = append(opts, agents.WithDangerouslyBypassApprovalsAndSandbox(true))
	}
	return agents.NewCodexAgent(opts...)
}

// newCopilotAgentFromConfig creates a CopilotAgent from config. If binaryPath
// is non-empty it overrides auto-detection; otherwise the binary is resolved
// from PATH (preferring standalone "copilot", falling back to "gh").
func newCopilotAgentFromConfig(cfg *config.Config, binaryPath ...string) *agents.CopilotAgent {
	if cfg == nil {
		return agents.NewCopilotAgent()
	}

	var binary string
	if len(binaryPath) > 0 && binaryPath[0] != "" {
		binary = binaryPath[0]
	} else {
		// Auto-detect: prefer standalone copilot, fallback to gh
		binary = "gh"
		if _, err := exec.LookPath("copilot"); err == nil {
			binary = "copilot"
		}
	}

	opts := []agents.CopilotOption{
		agents.WithCopilotBinaryPath(binary),
		agents.WithCopilotDangerouslySkipPermissions(cfg.Providers.Copilot.DangerouslySkipPermissions),
	}
	return agents.NewCopilotAgent(opts...)
}
