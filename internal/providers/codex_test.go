package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// codexRateLimitsJSON returns a Codex JSONL line with rate_limits in the real
// nested format: {"type":"event_msg","payload":{"type":"token_count","rate_limits":{...}}}.
func codexRateLimitsJSON(primary, secondary string) string {
	parts := []string{}
	if primary != "" {
		parts = append(parts, `"primary":`+primary)
	}
	if secondary != "" {
		parts = append(parts, `"secondary":`+secondary)
	}
	return `{"type":"event_msg","payload":{"type":"token_count","rate_limits":{` + strings.Join(parts, ",") + `}}}`
}

func TestCodexParseSessionJSONL_WithRateLimits(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	content := `{"type":"session_meta","payload":{"id":"test"}}
{"type":"response_item","payload":{"type":"message","role":"user"}}
` + codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	) + "\n" + `{"type":"response_item","payload":{"type":"message","role":"user"}}
` + codexRateLimitsJSON(
		`{"used_percent":45.5,"window_minutes":300,"resets_at":1769896400}`,
		`{"used_percent":12.5,"window_minutes":10080,"resets_at":1770483200}`,
	) + "\n"

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	limits, err := provider.ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL error: %v", err)
	}

	if limits == nil {
		t.Fatal("expected non-nil rate limits")
	}

	// Should have the most recent values (second rate_limits entry)
	if limits.Primary.UsedPercent != 45.5 {
		t.Errorf("Primary.UsedPercent = %.1f, want 45.5", limits.Primary.UsedPercent)
	}
	if limits.Primary.WindowMinutes != 300 {
		t.Errorf("Primary.WindowMinutes = %d, want 300", limits.Primary.WindowMinutes)
	}
	if limits.Primary.ResetsAt != 1769896400 {
		t.Errorf("Primary.ResetsAt = %d, want 1769896400", limits.Primary.ResetsAt)
	}

	if limits.Secondary.UsedPercent != 12.5 {
		t.Errorf("Secondary.UsedPercent = %.1f, want 12.5", limits.Secondary.UsedPercent)
	}
	if limits.Secondary.WindowMinutes != 10080 {
		t.Errorf("Secondary.WindowMinutes = %d, want 10080", limits.Secondary.WindowMinutes)
	}
	if limits.Secondary.ResetsAt != 1770483200 {
		t.Errorf("Secondary.ResetsAt = %d, want 1770483200", limits.Secondary.ResetsAt)
	}
}

func TestCodexParseSessionJSONL_NoRateLimits(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	content := `{"type":"session_meta","payload":{"id":"test"}}
{"type":"response_item","payload":{"type":"message","role":"user"}}
`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	limits, err := provider.ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL error: %v", err)
	}

	if limits != nil {
		t.Error("expected nil rate limits when none present")
	}
}

func TestCodexParseSessionJSONL_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	if err := os.WriteFile(sessionPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	limits, err := provider.ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL error: %v", err)
	}

	if limits != nil {
		t.Error("expected nil rate limits for empty file")
	}
}

func TestCodexParseSessionJSONL_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	content := `invalid json line
` + codexRateLimitsJSON(
		`{"used_percent":20.0,"window_minutes":300,"resets_at":1769896359}`,
		"",
	) + "\n" + `more invalid json
`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	limits, err := provider.ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL error: %v", err)
	}

	if limits == nil || limits.Primary == nil {
		t.Fatal("expected rate limits despite malformed lines")
	}
	if limits.Primary.UsedPercent != 20.0 {
		t.Errorf("Primary.UsedPercent = %.1f, want 20.0", limits.Primary.UsedPercent)
	}
}

func TestCodexListSessionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessions := []string{
		filepath.Join(sessionsDir, "session1.jsonl"),
		filepath.Join(sessionsDir, "session2.jsonl"),
	}
	for _, s := range sessions {
		if err := os.WriteFile(s, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a non-JSONL file
	if err := os.WriteFile(filepath.Join(sessionsDir, "notes.txt"), []byte("notes"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	files, err := provider.ListSessionFiles()
	if err != nil {
		t.Fatalf("ListSessionFiles error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 session files, got %d", len(files))
	}
}

func TestCodexListSessionFiles_NoSessions(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	files, err := provider.ListSessionFiles()
	if err != nil {
		t.Fatalf("ListSessionFiles error: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for missing sessions dir, got %v", files)
	}
}

func TestCodexFindMostRecentSession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	older := filepath.Join(sessionsDir, "older.jsonl")
	newer := filepath.Join(sessionsDir, "newer.jsonl")

	if err := os.WriteFile(older, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	pastTime := time.Now().Add(-time.Hour)
	os.Chtimes(older, pastTime, pastTime)

	if err := os.WriteFile(newer, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	recent, err := provider.FindMostRecentSession()
	if err != nil {
		t.Fatalf("FindMostRecentSession error: %v", err)
	}

	if recent != newer {
		t.Errorf("FindMostRecentSession = %q, want %q", recent, newer)
	}
}

func TestCodexFindMostRecentSession_NoSessions(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	recent, err := provider.FindMostRecentSession()
	if err != nil {
		t.Fatalf("FindMostRecentSession error: %v", err)
	}
	if recent != "" {
		t.Errorf("expected empty string for no sessions, got %q", recent)
	}
}

func TestCodexGetRateLimits(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	limits, err := provider.GetRateLimits()
	if err != nil {
		t.Fatalf("GetRateLimits error: %v", err)
	}

	if limits == nil {
		t.Fatal("expected non-nil rate limits")
	}
	if limits.Primary.UsedPercent != 34.0 {
		t.Errorf("Primary.UsedPercent = %.1f, want 34.0", limits.Primary.UsedPercent)
	}
}

func TestCodexGetRateLimits_NoSessions(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	limits, err := provider.GetRateLimits()
	if err != nil {
		t.Fatalf("GetRateLimits error: %v", err)
	}
	if limits != nil {
		t.Error("expected nil rate limits when no sessions")
	}
}

func TestCodexGetUsedPercent_Daily_RateLimitFallback(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// No today sessions with token data, so should fall back to rate limit
	provider := NewCodexWithPath(tmpDir)
	pct, err := provider.GetUsedPercent("daily", 700000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}

	if pct != 34.0 {
		t.Errorf("GetUsedPercent(daily) = %.1f, want 34.0 (rate limit fallback)", pct)
	}
}

func TestCodexGetUsedPercent_Daily_PrefersRateLimit(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()
	todayDir := filepath.Join(
		tmpDir, "sessions",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Session has both token data and rate limits (used_percent: 5.0).
	// Rate limit should be preferred over token-based calculation.
	sessionPath := filepath.Join(todayDir, "session.jsonl")
	content := `{"type":"session_meta","payload":{"id":"test"}}
` + codexTokenCountJSON(5000, 4000, 1000, 200, 6200) + "\n"
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	pct, err := provider.GetUsedPercent("daily", 700000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}

	// Rate limit used_percent=5.0 is preferred over token-based (2.2%)
	if pct != 5.0 {
		t.Errorf("GetUsedPercent(daily) = %.1f, want 5.0 (rate limit preferred)", pct)
	}
}

func TestCodexGetUsedPercent_Daily_TokenFallback(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()
	todayDir := filepath.Join(
		tmpDir, "sessions",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Session with token data but no rate limits — should fall back to tokens
	sessionPath := filepath.Join(todayDir, "session.jsonl")
	content := `{"type":"session_meta","payload":{"id":"test"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":5000,"cached_input_tokens":4000,"output_tokens":1000,"reasoning_output_tokens":200,"total_tokens":6200},"last_token_usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":20,"reasoning_output_tokens":0,"total_tokens":120}}}}
`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	pct, err := provider.GetUsedPercent("daily", 700000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}

	// No rate limits available, falls back to token-based:
	// dailyBudget = 700000/7 = 100000
	// billable = (5000-4000) + 1000 + 200 = 2200
	// pct = 2200/100000 * 100 = 2.2%
	expectedPct := float64(2200) / float64(100000) * 100
	if pct != expectedPct {
		t.Errorf("GetUsedPercent(daily) = %.4f, want %.4f (token fallback)", pct, expectedPct)
	}
}

func TestCodexGetUsedPercent_Weekly(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	pct, err := provider.GetUsedPercent("weekly", 700000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}

	if pct != 10.0 {
		t.Errorf("GetUsedPercent(weekly) = %.1f, want 10.0", pct)
	}
}

func TestCodexGetUsedPercent_InvalidMode(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	_, err := provider.GetUsedPercent("monthly", 700000)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestCodexGetUsedPercent_NoData(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	pct, err := provider.GetUsedPercent("daily", 700000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}
	if pct != 0 {
		t.Errorf("expected 0 for no data, got %.1f", pct)
	}
}

func TestCodexGetPrimaryUsedPercent(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":45.5,"window_minutes":300,"resets_at":1769896359}`,
		"",
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	pct, err := provider.GetPrimaryUsedPercent()
	if err != nil {
		t.Fatalf("GetPrimaryUsedPercent error: %v", err)
	}

	if pct != 45.5 {
		t.Errorf("GetPrimaryUsedPercent = %.1f, want 45.5", pct)
	}
}

func TestCodexGetSecondaryUsedPercent(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		"",
		`{"used_percent":15.5,"window_minutes":10080,"resets_at":1770483159}`,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	pct, err := provider.GetSecondaryUsedPercent()
	if err != nil {
		t.Fatalf("GetSecondaryUsedPercent error: %v", err)
	}

	if pct != 15.5 {
		t.Errorf("GetSecondaryUsedPercent = %.1f, want 15.5", pct)
	}
}

func TestCodexGetResetTime_Daily(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	resetTime, err := provider.GetResetTime("daily")
	if err != nil {
		t.Fatalf("GetResetTime error: %v", err)
	}

	expected := time.Unix(1769896359, 0)
	if !resetTime.Equal(expected) {
		t.Errorf("GetResetTime(daily) = %v, want %v", resetTime, expected)
	}
}

func TestCodexGetResetTime_Weekly(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	resetTime, err := provider.GetResetTime("weekly")
	if err != nil {
		t.Fatalf("GetResetTime error: %v", err)
	}

	expected := time.Unix(1770483159, 0)
	if !resetTime.Equal(expected) {
		t.Errorf("GetResetTime(weekly) = %v, want %v", resetTime, expected)
	}
}

func TestCodexGetResetTime_InvalidMode(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		"",
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	_, err := provider.GetResetTime("monthly")
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestCodexGetResetTime_NoData(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	resetTime, err := provider.GetResetTime("daily")
	if err != nil {
		t.Fatalf("GetResetTime error: %v", err)
	}
	if !resetTime.IsZero() {
		t.Errorf("expected zero time for no data, got %v", resetTime)
	}
}

func TestCodexGetWindowMinutes(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)

	dailyWindow, err := provider.GetWindowMinutes("daily")
	if err != nil {
		t.Fatalf("GetWindowMinutes(daily) error: %v", err)
	}
	if dailyWindow != 300 {
		t.Errorf("GetWindowMinutes(daily) = %d, want 300", dailyWindow)
	}

	weeklyWindow, err := provider.GetWindowMinutes("weekly")
	if err != nil {
		t.Fatalf("GetWindowMinutes(weekly) error: %v", err)
	}
	if weeklyWindow != 10080 {
		t.Errorf("GetWindowMinutes(weekly) = %d, want 10080", weeklyWindow)
	}
}

func TestCodexGetWindowMinutes_InvalidMode(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		"",
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	_, err := provider.GetWindowMinutes("monthly")
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestCodexRefreshRateLimits(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		"",
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)

	// First read
	limits, err := provider.GetRateLimits()
	if err != nil {
		t.Fatalf("GetRateLimits error: %v", err)
	}
	if limits.Primary.UsedPercent != 34.0 {
		t.Errorf("initial Primary.UsedPercent = %.1f, want 34.0", limits.Primary.UsedPercent)
	}

	// Update file
	newContent := codexRateLimitsJSON(
		`{"used_percent":50.0,"window_minutes":300,"resets_at":1769896400}`,
		"",
	)
	if err := os.WriteFile(sessionPath, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Cached value should still be 34.0
	limits, _ = provider.GetRateLimits()
	if limits.Primary.UsedPercent != 34.0 {
		t.Errorf("cached Primary.UsedPercent = %.1f, want 34.0", limits.Primary.UsedPercent)
	}

	// After refresh should be 50.0
	limits, err = provider.RefreshRateLimits()
	if err != nil {
		t.Fatalf("RefreshRateLimits error: %v", err)
	}
	if limits.Primary.UsedPercent != 50.0 {
		t.Errorf("refreshed Primary.UsedPercent = %.1f, want 50.0", limits.Primary.UsedPercent)
	}
}

func TestCodexProvider_Name(t *testing.T) {
	provider := NewCodex()
	if provider.Name() != "codex" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "codex")
	}
}

func TestCodexProvider_Cost(t *testing.T) {
	provider := NewCodex()
	input, output := provider.Cost()
	if input != 100 {
		t.Errorf("input cost = %d, want 100", input)
	}
	if output != 300 {
		t.Errorf("output cost = %d, want 300", output)
	}
}

func TestCodexProvider_DataPath(t *testing.T) {
	path := "/custom/path"
	provider := NewCodexWithPath(path)
	if provider.DataPath() != path {
		t.Errorf("DataPath() = %q, want %q", provider.DataPath(), path)
	}
}

// codexTokenCountJSON returns a Codex JSONL line with token_count info in the
// real format: {"type":"event_msg","payload":{"type":"token_count","info":{...},"rate_limits":{...}}}.
func codexTokenCountJSON(inputTokens, cachedInput, outputTokens, reasoningOutput, totalTokens int64) string {
	return fmt.Sprintf(`{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":%d,"cached_input_tokens":%d,"output_tokens":%d,"reasoning_output_tokens":%d,"total_tokens":%d},"last_token_usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":20,"reasoning_output_tokens":0,"total_tokens":120}},"rate_limits":{"primary":{"used_percent":5.0,"window_minutes":300,"resets_at":1770283138}}}}`,
		inputTokens, cachedInput, outputTokens, reasoningOutput, totalTokens)
}

// codexTokenCountNoInfoJSON returns a token_count event with info: null (early session event).
func codexTokenCountNoInfoJSON() string {
	return `{"type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":0.0,"window_minutes":300,"resets_at":1770264913}}}}`
}

func TestCodexParseSessionTokenUsage(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	// Two cumulative events: first and last
	// Event 1: input=1000, cached=800, output=200, reasoning=50
	// Event 2: input=2000, cached=1600, output=400, reasoning=100
	// Delta:   input=1000, cached=800, output=200, reasoning=50
	// Billable: (1000-800) + 200 + 50 = 450
	content := `{"type":"session_meta","payload":{"id":"test"}}
{"type":"response_item","payload":{"type":"message","role":"user"}}
` + codexTokenCountJSON(1000, 800, 200, 50, 1250) + "\n" +
		codexTokenCountJSON(2000, 1600, 400, 100, 2500) + "\n"

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	usage, err := provider.ParseSessionTokenUsage(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionTokenUsage error: %v", err)
	}

	if usage == nil {
		t.Fatal("expected non-nil token usage")
	}
	// Returns delta between last and first events
	if usage.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000 (delta)", usage.InputTokens)
	}
	if usage.CachedInputTokens != 800 {
		t.Errorf("CachedInputTokens = %d, want 800 (delta)", usage.CachedInputTokens)
	}
	if usage.OutputTokens != 200 {
		t.Errorf("OutputTokens = %d, want 200 (delta)", usage.OutputTokens)
	}
	if usage.ReasoningOutputTokens != 50 {
		t.Errorf("ReasoningOutputTokens = %d, want 50 (delta)", usage.ReasoningOutputTokens)
	}
	// TotalTokens = billable = (1000-800) non-cached input + 200 output + 50 reasoning = 450
	if usage.TotalTokens != 450 {
		t.Errorf("TotalTokens = %d, want 450 (billable)", usage.TotalTokens)
	}
}

func TestCodexParseSessionTokenUsage_NoData(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	// Stub session with no token data
	content := `{"type":"session_meta","payload":{"id":"test"}}
{"type":"response_item","payload":{"type":"message","role":"user"}}
`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	usage, err := provider.ParseSessionTokenUsage(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionTokenUsage error: %v", err)
	}
	if usage != nil {
		t.Error("expected nil token usage for stub session")
	}
}

func TestCodexParseSessionTokenUsage_NullInfo(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	// Session with token_count events that have info: null
	content := `{"type":"session_meta","payload":{"id":"test"}}
` + codexTokenCountNoInfoJSON() + "\n"

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	usage, err := provider.ParseSessionTokenUsage(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionTokenUsage error: %v", err)
	}
	if usage != nil {
		t.Error("expected nil token usage when info is null")
	}
}

func TestCodexFindMostRecentSessionWithData(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "04")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create older session WITH token data
	withData := filepath.Join(sessionsDir, "with-data.jsonl")
	dataContent := `{"type":"session_meta","payload":{"id":"data-session"}}
` + codexTokenCountJSON(5000, 4000, 1000, 200, 6200) + "\n"
	if err := os.WriteFile(withData, []byte(dataContent), 0644); err != nil {
		t.Fatal(err)
	}
	pastTime := time.Now().Add(-time.Hour)
	os.Chtimes(withData, pastTime, pastTime)

	// Create newer stub session WITHOUT token data
	stub := filepath.Join(sessionsDir, "stub.jsonl")
	stubContent := `{"type":"session_meta","payload":{"id":"stub-session"}}
{"type":"response_item","payload":{"type":"message","role":"user"}}
`
	if err := os.WriteFile(stub, []byte(stubContent), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)

	// FindMostRecentSession returns the stub (newest by mtime)
	recent, err := provider.FindMostRecentSession()
	if err != nil {
		t.Fatalf("FindMostRecentSession error: %v", err)
	}
	if recent != stub {
		t.Errorf("FindMostRecentSession = %q, want stub %q", recent, stub)
	}

	// FindMostRecentSessionWithData should skip the stub and return the one with data
	recentWithData, err := provider.FindMostRecentSessionWithData()
	if err != nil {
		t.Fatalf("FindMostRecentSessionWithData error: %v", err)
	}
	if recentWithData != withData {
		t.Errorf("FindMostRecentSessionWithData = %q, want %q", recentWithData, withData)
	}
}

func TestCodexFindMostRecentSessionWithData_NoSessions(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	recent, err := provider.FindMostRecentSessionWithData()
	if err != nil {
		t.Fatalf("FindMostRecentSessionWithData error: %v", err)
	}
	if recent != "" {
		t.Errorf("expected empty string, got %q", recent)
	}
}

func TestCodexFindMostRecentSessionWithData_AllStubs(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "04")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	stub := filepath.Join(sessionsDir, "stub.jsonl")
	if err := os.WriteFile(stub, []byte(`{"type":"session_meta","payload":{"id":"s1"}}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	recent, err := provider.FindMostRecentSessionWithData()
	if err != nil {
		t.Fatalf("FindMostRecentSessionWithData error: %v", err)
	}
	if recent != "" {
		t.Errorf("expected empty string when all stubs, got %q", recent)
	}
}

func TestCodexGetTodayTokenUsage(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()
	todayDir := filepath.Join(
		tmpDir, "sessions",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Session 1: two events (delta computed)
	// Event 1: input=1000, cached=800, output=200, reasoning=50
	// Event 2: input=3000, cached=2400, output=600, reasoning=150
	// Delta: input=2000, cached=1600, output=400, reasoning=100
	// Billable: (2000-1600) + 400 + 100 = 900
	s1 := filepath.Join(todayDir, "session1.jsonl")
	s1Content := `{"type":"session_meta","payload":{"id":"s1"}}
` + codexTokenCountJSON(1000, 800, 200, 50, 1250) + "\n" +
		codexTokenCountJSON(3000, 2400, 600, 150, 3750) + "\n"
	if err := os.WriteFile(s1, []byte(s1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Session 2: single event (first == last)
	// input=500, cached=400, output=100, reasoning=25
	// Billable: (500-400) + 100 + 25 = 225
	s2 := filepath.Join(todayDir, "session2.jsonl")
	s2Content := `{"type":"session_meta","payload":{"id":"s2"}}
` + codexTokenCountJSON(500, 400, 100, 25, 625) + "\n"
	if err := os.WriteFile(s2, []byte(s2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Session 3: stub, no token data
	s3 := filepath.Join(todayDir, "session3.jsonl")
	if err := os.WriteFile(s3, []byte(`{"type":"session_meta","payload":{"id":"s3"}}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	usage, err := provider.GetTodayTokenUsage()
	if err != nil {
		t.Fatalf("GetTodayTokenUsage error: %v", err)
	}

	if usage == nil {
		t.Fatal("expected non-nil token usage")
	}

	// Session 1 delta: input=2000, cached=1600, output=400, reasoning=100
	// Session 2 single: input=500, cached=400, output=100, reasoning=25
	// Sum: input=2500, cached=2000, output=500, reasoning=125
	if usage.InputTokens != 2500 {
		t.Errorf("InputTokens = %d, want 2500", usage.InputTokens)
	}
	if usage.CachedInputTokens != 2000 {
		t.Errorf("CachedInputTokens = %d, want 2000", usage.CachedInputTokens)
	}
	if usage.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500", usage.OutputTokens)
	}
	if usage.ReasoningOutputTokens != 125 {
		t.Errorf("ReasoningOutputTokens = %d, want 125", usage.ReasoningOutputTokens)
	}
	// TotalTokens = sum of billable: 900 + 225 = 1125
	if usage.TotalTokens != 1125 {
		t.Errorf("TotalTokens = %d, want 1125 (billable)", usage.TotalTokens)
	}
}

func TestCodexGetTodayTokenUsage_NoSessions(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	usage, err := provider.GetTodayTokenUsage()
	if err != nil {
		t.Fatalf("GetTodayTokenUsage error: %v", err)
	}
	if usage != nil {
		t.Error("expected nil token usage when no sessions")
	}
}

func TestCodexGetTodayTokenUsage_AllStubs(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()
	todayDir := filepath.Join(
		tmpDir, "sessions",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		t.Fatal(err)
	}

	stub := filepath.Join(todayDir, "stub.jsonl")
	if err := os.WriteFile(stub, []byte(`{"type":"session_meta","payload":{"id":"s1"}}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	usage, err := provider.GetTodayTokenUsage()
	if err != nil {
		t.Fatalf("GetTodayTokenUsage error: %v", err)
	}
	if usage != nil {
		t.Error("expected nil token usage when all stubs")
	}
}

func TestCodexListTodaySessionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()
	todayDir := filepath.Join(
		tmpDir, "sessions",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"a.jsonl", "b.jsonl", "c.txt"} {
		if err := os.WriteFile(filepath.Join(todayDir, name), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	provider := NewCodexWithPath(tmpDir)
	files, err := provider.ListTodaySessionFiles()
	if err != nil {
		t.Fatalf("ListTodaySessionFiles error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 jsonl files, got %d", len(files))
	}
}

func TestCodexListTodaySessionFiles_NoDir(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	files, err := provider.ListTodaySessionFiles()
	if err != nil {
		t.Fatalf("ListTodaySessionFiles error: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for missing dir, got %v", files)
	}
}

func TestCodexGetWeeklyTokenUsage(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	// Create sessions across 3 days, each with a single event
	// input=1000, cached=800, output=200, reasoning=50
	// Billable per session: (1000-800) + 200 + 50 = 450
	for i := 0; i < 3; i++ {
		date := now.AddDate(0, 0, -i)
		dateDir := filepath.Join(
			tmpDir, "sessions",
			fmt.Sprintf("%04d", date.Year()),
			fmt.Sprintf("%02d", int(date.Month())),
			fmt.Sprintf("%02d", date.Day()),
		)
		if err := os.MkdirAll(dateDir, 0755); err != nil {
			t.Fatal(err)
		}

		sessionPath := filepath.Join(dateDir, "session.jsonl")
		content := `{"type":"session_meta","payload":{"id":"test"}}
` + codexTokenCountJSON(1000, 800, 200, 50, 1250) + "\n"
		if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	provider := NewCodexWithPath(tmpDir)
	usage, err := provider.GetWeeklyTokenUsage()
	if err != nil {
		t.Fatalf("GetWeeklyTokenUsage error: %v", err)
	}
	if usage == nil {
		t.Fatal("expected non-nil weekly usage")
	}

	// 3 sessions * 450 billable = 1350
	expectedTotal := int64(3 * 450)
	if usage.TotalTokens != expectedTotal {
		t.Errorf("TotalTokens = %d, want %d (billable)", usage.TotalTokens, expectedTotal)
	}
}

func TestCodexGetWeeklyTokenUsage_NoData(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	usage, err := provider.GetWeeklyTokenUsage()
	if err != nil {
		t.Fatalf("GetWeeklyTokenUsage error: %v", err)
	}
	if usage != nil {
		t.Error("expected nil for no data")
	}
}

func TestCodexGetTodayTokens(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()
	todayDir := filepath.Join(
		tmpDir, "sessions",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Single event: input=5000, cached=4000, output=1000, reasoning=200
	// Billable: (5000-4000) + 1000 + 200 = 2200
	sessionPath := filepath.Join(todayDir, "session.jsonl")
	content := `{"type":"session_meta","payload":{"id":"test"}}
` + codexTokenCountJSON(5000, 4000, 1000, 200, 6200) + "\n"
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	tokens, err := provider.GetTodayTokens()
	if err != nil {
		t.Fatalf("GetTodayTokens error: %v", err)
	}
	if tokens != 2200 {
		t.Errorf("GetTodayTokens = %d, want 2200 (billable)", tokens)
	}
}

func TestCodexGetWeeklyTokens(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()
	todayDir := filepath.Join(
		tmpDir, "sessions",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Single event: billable = 2200
	sessionPath := filepath.Join(todayDir, "session.jsonl")
	content := `{"type":"session_meta","payload":{"id":"test"}}
` + codexTokenCountJSON(5000, 4000, 1000, 200, 6200) + "\n"
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	tokens, err := provider.GetWeeklyTokens()
	if err != nil {
		t.Fatalf("GetWeeklyTokens error: %v", err)
	}
	if tokens != 2200 {
		t.Errorf("GetWeeklyTokens = %d, want 2200 (billable)", tokens)
	}
}

func TestCodexParseSessionJSONL_LargeLines(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	// Create a file with a line exceeding the old 1MB scanner limit
	bigContent := strings.Repeat("x", 2*1024*1024)
	content := `{"type":"session_meta","payload":{"id":"test"}}
{"type":"response_item","payload":{"type":"message","content":"` + bigContent + `"}}
` + codexRateLimitsJSON(
		`{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}`,
		`{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}`,
	) + "\n"

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	limits, err := provider.ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL with large line: %v", err)
	}

	if limits == nil {
		t.Fatal("expected non-nil rate limits despite large line")
	}
	if limits.Primary.UsedPercent != 34.0 {
		t.Errorf("Primary.UsedPercent = %.1f, want 34.0", limits.Primary.UsedPercent)
	}
}
