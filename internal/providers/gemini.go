// gemini.go implements the Provider interface for Google Gemini CLI.
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GeminiSession represents a Gemini CLI session JSON file.
type GeminiSession struct {
	SessionID   string          `json:"sessionId"`
	ProjectHash string          `json:"projectHash"`
	StartTime   string          `json:"startTime"`   // ISO 8601
	LastUpdated string          `json:"lastUpdated"` // ISO 8601
	Messages    []GeminiMessage `json:"messages"`
}

// GeminiMessage represents a single message in a Gemini session.
type GeminiMessage struct {
	ID        string       `json:"id"`
	Timestamp string       `json:"timestamp"` // ISO 8601
	Type      string       `json:"type"`      // "user" or "gemini"
	Tokens    *GeminiTokens `json:"tokens,omitempty"`
	Model     string       `json:"model,omitempty"`
}

// GeminiTokens holds per-message token counts from a Gemini session.
type GeminiTokens struct {
	Input    int64 `json:"input"`
	Output   int64 `json:"output"`
	Cached   int64 `json:"cached"`
	Thoughts int64 `json:"thoughts"`
	Tool     int64 `json:"tool"`
	Total    int64 `json:"total"`
}

// Gemini wraps the Gemini CLI as a provider.
type Gemini struct {
	dataPath              string // Path to ~/.gemini
	mu                    sync.RWMutex
	lastUsedPercentSource string
}

// NewGemini creates a Gemini CLI provider.
func NewGemini() *Gemini {
	home, _ := os.UserHomeDir()
	return &Gemini{
		dataPath: filepath.Join(home, ".gemini"),
	}
}

// NewGeminiWithPath creates a Gemini provider with a custom data path.
func NewGeminiWithPath(dataPath string) *Gemini {
	return &Gemini{
		dataPath: dataPath,
	}
}

// Name returns "gemini".
func (g *Gemini) Name() string {
	return "gemini"
}

// Execute runs a task via Gemini CLI.
func (g *Gemini) Execute(ctx context.Context, task Task) (Result, error) {
	// TODO: Implement - spawn gemini CLI process
	return Result{}, nil
}

// Cost returns Gemini's token pricing (cents per 1K tokens).
// Based on Gemini 2.5 Pro pricing.
func (g *Gemini) Cost() (inputCents, outputCents int64) {
	// Gemini 2.5 Pro: $1.25/M input, $10/M output (>200K context)
	// Per 1K: 0.125 cents input, 1 cent output
	return 13, 100 // in hundredths of a cent for precision
}

// DataPath returns the configured data path.
func (g *Gemini) DataPath() string {
	return g.dataPath
}

// ListSessionFiles finds all session JSON files under tmp/<project_hash>/chats/.
func (g *Gemini) ListSessionFiles() ([]string, error) {
	tmpDir := filepath.Join(g.dataPath, "tmp")
	var sessions []string

	err := filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".json") && strings.Contains(path, "chats") {
			sessions = append(sessions, path)
		}
		return nil
	})

	if os.IsNotExist(err) {
		return nil, nil
	}
	return sessions, err
}

// ParseSessionFile reads and parses a Gemini session JSON file.
func ParseGeminiSession(path string) (*GeminiSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading gemini session: %w", err)
	}

	var session GeminiSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parsing gemini session: %w", err)
	}

	return &session, nil
}

// SessionTokenUsage computes total token usage for a single session file.
// Sums tokens across all gemini messages in the session.
func SessionTokenUsage(session *GeminiSession) *GeminiTokens {
	total := &GeminiTokens{}
	for _, msg := range session.Messages {
		if msg.Type != "gemini" || msg.Tokens == nil {
			continue
		}
		total.Input += msg.Tokens.Input
		total.Output += msg.Tokens.Output
		total.Cached += msg.Tokens.Cached
		total.Thoughts += msg.Tokens.Thoughts
		total.Tool += msg.Tokens.Tool
		total.Total += msg.Tokens.Total
	}
	return total
}

// GetTodayUsage returns today's total token usage across all sessions.
func (g *Gemini) GetTodayUsage() (int64, error) {
	usage, _, err := g.getTodayUsageWithSource()
	return usage, err
}

func (g *Gemini) getTodayUsageWithSource() (int64, string, error) {
	today := time.Now().Format("2006-01-02")
	tokens, err := g.scanTokensSince(today)
	if err != nil {
		return 0, "", err
	}
	return tokens, "session-files", nil
}

// GetWeeklyUsage returns the last 7 days total token usage.
func (g *Gemini) GetWeeklyUsage() (int64, error) {
	usage, _, err := g.getWeeklyUsageWithSource()
	return usage, err
}

func (g *Gemini) getWeeklyUsageWithSource() (int64, string, error) {
	cutoff := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	tokens, err := g.scanTokensSince(cutoff)
	if err != nil {
		return 0, "", err
	}
	return tokens, "session-files", nil
}

// scanTokensSince walks session files and sums tokens for sessions
// whose startTime is on or after cutoffDate (YYYY-MM-DD).
func (g *Gemini) scanTokensSince(cutoffDate string) (int64, error) {
	sessions, err := g.ListSessionFiles()
	if err != nil {
		return 0, err
	}

	cutoff, err := time.Parse("2006-01-02", cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("parsing cutoff date: %w", err)
	}

	var total int64
	for _, path := range sessions {
		// Quick mtime filter: skip files not modified since cutoff
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			continue
		}

		session, err := ParseGeminiSession(path)
		if err != nil {
			continue // skip corrupt files
		}

		// Parse session start time to check if it's in range
		startTime, err := time.Parse(time.RFC3339, session.StartTime)
		if err != nil {
			startTime, err = time.Parse(time.RFC3339Nano, session.StartTime)
			if err != nil {
				continue
			}
		}
		sessionDate := startTime.Local().Format("2006-01-02")
		if sessionDate < cutoffDate {
			continue
		}

		usage := SessionTokenUsage(session)
		total += usage.Total
	}

	return total, nil
}

// GetUsedPercent calculates the used percentage based on mode and budget.
func (g *Gemini) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	if weeklyBudget <= 0 {
		g.setLastUsedPercentSource("")
		return 0, fmt.Errorf("invalid weekly budget: %d", weeklyBudget)
	}

	switch mode {
	case "daily":
		usage, source, err := g.getTodayUsageWithSource()
		if err != nil {
			g.setLastUsedPercentSource("")
			return 0, err
		}
		g.setLastUsedPercentSource(source)
		dailyBudget := weeklyBudget / 7
		if dailyBudget <= 0 {
			return 0, nil
		}
		return float64(usage) / float64(dailyBudget) * 100, nil

	case "weekly":
		usage, source, err := g.getWeeklyUsageWithSource()
		if err != nil {
			g.setLastUsedPercentSource("")
			return 0, err
		}
		g.setLastUsedPercentSource(source)
		return float64(usage) / float64(weeklyBudget) * 100, nil

	default:
		g.setLastUsedPercentSource("")
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// LastUsedPercentSource reports where the last GetUsedPercent call sourced data.
func (g *Gemini) LastUsedPercentSource() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastUsedPercentSource
}

func (g *Gemini) setLastUsedPercentSource(source string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lastUsedPercentSource = source
}
