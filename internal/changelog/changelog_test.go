package changelog

import (
	"strings"
	"testing"
)

func TestParseCommit_ConventionalWithScope(t *testing.T) {
	ci := ParseCommit("abc123", "feat(api): add new endpoint (#42)")
	if ci.Prefix != "feat" {
		t.Errorf("prefix = %q, want %q", ci.Prefix, "feat")
	}
	if ci.Scope != "api" {
		t.Errorf("scope = %q, want %q", ci.Scope, "api")
	}
	if ci.Title != "add new endpoint (#42)" {
		t.Errorf("title = %q, want %q", ci.Title, "add new endpoint (#42)")
	}
	if ci.PR != 42 {
		t.Errorf("pr = %d, want %d", ci.PR, 42)
	}
	if ci.Hash != "abc123" {
		t.Errorf("hash = %q, want %q", ci.Hash, "abc123")
	}
}

func TestParseCommit_ConventionalWithoutScope(t *testing.T) {
	ci := ParseCommit("def456", "fix: resolve crash on startup")
	if ci.Prefix != "fix" {
		t.Errorf("prefix = %q, want %q", ci.Prefix, "fix")
	}
	if ci.Scope != "" {
		t.Errorf("scope = %q, want empty", ci.Scope)
	}
	if ci.Title != "resolve crash on startup" {
		t.Errorf("title = %q, want %q", ci.Title, "resolve crash on startup")
	}
	if ci.PR != 0 {
		t.Errorf("pr = %d, want 0", ci.PR)
	}
}

func TestParseCommit_NonConventional(t *testing.T) {
	ci := ParseCommit("789abc", "Update README (#10)")
	if ci.Prefix != "" {
		t.Errorf("prefix = %q, want empty", ci.Prefix)
	}
	if ci.Title != "Update README (#10)" {
		t.Errorf("title = %q, want %q", ci.Title, "Update README (#10)")
	}
	if ci.PR != 10 {
		t.Errorf("pr = %d, want 10", ci.PR)
	}
}

func TestClassifyCommit(t *testing.T) {
	tests := []struct {
		prefix string
		want   Category
	}{
		{"feat", CategoryFeatures},
		{"fix", CategoryFixes},
		{"docs", CategoryDocs},
		{"refactor", CategoryOther},
		{"chore", CategoryOther},
		{"ci", CategoryOther},
		{"test", CategoryOther},
		{"", CategoryOther},
		{"unknown", CategoryOther},
	}
	for _, tc := range tests {
		got := ClassifyCommit(tc.prefix)
		if got != tc.want {
			t.Errorf("ClassifyCommit(%q) = %q, want %q", tc.prefix, got, tc.want)
		}
	}
}

func TestGroupCommits(t *testing.T) {
	lines := []string{
		"aaa111 feat: add user profiles (#1)",
		"bbb222 fix: resolve login crash (#2)",
		"ccc333 docs: update API reference",
		"ddd444 chore: update dependencies",
		"eee555 feat(ui): redesign dashboard (#3)",
	}
	groups := GroupCommits(lines)

	// Should have Features first, then Bug Fixes, Documentation, Other
	if len(groups) != 4 {
		t.Fatalf("got %d groups, want 4", len(groups))
	}
	if groups[0].Category != CategoryFeatures {
		t.Errorf("first group = %q, want Features", groups[0].Category)
	}
	if len(groups[0].Commits) != 2 {
		t.Errorf("Features has %d commits, want 2", len(groups[0].Commits))
	}
	if groups[1].Category != CategoryFixes {
		t.Errorf("second group = %q, want Bug Fixes", groups[1].Category)
	}
	if groups[2].Category != CategoryDocs {
		t.Errorf("third group = %q, want Documentation", groups[2].Category)
	}
	if groups[3].Category != CategoryOther {
		t.Errorf("fourth group = %q, want Other", groups[3].Category)
	}
}

func TestGroupCommits_EmptyInput(t *testing.T) {
	groups := GroupCommits(nil)
	if len(groups) != 0 {
		t.Errorf("got %d groups, want 0", len(groups))
	}
}

func TestRenderMarkdown(t *testing.T) {
	groups := []CategoryGroup{
		{
			Category: CategoryFeatures,
			Commits: []CommitInfo{
				{Hash: "aaa", Title: "add user profiles (#1)", PR: 1},
				{Hash: "bbb", Title: "redesign dashboard (#3)", PR: 3},
			},
		},
		{
			Category: CategoryFixes,
			Commits: []CommitInfo{
				{Hash: "ccc", Title: "resolve login crash (#2)", PR: 2},
			},
		},
	}

	out := RenderMarkdown("v1.0.0", groups)

	if !strings.HasPrefix(out, "## [v1.0.0] - ") {
		t.Errorf("missing version header, got: %s", out)
	}
	if !strings.Contains(out, "### Features") {
		t.Error("missing Features section")
	}
	if !strings.Contains(out, "### Bug Fixes") {
		t.Error("missing Bug Fixes section")
	}
	if !strings.Contains(out, "- **add user profiles** (#1)") {
		t.Errorf("missing formatted entry, got:\n%s", out)
	}
	if !strings.Contains(out, "- **resolve login crash** (#2)") {
		t.Errorf("missing fix entry, got:\n%s", out)
	}
}

func TestRenderMarkdown_NoVersion(t *testing.T) {
	groups := []CategoryGroup{
		{Category: CategoryOther, Commits: []CommitInfo{{Title: "update deps"}}},
	}
	out := RenderMarkdown("", groups)
	if strings.Contains(out, "## [") {
		t.Errorf("should not have version header when version is empty, got:\n%s", out)
	}
	if !strings.Contains(out, "### Other") {
		t.Error("missing Other section")
	}
}

func TestRenderPlain(t *testing.T) {
	groups := []CategoryGroup{
		{
			Category: CategoryFeatures,
			Commits: []CommitInfo{
				{Title: "add profiles (#5)", PR: 5},
			},
		},
	}
	out := RenderPlain("v2.0.0", groups)
	if !strings.HasPrefix(out, "v2.0.0\n======") {
		t.Errorf("missing plain header, got:\n%s", out)
	}
	if !strings.Contains(out, "Features\n--------") {
		t.Error("missing plain Features section")
	}
	if !strings.Contains(out, "- add profiles (#5)") {
		t.Errorf("missing plain entry, got:\n%s", out)
	}
}

func TestPRNumberExtraction(t *testing.T) {
	tests := []struct {
		subject string
		wantPR  int
	}{
		{"feat: something (#99)", 99},
		{"fix: no pr here", 0},
		{"chore: multiple (#1) and (#2)", 1}, // takes first match
		{"Update README (#123)", 123},
	}
	for _, tc := range tests {
		ci := ParseCommit("abc", tc.subject)
		if ci.PR != tc.wantPR {
			t.Errorf("ParseCommit(%q).PR = %d, want %d", tc.subject, ci.PR, tc.wantPR)
		}
	}
}
