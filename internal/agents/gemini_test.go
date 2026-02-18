package agents

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestNewGeminiAgent_Defaults(t *testing.T) {
	agent := NewGeminiAgent()

	if agent.binaryPath != "gemini" {
		t.Errorf("binaryPath = %q, want %q", agent.binaryPath, "gemini")
	}
	if agent.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", agent.timeout, DefaultTimeout)
	}
	if agent.runner == nil {
		t.Error("expected non-nil runner")
	}
	if !agent.autoApprove {
		t.Error("expected autoApprove to be true by default")
	}
}

func TestNewGeminiAgent_WithOptions(t *testing.T) {
	mockRunner := &MockRunner{}
	agent := NewGeminiAgent(
		WithGeminiBinaryPath("/custom/gemini"),
		WithGeminiDefaultTimeout(5*time.Minute),
		WithGeminiRunner(mockRunner),
		WithGeminiAutoApprove(false),
	)

	if agent.binaryPath != "/custom/gemini" {
		t.Errorf("binaryPath = %q, want %q", agent.binaryPath, "/custom/gemini")
	}
	if agent.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v", agent.timeout, 5*time.Minute)
	}
	if agent.runner != mockRunner {
		t.Error("expected custom runner")
	}
	if agent.autoApprove {
		t.Error("expected autoApprove to be false")
	}
}

func TestGeminiAgent_Name(t *testing.T) {
	agent := NewGeminiAgent()
	if agent.Name() != "gemini" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "gemini")
	}
}

func TestGeminiAgent_Execute_Basic(t *testing.T) {
	mockRunner := &MockRunner{
		Stdout:   `{"response": "Hello!", "stats": {}}`,
		ExitCode: 0,
	}
	agent := NewGeminiAgent(WithGeminiRunner(mockRunner))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt:  "say hello",
		WorkDir: "/tmp",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Output != `{"response": "Hello!", "stats": {}}` {
		t.Errorf("unexpected output: %s", result.Output)
	}
	if result.JSON == nil {
		t.Error("expected JSON output to be parsed")
	}

	// Verify command args
	if mockRunner.CapturedName != "gemini" {
		t.Errorf("command = %q, want %q", mockRunner.CapturedName, "gemini")
	}

	// Should contain -p, -y, --output-format, json, and the prompt
	args := mockRunner.CapturedArgs
	hasP := false
	hasY := false
	hasFormat := false
	hasPrompt := false
	for i, a := range args {
		if a == "-p" {
			hasP = true
		}
		if a == "-y" {
			hasY = true
		}
		if a == "--output-format" && i+1 < len(args) && args[i+1] == "json" {
			hasFormat = true
		}
		if a == "say hello" {
			hasPrompt = true
		}
	}
	if !hasP {
		t.Error("missing -p flag")
	}
	if !hasY {
		t.Error("missing -y flag")
	}
	if !hasFormat {
		t.Error("missing --output-format json")
	}
	if !hasPrompt {
		t.Error("missing prompt in args")
	}
}

func TestGeminiAgent_Execute_NoAutoApprove(t *testing.T) {
	mockRunner := &MockRunner{Stdout: "ok", ExitCode: 0}
	agent := NewGeminiAgent(
		WithGeminiRunner(mockRunner),
		WithGeminiAutoApprove(false),
	)

	_, err := agent.Execute(context.Background(), ExecuteOptions{Prompt: "test"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	for _, a := range mockRunner.CapturedArgs {
		if a == "-y" {
			t.Error("-y flag should not be present when autoApprove is false")
		}
	}
}

func TestGeminiAgent_Execute_Timeout(t *testing.T) {
	mockRunner := &MockRunner{
		Delay: 2 * time.Second,
	}
	agent := NewGeminiAgent(WithGeminiRunner(mockRunner))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt:  "slow task",
		Timeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestGeminiAgent_Execute_WithFiles(t *testing.T) {
	tmpFile := t.TempDir() + "/test.go"
	if err := os.WriteFile(tmpFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	mockRunner := &MockRunner{Stdout: "done", ExitCode: 0}
	agent := NewGeminiAgent(WithGeminiRunner(mockRunner))

	_, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "review this",
		Files:  []string{tmpFile},
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if mockRunner.CapturedStdin == "" {
		t.Error("expected file context in stdin")
	}
}

func TestGeminiAgent_Execute_Error(t *testing.T) {
	mockRunner := &MockRunner{
		Stderr:   "something went wrong",
		ExitCode: 1,
		Err:      errors.New("exit status 1"),
	}
	agent := NewGeminiAgent(WithGeminiRunner(mockRunner))

	result, err := agent.Execute(context.Background(), ExecuteOptions{Prompt: "fail"})
	if err == nil {
		t.Fatal("expected error")
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}
