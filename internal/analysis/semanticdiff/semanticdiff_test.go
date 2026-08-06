package semanticdiff

import (
	"strings"
	"testing"
)

func TestParseNumstatLines(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantAdd int
		wantDel int
	}{
		{
			name:    "normal file",
			input:   "10\t5\tsrc/main.go\n",
			want:    1,
			wantAdd: 10,
			wantDel: 5,
		},
		{
			name:    "binary file",
			input:   "-\t-\tassets/logo.png\n",
			want:    1,
			wantAdd: 0,
			wantDel: 0,
		},
		{
			name:    "multiple files",
			input:   "10\t5\ta.go\n3\t1\tb.go\n",
			want:    2,
			wantAdd: 13,
			wantDel: 6,
		},
		{
			name:  "empty input",
			input: "",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := parseNumstatLines(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(files) != tt.want {
				t.Fatalf("got %d files, want %d", len(files), tt.want)
			}
			totalAdd, totalDel := 0, 0
			for _, f := range files {
				totalAdd += f.Additions
				totalDel += f.Deletions
			}
			if totalAdd != tt.wantAdd {
				t.Errorf("total additions = %d, want %d", totalAdd, tt.wantAdd)
			}
			if totalDel != tt.wantDel {
				t.Errorf("total deletions = %d, want %d", totalDel, tt.wantDel)
			}
		})
	}
}

func TestParseNumstatBinaryFlag(t *testing.T) {
	files, err := parseNumstatLines("-\t-\timage.png\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if !files[0].IsBinary {
		t.Error("expected IsBinary=true")
	}
}

func TestClassifyFromMessage(t *testing.T) {
	tests := []struct {
		subject string
		body    string
		want    Category
	}{
		{"fix: null pointer in handler", "", CategoryBugfix},
		{"feat: add user profile page", "", CategoryFeature},
		{"docs: update README", "", CategoryDocs},
		{"test: add integration tests", "", CategoryTest},
		{"refactor: extract helper", "", CategoryRefactor},
		{"chore: clean up old scripts", "", CategoryCleanup},
		{"ci: update workflow", "", CategoryConfig},
		{"deps: bump cobra to v1.8", "", CategoryDeps},
		{"Fix a bug in login", "", CategoryBugfix},
		{"Add new endpoint", "", CategoryFeature},
		{"something unrelated", "", CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			got, _ := classifyFromMessage(tt.subject, tt.body)
			if got != tt.want {
				t.Errorf("classifyFromMessage(%q) = %s, want %s", tt.subject, got, tt.want)
			}
		})
	}
}

func TestClassifyFileByPath(t *testing.T) {
	tests := []struct {
		path string
		want Category
	}{
		{"internal/foo/foo_test.go", CategoryTest},
		{"tests/integration.py", CategoryTest},
		{"README.md", CategoryDocs},
		{"docs/guide.rst", CategoryDocs},
		{"go.mod", CategoryDeps},
		{"package.json", CategoryDeps},
		{".github/workflows/ci.yml", CategoryConfig},
		{"Dockerfile", CategoryConfig},
		{"Makefile", CategoryConfig},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			f := FileChange{Path: tt.path}
			got, _ := classifyFile(f, "", "")
			if got != tt.want {
				t.Errorf("classifyFile(%q) = %s, want %s", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	if !isTestFile("cmd/foo_test.go", "foo_test.go") {
		t.Error("expected _test.go to be detected")
	}
	if !isTestFile("tests/main.py", "main.py") {
		t.Error("expected tests/ directory to be detected")
	}
	if !isTestFile("src/app.test.js", "app.test.js") {
		t.Error("expected .test.js to be detected")
	}
	if isTestFile("src/main.go", "main.go") {
		t.Error("expected non-test file to not be detected")
	}
}

func TestIsDocsFile(t *testing.T) {
	if !isDocsFile("readme.md", "readme.md", ".md") {
		t.Error("expected .md to be detected")
	}
	if !isDocsFile("docs/guide.rst", "guide.rst", ".rst") {
		t.Error("expected .rst to be detected")
	}
	if isDocsFile("src/main.go", "main.go", ".go") {
		t.Error("expected .go to not be docs")
	}
}

func TestIsDepsFile(t *testing.T) {
	if !isDepsFile("go.mod") {
		t.Error("expected go.mod to be deps")
	}
	if !isDepsFile("package-lock.json") {
		t.Error("expected package-lock.json to be deps")
	}
	if isDepsFile("main.go") {
		t.Error("expected main.go to not be deps")
	}
}

func TestMajorityCategory(t *testing.T) {
	changes := []ClassifiedChange{
		{Category: CategoryBugfix},
		{Category: CategoryBugfix},
		{Category: CategoryFeature},
	}
	got, _ := majorityCategory(changes)
	if got != CategoryBugfix {
		t.Errorf("got %s, want bugfix", got)
	}
}

func TestGenerateReport(t *testing.T) {
	cs := &ChangeSet{
		BaseRef: "abc1234",
		HeadRef: "def5678",
		Commits: []Commit{
			{Hash: "aaa1111", Subject: "feat: add widget", Author: "alice", Files: []FileChange{
				{Path: "widget.go", Additions: 50, Deletions: 0},
			}},
			{Hash: "bbb2222", Subject: "fix: null check", Author: "bob", Files: []FileChange{
				{Path: "handler.go", Additions: 2, Deletions: 1},
			}},
		},
		AllFiles: []FileChange{
			{Path: "widget.go", Additions: 50, Deletions: 0},
			{Path: "handler.go", Additions: 2, Deletions: 1},
		},
	}

	classified := ClassifyChangeSet(cs)
	report := GenerateReport(cs, classified)

	if report.Stats.TotalCommits != 2 {
		t.Errorf("expected 2 commits, got %d", report.Stats.TotalCommits)
	}
	if report.Stats.TotalFiles != 2 {
		t.Errorf("expected 2 files, got %d", report.Stats.TotalFiles)
	}
	if report.Stats.TotalAdded != 52 {
		t.Errorf("expected 52 additions, got %d", report.Stats.TotalAdded)
	}
	if report.Stats.TotalDeleted != 1 {
		t.Errorf("expected 1 deletion, got %d", report.Stats.TotalDeleted)
	}
	if len(report.Categories) == 0 {
		t.Fatal("expected at least one category group")
	}

	// Verify markdown rendering.
	md := RenderMarkdown(report)
	if !strings.Contains(md, "Semantic Diff") {
		t.Error("markdown should contain header")
	}
	if !strings.Contains(md, "TL;DR") {
		t.Error("markdown should contain TL;DR section")
	}
	if !strings.Contains(md, "widget.go") {
		t.Error("markdown should reference changed files")
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	r := &Report{
		BaseRef: "aaa",
		HeadRef: "bbb",
		Summary: "No changes found between the specified refs.",
	}
	md := RenderMarkdown(r)
	if !strings.Contains(md, "No changes") {
		t.Error("empty report should show no-changes message")
	}
}

func TestDetectHighlights(t *testing.T) {
	cs := &ChangeSet{
		AllFiles: []FileChange{
			{Path: "internal/api/handler.go", Additions: 10, Deletions: 5},
			{Path: "internal/auth/middleware.go", Additions: 3, Deletions: 2},
			{Path: "db/migrations/001_init.sql", Additions: 50, Deletions: 0},
			{Path: "README.md", Additions: 5, Deletions: 2},
		},
	}
	highlights := detectHighlights(cs, nil)
	found := map[string]bool{"api": false, "auth": false, "migration": false}
	for _, h := range highlights {
		if strings.Contains(h, "API surface") {
			found["api"] = true
		}
		if strings.Contains(h, "Security-sensitive") {
			found["auth"] = true
		}
		if strings.Contains(h, "Schema/migration") {
			found["migration"] = true
		}
	}
	for k, v := range found {
		if !v {
			t.Errorf("expected %s highlight to be detected", k)
		}
	}
}

func TestChangeSetTotals(t *testing.T) {
	cs := &ChangeSet{
		AllFiles: []FileChange{
			{Additions: 10, Deletions: 3},
			{Additions: 5, Deletions: 7},
		},
	}
	if cs.TotalAdditions() != 15 {
		t.Errorf("TotalAdditions = %d, want 15", cs.TotalAdditions())
	}
	if cs.TotalDeletions() != 10 {
		t.Errorf("TotalDeletions = %d, want 10", cs.TotalDeletions())
	}
}

func TestShortHash(t *testing.T) {
	if shortHash("abc1234567890") != "abc1234" {
		t.Error("expected 7-char short hash")
	}
	if shortHash("abc") != "abc" {
		t.Error("expected short input returned as-is")
	}
}

func TestCategoryLabel(t *testing.T) {
	if categoryLabel(CategoryFeature) != "New Features" {
		t.Error("unexpected label for feature")
	}
	if categoryLabel(CategoryUnknown) != "Other Changes" {
		t.Error("unexpected label for unknown")
	}
}

func TestBuildSummary(t *testing.T) {
	groups := []CategoryGroup{
		{Category: CategoryFeature, Label: "New Features", Commits: []ClassifiedCommit{{}}},
		{Category: CategoryBugfix, Label: "Bug Fixes", Commits: []ClassifiedCommit{{}, {}}},
	}
	stats := DiffStats{TotalCommits: 3, TotalFiles: 5, TotalAdded: 100, TotalDeleted: 20}
	summary := buildSummary(groups, stats)
	if !strings.Contains(summary, "3 commits") {
		t.Error("summary should mention commit count")
	}
	if !strings.Contains(summary, "+100/-20") {
		t.Error("summary should mention line changes")
	}
}
