package analysis

import (
	"testing"
	"time"
)

func TestParseAuthorsEmpty(t *testing.T) {
	parser := NewGitParser("/nonexistent")
	_, err := parser.ParseAuthors(ParseOptions{})
	if err == nil {
		t.Errorf("expected error for nonexistent repo")
	}
}

func TestCalculateMetricsEmpty(t *testing.T) {
	metrics := CalculateMetrics([]CommitAuthor{})
	if metrics.TotalContributors != 0 {
		t.Errorf("expected 0 contributors, got %d", metrics.TotalContributors)
	}
	if metrics.RiskLevel != "unknown" {
		t.Errorf("expected unknown risk level, got %s", metrics.RiskLevel)
	}
}

func TestCalculateMetricsSingleAuthor(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "Alice", Email: "alice@example.com", Commits: 100},
	}

	metrics := CalculateMetrics(authors)

	if metrics.TotalContributors != 1 {
		t.Errorf("expected 1 contributor, got %d", metrics.TotalContributors)
	}
	if metrics.RiskLevel != "critical" {
		t.Errorf("expected critical risk, got %s", metrics.RiskLevel)
	}
	if metrics.BusFactor != 1 {
		t.Errorf("expected bus factor 1, got %d", metrics.BusFactor)
	}
	if metrics.Top1Percent != 100.0 {
		t.Errorf("expected 100%% top 1, got %.1f%%", metrics.Top1Percent)
	}
}

func TestCalculateMetricsMultipleAuthors(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "Alice", Email: "alice@example.com", Commits: 50},
		{Name: "Bob", Email: "bob@example.com", Commits: 30},
		{Name: "Charlie", Email: "charlie@example.com", Commits: 20},
	}

	metrics := CalculateMetrics(authors)

	if metrics.TotalContributors != 3 {
		t.Errorf("expected 3 contributors, got %d", metrics.TotalContributors)
	}
	// Alice = 50 (50%), reaches exactly 50% threshold
	if metrics.BusFactor != 1 {
		t.Errorf("expected bus factor 1, got %d", metrics.BusFactor)
	}

	// Alice = 50%, Bob = 30%, Charlie = 20%
	// Top 1 = 50%, Top 2 = 80%, Top 3 = 100%
	if metrics.Top1Percent < 49 || metrics.Top1Percent > 51 {
		t.Errorf("expected ~50%% top 1, got %.1f%%", metrics.Top1Percent)
	}
	if metrics.Top3Percent < 99 {
		t.Errorf("expected ~100%% top 3, got %.1f%%", metrics.Top3Percent)
	}
}

func TestCalculateMetricsEvenDistribution(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "A", Email: "a@example.com", Commits: 25},
		{Name: "B", Email: "b@example.com", Commits: 25},
		{Name: "C", Email: "c@example.com", Commits: 25},
		{Name: "D", Email: "d@example.com", Commits: 25},
	}

	metrics := CalculateMetrics(authors)

	// Even distribution with 4 contributors is medium (not low) due to contributor count threshold
	if metrics.RiskLevel != "medium" {
		t.Errorf("expected medium risk for 4-person even distribution, got %s", metrics.RiskLevel)
	}
	// Each person = 25%, first 2 = 50%
	if metrics.BusFactor != 2 {
		t.Errorf("expected bus factor 2, got %d", metrics.BusFactor)
	}
}

func TestHerfindahlCalculation(t *testing.T) {
	// Test single author (should be 1 when normalized)
	authors1 := []CommitAuthor{
		{Name: "A", Email: "a@example.com", Commits: 100},
	}
	hhi1 := calculateHerfindahl(authors1, 100)
	if hhi1 < 0.99 {
		t.Errorf("single author should have HHI ~1.0, got %.3f", hhi1)
	}

	// Test equal distribution (should be 0)
	authors4 := []CommitAuthor{
		{Name: "A", Email: "a@example.com", Commits: 25},
		{Name: "B", Email: "b@example.com", Commits: 25},
		{Name: "C", Email: "c@example.com", Commits: 25},
		{Name: "D", Email: "d@example.com", Commits: 25},
	}
	hhi4 := calculateHerfindahl(authors4, 100)
	if hhi4 > 0.01 {
		t.Errorf("equal distribution should have HHI ~0.0, got %.3f", hhi4)
	}
}

func TestGiniCalculation(t *testing.T) {
	// Equal distribution should have Gini ~0
	authors := []CommitAuthor{
		{Name: "A", Email: "a@example.com", Commits: 25},
		{Name: "B", Email: "b@example.com", Commits: 25},
		{Name: "C", Email: "c@example.com", Commits: 25},
		{Name: "D", Email: "d@example.com", Commits: 25},
	}
	gini := calculateGini(authors, 100)
	if gini > 0.01 {
		t.Errorf("equal distribution should have Gini ~0.0, got %.3f", gini)
	}
}

func TestRepositoryExistsFalse(t *testing.T) {
	exists := RepositoryExists("/nonexistent/path")
	if exists {
		t.Errorf("nonexistent path should not exist")
	}
}

func TestRiskLevelScore(t *testing.T) {
	tests := []struct {
		level string
		want  int
	}{
		{"critical", 75},
		{"high", 50},
		{"medium", 25},
		{"low", 0},
		{"unknown", -1},
	}

	for _, tt := range tests {
		got := RiskLevelScore(tt.level)
		if got != tt.want {
			t.Errorf("RiskLevelScore(%q) = %d, want %d", tt.level, got, tt.want)
		}
	}
}

func TestParseOptionsWithFilters(t *testing.T) {
	since := time.Now().AddDate(0, -1, 0)
	until := time.Now()

	opts := ParseOptions{
		Since:    since,
		Until:    until,
		FilePath: "pkg/foo.go",
	}

	if opts.Since.IsZero() {
		t.Errorf("since should not be zero")
	}
	if opts.Until.IsZero() {
		t.Errorf("until should not be zero")
	}
	if opts.FilePath != "pkg/foo.go" {
		t.Errorf("expected file path 'pkg/foo.go', got %s", opts.FilePath)
	}
}
