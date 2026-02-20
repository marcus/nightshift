package commands

import (
	"testing"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/config"
)

func TestResolveProviderListAll(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
	}

	result, err := resolveProviderList(cfg, "")
	if err != nil {
		t.Fatalf("resolveProviderList: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(result))
	}
	if result[0] != "claude" || result[1] != "codex" {
		t.Fatalf("unexpected provider list: %v", result)
	}
}

func TestResolveProviderListClaude(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
	}

	result, err := resolveProviderList(cfg, "claude")
	if err != nil {
		t.Fatalf("resolveProviderList: %v", err)
	}

	if len(result) != 1 || result[0] != "claude" {
		t.Fatalf("expected [claude], got %v", result)
	}
}

func TestResolveProviderListCodex(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
	}

	result, err := resolveProviderList(cfg, "codex")
	if err != nil {
		t.Fatalf("resolveProviderList: %v", err)
	}

	if len(result) != 1 || result[0] != "codex" {
		t.Fatalf("expected [codex], got %v", result)
	}
}

func TestResolveProviderListDisabledProvider(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: false},
		},
	}

	_, err := resolveProviderList(cfg, "codex")
	if err == nil {
		t.Fatal("expected error for disabled provider")
	}
}

func TestResolveProviderListInvalidProvider(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
	}

	_, err := resolveProviderList(cfg, "invalid")
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
}

func TestResolveProviderListNoneEnabled(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: false},
			Codex:  config.ProviderConfig{Enabled: false},
		},
	}

	result, err := resolveProviderList(cfg, "")
	if err != nil {
		t.Fatalf("resolveProviderList: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected empty list, got %v", result)
	}
}

func TestResolveProviderListCaseInsensitive(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
	}

	result, err := resolveProviderList(cfg, "CLAUDE")
	if err != nil {
		t.Fatalf("resolveProviderList: %v", err)
	}

	if len(result) != 1 || result[0] != "claude" {
		t.Fatalf("expected [claude], got %v", result)
	}
}

func TestFormatTokens64Zero(t *testing.T) {
	result := formatTokens64(0)
	if result != "0" {
		t.Fatalf("expected '0', got '%s'", result)
	}
}

func TestFormatTokens64Hundreds(t *testing.T) {
	result := formatTokens64(500)
	if result != "500" {
		t.Fatalf("expected '500', got '%s'", result)
	}
}

func TestFormatTokens64Thousands(t *testing.T) {
	result := formatTokens64(1500)
	if result != "1.5K" {
		t.Fatalf("expected '1.5K', got '%s'", result)
	}
}

func TestFormatTokens64Millions(t *testing.T) {
	result := formatTokens64(1500000)
	if result != "1.5M" {
		t.Fatalf("expected '1.5M', got '%s'", result)
	}
}

func TestFormatBudgetMetaEmpty(t *testing.T) {
	est := &budget.BudgetEstimate{}
	result := formatBudgetMeta(*est)
	if result != "" {
		t.Fatalf("expected empty string, got '%s'", result)
	}
}

func TestFormatBudgetMetaWithSource(t *testing.T) {
	est := &budget.BudgetEstimate{Source: "config"}
	result := formatBudgetMeta(*est)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if !contains(result, "config") {
		t.Fatalf("expected 'config' in result, got '%s'", result)
	}
}

func TestFormatBudgetMetaWithConfidence(t *testing.T) {
	est := &budget.BudgetEstimate{
		Source:     "calibrated",
		Confidence: "high",
	}
	result := formatBudgetMeta(*est)
	if !contains(result, "calibrated") || !contains(result, "high") {
		t.Fatalf("expected 'calibrated' and 'high', got '%s'", result)
	}
}

func TestFormatBudgetMetaWithSampleCount(t *testing.T) {
	est := &budget.BudgetEstimate{
		Source:      "scraped",
		SampleCount: 42,
	}
	result := formatBudgetMeta(*est)
	if !contains(result, "scraped") || !contains(result, "42 samples") {
		t.Fatalf("expected 'scraped' and '42 samples', got '%s'", result)
	}
}

func TestFormatResetLineEmpty(t *testing.T) {
	result := formatResetLine("", "")
	if result != "" {
		t.Fatalf("expected empty string, got '%s'", result)
	}
}

func TestFormatResetLineSessionOnly(t *testing.T) {
	result := formatResetLine("9pm (America/Los_Angeles)", "")
	if !contains(result, "session") || !contains(result, "9pm") {
		t.Fatalf("expected session info, got '%s'", result)
	}
}

func TestFormatResetLineWeeklyOnly(t *testing.T) {
	result := formatResetLine("", "Feb 8 at 10am (America/Los_Angeles)")
	if !contains(result, "week") || !contains(result, "Feb 8") {
		t.Fatalf("expected weekly info, got '%s'", result)
	}
}

func TestFormatResetLineBoth(t *testing.T) {
	result := formatResetLine("9pm (America/Los_Angeles)", "Feb 8 at 10am (America/Los_Angeles)")
	if !contains(result, "session") || !contains(result, "week") || !contains(result, "·") {
		t.Fatalf("expected both session and weekly info separated by ·, got '%s'", result)
	}
}

func TestProgressBarZero(t *testing.T) {
	result := progressBar(0, 10)
	if !contains(result, "[") || !contains(result, "0.0%") {
		t.Fatalf("expected format with 0.0%%, got '%s'", result)
	}
}

func TestProgressBarFull(t *testing.T) {
	result := progressBar(100, 10)
	if !contains(result, "##########") || !contains(result, "100.0%") {
		t.Fatalf("expected full bar with 100.0%%, got '%s'", result)
	}
}

func TestProgressBarPartial(t *testing.T) {
	result := progressBar(50, 10)
	if !contains(result, "#####-----") || !contains(result, "50.0%") {
		t.Fatalf("expected half-filled bar with 50.0%%, got '%s'", result)
	}
}

func TestProgressBarNegative(t *testing.T) {
	result := progressBar(-10, 10)
	if !contains(result, "----------") || !contains(result, "-10.0%") {
		t.Fatalf("expected empty bar with -10.0%%, got '%s'", result)
	}
}

func TestProgressBarOver100(t *testing.T) {
	result := progressBar(150, 10)
	if !contains(result, "##########") || !contains(result, "150.0%") {
		t.Fatalf("expected full bar with 150.0%%, got '%s'", result)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
