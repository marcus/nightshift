package state

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/db"
)

// TestBackwardCompat_RunRecordJSONFields verifies that RunRecord JSON field
// names haven't changed. External tools and stored history may depend on
// these exact field names.
func TestBackwardCompat_RunRecordJSONFields(t *testing.T) {
	record := RunRecord{
		ID:         "run-123",
		StartTime:  time.Date(2026, 1, 15, 2, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 1, 15, 2, 30, 0, 0, time.UTC),
		Provider:   "claude",
		Project:    "/path/to/project",
		Tasks:      []string{"lint-fix", "test-gap"},
		TokensUsed: 50000,
		Status:     "success",
		Error:      "",
		Branch:     "main",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// Verify all expected JSON field names are present
	requiredFields := []string{
		"id", "start_time", "end_time", "project",
		"tasks", "tokens_used", "status",
	}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("RunRecord JSON missing required field %q", field)
		}
	}

	// Verify omitempty fields work correctly
	// Provider should be present when set
	if _, ok := raw["provider"]; !ok {
		t.Error("RunRecord JSON missing 'provider' field when set")
	}
	// Branch should be present when set
	if _, ok := raw["branch"]; !ok {
		t.Error("RunRecord JSON missing 'branch' field when set")
	}
}

// TestBackwardCompat_RunRecordOmitEmpty verifies that omitempty fields
// are correctly omitted when empty, maintaining backward compatibility
// with consumers that may not expect these fields.
func TestBackwardCompat_RunRecordOmitEmpty(t *testing.T) {
	record := RunRecord{
		ID:        "run-456",
		StartTime: time.Date(2026, 1, 15, 2, 0, 0, 0, time.UTC),
		Project:   "/path/to/project",
		Tasks:     []string{"lint-fix"},
		Status:    "success",
		// Provider, Error, Branch intentionally empty
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// omitempty fields should be absent when empty
	omitFields := []string{"provider", "error", "branch"}
	for _, field := range omitFields {
		if _, ok := raw[field]; ok {
			t.Errorf("RunRecord JSON should omit empty field %q", field)
		}
	}
}

// TestBackwardCompat_RunRecordDeserialization verifies that a JSON payload
// matching the v0.3.0 format (no provider/branch fields) can still be
// deserialized into the current RunRecord struct.
func TestBackwardCompat_RunRecordDeserialization(t *testing.T) {
	// Simulate v0.3.0 JSON that lacks provider and branch fields
	oldJSON := `{
		"id": "run-old",
		"start_time": "2026-01-15T02:00:00Z",
		"end_time": "2026-01-15T02:30:00Z",
		"project": "/old/project",
		"tasks": ["lint-fix"],
		"tokens_used": 25000,
		"status": "success"
	}`

	var record RunRecord
	if err := json.Unmarshal([]byte(oldJSON), &record); err != nil {
		t.Fatalf("Unmarshal old JSON: %v", err)
	}

	if record.ID != "run-old" {
		t.Errorf("ID = %q, want %q", record.ID, "run-old")
	}
	if record.Project != "/old/project" {
		t.Errorf("Project = %q, want %q", record.Project, "/old/project")
	}
	if record.Provider != "" {
		t.Errorf("Provider = %q, want empty (old format)", record.Provider)
	}
	if record.Branch != "" {
		t.Errorf("Branch = %q, want empty (old format)", record.Branch)
	}
	if record.Status != "success" {
		t.Errorf("Status = %q, want %q", record.Status, "success")
	}
	if len(record.Tasks) != 1 || record.Tasks[0] != "lint-fix" {
		t.Errorf("Tasks = %v, want [lint-fix]", record.Tasks)
	}
}

// TestBackwardCompat_ProjectStateJSONFields verifies that ProjectState JSON
// field names haven't changed.
func TestBackwardCompat_ProjectStateJSONFields(t *testing.T) {
	state := ProjectState{
		Path:        "/path/to/project",
		LastRun:     time.Date(2026, 1, 15, 2, 0, 0, 0, time.UTC),
		TaskHistory: map[string]time.Time{"lint-fix": time.Now()},
		RunCount:    5,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredFields := []string{"path", "last_run", "task_history", "run_count"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("ProjectState JSON missing required field %q", field)
		}
	}
}

// TestBackwardCompat_AssignedTaskJSONFields verifies that AssignedTask JSON
// field names haven't changed.
func TestBackwardCompat_AssignedTaskJSONFields(t *testing.T) {
	task := AssignedTask{
		TaskID:     "lint-fix:/path",
		Project:    "/path/to/project",
		TaskType:   "lint-fix",
		AssignedAt: time.Date(2026, 1, 15, 2, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredFields := []string{"task_id", "project", "task_type", "assigned_at"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("AssignedTask JSON missing required field %q", field)
		}
	}
}

// TestBackwardCompat_StalenessBonus verifies the staleness calculation
// formula hasn't changed, as task selection depends on it.
func TestBackwardCompat_StalenessBonus(t *testing.T) {
	s := newBackwardCompatTestState(t)
	project := "/compat/project"

	// Never-run task should get max bonus of 3.0
	bonus := s.StalenessBonus(project, "never-run-task")
	if bonus != 3.0 {
		t.Errorf("StalenessBonus for never-run task = %f, want 3.0", bonus)
	}

	// Recently run task should get low bonus
	s.RecordTaskRun(project, "recent-task")
	bonus = s.StalenessBonus(project, "recent-task")
	if bonus != 0.0 {
		t.Errorf("StalenessBonus for just-run task = %f, want 0.0", bonus)
	}
}

// TestBackwardCompat_DaysSinceLastRun verifies the sentinel value for
// never-run tasks (-1) hasn't changed.
func TestBackwardCompat_DaysSinceLastRun(t *testing.T) {
	s := newBackwardCompatTestState(t)

	days := s.DaysSinceLastRun("/some/project", "untracked-task")
	if days != -1 {
		t.Errorf("DaysSinceLastRun for untracked task = %d, want -1", days)
	}
}

// TestBackwardCompat_RunHistoryPersistence verifies that run records can
// be stored and retrieved with all fields intact across the current schema.
func TestBackwardCompat_RunHistoryPersistence(t *testing.T) {
	s := newBackwardCompatTestState(t)

	record := RunRecord{
		ID:         "compat-run-1",
		StartTime:  time.Now().Add(-time.Hour),
		EndTime:    time.Now(),
		Provider:   "claude",
		Project:    "/compat/project",
		Tasks:      []string{"lint-fix", "test-gap"},
		TokensUsed: 75000,
		Status:     "success",
		Branch:     "main",
	}

	s.AddRunRecord(record)

	history := s.GetRunHistory(1)
	if len(history) != 1 {
		t.Fatalf("GetRunHistory returned %d records, want 1", len(history))
	}

	got := history[0]
	if got.ID != record.ID {
		t.Errorf("ID = %q, want %q", got.ID, record.ID)
	}
	if got.Provider != record.Provider {
		t.Errorf("Provider = %q, want %q", got.Provider, record.Provider)
	}
	if got.Branch != record.Branch {
		t.Errorf("Branch = %q, want %q", got.Branch, record.Branch)
	}
	if got.TokensUsed != record.TokensUsed {
		t.Errorf("TokensUsed = %d, want %d", got.TokensUsed, record.TokensUsed)
	}
	if len(got.Tasks) != 2 {
		t.Errorf("Tasks length = %d, want 2", len(got.Tasks))
	}
}

// TestBackwardCompat_TodaySummaryStructure verifies that TodaySummary
// fields are preserved for consumers.
func TestBackwardCompat_TodaySummaryStructure(t *testing.T) {
	s := newBackwardCompatTestState(t)

	// Add a run for today
	s.AddRunRecord(RunRecord{
		ID:         "today-run-1",
		StartTime:  time.Now(),
		Project:    "/today/project",
		Tasks:      []string{"lint-fix"},
		TokensUsed: 10000,
		Status:     "success",
	})

	summary := s.GetTodaySummary()
	if summary.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", summary.TotalRuns)
	}
	if summary.SuccessfulRuns != 1 {
		t.Errorf("SuccessfulRuns = %d, want 1", summary.SuccessfulRuns)
	}
	if summary.TotalTokens != 10000 {
		t.Errorf("TotalTokens = %d, want 10000", summary.TotalTokens)
	}
	if summary.TaskCounts == nil {
		t.Error("TaskCounts should not be nil")
	}
}

// newBackwardCompatTestState creates a State backed by a temporary database for testing.
func newBackwardCompatTestState(t *testing.T) *State {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	s, err := New(database)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	return s
}
