package security

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultSandboxConfig(t *testing.T) {
	cfg := DefaultSandboxConfig()

	if cfg.AllowNetwork {
		t.Error("expected AllowNetwork to be false by default")
	}

	if cfg.MaxDuration != 30*time.Minute {
		t.Errorf("expected MaxDuration 30min, got %v", cfg.MaxDuration)
	}

	if !cfg.Cleanup {
		t.Error("expected Cleanup to be true by default")
	}
}

func TestNewSandbox(t *testing.T) {
	cfg := DefaultSandboxConfig()

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	if sandbox.TempDir() == "" {
		t.Error("expected temp dir to be set")
	}

	// Verify temp dir exists
	if _, err := os.Stat(sandbox.TempDir()); err != nil {
		t.Errorf("temp dir does not exist: %v", err)
	}
}

func TestSandbox_Execute(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.MaxDuration = 10 * time.Second

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	ctx := context.Background()

	// Test simple command
	result, err := sandbox.Execute(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success, got exit code %d, error: %s", result.ExitCode, result.Error)
	}

	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", result.Stdout)
	}
}

func TestSandbox_ExecuteWithTimeout(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.MaxDuration = 100 * time.Millisecond

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	ctx := context.Background()

	// Command that takes too long
	result, err := sandbox.Execute(ctx, "sleep", "10")
	_ = err // expected for timeout

	// Should not succeed
	if result != nil && result.Success() {
		t.Error("expected timeout or failure for long-running command")
	}
}

func TestSandbox_ExecuteCommandNotFound(t *testing.T) {
	cfg := DefaultSandboxConfig()

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	ctx := context.Background()

	_, err = sandbox.Execute(ctx, "nonexistent_command_12345")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestSandbox_ValidatePath(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.AllowedPaths = []string{"/tmp", "/var/tmp"}
	cfg.DeniedPaths = []string{"/etc", "/root"}

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	tests := []struct {
		path    string
		wantErr bool
	}{
		{"/tmp/test.txt", false},
		{"/var/tmp/data", false},
		{"/etc/passwd", true},
		{"/root/.ssh", true},
		{"/home/user/file", true}, // Not in allowed list
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := sandbox.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestSandbox_CreateTempFile(t *testing.T) {
	cfg := DefaultSandboxConfig()

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	f, err := sandbox.CreateTempFile("test-*.txt")
	if err != nil {
		t.Fatalf("CreateTempFile failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Verify file is in sandbox temp dir
	if !strings.HasPrefix(f.Name(), sandbox.TempDir()) {
		t.Errorf("temp file not in sandbox dir: %s", f.Name())
	}
}

func TestSandbox_CreateTempDir(t *testing.T) {
	cfg := DefaultSandboxConfig()

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	dir, err := sandbox.CreateTempDir("subdir-*")
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}

	// Verify dir is in sandbox temp dir
	if !strings.HasPrefix(dir, sandbox.TempDir()) {
		t.Errorf("temp dir not in sandbox dir: %s", dir)
	}

	// Verify dir exists
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("temp dir does not exist: %v", err)
	}
}

func TestSandbox_Cleanup(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.Cleanup = true

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}

	tempDir := sandbox.TempDir()

	// Create a file in temp dir
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Cleanup
	if err := sandbox.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify temp dir is gone
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("expected temp dir to be removed after cleanup")
	}
}

func TestSandbox_CleanupDisabled(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.Cleanup = false

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}

	tempDir := sandbox.TempDir()

	// Cleanup should not delete when disabled
	if err := sandbox.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify temp dir still exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("expected temp dir to remain when cleanup disabled")
	}

	// Manual cleanup for test
	_ = os.RemoveAll(tempDir)
}

func TestSandbox_BuildEnvironment(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.Environment = map[string]string{
		"CUSTOM_VAR": "custom_value",
	}

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	env := sandbox.buildEnvironment()

	// Check sandbox-specific vars
	hasSandboxVar := false
	hasCustomVar := false
	hasTmpDir := false

	for _, e := range env {
		if e == "NIGHTSHIFT_SANDBOX=1" {
			hasSandboxVar = true
		}
		if e == "CUSTOM_VAR=custom_value" {
			hasCustomVar = true
		}
		if strings.HasPrefix(e, "TMPDIR=") {
			hasTmpDir = true
		}
	}

	if !hasSandboxVar {
		t.Error("expected NIGHTSHIFT_SANDBOX env var")
	}

	if !hasCustomVar {
		t.Error("expected CUSTOM_VAR env var")
	}

	if !hasTmpDir {
		t.Error("expected TMPDIR env var")
	}
}

func TestSandbox_IsActive(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.MaxDuration = 5 * time.Second

	sandbox, err := NewSandbox(cfg)
	if err != nil {
		t.Fatalf("NewSandbox failed: %v", err)
	}
	defer func() { _ = sandbox.Cleanup() }()

	if sandbox.IsActive() {
		t.Error("expected sandbox to not be active initially")
	}

	// Start a command
	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		_, _ = sandbox.Execute(ctx, "sleep", "1")
		close(done)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	if !sandbox.IsActive() {
		t.Error("expected sandbox to be active during execution")
	}

	// Wait for completion
	<-done

	if sandbox.IsActive() {
		t.Error("expected sandbox to not be active after completion")
	}
}

func TestNewSandboxedAgent(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := SandboxConfig{
		WorkDir:      tmpDir,
		AllowNetwork: false,
		MaxDuration:  10 * time.Second,
		Cleanup:      true,
	}

	agent, err := NewSandboxedAgent(cfg)
	if err != nil {
		t.Fatalf("NewSandboxedAgent failed: %v", err)
	}
	defer func() { _ = agent.Close() }()

	if agent.Sandbox() == nil {
		t.Error("expected sandbox to be initialized")
	}
}
