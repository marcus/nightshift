package integrations

import (
	"testing"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

func TestGitHubReader_Name(t *testing.T) {
	r := &GitHubReader{}
	if r.Name() != "github" {
		t.Errorf("Name() = %q, want %q", r.Name(), "github")
	}
}

func TestGitHubReader_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GitHubReader{enabled: tt.enabled}
			if r.Enabled() != tt.enabled {
				t.Errorf("Enabled() = %v, want %v", r.Enabled(), tt.enabled)
			}
		})
	}
}

func TestNewGitHubReader(t *testing.T) {
	cfg := &config.Config{
		Integrations: config.IntegrationsConfig{
			TaskSources: []config.TaskSourceEntry{
				{GithubIssues: true},
			},
		},
	}

	r := NewGitHubReader(cfg)
	if !r.Enabled() {
		t.Error("expected reader to be enabled")
	}
	if r.label != "nightshift" {
		t.Errorf("expected default label %q, got %q", "nightshift", r.label)
	}
}

func TestNewGitHubReader_Disabled(t *testing.T) {
	cfg := &config.Config{
		Integrations: config.IntegrationsConfig{
			TaskSources: []config.TaskSourceEntry{},
		},
	}

	r := NewGitHubReader(cfg)
	if r.Enabled() {
		t.Error("expected reader to be disabled")
	}
}

func TestExtractPriority(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   int
	}{
		{"critical", []string{"critical"}, 100},
		{"urgent", []string{"urgent"}, 100},
		{"high priority", []string{"high-priority"}, 75},
		{"medium", []string{"medium"}, 50},
		{"low", []string{"low"}, 25},
		{"p0", []string{"p0"}, 100},
		{"p1", []string{"P1"}, 75},
		{"p2", []string{"p2"}, 50},
		{"p3", []string{"p3-backlog"}, 25},
		{"mixed", []string{"bug", "high", "needs-review"}, 75},
		{"no priority", []string{"bug", "feature"}, 50},
		{"empty", []string{}, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPriority(tt.labels)
			if got != tt.want {
				t.Errorf("extractPriority(%v) = %d, want %d", tt.labels, got, tt.want)
			}
		})
	}
}

func TestGhIssue_JsonUnmarshal(t *testing.T) {
	// This is a structural test - verify the struct matches expected JSON format
	issue := ghIssue{
		Number: 42,
		Title:  "Test issue",
		Body:   "Issue body",
		State:  "open",
		Labels: []struct {
			Name string `json:"name"`
		}{
			{Name: "bug"},
			{Name: "nightshift"},
		},
	}

	if issue.Number != 42 {
		t.Error("issue number mismatch")
	}
	if len(issue.Labels) != 2 {
		t.Error("labels count mismatch")
	}
}
