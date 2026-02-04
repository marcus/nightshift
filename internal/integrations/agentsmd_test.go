package integrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

func TestAgentsMDReader_Name(t *testing.T) {
	r := &AgentsMDReader{}
	if r.Name() != "agents.md" {
		t.Errorf("Name() = %q, want %q", r.Name(), "agents.md")
	}
}

func TestAgentsMDReader_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AgentsMDReader{enabled: tt.enabled}
			if r.Enabled() != tt.enabled {
				t.Errorf("Enabled() = %v, want %v", r.Enabled(), tt.enabled)
			}
		})
	}
}

func TestAgentsMDReader_Read_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nightshift-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	r := &AgentsMDReader{enabled: true}
	result, err := r.Read(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for missing file")
	}
}

func TestAgentsMDReader_Read_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nightshift-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `# Agent Configuration

## Allowed Actions
- Create files
- Run tests
- Submit PRs

## Forbidden Actions
- Delete production data
- Push to main directly

## Tool Restrictions
- No shell access
- Read-only for config files
`
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r := &AgentsMDReader{enabled: true}
	result, err := r.Read(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check metadata
	if result.Metadata["allowed_actions"] == nil {
		t.Error("expected allowed_actions in metadata")
	}
	if result.Metadata["forbidden_actions"] == nil {
		t.Error("expected forbidden_actions in metadata")
	}
	if result.Metadata["tool_restrictions"] == nil {
		t.Error("expected tool_restrictions in metadata")
	}

	// Check hints
	if len(result.Hints) == 0 {
		t.Error("expected hints to be extracted")
	}

	// Verify constraint hints exist
	var hasConstraint bool
	for _, h := range result.Hints {
		if h.Type == HintConstraint {
			hasConstraint = true
			break
		}
	}
	if !hasConstraint {
		t.Error("expected constraint hints from forbidden/restrictions")
	}
}

func TestAgentsMDReader_Read_LowercaseFilename(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nightshift-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := "# Agents"
	if err := os.WriteFile(filepath.Join(tmpDir, "agents.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r := &AgentsMDReader{enabled: true}
	result, err := r.Read(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result == nil {
		t.Error("expected to find agents.md")
	}
}

func TestNewAgentsMDReader(t *testing.T) {
	cfg := &config.Config{
		Integrations: config.IntegrationsConfig{
			AgentsMD: true,
		},
	}

	r := NewAgentsMDReader(cfg)
	if !r.Enabled() {
		t.Error("expected reader to be enabled")
	}
}

func TestParseAgentsMD(t *testing.T) {
	content := `# Agent Config

## Permitted Actions
- Run linters
- Create branches

## Never Do
- Force push
- Delete repos

## Tool Restrictions
- Limit API calls

## Safety
- Always backup first
`
	result := parseAgentsMD(content)

	if len(result.allowedActions) == 0 {
		t.Error("expected allowed actions")
	}
	if len(result.forbiddenActions) == 0 {
		t.Error("expected forbidden actions")
	}
	if len(result.toolRestrictions) == 0 {
		t.Error("expected tool restrictions")
	}
}
