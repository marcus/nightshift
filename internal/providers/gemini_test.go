package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewGemini_Defaults(t *testing.T) {
	g := NewGemini()
	if g.Name() != "gemini" {
		t.Errorf("Name() = %q, want %q", g.Name(), "gemini")
	}
	if g.DataPath() == "" {
		t.Error("expected non-empty DataPath")
	}
}

func TestNewGeminiWithPath(t *testing.T) {
	g := NewGeminiWithPath("/custom/path")
	if g.DataPath() != "/custom/path" {
		t.Errorf("DataPath() = %q, want %q", g.DataPath(), "/custom/path")
	}
}

func TestGemini_Cost(t *testing.T) {
	g := NewGemini()
	input, output := g.Cost()
	if input <= 0 || output <= 0 {
		t.Errorf("Cost() = (%d, %d), want positive values", input, output)
	}
}

func TestParseGeminiSession_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	session := GeminiSession{
		SessionID:   "test-session",
		ProjectHash: "abc123",
		StartTime:   time.Now().Format(time.RFC3339),
		LastUpdated: time.Now().Format(time.RFC3339),
		Messages: []GeminiMessage{
			{ID: "1", Type: "user", Timestamp: time.Now().Format(time.RFC3339)},
			{
				ID: "2", Type: "gemini", Timestamp: time.Now().Format(time.RFC3339),
				Model: "gemini-2.5-pro",
				Tokens: &GeminiTokens{
					Input: 1000, Output: 200, Cached: 500, Thoughts: 50, Tool: 0, Total: 1250,
				},
			},
			{
				ID: "3", Type: "gemini", Timestamp: time.Now().Format(time.RFC3339),
				Model: "gemini-2.5-pro",
				Tokens: &GeminiTokens{
					Input: 2000, Output: 300, Cached: 1000, Thoughts: 100, Tool: 0, Total: 2400,
				},
			},
		},
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseGeminiSession(sessionPath)
	if err != nil {
		t.Fatalf("ParseGeminiSession error: %v", err)
	}

	if parsed.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", parsed.SessionID, "test-session")
	}
	if len(parsed.Messages) != 3 {
		t.Errorf("messages count = %d, want 3", len(parsed.Messages))
	}
}

func TestParseGeminiSession_NotExist(t *testing.T) {
	_, err := ParseGeminiSession("/nonexistent/path/session.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSessionTokenUsage(t *testing.T) {
	session := &GeminiSession{
		Messages: []GeminiMessage{
			{Type: "user"},
			{
				Type:   "gemini",
				Tokens: &GeminiTokens{Input: 1000, Output: 200, Total: 1200},
			},
			{
				Type:   "gemini",
				Tokens: &GeminiTokens{Input: 2000, Output: 300, Total: 2300},
			},
			{Type: "user"},
			{
				Type:   "gemini",
				Tokens: nil, // no tokens (e.g., tool call)
			},
		},
	}

	usage := SessionTokenUsage(session)
	if usage.Input != 3000 {
		t.Errorf("Input = %d, want 3000", usage.Input)
	}
	if usage.Output != 500 {
		t.Errorf("Output = %d, want 500", usage.Output)
	}
	if usage.Total != 3500 {
		t.Errorf("Total = %d, want 3500", usage.Total)
	}
}

func TestSessionTokenUsage_Empty(t *testing.T) {
	session := &GeminiSession{
		Messages: []GeminiMessage{
			{Type: "user"},
		},
	}
	usage := SessionTokenUsage(session)
	if usage.Total != 0 {
		t.Errorf("Total = %d, want 0", usage.Total)
	}
}

func TestGemini_ListSessionFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the expected directory structure
	chatDir := filepath.Join(tmpDir, "tmp", "projecthash", "chats")
	if err := os.MkdirAll(chatDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a session file
	sessionPath := filepath.Join(chatDir, "session-2026-02-17.json")
	if err := os.WriteFile(sessionPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a non-session file (should be ignored)
	otherPath := filepath.Join(chatDir, "other.txt")
	if err := os.WriteFile(otherPath, []byte("not a session"), 0644); err != nil {
		t.Fatal(err)
	}

	g := NewGeminiWithPath(tmpDir)
	sessions, err := g.ListSessionFiles()
	if err != nil {
		t.Fatalf("ListSessionFiles error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("session count = %d, want 1", len(sessions))
	}
	if len(sessions) > 0 && sessions[0] != sessionPath {
		t.Errorf("session path = %q, want %q", sessions[0], sessionPath)
	}
}

func TestGemini_ListSessionFiles_NoDir(t *testing.T) {
	g := NewGeminiWithPath("/nonexistent/path")
	sessions, err := g.ListSessionFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestGemini_GetUsedPercent_Daily(t *testing.T) {
	tmpDir := t.TempDir()

	// Create session with today's date
	chatDir := filepath.Join(tmpDir, "tmp", "proj", "chats")
	if err := os.MkdirAll(chatDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	session := GeminiSession{
		SessionID: "test",
		StartTime: now.Format(time.RFC3339),
		Messages: []GeminiMessage{
			{
				Type:   "gemini",
				Tokens: &GeminiTokens{Total: 10000},
			},
		},
	}
	data, _ := json.Marshal(session)
	sessionPath := filepath.Join(chatDir, "session-today.json")
	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	g := NewGeminiWithPath(tmpDir)
	// weeklyBudget = 70000 -> dailyBudget = 10000 -> 10000/10000 = 100%
	pct, err := g.GetUsedPercent("daily", 70000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}
	if pct != 100.0 {
		t.Errorf("pct = %.1f, want 100.0", pct)
	}
}

func TestGemini_GetUsedPercent_InvalidMode(t *testing.T) {
	g := NewGeminiWithPath(t.TempDir())
	_, err := g.GetUsedPercent("invalid", 70000)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestGemini_GetUsedPercent_InvalidBudget(t *testing.T) {
	g := NewGeminiWithPath(t.TempDir())
	_, err := g.GetUsedPercent("daily", 0)
	if err == nil {
		t.Error("expected error for zero budget")
	}
}

func TestGemini_LastUsedPercentSource(t *testing.T) {
	g := NewGemini()
	if src := g.LastUsedPercentSource(); src != "" {
		t.Errorf("initial source = %q, want empty", src)
	}

	// Trigger a GetUsedPercent call to set the source
	tmpDir := t.TempDir()
	g = NewGeminiWithPath(tmpDir)
	_, _ = g.GetUsedPercent("daily", 70000)
	if src := g.LastUsedPercentSource(); src != "session-files" {
		t.Errorf("source = %q, want %q", src, "session-files")
	}
}
