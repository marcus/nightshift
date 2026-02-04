package integrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

func TestClaudeMDReader_Name(t *testing.T) {
	r := &ClaudeMDReader{}
	if r.Name() != "claude.md" {
		t.Errorf("Name() = %q, want %q", r.Name(), "claude.md")
	}
}

func TestClaudeMDReader_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ClaudeMDReader{enabled: tt.enabled}
			if r.Enabled() != tt.enabled {
				t.Errorf("Enabled() = %v, want %v", r.Enabled(), tt.enabled)
			}
		})
	}
}

func TestClaudeMDReader_Read_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nightshift-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	r := &ClaudeMDReader{enabled: true}
	result, err := r.Read(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for missing file")
	}
}

func TestClaudeMDReader_Read_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nightshift-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `# My Project

A test project.

## Conventions
- Use Go idioms
- Write tests for all code

## Tasks
- Implement feature X
- Fix bug Y
`
	if err := os.WriteFile(filepath.Join(tmpDir, "claude.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r := &ClaudeMDReader{enabled: true}
	result, err := r.Read(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check context includes content
	if result.Context == "" {
		t.Error("expected non-empty context")
	}

	// Check hints extracted
	if len(result.Hints) == 0 {
		t.Error("expected hints to be extracted")
	}

	// Check metadata
	if result.Metadata["source_file"] == nil {
		t.Error("expected source_file in metadata")
	}
}

func TestClaudeMDReader_Read_CaseSensitive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nightshift-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := "# Test"
	// Write CLAUDE.md (uppercase)
	if err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r := &ClaudeMDReader{enabled: true}
	result, err := r.Read(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result == nil {
		t.Error("expected to find CLAUDE.md")
	}
}

func TestNewClaudeMDReader(t *testing.T) {
	cfg := &config.Config{
		Integrations: config.IntegrationsConfig{
			ClaudeMD: true,
		},
	}

	r := NewClaudeMDReader(cfg)
	if !r.Enabled() {
		t.Error("expected reader to be enabled")
	}
}

func TestParseClaudeMD(t *testing.T) {
	content := `# Project

## Coding Style
- Use 4 spaces for indentation
- Prefer explicit error handling

## Tasks
- Add logging
- Fix tests

## Constraints
- Do not modify public APIs
`
	result := parseClaudeMD(content)

	if result.context == "" {
		t.Error("expected non-empty context")
	}

	// Check for convention hints
	var hasConvention, hasTask, hasConstraint bool
	for _, h := range result.hints {
		switch h.Type {
		case HintConvention:
			hasConvention = true
		case HintTaskSuggestion:
			hasTask = true
		case HintConstraint:
			hasConstraint = true
		}
	}

	if !hasConvention {
		t.Error("expected convention hints")
	}
	if !hasTask {
		t.Error("expected task hints")
	}
	if !hasConstraint {
		t.Error("expected constraint hints")
	}
}
