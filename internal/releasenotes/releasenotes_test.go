package releasenotes

import (
	"strings"
	"testing"
)

func TestParseLine_ConventionalFeat(t *testing.T) {
	t.Parallel()
	line := "abc123" + commitDelimiter + "feat(auth): add OAuth2 support (#42)" + commitDelimiter + "alice"
	c := parseLine(line)

	if c.Hash != "abc123" {
		t.Errorf("Hash = %q, want %q", c.Hash, "abc123")
	}
	if c.Type != TypeFeature {
		t.Errorf("Type = %v, want TypeFeature", c.Type)
	}
	if c.RawType != "feat" {
		t.Errorf("RawType = %q, want %q", c.RawType, "feat")
	}
	if c.Scope != "auth" {
		t.Errorf("Scope = %q, want %q", c.Scope, "auth")
	}
	if c.Description != "add OAuth2 support" {
		t.Errorf("Description = %q, want %q", c.Description, "add OAuth2 support")
	}
	if c.PRNumber != "42" {
		t.Errorf("PRNumber = %q, want %q", c.PRNumber, "42")
	}
	if c.Author != "alice" {
		t.Errorf("Author = %q, want %q", c.Author, "alice")
	}
	if c.Breaking {
		t.Error("Breaking = true, want false")
	}
}

func TestParseLine_ConventionalFix(t *testing.T) {
	t.Parallel()
	line := "def456" + commitDelimiter + "fix: correct nil pointer in scheduler" + commitDelimiter + "bob"
	c := parseLine(line)

	if c.Type != TypeFix {
		t.Errorf("Type = %v, want TypeFix", c.Type)
	}
	if c.Scope != "" {
		t.Errorf("Scope = %q, want empty", c.Scope)
	}
	if c.Description != "correct nil pointer in scheduler" {
		t.Errorf("Description = %q, want %q", c.Description, "correct nil pointer in scheduler")
	}
	if c.PRNumber != "" {
		t.Errorf("PRNumber = %q, want empty", c.PRNumber)
	}
}

func TestParseLine_BreakingChange(t *testing.T) {
	t.Parallel()
	line := "aaa111" + commitDelimiter + "feat!: remove deprecated API (#10)" + commitDelimiter + "carol"
	c := parseLine(line)

	if c.Type != TypeFeature {
		t.Errorf("Type = %v, want TypeFeature", c.Type)
	}
	if !c.Breaking {
		t.Error("Breaking = false, want true")
	}
	if c.PRNumber != "10" {
		t.Errorf("PRNumber = %q, want %q", c.PRNumber, "10")
	}
}

func TestParseLine_BreakingWithScope(t *testing.T) {
	t.Parallel()
	line := "bbb222" + commitDelimiter + "feat(config)!: change default format" + commitDelimiter + "dave"
	c := parseLine(line)

	if c.Type != TypeFeature {
		t.Errorf("Type = %v, want TypeFeature", c.Type)
	}
	if c.Scope != "config" {
		t.Errorf("Scope = %q, want %q", c.Scope, "config")
	}
	if !c.Breaking {
		t.Error("Breaking = false, want true")
	}
}

func TestParseLine_NonConventional(t *testing.T) {
	t.Parallel()
	line := "ghi789" + commitDelimiter + "Update README with examples" + commitDelimiter + "eve"
	c := parseLine(line)

	if c.Type != TypeOther {
		t.Errorf("Type = %v, want TypeOther", c.Type)
	}
	if c.Description != "Update README with examples" {
		t.Errorf("Description = %q, want %q", c.Description, "Update README with examples")
	}
	if c.RawType != "" {
		t.Errorf("RawType = %q, want empty", c.RawType)
	}
}

func TestParseLine_OtherType(t *testing.T) {
	t.Parallel()
	line := "ccc333" + commitDelimiter + "chore: update deps (#55)" + commitDelimiter + "frank"
	c := parseLine(line)

	if c.Type != TypeOther {
		t.Errorf("Type = %v, want TypeOther", c.Type)
	}
	if c.RawType != "chore" {
		t.Errorf("RawType = %q, want %q", c.RawType, "chore")
	}
	if c.PRNumber != "55" {
		t.Errorf("PRNumber = %q, want %q", c.PRNumber, "55")
	}
}

func TestParseLine_IssueRefs(t *testing.T) {
	t.Parallel()
	line := "ddd444" + commitDelimiter + "fix: handle edge case fixes #7 and closes #8" + commitDelimiter + "grace"
	c := parseLine(line)

	if len(c.IssueRefs) != 2 {
		t.Fatalf("IssueRefs len = %d, want 2", len(c.IssueRefs))
	}
	if c.IssueRefs[0] != "7" || c.IssueRefs[1] != "8" {
		t.Errorf("IssueRefs = %v, want [7 8]", c.IssueRefs)
	}
}

func TestParseCommits_SkipsEmpty(t *testing.T) {
	t.Parallel()
	lines := []string{
		"abc" + commitDelimiter + "feat: first" + commitDelimiter + "a",
		"",
		"def" + commitDelimiter + "fix: second" + commitDelimiter + "b",
	}
	commits := ParseCommits(lines)
	if len(commits) != 2 {
		t.Errorf("len = %d, want 2", len(commits))
	}
}

func TestGroupCommits_Order(t *testing.T) {
	t.Parallel()
	commits := []Commit{
		{Type: TypeOther, Description: "chore"},
		{Type: TypeFix, Description: "bugfix"},
		{Type: TypeFeature, Description: "new thing"},
		{Type: TypeFix, Description: "another fix"},
	}

	groups := GroupCommits(commits)

	if len(groups) != 3 {
		t.Fatalf("groups len = %d, want 3", len(groups))
	}
	if groups[0].Title != "Features" {
		t.Errorf("groups[0].Title = %q, want %q", groups[0].Title, "Features")
	}
	if groups[1].Title != "Bug Fixes" {
		t.Errorf("groups[1].Title = %q, want %q", groups[1].Title, "Bug Fixes")
	}
	if groups[2].Title != "Other" {
		t.Errorf("groups[2].Title = %q, want %q", groups[2].Title, "Other")
	}
	if len(groups[1].Commits) != 2 {
		t.Errorf("Bug Fixes commits = %d, want 2", len(groups[1].Commits))
	}
}

func TestGroupCommits_OmitsEmpty(t *testing.T) {
	t.Parallel()
	commits := []Commit{
		{Type: TypeFeature, Description: "feat only"},
	}
	groups := GroupCommits(commits)
	if len(groups) != 1 {
		t.Fatalf("groups len = %d, want 1", len(groups))
	}
	if groups[0].Title != "Features" {
		t.Errorf("Title = %q, want %q", groups[0].Title, "Features")
	}
}

func TestRender_Unreleased(t *testing.T) {
	t.Parallel()
	groups := []Group{
		{
			Title: "Features",
			Commits: []Commit{
				{Scope: "auth", Description: "add login flow", PRNumber: "42"},
			},
		},
		{
			Title: "Bug Fixes",
			Commits: []Commit{
				{Description: "fix crash on startup"},
			},
		},
	}

	out := Render("", groups)

	if !strings.HasPrefix(out, "## [Unreleased]\n") {
		t.Errorf("header missing, got: %s", out)
	}
	if !strings.Contains(out, "### Features\n") {
		t.Error("missing Features section")
	}
	if !strings.Contains(out, "- **auth** — add login flow (#42)\n") {
		t.Errorf("missing formatted feature commit, got:\n%s", out)
	}
	if !strings.Contains(out, "### Bug Fixes\n") {
		t.Error("missing Bug Fixes section")
	}
	if !strings.Contains(out, "- **fix crash on startup**\n") {
		t.Errorf("missing formatted fix commit, got:\n%s", out)
	}
}

func TestRender_WithVersion(t *testing.T) {
	t.Parallel()
	groups := []Group{
		{
			Title:   "Other",
			Commits: []Commit{{Description: "update deps"}},
		},
	}

	out := Render("v1.0.0", groups)

	if !strings.Contains(out, "## [v1.0.0] - ") {
		t.Errorf("version header missing, got: %s", out)
	}
}

func TestRender_BreakingMarker(t *testing.T) {
	t.Parallel()
	groups := []Group{
		{
			Title: "Features",
			Commits: []Commit{
				{Description: "new API", Breaking: true, PRNumber: "5"},
			},
		},
	}

	out := Render("v2.0.0", groups)

	if !strings.Contains(out, "**BREAKING**") {
		t.Errorf("missing BREAKING marker, got:\n%s", out)
	}
}

func TestRenderCommit_NoScope(t *testing.T) {
	t.Parallel()
	c := Commit{Description: "simple change", PRNumber: "99"}
	got := renderCommit(c)
	want := "- **simple change** (#99)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderCommit_WithScope(t *testing.T) {
	t.Parallel()
	c := Commit{Scope: "cli", Description: "add verbose flag", PRNumber: "10"}
	got := renderCommit(c)
	want := "- **cli** — add verbose flag (#10)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestClassifyType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  CommitType
	}{
		{"feat", TypeFeature},
		{"fix", TypeFix},
		{"chore", TypeOther},
		{"docs", TypeOther},
		{"refactor", TypeOther},
		{"ci", TypeOther},
		{"test", TypeOther},
	}
	for _, tt := range tests {
		if got := classifyType(tt.input); got != tt.want {
			t.Errorf("classifyType(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
