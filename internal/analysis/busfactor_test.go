package analysis

import (
	"strings"
	"testing"
	"time"
)

func TestReportGeneratorGenerate(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "Alice", Email: "alice@example.com", Commits: 60},
		{Name: "Bob", Email: "bob@example.com", Commits: 40},
	}
	metrics := CalculateMetrics(authors)

	gen := NewReportGenerator()
	report := gen.Generate("test-component", authors, metrics)

	if report.Component != "test-component" {
		t.Errorf("expected component 'test-component', got %s", report.Component)
	}
	if report.Metrics == nil {
		t.Errorf("metrics should not be nil")
	}
	if len(report.Contributors) != 2 {
		t.Errorf("expected 2 contributors, got %d", len(report.Contributors))
	}
	if len(report.Recommendations) == 0 {
		t.Errorf("expected recommendations to be generated")
	}
}

func TestRenderMarkdown(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "Alice", Email: "alice@example.com", Commits: 60},
		{Name: "Bob", Email: "bob@example.com", Commits: 40},
	}
	metrics := CalculateMetrics(authors)

	gen := NewReportGenerator()
	report := gen.Generate("test-component", authors, metrics)
	markdown := gen.RenderMarkdown(report)

	// Check for key sections
	if !strings.Contains(markdown, "# Bus Factor Analysis") {
		t.Errorf("markdown should contain title")
	}
	if !strings.Contains(markdown, "## Ownership Metrics") {
		t.Errorf("markdown should contain metrics section")
	}
	if !strings.Contains(markdown, "## Top Contributors") {
		t.Errorf("markdown should contain contributors section")
	}
	if !strings.Contains(markdown, "## Recommendations") {
		t.Errorf("markdown should contain recommendations section")
	}
	if !strings.Contains(markdown, "Alice") {
		t.Errorf("markdown should contain contributor name")
	}
}

func TestRecommendationsCritical(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "Alice", Email: "alice@example.com", Commits: 100},
	}
	metrics := CalculateMetrics(authors)

	gen := NewReportGenerator()
	report := gen.Generate("critical-component", authors, metrics)

	if len(report.Recommendations) == 0 {
		t.Errorf("expected recommendations for critical risk")
	}

	found := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "CRITICAL") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected CRITICAL recommendation")
	}
}

func TestRecommendationsLow(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "A", Email: "a@example.com", Commits: 20},
		{Name: "B", Email: "b@example.com", Commits: 20},
		{Name: "C", Email: "c@example.com", Commits: 20},
		{Name: "D", Email: "d@example.com", Commits: 20},
		{Name: "E", Email: "e@example.com", Commits: 20},
		{Name: "F", Email: "f@example.com", Commits: 20},
	}
	metrics := CalculateMetrics(authors)

	gen := NewReportGenerator()
	report := gen.Generate("healthy-component", authors, metrics)

	if metrics.RiskLevel != "low" {
		t.Errorf("expected low risk for 6 equal contributors, got %s", metrics.RiskLevel)
	}

	found := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "GOOD") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected GOOD recommendation for low risk")
	}
}

func TestProgressBar(t *testing.T) {
	gen := NewReportGenerator()

	tests := []struct {
		filled   int
		hasBlock bool
	}{
		{0, false},
		{5, true},
		{10, true},
		{20, true},
		{50, true},
	}

	for _, tt := range tests {
		bar := gen.progressBar(tt.filled)
		if len(bar) == 0 {
			t.Errorf("progress bar should not be empty")
		}
		if tt.hasBlock && tt.filled > 0 && !strings.Contains(bar, "█") {
			t.Errorf("expected progress bar to contain filled block for %d filled", tt.filled)
		}
		if strings.Contains(bar, "[") && strings.Contains(bar, "]") {
			// Valid format
		} else {
			t.Errorf("progress bar should be wrapped in brackets")
		}
	}
}

func TestReportTimestamp(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "Alice", Email: "alice@example.com", Commits: 100},
	}
	metrics := CalculateMetrics(authors)

	gen := NewReportGenerator()
	report := gen.Generate("test", authors, metrics)

	if report.Timestamp.IsZero() {
		t.Errorf("timestamp should not be zero")
	}
	if report.ReportedAt == "" {
		t.Errorf("reported_at should not be empty")
	}

	// Check timestamp is recent (within last minute)
	if time.Since(report.Timestamp) > time.Minute {
		t.Errorf("timestamp should be recent")
	}
}

func TestMetricsString(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "Alice", Email: "alice@example.com", Commits: 60},
		{Name: "Bob", Email: "bob@example.com", Commits: 40},
	}
	metrics := CalculateMetrics(authors)

	str := metrics.String()
	if str == "No contributors found" {
		t.Errorf("string representation should not be empty for valid metrics")
	}
	if !strings.Contains(str, "Risk:") {
		t.Errorf("string should contain risk level")
	}
	if !strings.Contains(str, "Bus Factor") {
		t.Errorf("string should contain bus factor")
	}
}
