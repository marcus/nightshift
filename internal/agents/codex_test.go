package agents

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewCodexAgent_Defaults(t *testing.T) {
	agent := NewCodexAgent()

	if agent.binaryPath != "codex" {
		t.Errorf("binaryPath = %q, want %q", agent.binaryPath, "codex")
	}
	if agent.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", agent.timeout, DefaultTimeout)
	}
	if agent.runner == nil {
		t.Error("expected non-nil runner")
	}
}

func TestNewCodexAgent_WithOptions(t *testing.T) {
	mockRunner := &MockRunner{}
	agent := NewCodexAgent(
		WithCodexBinaryPath("/custom/codex"),
		WithCodexDefaultTimeout(5*time.Minute),
		WithCodexRunner(mockRunner),
	)

	if agent.binaryPath != "/custom/codex" {
		t.Errorf("binaryPath = %q, want %q", agent.binaryPath, "/custom/codex")
	}
	if agent.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v", agent.timeout, 5*time.Minute)
	}
	if agent.runner != mockRunner {
		t.Error("expected custom runner")
	}
}

func TestCodexAgent_Name(t *testing.T) {
	agent := NewCodexAgent()
	if agent.Name() != "codex" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "codex")
	}
}

func TestCodexAgent_Execute_Success(t *testing.T) {
	mock := &MockRunner{
		Stdout:   "Task completed successfully",
		ExitCode: 0,
	}
	agent := NewCodexAgent(WithCodexRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt:  "fix the bug",
		WorkDir: "/project",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !result.IsSuccess() {
		t.Error("expected IsSuccess() to be true")
	}
	if result.Output != "Task completed successfully" {
		t.Errorf("Output = %q, want %q", result.Output, "Task completed successfully")
	}

	// Verify captured values
	if mock.CapturedName != "codex" {
		t.Errorf("binary = %q, want %q", mock.CapturedName, "codex")
	}
	if len(mock.CapturedArgs) != 3 || mock.CapturedArgs[0] != "exec" || mock.CapturedArgs[1] != "--dangerously-bypass-approvals-and-sandbox" || mock.CapturedArgs[2] != "fix the bug" {
		t.Errorf("args = %v, want [exec --dangerously-bypass-approvals-and-sandbox fix the bug]", mock.CapturedArgs)
	}
	if mock.CapturedDir != "/project" {
		t.Errorf("dir = %q, want %q", mock.CapturedDir, "/project")
	}
}

func TestCodexAgent_Execute_JSONOutput(t *testing.T) {
	mock := &MockRunner{
		Stdout:   `{"status":"success","files_changed":3}`,
		ExitCode: 0,
	}
	agent := NewCodexAgent(WithCodexRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "analyze code",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.JSON == nil {
		t.Error("expected JSON to be extracted")
	}
	if string(result.JSON) != `{"status":"success","files_changed":3}` {
		t.Errorf("JSON = %s", result.JSON)
	}
}

func TestCodexAgent_Execute_Timeout(t *testing.T) {
	mock := &MockRunner{
		Delay: 5 * time.Second, // Will be cancelled
	}
	agent := NewCodexAgent(
		WithCodexRunner(mock),
		WithCodexDefaultTimeout(50*time.Millisecond),
	)

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "long task",
	})

	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
	if !strings.Contains(result.Error, "timeout") {
		t.Errorf("Error = %q, want timeout message", result.Error)
	}
}

func TestCodexAgent_Execute_WithOptionsTimeout(t *testing.T) {
	mock := &MockRunner{
		Delay: 5 * time.Second,
	}
	agent := NewCodexAgent(
		WithCodexRunner(mock),
		WithCodexDefaultTimeout(10*time.Second), // Long default
	)

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt:  "task",
		Timeout: 50 * time.Millisecond, // Short override
	})

	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
}

func TestCodexAgent_Execute_ExitError(t *testing.T) {
	mock := &MockRunner{
		Stdout:   "",
		Stderr:   "command failed: no such file",
		ExitCode: 1,
		Err:      errors.New("exit status 1"),
	}
	agent := NewCodexAgent(WithCodexRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "bad task",
	})

	if err == nil {
		t.Error("expected error")
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.IsSuccess() {
		t.Error("expected IsSuccess() to be false")
	}
}

func TestCodexAgent_Execute_BinaryNotFound(t *testing.T) {
	mock := &MockRunner{
		Err: errors.New("executable file not found"),
	}
	agent := NewCodexAgent(
		WithCodexBinaryPath("/nonexistent/codex"),
		WithCodexRunner(mock),
	)

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "test",
	})

	if err == nil {
		t.Error("expected error for missing binary")
	}
	if result == nil {
		t.Fatal("expected result even on error")
		return
	}
	if result.Error == "" {
		t.Error("expected error message in result")
	}
}

func TestCodexAgent_Execute_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &MockRunner{
		Stdout:   "analyzed file",
		ExitCode: 0,
	}
	agent := NewCodexAgent(WithCodexRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "review code",
		Files:  []string{testFile},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(mock.CapturedStdin, "package main") {
		t.Error("expected file content in stdin")
	}
	if !strings.Contains(mock.CapturedStdin, "# Context Files") {
		t.Error("expected context header in stdin")
	}
	if result.Output != "analyzed file" {
		t.Errorf("Output = %q", result.Output)
	}
}

func TestCodexAgent_Execute_MissingFile(t *testing.T) {
	agent := NewCodexAgent(WithCodexRunner(&MockRunner{}))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "review",
		Files:  []string{"/nonexistent/file.go"},
	})

	if err == nil {
		t.Error("expected error for missing file")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestCodexAgent_ExecuteWithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("func main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &MockRunner{
		Stdout:   "ok",
		ExitCode: 0,
	}
	agent := NewCodexAgent(WithCodexRunner(mock))

	result, err := agent.ExecuteWithFiles(context.Background(), "analyze", []string{testFile}, tmpDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "ok" {
		t.Errorf("Output = %q", result.Output)
	}
	if mock.CapturedDir != tmpDir {
		t.Errorf("WorkDir = %q, want %q", mock.CapturedDir, tmpDir)
	}
}

func TestCodexAgent_buildFileContext(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.go")

	if err := os.WriteFile(file1, []byte("content 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	agent := NewCodexAgent()
	ctx, err := agent.buildFileContext([]string{file1, file2})
	if err != nil {
		t.Fatalf("buildFileContext error: %v", err)
	}

	if ctx == "" {
		t.Error("expected non-empty context")
	}
	if !strings.Contains(ctx, "content 1") {
		t.Error("context missing file1 content")
	}
	if !strings.Contains(ctx, "package main") {
		t.Error("context missing file2 content")
	}
	if !strings.Contains(ctx, "# Context Files") {
		t.Error("context missing header")
	}
}

func TestCodexAgent_buildFileContext_MissingFile(t *testing.T) {
	agent := NewCodexAgent()
	_, err := agent.buildFileContext([]string{"/nonexistent/file.txt"})
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestCodexAgent_extractJSON(t *testing.T) {
	agent := NewCodexAgent()

	tests := []struct {
		name     string
		input    string
		wantJSON bool
	}{
		{"plain json", `{"key":"value"}`, true},
		{"json array", `[1,2,3]`, true},
		{"json with prefix", `Some text {"key":"value"}`, true},
		{"json with suffix", `{"key":"value"} more text`, true},
		{"json with both", `prefix {"key":"value"} suffix`, true},
		{"no json", `plain text no json here`, false},
		{"invalid json", `{"key":}`, false},
		{"nested json", `{"outer":{"inner":"value"}}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.extractJSON([]byte(tt.input))
			if tt.wantJSON && result == nil {
				t.Error("expected JSON, got nil")
			}
			if !tt.wantJSON && result != nil {
				t.Errorf("expected nil, got %s", result)
			}
		})
	}
}

func TestCodexAgent_Available(t *testing.T) {
	// Test with known available binary
	agent := NewCodexAgent(WithCodexBinaryPath("echo"))
	if !agent.Available() {
		t.Error("expected echo to be available")
	}

	// Test with nonexistent binary
	agent = NewCodexAgent(WithCodexBinaryPath("/nonexistent/binary"))
	if agent.Available() {
		t.Error("expected nonexistent binary to not be available")
	}
}

func TestCodexAgent_Version(t *testing.T) {
	agent := NewCodexAgent(WithCodexBinaryPath("/nonexistent/codex"))
	_, err := agent.Version()
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

func TestCodexAgent_ContextCancellation(t *testing.T) {
	mock := &MockRunner{
		Delay: 5 * time.Second,
	}
	agent := NewCodexAgent(
		WithCodexRunner(mock),
		WithCodexDefaultTimeout(10*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := agent.Execute(ctx, ExecuteOptions{
		Prompt: "task",
	})

	if err == nil {
		t.Error("expected error for cancelled context")
	}
	if result == nil {
		t.Fatal("expected result")
	}
}

func TestCodexAgent_ImplementsAgentInterface(t *testing.T) {
	// Verify CodexAgent implements the Agent interface
	var _ Agent = (*CodexAgent)(nil)
}
