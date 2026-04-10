package agents

import (
	"context"
	"testing"
	"time"
)

// Backward-compatibility tests for the agents package.
// These tests verify that the Agent interface, ExecuteOptions, and
// ExecuteResult structs maintain their public API surface.

// TestBackwardCompat_DefaultTimeout verifies the default timeout constant
// hasn't changed, since scripts and configs may rely on the 30-minute default.
func TestBackwardCompat_DefaultTimeout(t *testing.T) {
	expected := 30 * time.Minute
	if DefaultTimeout != expected {
		t.Errorf("DefaultTimeout = %v, want %v — changing the default timeout breaks existing behavior",
			DefaultTimeout, expected)
	}
}

// TestBackwardCompat_ExecuteOptionsFields verifies that ExecuteOptions
// can be constructed with all expected fields. A compilation failure here
// means a field was removed or renamed.
func TestBackwardCompat_ExecuteOptionsFields(t *testing.T) {
	opts := ExecuteOptions{
		Prompt:  "test prompt",
		WorkDir: "/tmp/test",
		Files:   []string{"file1.go", "file2.go"},
		Timeout: 10 * time.Minute,
	}

	if opts.Prompt != "test prompt" {
		t.Error("ExecuteOptions.Prompt field not working")
	}
	if opts.WorkDir != "/tmp/test" {
		t.Error("ExecuteOptions.WorkDir field not working")
	}
	if len(opts.Files) != 2 {
		t.Error("ExecuteOptions.Files field not working")
	}
	if opts.Timeout != 10*time.Minute {
		t.Error("ExecuteOptions.Timeout field not working")
	}
}

// TestBackwardCompat_ExecuteResultFields verifies that ExecuteResult
// can be constructed with all expected fields and that IsSuccess()
// works as expected.
func TestBackwardCompat_ExecuteResultFields(t *testing.T) {
	// Successful result
	success := &ExecuteResult{
		Output:   "task completed",
		JSON:     []byte(`{"ok":true}`),
		ExitCode: 0,
		Duration: 5 * time.Second,
		Error:    "",
	}

	if !success.IsSuccess() {
		t.Error("IsSuccess() should return true for exit code 0 and no error")
	}

	// Failed result (non-zero exit code)
	failedExit := &ExecuteResult{
		Output:   "",
		ExitCode: 1,
		Error:    "",
	}

	if failedExit.IsSuccess() {
		t.Error("IsSuccess() should return false for non-zero exit code")
	}

	// Failed result (error message)
	failedError := &ExecuteResult{
		Output:   "",
		ExitCode: 0,
		Error:    "something went wrong",
	}

	if failedError.IsSuccess() {
		t.Error("IsSuccess() should return false when Error is set")
	}
}

// testAgent implements the Agent interface for compile-time verification.
type testAgent struct{}

func (a *testAgent) Name() string { return "test" }
func (a *testAgent) Execute(_ context.Context, _ ExecuteOptions) (*ExecuteResult, error) {
	return &ExecuteResult{}, nil
}

// TestBackwardCompat_AgentInterfaceCompiles verifies that the Agent
// interface has the expected method signatures. This is a compile-time
// check — if the interface changes, this test won't compile.
func TestBackwardCompat_AgentInterfaceCompiles(t *testing.T) {
	// This is a compile-time assertion that the interface shape is preserved.
	// The testAgent type above implements Agent, and if Agent's methods change,
	// this assignment will fail to compile.
	var _ Agent = (*testAgent)(nil)

	// Verify the interface can't be satisfied with just Name() — Execute() is required
	t.Log("Agent interface requires both Name() and Execute() methods")
}
