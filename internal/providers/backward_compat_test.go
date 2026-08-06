package providers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Backward-compatibility tests for the providers package.
// These tests verify that provider interfaces, struct shapes, and
// JSON serialization are stable across versions.

// TestBackwardCompat_ProviderInterfaceShape verifies that all three
// providers (Claude, Codex, Copilot) satisfy the Provider interface.
// If the interface changes, this test won't compile.
func TestBackwardCompat_ProviderInterfaceShape(t *testing.T) {
	var _ Provider = (*Claude)(nil)
	var _ Provider = (*Codex)(nil)
	var _ Provider = (*Copilot)(nil)
}

// TestBackwardCompat_ProviderNames verifies that provider Name() methods
// return stable values. These are used in config files, database records,
// and log entries.
func TestBackwardCompat_ProviderNames(t *testing.T) {
	tests := []struct {
		provider Provider
		wantName string
	}{
		{NewClaude(), "claude"},
		{NewCodex(), "codex"},
		{NewCopilot(), "copilot"},
	}

	for _, tt := range tests {
		if got := tt.provider.Name(); got != tt.wantName {
			t.Errorf("%T.Name() = %q, want %q — provider names are referenced in configs and database",
				tt.provider, got, tt.wantName)
		}
	}
}

// TestBackwardCompat_ClaudeWithCustomPath verifies that NewClaudeWithPath
// works and that DataPath() returns the configured path.
func TestBackwardCompat_ClaudeWithCustomPath(t *testing.T) {
	c := NewClaudeWithPath("/custom/claude")
	if c.DataPath() != "/custom/claude" {
		t.Errorf("Claude.DataPath() = %q, want /custom/claude", c.DataPath())
	}
}

// TestBackwardCompat_CodexWithCustomPath verifies NewCodexWithPath.
func TestBackwardCompat_CodexWithCustomPath(t *testing.T) {
	c := NewCodexWithPath("/custom/codex")
	if c.DataPath() != "/custom/codex" {
		t.Errorf("Codex.DataPath() = %q, want /custom/codex", c.DataPath())
	}
}

// TestBackwardCompat_CopilotWithCustomPath verifies NewCopilotWithPath.
func TestBackwardCompat_CopilotWithCustomPath(t *testing.T) {
	c := NewCopilotWithPath("/custom/copilot")
	if c.DataPath() != "/custom/copilot" {
		t.Errorf("Copilot.DataPath() = %q, want /custom/copilot", c.DataPath())
	}
}

// TestBackwardCompat_ProviderCostMethod verifies that Cost() returns
// values for all providers (even if zero for Copilot).
func TestBackwardCompat_ProviderCostMethod(t *testing.T) {
	// Claude should have non-zero cost (token-based billing)
	claudeIn, claudeOut := NewClaude().Cost()
	if claudeIn <= 0 || claudeOut <= 0 {
		t.Errorf("Claude.Cost() = (%d, %d), expected positive values", claudeIn, claudeOut)
	}

	// Codex should have non-zero cost (token-based billing)
	codexIn, codexOut := NewCodex().Cost()
	if codexIn <= 0 || codexOut <= 0 {
		t.Errorf("Codex.Cost() = (%d, %d), expected positive values", codexIn, codexOut)
	}

	// Copilot returns 0 (request-based, not token-based)
	copilotIn, copilotOut := NewCopilot().Cost()
	if copilotIn != 0 || copilotOut != 0 {
		t.Errorf("Copilot.Cost() = (%d, %d), expected (0, 0) for request-based billing",
			copilotIn, copilotOut)
	}
}

// TestBackwardCompat_StatsCacheJSONParsing verifies that the StatsCache
// JSON structure matches Claude Code's stats-cache.json format.
func TestBackwardCompat_StatsCacheJSONParsing(t *testing.T) {
	// Simulate a stats-cache.json from Claude Code
	jsonData := `{
		"version": 1,
		"dailyActivity": [
			{"date": "2026-04-10", "messageCount": 42, "sessionCount": 3, "toolCallCount": 100}
		],
		"dailyModelTokens": [
			{"date": "2026-04-10", "tokensByModel": {"claude-opus-4-6": 50000, "claude-sonnet-4-6": 30000}}
		]
	}`

	var stats StatsCache
	if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
		t.Fatalf("Failed to parse StatsCache JSON: %v", err)
	}

	if stats.Version != 1 {
		t.Errorf("Version = %d, want 1", stats.Version)
	}
	if len(stats.DailyActivity) != 1 {
		t.Fatalf("DailyActivity length = %d, want 1", len(stats.DailyActivity))
	}
	if stats.DailyActivity[0].Date != "2026-04-10" {
		t.Errorf("DailyActivity[0].Date = %q, want 2026-04-10", stats.DailyActivity[0].Date)
	}
	if stats.DailyActivity[0].MessageCount != 42 {
		t.Errorf("MessageCount = %d, want 42", stats.DailyActivity[0].MessageCount)
	}

	// TokensByDate should sum across models
	byDate := stats.TokensByDate()
	if byDate["2026-04-10"] != 80000 {
		t.Errorf("TokensByDate[2026-04-10] = %d, want 80000", byDate["2026-04-10"])
	}

	// GetDailyStat should return combined data
	stat := stats.GetDailyStat("2026-04-10")
	if stat == nil {
		t.Fatal("GetDailyStat returned nil for existing date")
	}
	if stat.MessageCount != 42 {
		t.Errorf("DailyStat.MessageCount = %d, want 42", stat.MessageCount)
	}
}

// TestBackwardCompat_TokenUsageTotalTokens verifies that TotalTokens()
// sums all four token fields correctly.
func TestBackwardCompat_TokenUsageTotalTokens(t *testing.T) {
	usage := &TokenUsage{
		InputTokens:              1000,
		OutputTokens:             2000,
		CacheReadInputTokens:     500,
		CacheCreationInputTokens: 300,
	}

	total := usage.TotalTokens()
	expected := int64(3800)
	if total != expected {
		t.Errorf("TotalTokens() = %d, want %d", total, expected)
	}
}

// TestBackwardCompat_ParseStatsCacheEmptyFile verifies that a missing
// stats-cache.json file returns an empty StatsCache (not an error).
func TestBackwardCompat_ParseStatsCacheEmptyFile(t *testing.T) {
	stats, err := ParseStatsCache("/nonexistent/path/stats-cache.json")
	if err != nil {
		t.Fatalf("ParseStatsCache for missing file should not error: %v", err)
	}
	if stats == nil {
		t.Fatal("ParseStatsCache should return empty StatsCache, not nil")
	}
	if len(stats.DailyActivity) != 0 {
		t.Error("Empty StatsCache should have no DailyActivity")
	}
}

// TestBackwardCompat_CopilotUsageDataJSON verifies that the Copilot
// usage tracking file format is stable.
func TestBackwardCompat_CopilotUsageDataJSON(t *testing.T) {
	data := CopilotUsageData{
		RequestCount: 42,
		Month:        "2026-04",
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal(CopilotUsageData): %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, key := range []string{"request_count", "last_reset", "month"} {
		if _, ok := m[key]; !ok {
			t.Errorf("CopilotUsageData JSON missing key %q — this breaks usage file compatibility", key)
		}
	}
}

// TestBackwardCompat_CopilotUsageDataRoundTrip verifies that usage data
// can be written and read back correctly (file format stability).
func TestBackwardCompat_CopilotUsageDataRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewCopilotWithPath(tmpDir)

	// Load should return default data when file doesn't exist
	data, err := c.LoadUsageData()
	if err != nil {
		t.Fatalf("LoadUsageData for new install: %v", err)
	}
	if data.RequestCount != 0 {
		t.Errorf("New install RequestCount = %d, want 0", data.RequestCount)
	}

	// Save and reload
	data.RequestCount = 10
	if err := c.SaveUsageData(data); err != nil {
		t.Fatalf("SaveUsageData: %v", err)
	}

	reloaded, err := c.LoadUsageData()
	if err != nil {
		t.Fatalf("LoadUsageData after save: %v", err)
	}
	if reloaded.RequestCount != 10 {
		t.Errorf("After reload RequestCount = %d, want 10", reloaded.RequestCount)
	}
}

// TestBackwardCompat_CodexRateLimitsJSON verifies that Codex rate limit
// JSON parsing is stable.
func TestBackwardCompat_CodexRateLimitsJSON(t *testing.T) {
	jsonData := `{
		"primary": {"used_percent": 45.5, "window_minutes": 300, "resets_at": 1712700000},
		"secondary": {"used_percent": 12.0, "window_minutes": 10080, "resets_at": 1713000000}
	}`

	var limits CodexRateLimits
	if err := json.Unmarshal([]byte(jsonData), &limits); err != nil {
		t.Fatalf("Failed to parse CodexRateLimits: %v", err)
	}

	if limits.Primary == nil {
		t.Fatal("Primary rate limit should not be nil")
	}
	if limits.Primary.UsedPercent != 45.5 {
		t.Errorf("Primary.UsedPercent = %f, want 45.5", limits.Primary.UsedPercent)
	}
	if limits.Primary.WindowMinutes != 300 {
		t.Errorf("Primary.WindowMinutes = %d, want 300", limits.Primary.WindowMinutes)
	}
	if limits.Secondary == nil {
		t.Fatal("Secondary rate limit should not be nil")
	}
	if limits.Secondary.UsedPercent != 12.0 {
		t.Errorf("Secondary.UsedPercent = %f, want 12.0", limits.Secondary.UsedPercent)
	}
}

// TestBackwardCompat_CodexSessionJSONLParsing verifies that Codex session
// JSONL file parsing handles the expected format.
func TestBackwardCompat_CodexSessionJSONLParsing(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a minimal Codex session JSONL
	lines := `{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":500,"reasoning_output_tokens":100,"total_tokens":1400},"last_token_usage":null},"rate_limits":{"primary":{"used_percent":10.0,"window_minutes":300,"resets_at":1712700000},"secondary":{"used_percent":5.0,"window_minutes":10080,"resets_at":1713000000}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":2000,"cached_input_tokens":400,"output_tokens":1000,"reasoning_output_tokens":200,"total_tokens":2800},"last_token_usage":null},"rate_limits":{"primary":{"used_percent":20.0,"window_minutes":300,"resets_at":1712700000},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1713000000}}}}
`
	if err := os.WriteFile(sessionPath, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCodexWithPath(tmpDir)
	limits, err := c.ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL: %v", err)
	}

	// Should get the LAST entry's rate limits
	if limits == nil {
		t.Fatal("Expected rate limits from session")
	}
	if limits.Primary.UsedPercent != 20.0 {
		t.Errorf("Primary.UsedPercent = %f, want 20.0 (last entry)", limits.Primary.UsedPercent)
	}

	// Token usage parsing
	usage, err := c.ParseSessionTokenUsage(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionTokenUsage: %v", err)
	}
	if usage == nil {
		t.Fatal("Expected token usage from session")
	}

	// Delta: last - first
	// Input: 2000-1000=1000, Cached: 400-200=200, Output: 1000-500=500, Reasoning: 200-100=100
	// Billable = (1000-200) + 500 + 100 = 1200
	if usage.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000 (delta)", usage.InputTokens)
	}
	if usage.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500 (delta)", usage.OutputTokens)
	}
}

// TestBackwardCompat_ProviderExecuteSignature verifies that all providers
// accept the same Execute(ctx, Task) signature. A compilation check.
func TestBackwardCompat_ProviderExecuteSignature(t *testing.T) {
	ctx := context.Background()
	task := Task{}

	// These calls verify the method signatures haven't changed.
	// They will return empty results since providers aren't configured,
	// but we're testing the interface, not the implementation.
	claude := NewClaudeWithPath(t.TempDir())
	_, _ = claude.Execute(ctx, task)

	codex := NewCodexWithPath(t.TempDir())
	_, _ = codex.Execute(ctx, task)

	copilot := NewCopilotWithPath(t.TempDir())
	_, _ = copilot.Execute(ctx, task)
}
