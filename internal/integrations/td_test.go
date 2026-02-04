package integrations

import (
	"testing"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

func TestTDReader_Name(t *testing.T) {
	r := &TDReader{}
	if r.Name() != "td" {
		t.Errorf("Name() = %q, want %q", r.Name(), "td")
	}
}

func TestTDReader_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TDReader{enabled: tt.enabled}
			if r.Enabled() != tt.enabled {
				t.Errorf("Enabled() = %v, want %v", r.Enabled(), tt.enabled)
			}
		})
	}
}

func TestNewTDReader(t *testing.T) {
	cfg := &config.Config{
		Integrations: config.IntegrationsConfig{
			TaskSources: []config.TaskSourceEntry{
				{
					TD: &config.TDConfig{
						Enabled:    true,
						TeachAgent: true,
					},
				},
			},
		},
	}

	r := NewTDReader(cfg)
	if !r.Enabled() {
		t.Error("expected reader to be enabled")
	}
	if !r.teachAgent {
		t.Error("expected teachAgent to be true")
	}
}

func TestNewTDReader_Disabled(t *testing.T) {
	cfg := &config.Config{
		Integrations: config.IntegrationsConfig{
			TaskSources: []config.TaskSourceEntry{},
		},
	}

	r := NewTDReader(cfg)
	if r.Enabled() {
		t.Error("expected reader to be disabled")
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"critical", 100},
		{"urgent", 100},
		{"high", 75},
		{"medium", 50},
		{"normal", 50},
		{"low", 25},
		{"HIGH", 75},
		{"100", 100},
		{"25", 25},
		{"unknown", 50},
		{"", 50},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parsePriority(tt.input)
			if got != tt.want {
				t.Errorf("parsePriority(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestTDUsageContext(t *testing.T) {
	// Verify the context string contains key commands
	if tdUsageContext == "" {
		t.Error("expected non-empty usage context")
	}

	mustContain := []string{
		"td list",
		"td assign",
		"td complete",
	}

	for _, s := range mustContain {
		if !contains(tdUsageContext, s) {
			t.Errorf("usage context should contain %q", s)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
