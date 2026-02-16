package agents

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// mockExitError simulates an exec.ExitError for testing.
type mockExitError struct {
	code int
}

func (e *mockExitError) Error() string {
	return "exit status " + string(rune(e.code))
}

// writeTestFile is a helper to write test files.
func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestNewCopilotAgent_Defaults(t *testing.T) {
	agent := NewCopilotAgent()

	if agent.binaryPath != "gh" {
		t.Errorf("binaryPath = %q, want %q", agent.binaryPath, "gh")
	}
	if agent.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", agent.timeout, DefaultTimeout)
	}
	if agent.runner == nil {
		t.Error("expected non-nil runner")
	}
}

func TestNewCopilotAgent_WithOptions(t *testing.T) {
	mockRunner := &MockRunner{}
	agent := NewCopilotAgent(
		WithCopilotBinaryPath("/custom/gh"),
		WithCopilotDefaultTimeout(5*time.Minute),
		WithCopilotRunner(mockRunner),
	)

	if agent.binaryPath != "/custom/gh" {
		t.Errorf("binaryPath = %q, want %q", agent.binaryPath, "/custom/gh")
	}
	if agent.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v", agent.timeout, 5*time.Minute)
	}
	if agent.runner != mockRunner {
		t.Error("expected custom runner")
	}
}

func TestCopilotAgent_Name(t *testing.T) {
	agent := NewCopilotAgent()
	if agent.Name() != "copilot" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "copilot")
	}
}

func TestCopilotAgent_Execute_Success(t *testing.T) {
	mock := &MockRunner{
		Stdout:   "Here's a solution to your problem",
		ExitCode: 0,
	}
	agent := NewCopilotAgent(WithCopilotRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt:  "how to list files",
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
	if result.Output != "Here's a solution to your problem" {
		t.Errorf("Output = %q, want %q", result.Output, "Here's a solution to your problem")
	}

	// Verify captured values
	if mock.CapturedName != "gh" {
		t.Errorf("binary = %q, want %q", mock.CapturedName, "gh")
	}
	expectedArgs := []string{"copilot", "suggest", "-t", "shell", "how to list files"}
	if len(mock.CapturedArgs) != len(expectedArgs) {
		t.Errorf("args length = %d, want %d", len(mock.CapturedArgs), len(expectedArgs))
	} else {
		for i, arg := range expectedArgs {
			if mock.CapturedArgs[i] != arg {
				t.Errorf("args[%d] = %q, want %q", i, mock.CapturedArgs[i], arg)
			}
		}
	}
	if mock.CapturedDir != "/project" {
		t.Errorf("dir = %q, want %q", mock.CapturedDir, "/project")
	}
}

func TestCopilotAgent_Execute_JSONOutput(t *testing.T) {
	mock := &MockRunner{
		Stdout:   `{"suggestion":"ls -la","explanation":"Lists all files"}`,
		ExitCode: 0,
	}
	agent := NewCopilotAgent(WithCopilotRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "list files",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.JSON == nil {
		t.Error("expected JSON to be extracted")
	}
}

func TestCopilotAgent_Execute_Error(t *testing.T) {
	mock := &MockRunner{
		Stderr:   "GitHub Copilot extension not installed",
		ExitCode: 1,
		Err:      &mockExitError{code: 1},
	}
	agent := NewCopilotAgent(WithCopilotRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "test prompt",
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

func TestCopilotAgent_Execute_Timeout(t *testing.T) {
	mock := &MockRunner{
		Delay:    5 * time.Second,
		ExitCode: -1,
	}
	agent := NewCopilotAgent(WithCopilotRunner(mock))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := agent.Execute(ctx, ExecuteOptions{
		Prompt: "test prompt",
	})

	if err == nil {
		t.Error("expected timeout error")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1 for timeout", result.ExitCode)
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestCopilotAgent_Execute_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"
	content := "test content"
	if err := writeTestFile(testFile, content); err != nil {
		t.Fatal(err)
	}

	mock := &MockRunner{
		Stdout:   "Response with context",
		ExitCode: 0,
	}
	agent := NewCopilotAgent(WithCopilotRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "analyze this file",
		Files:  []string{testFile},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	// Verify file content was included in stdin
	if mock.CapturedStdin == "" {
		t.Error("expected file context in stdin")
	}
	if !containsString(mock.CapturedStdin, "test content") {
		t.Error("expected file content in stdin")
	}
}

func TestCopilotAgent_ExecuteWithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"
	if err := writeTestFile(testFile, "content"); err != nil {
		t.Fatal(err)
	}

	mock := &MockRunner{
		Stdout:   "Response",
		ExitCode: 0,
	}
	agent := NewCopilotAgent(WithCopilotRunner(mock))

	result, err := agent.ExecuteWithFiles(context.Background(), "test prompt", []string{testFile}, "/workdir")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if mock.CapturedDir != "/workdir" {
		t.Errorf("dir = %q, want %q", mock.CapturedDir, "/workdir")
	}
}

func TestCopilotAgent_ExtractJSON(t *testing.T) {
	agent := NewCopilotAgent()

	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "pure JSON object",
			output: `{"key":"value"}`,
			want:   true,
		},
		{
			name:   "JSON in text",
			output: `Some text {"key":"value"} more text`,
			want:   true,
		},
		{
			name:   "JSON array",
			output: `[1,2,3]`,
			want:   true,
		},
		{
			name:   "no JSON",
			output: "plain text response",
			want:   false,
		},
		{
			name:   "invalid JSON",
			output: `{"key":}`,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.extractJSON([]byte(tt.output))
			hasJSON := result != nil
			if hasJSON != tt.want {
				t.Errorf("extractJSON() returned JSON = %v, want %v", hasJSON, tt.want)
			}
		})
	}
}
