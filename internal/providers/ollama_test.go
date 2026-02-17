package providers

import (
	"testing"
	"time"
)

func TestNewOllama(t *testing.T) {
	provider := NewOllama()
	if provider == nil {
		t.Fatal("NewOllama returned nil")
	}
	if provider.Name() != "ollama" {
		t.Errorf("Expected name 'ollama', got '%s'", provider.Name())
	}
}

func TestNewOllamaWithPath(t *testing.T) {
	dataPath := "/custom/path"
	provider := NewOllamaWithPath(dataPath)
	if provider == nil {
		t.Fatal("NewOllamaWithPath returned nil")
	}
	if provider.dataPath != dataPath {
		t.Errorf("Expected dataPath '%s', got '%s'", dataPath, provider.dataPath)
	}
}

func TestOllamaCost(t *testing.T) {
	provider := NewOllama()
	inputCents, outputCents := provider.Cost()
	if inputCents != 0 || outputCents != 0 {
		t.Errorf("Expected cost (0, 0), got (%d, %d)", inputCents, outputCents)
	}
}

func TestOllamaLastUsedPercentSource(t *testing.T) {
	provider := NewOllama()
	source := provider.LastUsedPercentSource()
	if source != "ollama-web-scrape" {
		t.Errorf("Expected source 'ollama-web-scrape', got '%s'", source)
	}
}

func TestParsePct(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"integer", "45", 45.0, false},
		{"float", "45.5", 45.5, false},
		{"zero", "0", 0.0, false},
		{"hundred", "100", 100.0, false},
		{"negative", "-5", 0.0, true},
		{"over hundred", "150", 0.0, true},
		{"invalid", "abc", 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parsePct() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseOllamaSettingsHTML(t *testing.T) {
	html := `<span class="text-sm">Session usage</span>
      <span class="text-sm">8.6% used</span>
    </div>
    
    <div
      class="text-xs text-neutral-500 mt-1 local-time"
      data-time="2026-02-11T10:00:00Z"
    >
      Resets in 6 minutes
    </div>
  </div>

  <div>
    <div class="flex justify-between mb-2">
      <span class="text-sm">Weekly usage</span>
      <span class="text-sm">13.6% used</span>
    </div>
    <div
      class="text-xs text-neutral-500 mt-1 local-time"
      data-time="2026-02-16T00:00:00Z"
    >
      Resets in...`

	sessionPct, weeklyPct, sessionReset, weeklyReset, err := parseOllamaSettingsHTML(html)
	if err != nil {
		t.Fatalf("parseOllamaSettingsHTML() error = %v", err)
	}

	if sessionPct != 8.6 {
		t.Errorf("Expected sessionPct 8.6, got %.1f", sessionPct)
	}
	if weeklyPct != 13.6 {
		t.Errorf("Expected weeklyPct 13.6, got %.1f", weeklyPct)
	}

	expectedSessionReset := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	if !sessionReset.Equal(expectedSessionReset) {
		t.Errorf("Expected sessionReset %v, got %v", expectedSessionReset, sessionReset)
	}

	expectedWeeklyReset := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	if !weeklyReset.Equal(expectedWeeklyReset) {
		t.Errorf("Expected weeklyReset %v, got %v", expectedWeeklyReset, weeklyReset)
	}
}

func TestGetUsedPercent_InvalidMode(t *testing.T) {
	provider := NewOllama()
	_, err := provider.GetUsedPercent("invalid", 1000000)
	if err == nil {
		t.Error("Expected error for invalid mode, got nil")
	}
}

func TestGetResetTime_InvalidMode(t *testing.T) {
	provider := NewOllama()
	_, err := provider.GetResetTime("invalid")
	if err == nil {
		t.Error("Expected error for invalid mode, got nil")
	}
}