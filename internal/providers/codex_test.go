package providers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodexParseSessionJSONL_WithRateLimits(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	// Simulated Codex session JSONL with rate_limits
	content := `{"type":"user","content":"hello"}
{"type":"assistant","content":"Hi!"}
{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}}}
{"type":"user","content":"another message"}
{"rate_limits":{"primary":{"used_percent":45.5,"window_minutes":300,"resets_at":1769896400},"secondary":{"used_percent":12.5,"window_minutes":10080,"resets_at":1770483200}}}
`

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

	content := `{"type":"user","content":"hello"}
{"type":"assistant","content":"Hi!"}
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
{"rate_limits":{"primary":{"used_percent":20.0,"window_minutes":300,"resets_at":1769896359}}}
more invalid json
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

	// Create session files
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

	// Create sessions with different modification times
	older := filepath.Join(sessionsDir, "older.jsonl")
	newer := filepath.Join(sessionsDir, "newer.jsonl")

	if err := os.WriteFile(older, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	// Set older file to past time
	pastTime := time.Now().Add(-time.Hour)
	os.Chtimes(older, pastTime, pastTime)

	// Create newer file
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
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}}}`

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

func TestCodexGetUsedPercent_Daily(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}}}`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	pct, err := provider.GetUsedPercent("daily")
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}

	if pct != 34.0 {
		t.Errorf("GetUsedPercent(daily) = %.1f, want 34.0", pct)
	}
}

func TestCodexGetUsedPercent_Weekly(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "02", "03")

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(sessionsDir, "session.jsonl")
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}}}`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	pct, err := provider.GetUsedPercent("weekly")
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
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}}}`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewCodexWithPath(tmpDir)
	_, err := provider.GetUsedPercent("monthly")
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestCodexGetUsedPercent_NoData(t *testing.T) {
	provider := NewCodexWithPath(t.TempDir())
	pct, err := provider.GetUsedPercent("daily")
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
	content := `{"rate_limits":{"primary":{"used_percent":45.5,"window_minutes":300,"resets_at":1769896359}}}`

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
	content := `{"rate_limits":{"secondary":{"used_percent":15.5,"window_minutes":10080,"resets_at":1770483159}}}`

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
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}}}`

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
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}}}`

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
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}}}`

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
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359},"secondary":{"used_percent":10.0,"window_minutes":10080,"resets_at":1770483159}}}`

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
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}}}`

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
	content := `{"rate_limits":{"primary":{"used_percent":34.0,"window_minutes":300,"resets_at":1769896359}}}`

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
	newContent := `{"rate_limits":{"primary":{"used_percent":50.0,"window_minutes":300,"resets_at":1769896400}}}`
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
