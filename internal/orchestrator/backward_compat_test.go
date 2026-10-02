package orchestrator

import (
	"encoding/json"
	"testing"
	"time"
)

// Backward-compatibility tests for the orchestrator package.
// These tests verify that the output structs serialize to JSON correctly,
// since external tools and reports parse these structures.

// TestBackwardCompat_TaskStatusValues verifies that TaskStatus string
// values are stable. These appear in logs, reports, and state database.
func TestBackwardCompat_TaskStatusValues(t *testing.T) {
	statuses := map[TaskStatus]string{
		StatusPending:   "pending",
		StatusPlanning:  "planning",
		StatusExecuting: "executing",
		StatusReviewing: "reviewing",
		StatusCompleted: "completed",
		StatusFailed:    "failed",
		StatusAbandoned: "abandoned",
	}

	for status, want := range statuses {
		if string(status) != want {
			t.Errorf("TaskStatus %q has value %q, want %q — status values appear in logs and database",
				want, string(status), want)
		}
	}
}

// TestBackwardCompat_TaskResultJSONKeys verifies that TaskResult serializes
// to JSON with the expected key names. External tools may parse this output.
func TestBackwardCompat_TaskResultJSONKeys(t *testing.T) {
	result := TaskResult{
		TaskID:     "test-123",
		Status:     StatusCompleted,
		Iterations: 2,
		Plan: &PlanOutput{
			Steps:       []string{"step1", "step2"},
			Files:       []string{"file.go"},
			Description: "test plan",
		},
		Output:     "done",
		OutputType: "PR",
		OutputRef:  "https://github.com/test/repo/pull/1",
		Error:      "",
		Duration:   5 * time.Minute,
		Logs:       []LogEntry{{Message: "started"}},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(TaskResult): %v", err)
	}

	// Unmarshal into generic map to check key names
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredKeys := []string{
		"task_id", "status", "iterations", "plan",
		"output", "output_type", "output_ref", "duration", "logs",
	}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("TaskResult JSON missing key %q — this breaks external consumers", key)
		}
	}

	// Verify status value is correct
	if m["status"] != "completed" {
		t.Errorf("TaskResult.status = %v, want completed", m["status"])
	}
}

// TestBackwardCompat_PlanOutputJSONKeys verifies PlanOutput serialization.
func TestBackwardCompat_PlanOutputJSONKeys(t *testing.T) {
	plan := PlanOutput{
		Steps:       []string{"analyze", "implement", "test"},
		Files:       []string{"main.go", "main_test.go"},
		Description: "fix the bug",
		Raw:         "raw plan output",
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal(PlanOutput): %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, key := range []string{"steps", "files", "description"} {
		if _, ok := m[key]; !ok {
			t.Errorf("PlanOutput JSON missing key %q", key)
		}
	}
}

// TestBackwardCompat_ImplementOutputJSONKeys verifies ImplementOutput serialization.
func TestBackwardCompat_ImplementOutputJSONKeys(t *testing.T) {
	impl := ImplementOutput{
		FilesModified: []string{"main.go"},
		Summary:       "fixed the bug",
		Raw:           "raw output",
	}

	data, err := json.Marshal(impl)
	if err != nil {
		t.Fatalf("json.Marshal(ImplementOutput): %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, key := range []string{"files_modified", "summary"} {
		if _, ok := m[key]; !ok {
			t.Errorf("ImplementOutput JSON missing key %q", key)
		}
	}
}

// TestBackwardCompat_ReviewOutputJSONKeys verifies ReviewOutput serialization.
func TestBackwardCompat_ReviewOutputJSONKeys(t *testing.T) {
	review := ReviewOutput{
		Passed:   true,
		Feedback: "looks good",
		Issues:   []string{},
		Raw:      "raw review",
	}

	data, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("json.Marshal(ReviewOutput): %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, key := range []string{"passed", "feedback"} {
		if _, ok := m[key]; !ok {
			t.Errorf("ReviewOutput JSON missing key %q", key)
		}
	}
}

// TestBackwardCompat_OrchestratorConstants verifies that orchestration
// constants haven't changed unexpectedly.
func TestBackwardCompat_OrchestratorConstants(t *testing.T) {
	if DefaultMaxIterations != 3 {
		t.Errorf("DefaultMaxIterations = %d, want 3", DefaultMaxIterations)
	}
	if DefaultAgentTimeout != 30*time.Minute {
		t.Errorf("DefaultAgentTimeout = %v, want 30m", DefaultAgentTimeout)
	}
}
