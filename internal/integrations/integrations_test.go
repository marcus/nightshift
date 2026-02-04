package integrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		Integrations: config.IntegrationsConfig{
			ClaudeMD: true,
			AgentsMD: true,
		},
	}

	m := NewManager(cfg)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if len(m.readers) != 4 {
		t.Errorf("expected 4 readers, got %d", len(m.readers))
	}
}

func TestHintTypeString(t *testing.T) {
	tests := []struct {
		h    HintType
		want string
	}{
		{HintTaskSuggestion, "task_suggestion"},
		{HintConvention, "convention"},
		{HintConstraint, "constraint"},
		{HintContext, "context"},
		{HintType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.h.String(); got != tt.want {
			t.Errorf("HintType(%d).String() = %q, want %q", tt.h, got, tt.want)
		}
	}
}

func TestReaderError(t *testing.T) {
	err := ReaderError{Reader: "test", Err: os.ErrNotExist}
	want := "test: file does not exist"
	if err.Error() != want {
		t.Errorf("ReaderError.Error() = %q, want %q", err.Error(), want)
	}
}

func TestManagerReadAll(t *testing.T) {
	// Create temp dir with test files
	tmpDir, err := os.MkdirTemp("", "nightshift-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a claude.md file
	claudeMD := `# Project

This is a test project.

## Conventions
- Use Go idioms
- Write tests
`
	if err := os.WriteFile(filepath.Join(tmpDir, "claude.md"), []byte(claudeMD), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Integrations: config.IntegrationsConfig{
			ClaudeMD: true,
			AgentsMD: true,
		},
	}

	m := NewManager(cfg)
	result, err := m.ReadAll(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if result == nil {
		t.Fatal("ReadAll returned nil result")
	}

	// Should have claude.md result
	if _, ok := result.Results["claude.md"]; !ok {
		t.Error("expected claude.md result")
	}

	// Check combined context includes claude.md
	if result.CombinedContext == "" {
		t.Error("expected non-empty CombinedContext")
	}
}
