package security

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	if logger.sessionID == "" {
		t.Error("expected session ID to be set")
	}

	if logger.file == nil {
		t.Error("expected log file to be open")
	}
}

func TestAuditLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	event := AuditEvent{
		EventType: AuditAgentStart,
		Agent:     "claude",
		TaskID:    "task-123",
		Project:   "/path/to/project",
		Action:    "start",
	}

	if err := logger.Log(event); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify log file contains data
	files, err := logger.GetLogFiles()
	if err != nil {
		t.Fatalf("GetLogFiles failed: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least one log file")
	}

	// Read and verify event
	events, err := ReadEvents(files[0])
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	e := events[0]
	if e.EventType != AuditAgentStart {
		t.Errorf("expected EventType %s, got %s", AuditAgentStart, e.EventType)
	}
	if e.Agent != "claude" {
		t.Errorf("expected Agent 'claude', got %s", e.Agent)
	}
	if e.TaskID != "task-123" {
		t.Errorf("expected TaskID 'task-123', got %s", e.TaskID)
	}
	if e.SessionID == "" {
		t.Error("expected SessionID to be set")
	}
	if e.RequestID == "" {
		t.Error("expected RequestID to be set")
	}
}

func TestAuditLogger_LogOperation(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	op := Operation{
		Type:   OpFileWrite,
		Target: "/path/to/file.go",
		Agent:  "claude",
		TaskID: "task-456",
	}

	if err := logger.LogOperation(op); err != nil {
		t.Fatalf("LogOperation failed: %v", err)
	}

	files, _ := logger.GetLogFiles()
	events, _ := ReadEvents(files[0])

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	e := events[0]
	if e.EventType != AuditFileWrite {
		t.Errorf("expected EventType %s, got %s", AuditFileWrite, e.EventType)
	}
}

func TestAuditLogger_LogAgentStart(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	if err := logger.LogAgentStart("claude", "task-1", "/project"); err != nil {
		t.Fatalf("LogAgentStart failed: %v", err)
	}

	files, _ := logger.GetLogFiles()
	events, _ := ReadEvents(files[0])

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	e := events[0]
	if e.EventType != AuditAgentStart {
		t.Errorf("expected EventType %s, got %s", AuditAgentStart, e.EventType)
	}
	if e.Action != "start" {
		t.Errorf("expected Action 'start', got %s", e.Action)
	}
}

func TestAuditLogger_LogAgentComplete(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	duration := 5 * time.Second
	tokensUsed := 1500

	if err := logger.LogAgentComplete("claude", "task-1", "/project", duration, tokensUsed, "success"); err != nil {
		t.Fatalf("LogAgentComplete failed: %v", err)
	}

	files, _ := logger.GetLogFiles()
	events, _ := ReadEvents(files[0])

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	e := events[0]
	if e.EventType != AuditAgentComplete {
		t.Errorf("expected EventType %s, got %s", AuditAgentComplete, e.EventType)
	}
	if e.Duration != duration {
		t.Errorf("expected Duration %v, got %v", duration, e.Duration)
	}
	if e.TokensUsed != tokensUsed {
		t.Errorf("expected TokensUsed %d, got %d", tokensUsed, e.TokensUsed)
	}
	if e.Result != "success" {
		t.Errorf("expected Result 'success', got %s", e.Result)
	}
}

func TestAuditLogger_LogAgentError(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	testErr := &CredentialError{Credential: "test", Message: "test error"}

	if err := logger.LogAgentError("claude", "task-1", "/project", testErr); err != nil {
		t.Fatalf("LogAgentError failed: %v", err)
	}

	files, _ := logger.GetLogFiles()
	events, _ := ReadEvents(files[0])

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	e := events[0]
	if e.EventType != AuditAgentError {
		t.Errorf("expected EventType %s, got %s", AuditAgentError, e.EventType)
	}
	if e.Error == "" {
		t.Error("expected Error to be set")
	}
}

func TestAuditLogger_LogFileModification(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	if err := logger.LogFileModification(AuditFileWrite, "/path/to/file.go", "claude", "task-1"); err != nil {
		t.Fatalf("LogFileModification failed: %v", err)
	}

	files, _ := logger.GetLogFiles()
	events, _ := ReadEvents(files[0])

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	e := events[0]
	if e.Target != "/path/to/file.go" {
		t.Errorf("expected Target '/path/to/file.go', got %s", e.Target)
	}
}

func TestAuditLogger_LogGitOperation(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	metadata := map[string]string{
		"commit_hash": "abc123",
		"message":     "test commit",
	}

	if err := logger.LogGitOperation("commit", "/repo", "main", "claude", "task-1", metadata); err != nil {
		t.Fatalf("LogGitOperation failed: %v", err)
	}

	files, _ := logger.GetLogFiles()
	events, _ := ReadEvents(files[0])

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	e := events[0]
	if e.EventType != AuditGitOperation {
		t.Errorf("expected EventType %s, got %s", AuditGitOperation, e.EventType)
	}
	if e.Action != "commit" {
		t.Errorf("expected Action 'commit', got %s", e.Action)
	}
	if e.Metadata["commit_hash"] != "abc123" {
		t.Errorf("expected commit_hash 'abc123', got %s", e.Metadata["commit_hash"])
	}
}

func TestAuditLogger_LogSecurityCheck(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test allowed
	if err := logger.LogSecurityCheck("write_access", "/path", "allowed", true); err != nil {
		t.Fatalf("LogSecurityCheck failed: %v", err)
	}

	// Test denied
	if err := logger.LogSecurityCheck("git_push", "origin", "denied", false); err != nil {
		t.Fatalf("LogSecurityCheck failed: %v", err)
	}

	files, _ := logger.GetLogFiles()
	events, _ := ReadEvents(files[0])

	if len(events) < 2 {
		t.Fatal("expected at least 2 events")
	}

	if events[0].EventType != AuditSecurityCheck {
		t.Errorf("expected EventType %s, got %s", AuditSecurityCheck, events[0].EventType)
	}

	if events[1].EventType != AuditSecurityDenied {
		t.Errorf("expected EventType %s, got %s", AuditSecurityDenied, events[1].EventType)
	}
}

func TestAuditLogger_GetLogFiles(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write something
	_ = logger.Log(AuditEvent{EventType: AuditAgentStart})

	files, err := logger.GetLogFiles()
	if err != nil {
		t.Fatalf("GetLogFiles failed: %v", err)
	}

	if len(files) == 0 {
		t.Error("expected at least one log file")
	}

	for _, f := range files {
		if filepath.Ext(f) != ".jsonl" {
			t.Errorf("expected .jsonl extension, got %s", filepath.Ext(f))
		}
	}
}

func TestReadEvents(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	// Write multiple events
	for i := 0; i < 5; i++ {
		_ = logger.Log(AuditEvent{
			EventType: AuditAgentStart,
			TaskID:    "task-" + string(rune('0'+i)),
		})
	}
	_ = logger.Close()

	files, _ := logger.GetLogFiles()
	events, err := ReadEvents(files[0])
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}

	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}
}

func TestReadEvents_NonExistent(t *testing.T) {
	_, err := ReadEvents("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadEvents_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")

	// Write malformed JSON
	content := `{"valid": "json"}
not valid json
{"also": "valid"}`
	_ = os.WriteFile(path, []byte(content), 0644)

	events, err := ReadEvents(path)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}

	// Should skip malformed lines
	if len(events) != 2 {
		t.Errorf("expected 2 events (skipping malformed), got %d", len(events))
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"line1\nline2\nline3", 3},
		{"single", 1},
		{"with\ntrailing\n", 2},
		{"", 0},
		{"\n\n\n", 0},
	}

	for _, tt := range tests {
		lines := splitLines([]byte(tt.input))
		if len(lines) != tt.expected {
			t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(lines), tt.expected)
		}
	}
}
