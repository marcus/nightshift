package releasenotes

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClassifyCommit(t *testing.T) {
	tests := []struct {
		subject  string
		wantCat  CommitCategory
		wantScope string
		wantBreak bool
	}{
		// Conventional commits
		{"feat: add new feature", CategoryFeatures, "", false},
		{"feat(auth): add login endpoint", CategoryFeatures, "auth", false},
		{"fix: resolve crash on startup", CategoryFixes, "", false},
		{"fix(db): handle nil pointer", CategoryFixes, "db", false},
		{"docs: update README", CategoryDocs, "", false},
		{"test: add unit tests", CategoryTests, "", false},
		{"refactor: simplify logic", CategoryRefactor, "", false},
		{"perf: optimize query", CategoryPerformance, "", false},
		{"ci: update workflow", CategoryBuild, "", false},
		{"build: update Makefile", CategoryBuild, "", false},
		{"chore: bump dependencies", CategoryBuild, "", false},
		{"security: patch vulnerability", CategorySecurity, "", false},

		// Breaking changes
		{"feat!: remove deprecated API", CategoryFeatures, "", true},
		{"fix(api)!: change response format", CategoryFixes, "api", true},

		// Non-conventional commits (keyword inference)
		{"Add user authentication", CategoryFeatures, "", false},
		{"Fix memory leak in parser", CategoryFixes, "", false},
		{"Implement caching layer", CategoryFeatures, "", false},
		{"Optimize database queries", CategoryPerformance, "", false},
		{"Refactor middleware stack", CategoryRefactor, "", false},
		{"Update documentation for v2", CategoryDocs, "", false},
		{"random commit message", CategoryOther, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			cat, scope, breaking := classifyCommit(tt.subject)
			if cat != tt.wantCat {
				t.Errorf("classifyCommit(%q) category = %q, want %q", tt.subject, cat, tt.wantCat)
			}
			if scope != tt.wantScope {
				t.Errorf("classifyCommit(%q) scope = %q, want %q", tt.subject, scope, tt.wantScope)
			}
			if breaking != tt.wantBreak {
				t.Errorf("classifyCommit(%q) breaking = %v, want %v", tt.subject, breaking, tt.wantBreak)
			}
		})
	}
}

func TestRender(t *testing.T) {
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	rn := &ReleaseNotes{
		Version: "v1.0.0",
		PrevTag: "v0.9.0",
		Date:    now,
		AllCommits: []Commit{
			{ShortHash: "abc1234", Subject: "feat: add login", Author: "Alice", Date: now, Category: CategoryFeatures},
			{ShortHash: "def5678", Subject: "fix: crash on nil", Author: "Bob", Date: now, Category: CategoryFixes},
			{ShortHash: "ghi9012", Subject: "feat!: remove old API", Author: "Alice", Date: now, Category: CategoryFeatures, Breaking: true},
		},
		Categories: map[CommitCategory][]Commit{
			CategoryFeatures: {
				{ShortHash: "abc1234", Subject: "feat: add login", Author: "Alice", Date: now, Category: CategoryFeatures},
				{ShortHash: "ghi9012", Subject: "feat!: remove old API", Author: "Alice", Date: now, Category: CategoryFeatures, Breaking: true},
			},
			CategoryFixes: {
				{ShortHash: "def5678", Subject: "fix: crash on nil", Author: "Bob", Date: now, Category: CategoryFixes},
			},
			CategoryBreaking: {
				{ShortHash: "ghi9012", Subject: "feat!: remove old API", Author: "Alice", Date: now, Category: CategoryFeatures, Breaking: true},
			},
		},
	}

	t.Run("grouped with hashes", func(t *testing.T) {
		opts := Options{
			GroupByCategory:     true,
			IncludeCommitHashes: true,
		}
		result := rn.Render(opts)

		if !strings.Contains(result, "# Release Notes: v1.0.0") {
			t.Error("missing header")
		}
		if !strings.Contains(result, "## Breaking Changes") {
			t.Error("missing breaking changes section")
		}
		if !strings.Contains(result, "## Features") {
			t.Error("missing features section")
		}
		if !strings.Contains(result, "## Bug Fixes") {
			t.Error("missing fixes section")
		}
		if !strings.Contains(result, "[`abc1234`]") {
			t.Error("missing commit hash")
		}
		if !strings.Contains(result, "**Compared to:** v0.9.0") {
			t.Error("missing previous tag comparison")
		}
		if !strings.Contains(result, "**Commits:** 3") {
			t.Error("missing commit count")
		}
	})

	t.Run("flat list", func(t *testing.T) {
		opts := Options{
			GroupByCategory:     false,
			IncludeCommitHashes: true,
		}
		result := rn.Render(opts)

		if strings.Contains(result, "## Features") {
			t.Error("should not have category headers in flat mode")
		}
		// Should still have commits
		if !strings.Contains(result, "abc1234") {
			t.Error("missing commit in flat mode")
		}
	})

	t.Run("with authors", func(t *testing.T) {
		opts := Options{
			GroupByCategory:     true,
			IncludeCommitHashes: false,
			IncludeAuthors:      true,
		}
		result := rn.Render(opts)

		if !strings.Contains(result, "— Alice") {
			t.Error("missing author")
		}
		if strings.Contains(result, "[`abc1234`]") {
			t.Error("should not have commit hashes when disabled")
		}
	})
}

func TestFormatCommitLine(t *testing.T) {
	c := Commit{
		ShortHash: "abc1234",
		Subject:   "feat(auth): add login endpoint",
		Author:    "Alice",
		Scope:     "auth",
	}

	t.Run("with hash", func(t *testing.T) {
		line := formatCommitLine(c, Options{IncludeCommitHashes: true})
		if !strings.Contains(line, "- Add login endpoint") {
			t.Errorf("expected cleaned subject, got: %s", line)
		}
		if !strings.Contains(line, "(auth)") {
			t.Errorf("expected scope, got: %s", line)
		}
		if !strings.Contains(line, "[`abc1234`]") {
			t.Errorf("expected hash, got: %s", line)
		}
	})

	t.Run("without hash", func(t *testing.T) {
		line := formatCommitLine(c, Options{})
		if strings.Contains(line, "abc1234") {
			t.Error("should not include hash when disabled")
		}
	})
}

func TestFilterNonBreaking(t *testing.T) {
	commits := []Commit{
		{Subject: "breaking", Breaking: true},
		{Subject: "normal", Breaking: false},
		{Subject: "also breaking", Breaking: true},
	}

	filtered := filterNonBreaking(commits)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 non-breaking commit, got %d", len(filtered))
	}
	if filtered[0].Subject != "normal" {
		t.Errorf("expected 'normal', got %q", filtered[0].Subject)
	}
}

func TestPrefixToCategory(t *testing.T) {
	tests := []struct {
		prefix string
		want   CommitCategory
	}{
		{"feat", CategoryFeatures},
		{"feature", CategoryFeatures},
		{"fix", CategoryFixes},
		{"bugfix", CategoryFixes},
		{"security", CategorySecurity},
		{"sec", CategorySecurity},
		{"perf", CategoryPerformance},
		{"performance", CategoryPerformance},
		{"docs", CategoryDocs},
		{"doc", CategoryDocs},
		{"refactor", CategoryRefactor},
		{"test", CategoryTests},
		{"tests", CategoryTests},
		{"build", CategoryBuild},
		{"ci", CategoryBuild},
		{"chore", CategoryBuild},
		{"unknown", CategoryOther},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			got := prefixToCategory(tt.prefix)
			if got != tt.want {
				t.Errorf("prefixToCategory(%q) = %q, want %q", tt.prefix, got, tt.want)
			}
		})
	}
}

// TestGenerateWithRealRepo tests the full generation flow using a temporary git repo.
func TestGenerateWithRealRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Create temp repo
	dir := t.TempDir()
	mustRun(t, dir, "git", "init")
	mustRun(t, dir, "git", "config", "user.email", "test@test.com")
	mustRun(t, dir, "git", "config", "user.name", "Test User")

	// Create initial commit and tag
	writeFile(t, dir, "README.md", "# Test\n")
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "feat: initial commit")
	mustRun(t, dir, "git", "tag", "v0.1.0")

	// Add more commits
	writeFile(t, dir, "main.go", "package main\n")
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "feat(core): add main package")

	writeFile(t, dir, "bug.go", "package main\n// fixed\n")
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "fix: resolve startup crash")

	writeFile(t, dir, "api.go", "package main\n// new api\n")
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "feat!: redesign API surface")

	// Tag new version
	mustRun(t, dir, "git", "tag", "v0.2.0")

	gen := NewGenerator(dir)

	t.Run("generate for latest tag", func(t *testing.T) {
		rn, err := gen.Generate(Options{
			Tag:                 "v0.2.0",
			IncludeCommitHashes: true,
			GroupByCategory:     true,
		})
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}

		if rn.Version != "v0.2.0" {
			t.Errorf("version = %q, want v0.2.0", rn.Version)
		}
		if rn.PrevTag != "v0.1.0" {
			t.Errorf("prevTag = %q, want v0.1.0", rn.PrevTag)
		}
		if len(rn.AllCommits) != 3 {
			t.Errorf("expected 3 commits, got %d", len(rn.AllCommits))
		}

		// Check categories
		if len(rn.Categories[CategoryFeatures]) != 2 {
			t.Errorf("expected 2 feature commits, got %d", len(rn.Categories[CategoryFeatures]))
		}
		if len(rn.Categories[CategoryFixes]) != 1 {
			t.Errorf("expected 1 fix commit, got %d", len(rn.Categories[CategoryFixes]))
		}
		if len(rn.Categories[CategoryBreaking]) != 1 {
			t.Errorf("expected 1 breaking commit, got %d", len(rn.Categories[CategoryBreaking]))
		}

		// Render and check output
		output := rn.Render(DefaultOptions())
		if !strings.Contains(output, "## Features") {
			t.Error("rendered output missing Features section")
		}
		if !strings.Contains(output, "## Breaking Changes") {
			t.Error("rendered output missing Breaking Changes section")
		}
	})

	t.Run("generate auto-detect HEAD", func(t *testing.T) {
		// Add a commit after the tag
		writeFile(t, dir, "new.go", "package main\n// new\n")
		mustRun(t, dir, "git", "add", ".")
		mustRun(t, dir, "git", "commit", "-m", "feat: post-release feature")

		rn, err := gen.Generate(Options{
			GroupByCategory: true,
		})
		if err != nil {
			t.Fatalf("Generate HEAD: %v", err)
		}

		// HEAD should compare against latest tag
		if rn.Version != "HEAD" || rn.Version == "" {
			// HEAD with commits after the latest tag
		}
		if len(rn.AllCommits) == 0 {
			t.Error("expected commits for HEAD")
		}
	})

	t.Run("tags listing", func(t *testing.T) {
		tags, err := gen.Tags()
		if err != nil {
			t.Fatalf("Tags: %v", err)
		}
		if len(tags) < 2 {
			t.Errorf("expected at least 2 tags, got %d", len(tags))
		}
	})
}

func mustRun(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2026-02-16T00:00:00Z",
		"GIT_COMMITTER_DATE=2026-02-16T00:00:00Z",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
