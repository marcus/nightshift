package reporting

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultRunReportPath(t *testing.T) {
	ts := time.Date(2026, 2, 10, 14, 30, 45, 0, time.UTC)
	path := DefaultRunReportPath(ts)

	if !contains(path, "run-2026-02-10-143045") {
		t.Fatalf("path should contain run-2026-02-10-143045, got %s", path)
	}
	if !contains(path, ".md") {
		t.Fatalf("path should end with .md, got %s", path)
	}
}

func TestRenderRunReportBasic(t *testing.T) {
	results := &RunResults{
		StartTime:      time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:        time.Date(2026, 2, 10, 10, 5, 0, 0, time.UTC),
		StartBudget:    100000,
		UsedBudget:     20000,
		RemainingBudget: 80000,
		Tasks: []TaskResult{
			{
				Project:   "test-project",
				Title:     "Fix bug",
				TaskType:  "bug-fix",
				Status:    "completed",
				TokensUsed: 5000,
				Duration:  time.Minute * 3,
			},
		},
	}

	report, err := RenderRunReport(results, "")
	if err != nil {
		t.Fatalf("RenderRunReport: %v", err)
	}

	if !contains(report, "Nightshift Run") {
		t.Fatalf("report should contain 'Nightshift Run'")
	}
	if !contains(report, "Summary") {
		t.Fatalf("report should contain 'Summary'")
	}
	if !contains(report, "test-project") {
		t.Fatalf("report should contain project name")
	}
	if !contains(report, "Fix bug") {
		t.Fatalf("report should contain task title")
	}
	if !contains(report, "Tasks Completed") {
		t.Fatalf("report should contain 'Tasks Completed'")
	}
}

func TestRenderRunReportWithLogPath(t *testing.T) {
	results := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 10, 5, 0, 0, time.UTC),
		RemainingBudget: 100000,
	}

	logPath := "/var/log/nightshift.log"
	report, err := RenderRunReport(results, logPath)
	if err != nil {
		t.Fatalf("RenderRunReport: %v", err)
	}

	if !contains(report, logPath) {
		t.Fatalf("report should contain log path")
	}
}

func TestRenderRunReportNilResults(t *testing.T) {
	_, err := RenderRunReport(nil, "")
	if err == nil {
		t.Fatal("expected error for nil results")
	}
}

func TestRenderRunReportMultipleStatuses(t *testing.T) {
	results := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 10, 30, 0, 0, time.UTC),
		RemainingBudget: 100000,
		Tasks: []TaskResult{
			{Title: "Task 1", Status: "completed"},
			{Title: "Task 2", Status: "completed"},
			{Title: "Task 3", Status: "failed"},
			{Title: "Task 4", Status: "skipped", SkipReason: "Not enough budget"},
		},
	}

	report, err := RenderRunReport(results, "")
	if err != nil {
		t.Fatalf("RenderRunReport: %v", err)
	}

	if !contains(report, "Tasks Completed") {
		t.Fatalf("report should contain completed tasks section")
	}
	if !contains(report, "Tasks Failed") {
		t.Fatalf("report should contain failed tasks section")
	}
	if !contains(report, "Tasks Skipped") {
		t.Fatalf("report should contain skipped tasks section")
	}
	if !contains(report, "Not enough budget") {
		t.Fatalf("report should contain skip reason")
	}
}

func TestRenderRunReportTaskDetails(t *testing.T) {
	results := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 11, 0, 0, 0, time.UTC),
		RemainingBudget: 100000,
		Tasks: []TaskResult{
			{
				Project:     "project-a",
				Title:       "Refactor API",
				TaskType:    "refactor",
				Status:      "completed",
				TokensUsed:  50000,
				Duration:    time.Hour,
				OutputRef:   "pr-123",
			},
		},
	}

	report, err := RenderRunReport(results, "")
	if err != nil {
		t.Fatalf("RenderRunReport: %v", err)
	}

	if !contains(report, "project-a") {
		t.Fatalf("report should contain project name")
	}
	if !contains(report, "Refactor API") {
		t.Fatalf("report should contain task title")
	}
	if !contains(report, "refactor") {
		t.Fatalf("report should contain task type")
	}
	if !contains(report, "50") && !contains(report, "50K") {
		t.Fatalf("report should contain token count")
	}
	if !contains(report, "pr-123") {
		t.Fatalf("report should contain output reference")
	}
}

func TestSaveRunReport(t *testing.T) {
	tmpdir := t.TempDir()
	reportPath := filepath.Join(tmpdir, "reports", "run.md")

	results := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 10, 30, 0, 0, time.UTC),
		RemainingBudget: 100000,
	}

	err := SaveRunReport(results, reportPath, "")
	if err != nil {
		t.Fatalf("SaveRunReport: %v", err)
	}

	// Verify file exists and contains expected content
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !contains(string(content), "Nightshift Run") {
		t.Fatalf("saved report should contain 'Nightshift Run'")
	}
}

func TestSaveRunReportCreatesDir(t *testing.T) {
	tmpdir := t.TempDir()
	reportPath := filepath.Join(tmpdir, "deep", "nested", "path", "run.md")

	results := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 10, 30, 0, 0, time.UTC),
		RemainingBudget: 100000,
	}

	err := SaveRunReport(results, reportPath, "")
	if err != nil {
		t.Fatalf("SaveRunReport: %v", err)
	}

	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("file should exist at %s", reportPath)
	}
}

func TestDefaultRunResultsPath(t *testing.T) {
	ts := time.Date(2026, 2, 10, 14, 30, 45, 0, time.UTC)
	path := DefaultRunResultsPath(ts)

	if !contains(path, "run-2026-02-10-143045") {
		t.Fatalf("path should contain run-2026-02-10-143045, got %s", path)
	}
	if !contains(path, ".json") {
		t.Fatalf("path should end with .json, got %s", path)
	}
}

func TestDefaultReportsDir(t *testing.T) {
	dir := DefaultReportsDir()

	if !contains(dir, "nightshift") {
		t.Fatalf("reports dir should contain 'nightshift', got %s", dir)
	}
	if !contains(dir, "reports") {
		t.Fatalf("reports dir should contain 'reports', got %s", dir)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
