package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

func TestNewGenerator(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)
	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.cfg != cfg {
		t.Error("Generator config not set correctly")
	}
}

func TestGenerate(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)

	testDate := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	results := &RunResults{
		Date:            testDate,
		StartBudget:     100000,
		UsedBudget:      45234,
		RemainingBudget: 54766,
		StartTime:       testDate.Add(-6 * time.Hour),
		EndTime:         testDate,
		Tasks: []TaskResult{
			{
				Project:    "/home/user/projects/myproject",
				TaskType:   "lint-fix",
				Title:      "Fixed 12 linting issues",
				Status:     "completed",
				OutputType: "PR",
				OutputRef:  "#123",
				TokensUsed: 30000,
			},
			{
				Project:    "/home/user/projects/myproject",
				TaskType:   "dead-code",
				Title:      "Found 3 dead code blocks",
				Status:     "completed",
				OutputType: "Report",
				OutputRef:  "dead-code.md",
				TokensUsed: 10234,
			},
			{
				Project:    "/home/user/projects/library",
				TaskType:   "test-gap",
				Title:      "Test coverage gaps identified",
				Status:     "completed",
				OutputType: "Analysis",
				OutputRef:  "coverage-report",
				TokensUsed: 5000,
			},
			{
				Project:    "/home/user/projects/migration",
				TaskType:   "migration-rehearsal",
				Title:      "migration-rehearsal",
				Status:     "skipped",
				SkipReason: "estimated: 200k tokens",
			},
		},
	}

	summary, err := gen.Generate(results)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify summary fields
	if summary.BudgetStart != 100000 {
		t.Errorf("BudgetStart = %d, want 100000", summary.BudgetStart)
	}
	if summary.BudgetUsed != 45234 {
		t.Errorf("BudgetUsed = %d, want 45234", summary.BudgetUsed)
	}
	if summary.BudgetRemaining != 54766 {
		t.Errorf("BudgetRemaining = %d, want 54766", summary.BudgetRemaining)
	}
	if len(summary.CompletedTasks) != 3 {
		t.Errorf("CompletedTasks count = %d, want 3", len(summary.CompletedTasks))
	}
	if len(summary.SkippedTasks) != 1 {
		t.Errorf("SkippedTasks count = %d, want 1", len(summary.SkippedTasks))
	}

	// Verify project counts
	if summary.ProjectCounts["/home/user/projects/myproject"] != 2 {
		t.Errorf("myproject task count = %d, want 2", summary.ProjectCounts["/home/user/projects/myproject"])
	}
	if summary.ProjectCounts["/home/user/projects/library"] != 1 {
		t.Errorf("library task count = %d, want 1", summary.ProjectCounts["/home/user/projects/library"])
	}

	// Verify markdown content contains expected sections
	content := summary.Content
	if !strings.Contains(content, "# Nightshift Summary - 2024-01-15") {
		t.Error("Content missing header")
	}
	if !strings.Contains(content, "## Budget") {
		t.Error("Content missing Budget section")
	}
	if !strings.Contains(content, "100,000 tokens") {
		t.Error("Content missing formatted start budget")
	}
	if !strings.Contains(content, "45%") {
		t.Error("Content missing used percentage")
	}
	if !strings.Contains(content, "## Projects Processed") {
		t.Error("Content missing Projects section")
	}
	if !strings.Contains(content, "**myproject** (2 tasks)") {
		t.Error("Content missing myproject task count")
	}
	if !strings.Contains(content, "## Tasks Completed") {
		t.Error("Content missing Tasks Completed section")
	}
	if !strings.Contains(content, "[PR #123]") {
		t.Error("Content missing PR reference")
	}
	if !strings.Contains(content, "## Tasks Skipped") {
		t.Error("Content missing Tasks Skipped section")
	}
	if !strings.Contains(content, "## What's Next?") {
		t.Error("Content missing What's Next section")
	}
}

func TestGenerateNilResults(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)

	_, err := gen.Generate(nil)
	if err == nil {
		t.Error("Generate should fail with nil results")
	}
}

func TestGenerateEmptyResults(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)

	results := &RunResults{
		Date:            time.Now(),
		StartBudget:     50000,
		UsedBudget:      0,
		RemainingBudget: 50000,
		Tasks:           []TaskResult{},
	}

	summary, err := gen.Generate(results)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(summary.CompletedTasks) != 0 {
		t.Error("Expected no completed tasks")
	}
	if len(summary.ProjectCounts) != 0 {
		t.Error("Expected no projects")
	}

	// Should still have budget section
	if !strings.Contains(summary.Content, "## Budget") {
		t.Error("Content should have Budget section even with no tasks")
	}
}

func TestSave(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)

	summary := &Summary{
		Date:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Content: "# Test Summary\n\nThis is a test.",
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "nightshift-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "summaries", "test-summary.md")

	err = gen.Save(summary, path)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists and has correct content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(content) != summary.Content {
		t.Errorf("Saved content = %q, want %q", string(content), summary.Content)
	}
}

func TestSaveNilSummary(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)

	err := gen.Save(nil, "/tmp/test.md")
	if err == nil {
		t.Error("Save should fail with nil summary")
	}
}

func TestDefaultSummaryPath(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	path := DefaultSummaryPath(date)

	if !strings.Contains(path, "nightshift") {
		t.Error("Path should contain 'nightshift'")
	}
	if !strings.Contains(path, "summaries") {
		t.Error("Path should contain 'summaries'")
	}
	if !strings.Contains(path, "summary-2024-01-15.md") {
		t.Errorf("Path should contain date-formatted filename, got %s", path)
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{10000, "10,000"},
		{100000, "100,000"},
		{1000000, "1000,000"},
	}

	for _, tt := range tests {
		got := formatTokens(tt.input)
		if got != tt.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m 30s"},
		{5 * time.Minute, "5m 0s"},
		{65 * time.Minute, "1h 5m"},
		{2 * time.Hour, "2h 0m"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSendNotificationsDisabled(t *testing.T) {
	cfg := &config.Config{
		Reporting: config.ReportingConfig{
			MorningSummary: false,
		},
	}
	gen := NewGenerator(cfg)

	summary := &Summary{
		Date:    time.Now(),
		Content: "Test",
	}

	// Should return nil when notifications are disabled
	err := gen.SendNotifications(summary)
	if err != nil {
		t.Errorf("SendNotifications should succeed when disabled, got: %v", err)
	}
}

func TestSendNotificationsNilSummary(t *testing.T) {
	cfg := &config.Config{
		Reporting: config.ReportingConfig{
			MorningSummary: true,
		},
	}
	gen := NewGenerator(cfg)

	err := gen.SendNotifications(nil)
	if err == nil {
		t.Error("SendNotifications should fail with nil summary")
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~/test", filepath.Join(home, "test")},
		{"~/deep/path/here", filepath.Join(home, "deep/path/here")},
	}

	for _, tt := range tests {
		got := expandPath(tt.input)
		if got != tt.want {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateWhatsNext(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)

	summary := &Summary{
		CompletedTasks: []TaskResult{
			{
				Project:    "/home/user/myproject",
				Title:      "Fixed linting",
				OutputType: "PR",
				OutputRef:  "#123",
			},
			{
				Project:    "/home/user/library",
				Title:      "Dead code report",
				TaskType:   "dead-code",
				OutputType: "Report",
				OutputRef:  "report.md",
			},
		},
		SkippedTasks: []TaskResult{
			{
				Title:      "Migration test",
				SkipReason: "insufficient budget",
			},
		},
	}

	items := gen.generateWhatsNext(summary)

	// Should have items for PR and Report
	if len(items) < 2 {
		t.Errorf("Expected at least 2 what's next items, got %d", len(items))
	}

	hasReviewPR := false
	hasReport := false
	hasBudgetSuggestion := false

	for _, item := range items {
		if strings.Contains(item, "Review #123") {
			hasReviewPR = true
		}
		if strings.Contains(item, "dead-code") {
			hasReport = true
		}
		if strings.Contains(item, "budget") {
			hasBudgetSuggestion = true
		}
	}

	if !hasReviewPR {
		t.Error("Missing PR review suggestion")
	}
	if !hasReport {
		t.Error("Missing report review suggestion")
	}
	if !hasBudgetSuggestion {
		t.Error("Missing budget increase suggestion for skipped task")
	}
}

func TestFormatSlackSummary(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)

	summary := &Summary{
		BudgetStart: 100000,
		BudgetUsed:  45000,
		CompletedTasks: []TaskResult{
			{Title: "Task 1"},
			{Title: "Task 2"},
		},
		ProjectCounts: map[string]int{
			"project1": 1,
			"project2": 1,
		},
		SkippedTasks: []TaskResult{
			{Title: "Skipped task"},
		},
	}

	result := gen.formatSlackSummary(summary)

	if !strings.Contains(result, "Budget:") {
		t.Error("Slack summary missing budget")
	}
	if !strings.Contains(result, "45%") {
		t.Error("Slack summary missing percentage")
	}
	if !strings.Contains(result, "Tasks Completed:") {
		t.Error("Slack summary missing completed tasks")
	}
	if !strings.Contains(result, "Projects:") {
		t.Error("Slack summary missing projects")
	}
	if !strings.Contains(result, "Skipped") {
		t.Error("Slack summary missing skipped info")
	}
}

func TestGenerateWithFailedTasks(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(cfg)

	results := &RunResults{
		Date:            time.Now(),
		StartBudget:     50000,
		UsedBudget:      10000,
		RemainingBudget: 40000,
		Tasks: []TaskResult{
			{
				Project:    "/home/user/project",
				Title:      "Lint fix",
				Status:     "completed",
				TokensUsed: 5000,
			},
			{
				Project:    "/home/user/project",
				Title:      "Test run",
				Status:     "failed",
				SkipReason: "agent timeout",
				TokensUsed: 5000,
			},
		},
	}

	summary, err := gen.Generate(results)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(summary.FailedTasks) != 1 {
		t.Errorf("FailedTasks count = %d, want 1", len(summary.FailedTasks))
	}
	if len(summary.CompletedTasks) != 1 {
		t.Errorf("CompletedTasks count = %d, want 1", len(summary.CompletedTasks))
	}

	if !strings.Contains(summary.Content, "## Tasks Failed") {
		t.Error("Content missing Tasks Failed section")
	}
}
