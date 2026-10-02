package analysis

import (
	"strings"
	"testing"
)

func TestTruncateToDepth(t *testing.T) {
	tests := []struct {
		path  string
		depth int
		want  string
	}{
		{"internal/analysis/silo.go", 2, "internal/analysis"},
		{"internal/analysis/silo.go", 1, "internal"},
		{"cmd/nightshift/commands/silo.go", 2, "cmd/nightshift"},
		{"cmd/nightshift/commands/silo.go", 3, "cmd/nightshift/commands"},
		{"README.md", 2, "."},
		{"pkg/foo.go", 1, "pkg"},
		{"a/b/c/d/e.go", 2, "a/b"},
	}

	for _, tt := range tests {
		got := truncateToDepth(tt.path, tt.depth)
		if got != tt.want {
			t.Errorf("truncateToDepth(%q, %d) = %q, want %q", tt.path, tt.depth, got, tt.want)
		}
	}
}

func TestCalculateSilosEmpty(t *testing.T) {
	entries := CalculateSilos(nil, 5)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for nil input, got %d", len(entries))
	}
}

func TestCalculateSilosSingleOwner(t *testing.T) {
	dirAuthors := map[string][]CommitAuthor{
		"internal/core": {
			{Name: "Alice", Email: "alice@example.com", Commits: 50},
		},
	}

	entries := CalculateSilos(dirAuthors, 1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Directory != "internal/core" {
		t.Errorf("expected directory 'internal/core', got %q", e.Directory)
	}
	if e.RiskLevel != "critical" {
		t.Errorf("single owner should be critical risk, got %q", e.RiskLevel)
	}
	if e.SiloScore < 0.99 {
		t.Errorf("single owner should have silo score ~1.0, got %.2f", e.SiloScore)
	}
	if e.ContributorCount != 1 {
		t.Errorf("expected 1 contributor, got %d", e.ContributorCount)
	}
}

func TestCalculateSilosWellDistributed(t *testing.T) {
	dirAuthors := map[string][]CommitAuthor{
		"pkg/shared": {
			{Name: "A", Email: "a@example.com", Commits: 20},
			{Name: "B", Email: "b@example.com", Commits: 20},
			{Name: "C", Email: "c@example.com", Commits: 20},
			{Name: "D", Email: "d@example.com", Commits: 20},
			{Name: "E", Email: "e@example.com", Commits: 20},
		},
	}

	entries := CalculateSilos(dirAuthors, 5)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.RiskLevel != "low" {
		t.Errorf("well-distributed should be low risk, got %q", e.RiskLevel)
	}
	if e.SiloScore > 0.3 {
		t.Errorf("well-distributed should have low silo score, got %.2f", e.SiloScore)
	}
}

func TestCalculateSilosMinCommitsFilter(t *testing.T) {
	dirAuthors := map[string][]CommitAuthor{
		"internal/core": {
			{Name: "Alice", Email: "alice@example.com", Commits: 50},
		},
		"docs": {
			{Name: "Bob", Email: "bob@example.com", Commits: 2},
		},
	}

	entries := CalculateSilos(dirAuthors, 5)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (docs should be filtered), got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].Directory != "internal/core" {
		t.Errorf("expected internal/core entry, got %q", entries[0].Directory)
	}
}

func TestCalculateSilosSortedBySeverity(t *testing.T) {
	dirAuthors := map[string][]CommitAuthor{
		"pkg/shared": {
			{Name: "A", Email: "a@example.com", Commits: 10},
			{Name: "B", Email: "b@example.com", Commits: 10},
			{Name: "C", Email: "c@example.com", Commits: 10},
			{Name: "D", Email: "d@example.com", Commits: 10},
			{Name: "E", Email: "e@example.com", Commits: 10},
		},
		"internal/core": {
			{Name: "Alice", Email: "alice@example.com", Commits: 50},
		},
	}

	entries := CalculateSilos(dirAuthors, 5)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Worst silo should come first
	if entries[0].Directory != "internal/core" {
		t.Errorf("expected single-owner dir first, got %q", entries[0].Directory)
	}
	if entries[0].SiloScore <= entries[1].SiloScore {
		t.Errorf("entries should be sorted by silo score descending: %.2f <= %.2f",
			entries[0].SiloScore, entries[1].SiloScore)
	}
}

func TestAssessSiloRisk(t *testing.T) {
	tests := []struct {
		name    string
		authors []CommitAuthor
		total   int
		want    string
	}{
		{
			name:    "empty",
			authors: nil,
			total:   0,
			want:    "unknown",
		},
		{
			name:    "single contributor",
			authors: []CommitAuthor{{Commits: 100}},
			total:   100,
			want:    "critical",
		},
		{
			name: "dominant >80%",
			authors: []CommitAuthor{
				{Commits: 85},
				{Commits: 15},
			},
			total: 100,
			want:  "critical",
		},
		{
			name: "dominant >60%",
			authors: []CommitAuthor{
				{Commits: 70},
				{Commits: 20},
				{Commits: 10},
			},
			total: 100,
			want:  "high",
		},
		{
			name: "two contributors only",
			authors: []CommitAuthor{
				{Commits: 50},
				{Commits: 50},
			},
			total: 100,
			want:  "high",
		},
		{
			name: "moderate concentration",
			authors: []CommitAuthor{
				{Commits: 45},
				{Commits: 25},
				{Commits: 20},
				{Commits: 10},
			},
			total: 100,
			want:  "medium",
		},
		{
			name: "well distributed",
			authors: []CommitAuthor{
				{Commits: 25},
				{Commits: 25},
				{Commits: 25},
				{Commits: 15},
				{Commits: 10},
			},
			total: 100,
			want:  "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assessSiloRisk(tt.authors, tt.total)
			if got != tt.want {
				t.Errorf("assessSiloRisk() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalculateSiloScore(t *testing.T) {
	// Single contributor should be 1.0
	score1 := calculateSiloScore([]CommitAuthor{{Commits: 100}}, 100)
	if score1 < 0.99 {
		t.Errorf("single contributor should have score ~1.0, got %.2f", score1)
	}

	// Even distribution among many should be low
	many := []CommitAuthor{
		{Commits: 20}, {Commits: 20}, {Commits: 20}, {Commits: 20}, {Commits: 20},
	}
	scoreLow := calculateSiloScore(many, 100)
	if scoreLow > 0.3 {
		t.Errorf("even distribution should have low score, got %.2f", scoreLow)
	}

	// Empty should be 0
	score0 := calculateSiloScore(nil, 0)
	if score0 != 0 {
		t.Errorf("empty should be 0, got %.2f", score0)
	}
}

func TestSiloReportGenerate(t *testing.T) {
	entries := []SiloEntry{
		{
			Directory:        "internal/core",
			TopContributors:  []CommitAuthor{{Name: "Alice", Email: "alice@example.com", Commits: 50}},
			TotalCommits:     50,
			ContributorCount: 1,
			SiloScore:        1.0,
			RiskLevel:        "critical",
		},
	}

	gen := NewSiloReportGenerator()
	report := gen.Generate("/repo", 2, entries)

	if report.TotalDirs != 1 {
		t.Errorf("expected 1 total dir, got %d", report.TotalDirs)
	}
	if report.CriticalCount != 1 {
		t.Errorf("expected 1 critical, got %d", report.CriticalCount)
	}
	if len(report.Recommendations) == 0 {
		t.Errorf("expected recommendations")
	}
	if report.Timestamp.IsZero() {
		t.Errorf("timestamp should not be zero")
	}
}

func TestSiloReportRenderMarkdown(t *testing.T) {
	entries := []SiloEntry{
		{
			Directory:        "internal/core",
			TopContributors:  []CommitAuthor{{Name: "Alice", Email: "alice@example.com", Commits: 50}},
			TotalCommits:     50,
			ContributorCount: 1,
			SiloScore:        1.0,
			RiskLevel:        "critical",
		},
		{
			Directory: "pkg/shared",
			TopContributors: []CommitAuthor{
				{Name: "A", Email: "a@example.com", Commits: 10},
				{Name: "B", Email: "b@example.com", Commits: 10},
			},
			TotalCommits:     50,
			ContributorCount: 5,
			SiloScore:        0.2,
			RiskLevel:        "low",
		},
	}

	gen := NewSiloReportGenerator()
	report := gen.Generate("/repo", 2, entries)
	markdown := gen.RenderMarkdown(report)

	if !strings.Contains(markdown, "Knowledge Silo Analysis") {
		t.Errorf("markdown should contain title")
	}
	if !strings.Contains(markdown, "Directory Silo Risk") {
		t.Errorf("markdown should contain silo risk table")
	}
	if !strings.Contains(markdown, "internal/core") {
		t.Errorf("markdown should contain directory name")
	}
	if !strings.Contains(markdown, "Alice") {
		t.Errorf("markdown should contain contributor name")
	}
	if !strings.Contains(markdown, "Recommendations") {
		t.Errorf("markdown should contain recommendations")
	}
	if !strings.Contains(markdown, "CRITICAL") {
		t.Errorf("markdown should contain critical recommendation")
	}
}

func TestSiloReportNoSilos(t *testing.T) {
	entries := []SiloEntry{
		{
			Directory: "pkg/shared",
			TopContributors: []CommitAuthor{
				{Name: "A", Email: "a@example.com", Commits: 20},
			},
			TotalCommits:     100,
			ContributorCount: 5,
			SiloScore:        0.2,
			RiskLevel:        "low",
		},
	}

	gen := NewSiloReportGenerator()
	report := gen.Generate("/repo", 2, entries)

	foundGood := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "GOOD") {
			foundGood = true
			break
		}
	}
	if !foundGood {
		t.Errorf("expected GOOD recommendation when no critical/high silos")
	}
}

func TestSiloEntryTopContributorsCapped(t *testing.T) {
	dirAuthors := map[string][]CommitAuthor{
		"internal/big": {
			{Name: "A", Email: "a@example.com", Commits: 50},
			{Name: "B", Email: "b@example.com", Commits: 30},
			{Name: "C", Email: "c@example.com", Commits: 20},
			{Name: "D", Email: "d@example.com", Commits: 10},
			{Name: "E", Email: "e@example.com", Commits: 5},
		},
	}

	entries := CalculateSilos(dirAuthors, 1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// TopContributors should be capped at 3
	if len(entries[0].TopContributors) != 3 {
		t.Errorf("expected 3 top contributors, got %d", len(entries[0].TopContributors))
	}
}
