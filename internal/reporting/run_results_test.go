package reporting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveRunResults(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "results.json")

	results := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 10, 30, 0, 0, time.UTC),
		StartBudget:     100000,
		UsedBudget:      30000,
		RemainingBudget: 70000,
		Tasks: []TaskResult{
			{
				Project:    "test-project",
				Title:      "Test task",
				TaskType:   "analysis",
				Status:     "completed",
				TokensUsed: 15000,
				Duration:   time.Minute * 15,
			},
		},
	}

	err := SaveRunResults(results, path)
	if err != nil {
		t.Fatalf("SaveRunResults: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist at %s: %v", path, err)
	}

	// Verify content can be read back
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var loaded RunResults
	if err := json.Unmarshal(content, &loaded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if loaded.StartBudget != 100000 {
		t.Fatalf("StartBudget = %d, want 100000", loaded.StartBudget)
	}
	if len(loaded.Tasks) != 1 {
		t.Fatalf("Tasks count = %d, want 1", len(loaded.Tasks))
	}
}

func TestSaveRunResultsNilResults(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "results.json")

	err := SaveRunResults(nil, path)
	if err == nil {
		t.Fatal("expected error for nil results")
	}
}

func TestSaveRunResultsCreatesDir(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "deep", "nested", "path", "results.json")

	results := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 10, 30, 0, 0, time.UTC),
		RemainingBudget: 50000,
	}

	err := SaveRunResults(results, path)
	if err != nil {
		t.Fatalf("SaveRunResults: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist at %s", path)
	}
}

func TestLoadRunResults(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "results.json")

	original := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 10, 45, 0, 0, time.UTC),
		StartBudget:     200000,
		UsedBudget:      50000,
		RemainingBudget: 150000,
		Tasks: []TaskResult{
			{
				Project:    "my-project",
				Title:      "Implement feature",
				TaskType:   "feature",
				Status:     "completed",
				TokensUsed: 25000,
				Duration:   time.Minute * 30,
			},
		},
	}

	if err := SaveRunResults(original, path); err != nil {
		t.Fatalf("SaveRunResults: %v", err)
	}

	loaded, err := LoadRunResults(path)
	if err != nil {
		t.Fatalf("LoadRunResults: %v", err)
	}

	if loaded.StartBudget != original.StartBudget {
		t.Fatalf("StartBudget = %d, want %d", loaded.StartBudget, original.StartBudget)
	}
	if loaded.UsedBudget != original.UsedBudget {
		t.Fatalf("UsedBudget = %d, want %d", loaded.UsedBudget, original.UsedBudget)
	}
	if len(loaded.Tasks) != 1 {
		t.Fatalf("Tasks count = %d, want 1", len(loaded.Tasks))
	}
	if loaded.Tasks[0].Title != "Implement feature" {
		t.Fatalf("Task title = %s, want 'Implement feature'", loaded.Tasks[0].Title)
	}
}

func TestLoadRunResultsNotFound(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "nonexistent.json")

	_, err := LoadRunResults(path)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadRunResultsInvalidJSON(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "invalid.json")

	if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := LoadRunResults(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "roundtrip.json")

	original := &RunResults{
		StartTime:       time.Date(2026, 2, 10, 9, 15, 30, 0, time.UTC),
		EndTime:         time.Date(2026, 2, 10, 11, 45, 15, 0, time.UTC),
		StartBudget:     500000,
		UsedBudget:      125000,
		RemainingBudget: 375000,
		Tasks: []TaskResult{
			{
				Project:     "project-1",
				Title:       "Task A",
				TaskType:    "type-a",
				Status:      "completed",
				TokensUsed:  50000,
				Duration:    time.Minute * 45,
				OutputRef:   "ref-1",
				SkipReason:  "",
			},
			{
				Project:     "project-2",
				Title:       "Task B",
				TaskType:    "type-b",
				Status:      "skipped",
				TokensUsed:  0,
				Duration:    0,
				OutputRef:   "",
				SkipReason:  "Low priority",
			},
		},
	}

	if err := SaveRunResults(original, path); err != nil {
		t.Fatalf("SaveRunResults: %v", err)
	}

	loaded, err := LoadRunResults(path)
	if err != nil {
		t.Fatalf("LoadRunResults: %v", err)
	}

	if !loaded.StartTime.Equal(original.StartTime) {
		t.Fatalf("StartTime mismatch")
	}
	if !loaded.EndTime.Equal(original.EndTime) {
		t.Fatalf("EndTime mismatch")
	}
	if loaded.RemainingBudget != original.RemainingBudget {
		t.Fatalf("RemainingBudget mismatch")
	}
	if len(loaded.Tasks) != 2 {
		t.Fatalf("Tasks count mismatch")
	}
	if loaded.Tasks[1].SkipReason != "Low priority" {
		t.Fatalf("SkipReason mismatch")
	}
}
