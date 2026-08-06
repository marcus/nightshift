package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListComponentPaths(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{
		"cmd/alpha",
		"cmd/beta",
		"internal/one",
		"internal/two",
		"internal/.hidden",
	} {
		if err := os.MkdirAll(filepath.Join(dir, p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}
	// File at a root level — should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "cmd", "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := ListComponentPaths(dir, []string{"cmd", "internal", "missing"})
	if err != nil {
		t.Fatalf("ListComponentPaths: %v", err)
	}

	want := []string{"cmd/alpha", "cmd/beta", "internal/one", "internal/two"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("index %d: want %q got %q", i, w, got[i])
		}
	}
}

func TestAnalyzeComponentsSortAndSilo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_DATE=2024-01-01T00:00:00Z",
			"GIT_COMMITTER_DATE=2024-01-01T00:00:00Z",
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(name, email, path, content string) {
		t.Helper()
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		runGit("add", path)
		cmd := exec.Command("git",
			"-c", "user.name="+name,
			"-c", "user.email="+email,
			"commit", "-m", "change "+path)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_DATE=2024-01-01T00:00:00Z",
			"GIT_COMMITTER_DATE=2024-01-01T00:00:00Z",
			"GIT_AUTHOR_NAME="+name,
			"GIT_AUTHOR_EMAIL="+email,
			"GIT_COMMITTER_NAME="+name,
			"GIT_COMMITTER_EMAIL="+email,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("commit: %v\n%s", err, out)
		}
	}

	runGit("init", "-q", "-b", "main")

	// internal/solo: single contributor → critical, knowledge silo.
	commit("Solo", "solo@example.com", "internal/solo/file1.go", "package solo\n")
	commit("Solo", "solo@example.com", "internal/solo/file2.go", "package solo\n")

	// internal/shared: three contributors evenly distributed → bus factor 2.
	commit("Alice", "alice@example.com", "internal/shared/a.go", "package shared\n")
	commit("Bob", "bob@example.com", "internal/shared/b.go", "package shared\n")
	commit("Carol", "carol@example.com", "internal/shared/c.go", "package shared\n")

	results, err := AnalyzeComponents(dir, []string{"internal"}, ParseOptions{})
	if err != nil {
		t.Fatalf("AnalyzeComponents: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 components, got %d: %+v", len(results), results)
	}

	// Riskiest (solo) must come first.
	if results[0].Path != "internal/solo" {
		t.Errorf("expected internal/solo first, got %s", results[0].Path)
	}
	if !results[0].KnowledgeSilo {
		t.Errorf("internal/solo should be flagged as knowledge silo")
	}
	if results[0].Metrics.RiskLevel != "critical" {
		t.Errorf("internal/solo expected critical, got %s", results[0].Metrics.RiskLevel)
	}

	if results[1].Path != "internal/shared" {
		t.Errorf("expected internal/shared second, got %s", results[1].Path)
	}
	if results[1].KnowledgeSilo {
		t.Errorf("internal/shared should not be a silo (bus factor > 1)")
	}
}

func TestRenderMarkdownWithComponentBreakdown(t *testing.T) {
	authors := []CommitAuthor{
		{Name: "Alice", Email: "alice@example.com", Commits: 60},
		{Name: "Bob", Email: "bob@example.com", Commits: 40},
	}
	gen := NewReportGenerator()
	report := gen.Generate("repo", authors, CalculateMetrics(authors))

	soloAuthors := []CommitAuthor{{Name: "Solo", Email: "solo@example.com", Commits: 5}}
	report.ComponentBreakdown = []ComponentMetrics{
		{
			Path:          "internal/solo",
			Authors:       soloAuthors,
			Metrics:       CalculateMetrics(soloAuthors),
			KnowledgeSilo: true,
		},
		{
			Path:          "internal/shared",
			Authors:       authors,
			Metrics:       CalculateMetrics(authors),
			KnowledgeSilo: false,
		},
	}

	md := gen.RenderMarkdown(report)
	if !strings.Contains(md, "## Per-Component Bus Factor") {
		t.Errorf("missing per-component section: %s", md)
	}
	if !strings.Contains(md, "`internal/solo`") {
		t.Errorf("missing internal/solo row")
	}
	if !strings.Contains(md, "Knowledge silos detected: 1") {
		t.Errorf("missing silo summary in: %s", md)
	}

	// Section must be absent when breakdown empty.
	report2 := gen.Generate("repo", authors, CalculateMetrics(authors))
	md2 := gen.RenderMarkdown(report2)
	if strings.Contains(md2, "Per-Component Bus Factor") {
		t.Errorf("per-component section should not appear when breakdown empty")
	}
}
