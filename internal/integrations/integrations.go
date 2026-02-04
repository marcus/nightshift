// Package integrations provides readers for external configuration and task sources.
// Supports claude.md, agents.md, td task management, and GitHub issues.
package integrations

import (
	"context"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

// Reader loads context and tasks from an integration source.
type Reader interface {
	// Name returns the integration identifier.
	Name() string

	// Enabled returns true if this integration is configured.
	Enabled() bool

	// Read loads data from the integration source.
	// Returns nil if the source doesn't exist (not an error).
	Read(ctx context.Context, projectPath string) (*Result, error)
}

// Result holds data loaded from an integration.
type Result struct {
	// Context provides text to include in agent prompts.
	Context string

	// Tasks are work items from task sources (td, GitHub issues).
	Tasks []TaskItem

	// Hints are lightweight suggestions extracted from config files.
	Hints []Hint

	// Metadata holds source-specific data.
	Metadata map[string]any
}

// TaskItem represents a task from an external source.
type TaskItem struct {
	ID          string            // Unique identifier (e.g., "td-abc123", "gh-42")
	Title       string            // Task title
	Description string            // Full description/body
	Priority    int               // Priority (higher = more important)
	Labels      []string          // Tags/labels
	Source      string            // Source name (e.g., "td", "github")
	Metadata    map[string]string // Source-specific fields
}

// Hint is a lightweight suggestion from a config file.
type Hint struct {
	Type    HintType // Type of hint
	Content string   // Hint content
	Source  string   // Where the hint came from
}

// HintType categorizes hints.
type HintType int

const (
	HintTaskSuggestion HintType = iota // Suggested task to run
	HintConvention                     // Coding convention
	HintConstraint                     // Safety constraint
	HintContext                        // Background context
)

func (h HintType) String() string {
	switch h {
	case HintTaskSuggestion:
		return "task_suggestion"
	case HintConvention:
		return "convention"
	case HintConstraint:
		return "constraint"
	case HintContext:
		return "context"
	default:
		return "unknown"
	}
}

// Manager coordinates multiple integration readers.
type Manager struct {
	readers []Reader
	config  *config.Config
}

// NewManager creates a manager with the configured integrations.
func NewManager(cfg *config.Config) *Manager {
	m := &Manager{config: cfg}

	// Add configured readers
	m.readers = append(m.readers, NewClaudeMDReader(cfg))
	m.readers = append(m.readers, NewAgentsMDReader(cfg))
	m.readers = append(m.readers, NewTDReader(cfg))
	m.readers = append(m.readers, NewGitHubReader(cfg))

	return m
}

// ReadAll gathers results from all enabled integrations.
func (m *Manager) ReadAll(ctx context.Context, projectPath string) (*AggregatedResult, error) {
	agg := &AggregatedResult{
		Results: make(map[string]*Result),
	}

	for _, r := range m.readers {
		if !r.Enabled() {
			continue
		}

		result, err := r.Read(ctx, projectPath)
		if err != nil {
			// Log error but continue with other readers
			agg.Errors = append(agg.Errors, ReaderError{
				Reader: r.Name(),
				Err:    err,
			})
			continue
		}

		if result != nil {
			agg.Results[r.Name()] = result
			agg.AllTasks = append(agg.AllTasks, result.Tasks...)
			agg.AllHints = append(agg.AllHints, result.Hints...)
			if result.Context != "" {
				agg.CombinedContext += "\n\n## " + r.Name() + "\n" + result.Context
			}
		}
	}

	return agg, nil
}

// AggregatedResult combines results from all integrations.
type AggregatedResult struct {
	// Results by reader name.
	Results map[string]*Result

	// AllTasks from all sources.
	AllTasks []TaskItem

	// AllHints from all sources.
	AllHints []Hint

	// CombinedContext for agent prompts.
	CombinedContext string

	// Errors encountered (non-fatal).
	Errors []ReaderError
}

// ReaderError records a failed reader.
type ReaderError struct {
	Reader string
	Err    error
}

func (e ReaderError) Error() string {
	return e.Reader + ": " + e.Err.Error()
}
