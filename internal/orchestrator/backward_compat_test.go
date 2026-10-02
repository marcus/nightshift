package orchestrator

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBackwardCompat_TaskStatusConstants verifies that TaskStatus string
// values haven't changed. These are stored in databases and used in CLI
// output and reporting.
func TestBackwardCompat_TaskStatusConstants(t *testing.T) {
	expected := map[TaskStatus]string{
		StatusPending:   "pending",
		StatusPlanning:  "planning",
		StatusExecuting: "executing",
		StatusReviewing: "reviewing",
		StatusCompleted: "completed",
		StatusFailed:    "failed",
		StatusAbandoned: "abandoned",
	}

	for status, wantStr := range expected {
		if string(status) != wantStr {
			t.Errorf("TaskStatus constant changed: got %q, want %q", string(status), wantStr)
		}
	}
}

// TestBackwardCompat_TaskResultJSONFields verifies that TaskResult JSON
// field names haven't changed. External tools parsing nightshift output
// depend on these field names.
func TestBackwardCompat_TaskResultJSONFields(t *testing.T) {
	result := TaskResult{
		TaskID:     "test-task-1",
		Status:     StatusCompleted,
		Iterations: 2,
		Plan: &PlanOutput{
			Steps:       []string{"step 1", "step 2"},
			Files:       []string{"file1.go"},
			Description: "test plan",
		},
		Output:     "task output",
		OutputType: "PR",
		OutputRef:  "https://github.com/example/repo/pull/1",
		Error:      "",
		Duration:   5 * time.Minute,
		Logs: []LogEntry{
			{Time: time.Now(), Level: "info", Message: "started"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredFields := []string{
		"task_id", "status", "iterations", "duration", "logs",
	}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("TaskResult JSON missing required field %q", field)
		}
	}

	// Optional fields with omitempty should be present when set
	optionalPresent := []string{"plan", "output", "output_type", "output_ref"}
	for _, field := range optionalPresent {
		if _, ok := raw[field]; !ok {
			t.Errorf("TaskResult JSON missing optional field %q (should be present when set)", field)
		}
	}
}

// TestBackwardCompat_PlanOutputJSONFields verifies PlanOutput JSON structure.
func TestBackwardCompat_PlanOutputJSONFields(t *testing.T) {
	plan := PlanOutput{
		Steps:       []string{"analyze", "implement", "test"},
		Files:       []string{"main.go", "main_test.go"},
		Description: "Fix the bug",
		Raw:         "raw plan text",
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredFields := []string{"steps", "files", "description"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("PlanOutput JSON missing required field %q", field)
		}
	}
}

// TestBackwardCompat_ReviewOutputJSONFields verifies ReviewOutput JSON structure.
func TestBackwardCompat_ReviewOutputJSONFields(t *testing.T) {
	review := ReviewOutput{
		Passed:   true,
		Feedback: "looks good",
		Issues:   []string{"minor: consider renaming"},
		Raw:      "raw review",
	}

	data, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredFields := []string{"passed", "feedback"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("ReviewOutput JSON missing required field %q", field)
		}
	}
}

// TestBackwardCompat_ImplementOutputJSONFields verifies ImplementOutput JSON structure.
func TestBackwardCompat_ImplementOutputJSONFields(t *testing.T) {
	impl := ImplementOutput{
		FilesModified: []string{"main.go", "handler.go"},
		Summary:       "Fixed the auth bug",
		Raw:           "raw output",
	}

	data, err := json.Marshal(impl)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredFields := []string{"files_modified", "summary"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("ImplementOutput JSON missing required field %q", field)
		}
	}
}

// TestBackwardCompat_LogEntryJSONFields verifies LogEntry JSON structure.
func TestBackwardCompat_LogEntryJSONFields(t *testing.T) {
	entry := LogEntry{
		Time:    time.Date(2026, 1, 15, 2, 0, 0, 0, time.UTC),
		Level:   "info",
		Message: "task started",
		Fields:  map[string]any{"task_id": "test-1"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredFields := []string{"time", "level", "message"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("LogEntry JSON missing required field %q", field)
		}
	}
}

// TestBackwardCompat_TaskResultDeserialization verifies that JSON from
// an earlier version (without output_type/output_ref) still deserializes.
func TestBackwardCompat_TaskResultDeserialization(t *testing.T) {
	// Simulate older format without output_type and output_ref
	oldJSON := `{
		"task_id": "old-task",
		"status": "completed",
		"iterations": 1,
		"output": "some output",
		"duration": 300000000000,
		"logs": []
	}`

	var result TaskResult
	if err := json.Unmarshal([]byte(oldJSON), &result); err != nil {
		t.Fatalf("Unmarshal old TaskResult JSON: %v", err)
	}

	if result.TaskID != "old-task" {
		t.Errorf("TaskID = %q, want %q", result.TaskID, "old-task")
	}
	if result.Status != StatusCompleted {
		t.Errorf("Status = %q, want %q", result.Status, StatusCompleted)
	}
	if result.OutputType != "" {
		t.Errorf("OutputType = %q, want empty (old format)", result.OutputType)
	}
	if result.OutputRef != "" {
		t.Errorf("OutputRef = %q, want empty (old format)", result.OutputRef)
	}
}

// TestBackwardCompat_DefaultConstants verifies orchestrator constants.
func TestBackwardCompat_DefaultConstants(t *testing.T) {
	if DefaultMaxIterations != 3 {
		t.Errorf("DefaultMaxIterations = %d, want 3", DefaultMaxIterations)
	}
	if DefaultAgentTimeout != 30*time.Minute {
		t.Errorf("DefaultAgentTimeout = %v, want 30m", DefaultAgentTimeout)
	}
}
