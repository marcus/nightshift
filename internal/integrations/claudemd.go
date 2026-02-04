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

// ClaudeMDReader reads claude.md files for project context.
type ClaudeMDReader struct {
	enabled bool
}

// NewClaudeMDReader creates a reader based on config.
func NewClaudeMDReader(cfg *config.Config) *ClaudeMDReader {
	return &ClaudeMDReader{
		enabled: cfg.Integrations.ClaudeMD,
	}
}

func (r *ClaudeMDReader) Name() string {
	return "claude.md"
}

func (r *ClaudeMDReader) Enabled() bool {
	return r.enabled
}

// Read looks for claude.md in project root and extracts context.
func (r *ClaudeMDReader) Read(ctx context.Context, projectPath string) (*Result, error) {
	// Try multiple possible locations
	candidates := []string{
		filepath.Join(projectPath, "claude.md"),
		filepath.Join(projectPath, "CLAUDE.md"),
		filepath.Join(projectPath, ".claude.md"),
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
		return nil, nil // No claude.md found, not an error
	}

	result := &Result{
		Metadata: map[string]any{
			"source_file": foundPath,
		},
	}

	// Parse the content
	parsed := parseClaudeMD(string(content))
	result.Context = parsed.context
	result.Hints = parsed.hints

	return result, nil
}

type claudeMDParsed struct {
	context string
	hints   []Hint
}

// parseClaudeMD extracts structured data from claude.md content.
func parseClaudeMD(content string) claudeMDParsed {
	result := claudeMDParsed{
		context: content, // Full content as context
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentSection string
	sectionPatterns := map[string]HintType{
		"convention":  HintConvention,
		"coding":      HintConvention,
		"style":       HintConvention,
		"task":        HintTaskSuggestion,
		"todo":        HintTaskSuggestion,
		"constraint":  HintConstraint,
		"restriction": HintConstraint,
		"safety":      HintConstraint,
	}

	// Patterns for extracting hints
	headerRE := regexp.MustCompile(`^#+\s*(.+)`)
	bulletRE := regexp.MustCompile(`^[-*]\s+(.+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Detect section headers
		if match := headerRE.FindStringSubmatch(line); match != nil {
			header := strings.ToLower(match[1])
			for pattern, hintType := range sectionPatterns {
				if strings.Contains(header, pattern) {
					currentSection = pattern
					_ = hintType // Used below
					break
				}
			}
			continue
		}

		// Extract bullet points as hints when in relevant sections
		if currentSection != "" {
			if match := bulletRE.FindStringSubmatch(line); match != nil {
				hintType := HintContext
				for pattern, ht := range sectionPatterns {
					if strings.Contains(currentSection, pattern) {
						hintType = ht
						break
					}
				}
				result.hints = append(result.hints, Hint{
					Type:    hintType,
					Content: match[1],
					Source:  "claude.md",
				})
			}
		}
	}

	return result
}
