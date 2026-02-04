package integrations

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

// AgentsMDReader reads agents.md files for agent behavior configuration.
type AgentsMDReader struct {
	enabled bool
}

// NewAgentsMDReader creates a reader based on config.
func NewAgentsMDReader(cfg *config.Config) *AgentsMDReader {
	return &AgentsMDReader{
		enabled: cfg.Integrations.AgentsMD,
	}
}

func (r *AgentsMDReader) Name() string {
	return "agents.md"
}

func (r *AgentsMDReader) Enabled() bool {
	return r.enabled
}

// Read looks for agents.md or AGENTS.md and extracts behavior preferences.
func (r *AgentsMDReader) Read(ctx context.Context, projectPath string) (*Result, error) {
	// Try multiple possible locations
	candidates := []string{
		filepath.Join(projectPath, "AGENTS.md"),
		filepath.Join(projectPath, "agents.md"),
		filepath.Join(projectPath, ".agents.md"),
	}

	var content []byte
	var foundPath string
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			content = data
			foundPath = path
			break
		}
	}

	if content == nil {
		return nil, nil // No agents.md found, not an error
	}

	result := &Result{
		Metadata: map[string]any{
			"source_file": foundPath,
		},
	}

	// Parse the content
	parsed := parseAgentsMD(string(content))
	result.Context = parsed.context
	result.Hints = parsed.hints
	result.Metadata["tool_restrictions"] = parsed.toolRestrictions
	result.Metadata["allowed_actions"] = parsed.allowedActions
	result.Metadata["forbidden_actions"] = parsed.forbiddenActions

	return result, nil
}

type agentsMDParsed struct {
	context          string
	hints            []Hint
	toolRestrictions []string
	allowedActions   []string
	forbiddenActions []string
}

// parseAgentsMD extracts agent configuration from agents.md content.
func parseAgentsMD(content string) agentsMDParsed {
	result := agentsMDParsed{
		context: content,
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentSection string

	// Section detection patterns
	sectionPatterns := map[string]string{
		"allow":      "allowed",
		"permitted":  "allowed",
		"can":        "allowed",
		"forbidden":  "forbidden",
		"prohibited": "forbidden",
		"never":      "forbidden",
		"don't":      "forbidden",
		"tool":       "tools",
		"restrict":   "tools",
		"safety":     "safety",
		"constraint": "safety",
	}

	headerRE := regexp.MustCompile(`^#+\s*(.+)`)
	bulletRE := regexp.MustCompile(`^[-*]\s+(.+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Detect section headers
		if match := headerRE.FindStringSubmatch(line); match != nil {
			header := strings.ToLower(match[1])
			currentSection = ""
			for pattern, section := range sectionPatterns {
				if strings.Contains(header, pattern) {
					currentSection = section
					break
				}
			}
			continue
		}

		// Extract bullet points based on section
		if match := bulletRE.FindStringSubmatch(line); match != nil {
			item := strings.TrimSpace(match[1])

			switch currentSection {
			case "allowed":
				result.allowedActions = append(result.allowedActions, item)
				result.hints = append(result.hints, Hint{
					Type:    HintContext,
					Content: "Allowed: " + item,
					Source:  "agents.md",
				})
			case "forbidden":
				result.forbiddenActions = append(result.forbiddenActions, item)
				result.hints = append(result.hints, Hint{
					Type:    HintConstraint,
					Content: "Forbidden: " + item,
					Source:  "agents.md",
				})
			case "tools":
				result.toolRestrictions = append(result.toolRestrictions, item)
				result.hints = append(result.hints, Hint{
					Type:    HintConstraint,
					Content: "Tool restriction: " + item,
					Source:  "agents.md",
				})
			case "safety":
				result.hints = append(result.hints, Hint{
					Type:    HintConstraint,
					Content: item,
					Source:  "agents.md",
				})
			default:
				// Generic hint
				result.hints = append(result.hints, Hint{
					Type:    HintContext,
					Content: item,
					Source:  "agents.md",
				})
			}
		}
	}

	return result
}
