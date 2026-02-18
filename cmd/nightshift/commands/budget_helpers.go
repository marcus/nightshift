package commands

import (
	"fmt"
	"strings"

	"github.com/marcus/nightshift/internal/config"
)

func resolveProviderList(cfg *config.Config, filter string) ([]string, error) {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter != "" {
		switch filter {
		case "claude":
			if !cfg.Providers.Claude.Enabled {
				return nil, fmt.Errorf("claude provider not enabled")
			}
		case "codex":
			if !cfg.Providers.Codex.Enabled {
				return nil, fmt.Errorf("codex provider not enabled")
			}
		case "gemini":
			if !cfg.Providers.Gemini.Enabled {
				return nil, fmt.Errorf("gemini provider not enabled")
			}
		default:
			return nil, fmt.Errorf("unknown provider: %s (valid: claude, codex, gemini)", filter)
		}
		return []string{filter}, nil
	}

	providerList := []string{}
	if cfg.Providers.Claude.Enabled {
		providerList = append(providerList, "claude")
	}
	if cfg.Providers.Codex.Enabled {
		providerList = append(providerList, "codex")
	}
	if cfg.Providers.Gemini.Enabled {
		providerList = append(providerList, "gemini")
	}

	return providerList, nil
}
