package analysis

import (
	"strings"
	"testing"
	"time"
)

func TestDirectoryAtDepth(t *testing.T) {
	tests := []struct {
		path  string
		depth int
		want  string
	}{
		{"cmd/nightshift/main.go", 1, "cmd"},
		{"cmd/nightshift/main.go", 2, "cmd/nightshift"},
		{"cmd/nightshift/main.go", 3, "cmd/nightshift"},
		{"internal/analysis/ownership.go", 1, "internal"},
		{"internal/analysis/ownership.go", 2, "internal/analysis"},
		{"main.go", 1, ""}, // root-level file, no directory
		{"main.go", 2, ""}, // root-level file, no directory
		{"a/b.go", 1, "a"},
		{"a/b/c/d.go", 2, "a/b"},
	}

	for _, tt := range tests {
		got := directoryAtDepth(tt.path, tt.depth)
		if got != tt.want {
			t.Errorf("directoryAtDepth(%q, %d) = %q, want %q", tt.path, tt.depth, got, tt.want)
		}
	}
}

func TestSuggestBoundaries(t *testing.T) {
	dirs := []DirectoryOwnership{
		{
			Dir:           "cmd",
			DominantOwner: "Alice",
			Confidence:    0.8,
			TotalCommits:  40,
			Authors: map[string]*AuthorStats{
				"alice@example.com": {Name: "Alice", Email: "alice@example.com", Commits: 32},
				"bob@example.com":   {Name: "Bob", Email: "bob@example.com", Commits: 8},
			},
		},
		{
			Dir:           "internal/analysis",
			DominantOwner: "Alice",
			Confidence:    0.6,
			TotalCommits:  30,
			Authors: map[string]*AuthorStats{
				"alice@example.com": {Name: "Alice", Email: "alice@example.com", Commits: 18},
				"bob@example.com":   {Name: "Bob", Email: "bob@example.com", Commits: 12},
			},
		},
		{
			Dir:           "internal/db",
			DominantOwner: "Bob",
			Confidence:    0.7,
			TotalCommits:  20,
			Authors: map[string]*AuthorStats{
				"alice@example.com": {Name: "Alice", Email: "alice@example.com", Commits: 6},
				"bob@example.com":   {Name: "Bob", Email: "bob@example.com", Commits: 14},
			},
		},
	}

	boundaries := SuggestBoundaries(dirs)

	if len(boundaries) != 2 {
		t.Fatalf("expected 2 boundaries, got %d", len(boundaries))
	}

	// Alice should be first (more total commits)
	if boundaries[0].Owner != "Alice" {
		t.Errorf("expected first boundary owner to be Alice, got %s", boundaries[0].Owner)
	}
	if len(boundaries[0].Directories) != 2 {
		t.Errorf("expected Alice to own 2 directories, got %d", len(boundaries[0].Directories))
	}

	if boundaries[1].Owner != "Bob" {
		t.Errorf("expected second boundary owner to be Bob, got %s", boundaries[1].Owner)
	}
	if len(boundaries[1].Directories) != 1 {
		t.Errorf("expected Bob to own 1 directory, got %d", len(boundaries[1].Directories))
	}
}

func TestSuggestBoundariesEmpty(t *testing.T) {
	boundaries := SuggestBoundaries(nil)
	if len(boundaries) != 0 {
		t.Errorf("expected 0 boundaries for nil input, got %d", len(boundaries))
	}
}

func TestSuggestBoundariesSingleOwner(t *testing.T) {
	dirs := []DirectoryOwnership{
		{
			Dir:           "src",
			DominantOwner: "Alice",
			Confidence:    1.0,
			TotalCommits:  100,
			Authors: map[string]*AuthorStats{
				"alice@example.com": {Name: "Alice", Email: "alice@example.com", Commits: 100},
			},
		},
	}

	boundaries := SuggestBoundaries(dirs)
	if len(boundaries) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(boundaries))
	}
	if boundaries[0].TotalCommit != 100 {
		t.Errorf("expected total commits 100, got %d", boundaries[0].TotalCommit)
	}
}

func TestFindUnclearOwnership(t *testing.T) {
	dirs := []DirectoryOwnership{
		{Dir: "clear", Confidence: 0.8},
		{Dir: "unclear1", Confidence: 0.3},
		{Dir: "borderline", Confidence: 0.5},
		{Dir: "unclear2", Confidence: 0.1},
	}

	unclear := FindUnclearOwnership(dirs, 0.5)
	if len(unclear) != 2 {
		t.Fatalf("expected 2 unclear directories, got %d", len(unclear))
	}
	if unclear[0].Dir != "unclear1" {
		t.Errorf("expected first unclear dir to be 'unclear1', got %s", unclear[0].Dir)
	}
	if unclear[1].Dir != "unclear2" {
		t.Errorf("expected second unclear dir to be 'unclear2', got %s", unclear[1].Dir)
	}
}

func TestFindUnclearOwnershipAllClear(t *testing.T) {
	dirs := []DirectoryOwnership{
		{Dir: "a", Confidence: 0.9},
		{Dir: "b", Confidence: 0.7},
	}

	unclear := FindUnclearOwnership(dirs, 0.5)
	if len(unclear) != 0 {
		t.Errorf("expected 0 unclear directories, got %d", len(unclear))
	}
}

func TestGenerateCODEOWNERS(t *testing.T) {
	boundaries := []OwnershipBoundary{
		{
			Owner:       "Alice",
			Email:       "alice@example.com",
			Directories: []string{"cmd", "internal/analysis"},
		},
		{
			Owner:       "Bob",
			Email:       "bob@example.com",
			Directories: []string{"internal/db"},
		},
	}

	output := GenerateCODEOWNERS(boundaries)

	if !strings.Contains(output, "CODEOWNERS") {
		t.Error("output should contain CODEOWNERS header")
	}
	if !strings.Contains(output, "/cmd/ alice@example.com") {
		t.Error("output should contain /cmd/ alice@example.com")
	}
	if !strings.Contains(output, "/internal/analysis/ alice@example.com") {
		t.Error("output should contain /internal/analysis/ alice@example.com")
	}
	if !strings.Contains(output, "/internal/db/ bob@example.com") {
		t.Error("output should contain /internal/db/ bob@example.com")
	}
}

func TestGenerateCODEOWNERSEmpty(t *testing.T) {
	output := GenerateCODEOWNERS(nil)
	if !strings.Contains(output, "CODEOWNERS") {
		t.Error("even empty output should contain header")
	}
	// Should not contain any path entries
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "/") {
			t.Errorf("empty boundaries should not produce path entries, got: %s", line)
		}
	}
}

func TestRenderOwnershipMarkdown(t *testing.T) {
	report := &OwnershipReport{
		Timestamp: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
		RepoPath:  "/test/repo",
		Directories: []DirectoryOwnership{
			{
				Dir:           "cmd",
				DominantOwner: "Alice",
				Confidence:    0.8,
				TotalCommits:  40,
				Authors: map[string]*AuthorStats{
					"alice@example.com": {Name: "Alice", Email: "alice@example.com", Commits: 32},
					"bob@example.com":   {Name: "Bob", Email: "bob@example.com", Commits: 8},
				},
			},
		},
		Boundaries: []OwnershipBoundary{
			{Owner: "Alice", Email: "alice@example.com", Directories: []string{"cmd"}, TotalCommit: 40},
		},
		Cohesion: []CohesionPair{
			{DirA: "cmd", DirB: "internal", CoChanges: 15, Strength: 0.6},
		},
		Unclear: []DirectoryOwnership{
			{Dir: "docs", DominantOwner: "Carol", Confidence: 0.3, TotalCommits: 10,
				Authors: map[string]*AuthorStats{
					"carol@example.com": {Name: "Carol", Email: "carol@example.com", Commits: 3},
				}},
		},
	}

	md := RenderOwnershipMarkdown(report)

	sections := []string{
		"# Ownership Boundary Analysis",
		"## Summary",
		"## Ownership Boundaries",
		"## Directory Ownership Details",
		"## Unclear Ownership",
		"## Co-Change Cohesion",
	}
	for _, s := range sections {
		if !strings.Contains(md, s) {
			t.Errorf("markdown should contain section %q", s)
		}
	}

	if !strings.Contains(md, "Alice") {
		t.Error("markdown should mention Alice")
	}
	if !strings.Contains(md, "cmd") {
		t.Error("markdown should mention cmd directory")
	}
	if !strings.Contains(md, "2026-03-25") {
		t.Error("markdown should contain timestamp")
	}
}

func TestRenderOwnershipMarkdownEmpty(t *testing.T) {
	report := &OwnershipReport{
		Timestamp: time.Now(),
		RepoPath:  "/test/repo",
	}

	md := RenderOwnershipMarkdown(report)
	if !strings.Contains(md, "# Ownership Boundary Analysis") {
		t.Error("empty report should still have title")
	}
	if !strings.Contains(md, "Directories analyzed**: 0") {
		t.Error("empty report should show 0 directories")
	}
}
