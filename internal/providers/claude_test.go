package providers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseStatsCache_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	statsPath := filepath.Join(tmpDir, "stats-cache.json")

	content := `{
		"dailyStats": {
			"2026-02-03": {
				"messageCount": 42,
				"sessionCount": 5,
				"toolCallCount": 100,
				"tokensByModel": {
					"claude-opus-4-5-20251101": 150000,
					"claude-sonnet-4-20250514": 50000
				}
			},
			"2026-02-02": {
				"messageCount": 30,
				"sessionCount": 3,
				"toolCallCount": 80,
				"tokensByModel": {
					"claude-opus-4-5-20251101": 100000
				}
			}
		}
	}`

	if err := os.WriteFile(statsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stats, err := ParseStatsCache(statsPath)
	if err != nil {
		t.Fatalf("ParseStatsCache error: %v", err)
	}

	if len(stats.DailyStats) != 2 {
		t.Errorf("expected 2 daily stats, got %d", len(stats.DailyStats))
	}

	stat := stats.DailyStats["2026-02-03"]
	if stat.MessageCount != 42 {
		t.Errorf("MessageCount = %d, want 42", stat.MessageCount)
	}
	if stat.SessionCount != 5 {
		t.Errorf("SessionCount = %d, want 5", stat.SessionCount)
	}
	if stat.ToolCallCount != 100 {
		t.Errorf("ToolCallCount = %d, want 100", stat.ToolCallCount)
	}

	tokens := sumTokensByModel(stat.TokensByModel)
	if tokens != 200000 {
		t.Errorf("TotalTokens = %d, want 200000", tokens)
	}
}

func TestParseStatsCache_NotExist(t *testing.T) {
	stats, err := ParseStatsCache("/nonexistent/path/stats-cache.json")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if stats.DailyStats == nil {
		t.Error("expected initialized DailyStats map")
	}
	if len(stats.DailyStats) != 0 {
		t.Errorf("expected empty DailyStats, got %d entries", len(stats.DailyStats))
	}
}

func TestParseStatsCache_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	statsPath := filepath.Join(tmpDir, "stats-cache.json")

	if err := os.WriteFile(statsPath, []byte("invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseStatsCache(statsPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseSessionJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	// Simulated session JSONL with multiple messages
	content := `{"type":"user","content":"hello"}
{"type":"assistant","content":"Hi!","usage":{"inputTokens":100,"outputTokens":50,"cacheReadInputTokens":20,"cacheCreationInputTokens":10}}
{"type":"user","content":"how are you?"}
{"type":"assistant","content":"Good!","usage":{"inputTokens":200,"outputTokens":100,"cacheReadInputTokens":30,"cacheCreationInputTokens":0}}
`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	usage, err := ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL error: %v", err)
	}

	if usage.InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300", usage.InputTokens)
	}
	if usage.OutputTokens != 150 {
		t.Errorf("OutputTokens = %d, want 150", usage.OutputTokens)
	}
	if usage.CacheReadInputTokens != 50 {
		t.Errorf("CacheReadInputTokens = %d, want 50", usage.CacheReadInputTokens)
	}
	if usage.CacheCreationInputTokens != 10 {
		t.Errorf("CacheCreationInputTokens = %d, want 10", usage.CacheCreationInputTokens)
	}
	if usage.TotalTokens() != 510 {
		t.Errorf("TotalTokens = %d, want 510", usage.TotalTokens())
	}
}

func TestParseSessionJSONL_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	if err := os.WriteFile(sessionPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	usage, err := ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL error: %v", err)
	}
	if usage.TotalTokens() != 0 {
		t.Errorf("expected 0 tokens for empty file, got %d", usage.TotalTokens())
	}
}

func TestParseSessionJSONL_NoUsage(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.jsonl")

	content := `{"type":"user","content":"hello"}
{"type":"assistant","content":"Hi!"}
`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	usage, err := ParseSessionJSONL(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionJSONL error: %v", err)
	}
	if usage.TotalTokens() != 0 {
		t.Errorf("expected 0 tokens when no usage field, got %d", usage.TotalTokens())
	}
}

func TestClaudeProvider_GetTodayUsage(t *testing.T) {
	tmpDir := t.TempDir()
	statsPath := filepath.Join(tmpDir, "stats-cache.json")

	today := time.Now().Format("2006-01-02")
	content := `{
		"dailyStats": {
			"` + today + `": {
				"messageCount": 10,
				"sessionCount": 2,
				"toolCallCount": 25,
				"tokensByModel": {
					"claude-opus-4": 75000,
					"claude-sonnet-4": 25000
				}
			}
		}
	}`

	if err := os.WriteFile(statsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewClaudeWithPath(tmpDir)
	usage, err := provider.GetTodayUsage()
	if err != nil {
		t.Fatalf("GetTodayUsage error: %v", err)
	}

	if usage != 100000 {
		t.Errorf("GetTodayUsage = %d, want 100000", usage)
	}
}

func TestClaudeProvider_GetTodayUsage_NoData(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewClaudeWithPath(tmpDir)

	usage, err := provider.GetTodayUsage()
	if err != nil {
		t.Fatalf("GetTodayUsage error: %v", err)
	}
	if usage != 0 {
		t.Errorf("expected 0 usage for missing data, got %d", usage)
	}
}

func TestClaudeProvider_GetWeeklyUsage(t *testing.T) {
	tmpDir := t.TempDir()
	statsPath := filepath.Join(tmpDir, "stats-cache.json")

	now := time.Now()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	eightDaysAgo := now.AddDate(0, 0, -8).Format("2006-01-02") // Outside window

	content := `{
		"dailyStats": {
			"` + today + `": {"tokensByModel": {"model": 100000}},
			"` + yesterday + `": {"tokensByModel": {"model": 80000}},
			"` + twoDaysAgo + `": {"tokensByModel": {"model": 60000}},
			"` + eightDaysAgo + `": {"tokensByModel": {"model": 999999}}
		}
	}`

	if err := os.WriteFile(statsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewClaudeWithPath(tmpDir)
	usage, err := provider.GetWeeklyUsage()
	if err != nil {
		t.Fatalf("GetWeeklyUsage error: %v", err)
	}

	// Should only include last 7 days (today, yesterday, twoDaysAgo)
	expected := int64(100000 + 80000 + 60000)
	if usage != expected {
		t.Errorf("GetWeeklyUsage = %d, want %d", usage, expected)
	}
}

func TestClaudeProvider_GetUsedPercent_Daily(t *testing.T) {
	tmpDir := t.TempDir()
	statsPath := filepath.Join(tmpDir, "stats-cache.json")

	today := time.Now().Format("2006-01-02")
	content := `{
		"dailyStats": {
			"` + today + `": {"tokensByModel": {"model": 50000}}
		}
	}`

	if err := os.WriteFile(statsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewClaudeWithPath(tmpDir)

	// Weekly budget of 700000 means daily = 100000
	// Used 50000 of 100000 = 50%
	pct, err := provider.GetUsedPercent("daily", 700000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}

	if pct != 50.0 {
		t.Errorf("GetUsedPercent(daily) = %.2f, want 50.0", pct)
	}
}

func TestClaudeProvider_GetUsedPercent_Weekly(t *testing.T) {
	tmpDir := t.TempDir()
	statsPath := filepath.Join(tmpDir, "stats-cache.json")

	now := time.Now()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	content := `{
		"dailyStats": {
			"` + today + `": {"tokensByModel": {"model": 100000}},
			"` + yesterday + `": {"tokensByModel": {"model": 100000}}
		}
	}`

	if err := os.WriteFile(statsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewClaudeWithPath(tmpDir)

	// Weekly budget of 1000000, used 200000 = 20%
	pct, err := provider.GetUsedPercent("weekly", 1000000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}

	if pct != 20.0 {
		t.Errorf("GetUsedPercent(weekly) = %.2f, want 20.0", pct)
	}
}

func TestClaudeProvider_GetUsedPercent_InvalidMode(t *testing.T) {
	provider := NewClaudeWithPath(t.TempDir())
	_, err := provider.GetUsedPercent("monthly", 700000)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestClaudeProvider_GetUsedPercent_InvalidBudget(t *testing.T) {
	provider := NewClaudeWithPath(t.TempDir())
	_, err := provider.GetUsedPercent("daily", 0)
	if err == nil {
		t.Error("expected error for zero budget")
	}
	_, err = provider.GetUsedPercent("daily", -100)
	if err == nil {
		t.Error("expected error for negative budget")
	}
}

func TestClaudeProvider_GetDailyStats(t *testing.T) {
	tmpDir := t.TempDir()
	statsPath := filepath.Join(tmpDir, "stats-cache.json")

	content := `{
		"dailyStats": {
			"2026-02-03": {
				"messageCount": 42,
				"sessionCount": 5,
				"toolCallCount": 100,
				"tokensByModel": {"model": 200000}
			}
		}
	}`

	if err := os.WriteFile(statsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewClaudeWithPath(tmpDir)

	stat, err := provider.GetDailyStats("2026-02-03")
	if err != nil {
		t.Fatalf("GetDailyStats error: %v", err)
	}
	if stat == nil {
		t.Fatal("expected non-nil stat")
	}
	if stat.MessageCount != 42 {
		t.Errorf("MessageCount = %d, want 42", stat.MessageCount)
	}

	// Non-existent date
	stat, err = provider.GetDailyStats("2020-01-01")
	if err != nil {
		t.Fatalf("GetDailyStats error: %v", err)
	}
	if stat != nil {
		t.Error("expected nil stat for non-existent date")
	}
}

func TestClaudeProvider_ListSessionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects", "some-project")

	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some session files
	sessions := []string{
		filepath.Join(projectsDir, "session1.jsonl"),
		filepath.Join(projectsDir, "session2.jsonl"),
	}
	for _, s := range sessions {
		if err := os.WriteFile(s, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a non-JSONL file (should be ignored)
	if err := os.WriteFile(filepath.Join(projectsDir, "notes.txt"), []byte("notes"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := NewClaudeWithPath(tmpDir)
	files, err := provider.ListSessionFiles()
	if err != nil {
		t.Fatalf("ListSessionFiles error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 session files, got %d", len(files))
	}
}

func TestClaudeProvider_ListSessionFiles_NoProjects(t *testing.T) {
	provider := NewClaudeWithPath(t.TempDir())
	files, err := provider.ListSessionFiles()
	if err != nil {
		t.Fatalf("ListSessionFiles error: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for missing projects dir, got %v", files)
	}
}

func TestClaudeProvider_Name(t *testing.T) {
	provider := NewClaude()
	if provider.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "claude")
	}
}

func TestClaudeProvider_Cost(t *testing.T) {
	provider := NewClaude()
	input, output := provider.Cost()
	if input != 150 {
		t.Errorf("input cost = %d, want 150", input)
	}
	if output != 750 {
		t.Errorf("output cost = %d, want 750", output)
	}
}

func TestClaudeProvider_DataPath(t *testing.T) {
	path := "/custom/path"
	provider := NewClaudeWithPath(path)
	if provider.DataPath() != path {
		t.Errorf("DataPath() = %q, want %q", provider.DataPath(), path)
	}
}

func TestTokenUsage_TotalTokens(t *testing.T) {
	usage := &TokenUsage{
		InputTokens:              100,
		OutputTokens:             50,
		CacheReadInputTokens:     25,
		CacheCreationInputTokens: 10,
	}
	if usage.TotalTokens() != 185 {
		t.Errorf("TotalTokens() = %d, want 185", usage.TotalTokens())
	}
}
