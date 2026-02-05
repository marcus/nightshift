// claude.go implements the Provider interface for Claude Code CLI.
package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StatsCache represents the stats-cache.json structure from Claude Code.
// The file uses two arrays: dailyActivity (message/session/tool counts)
// and dailyModelTokens (per-model token counts keyed by date).
type StatsCache struct {
	Version          int                `json:"version"`
	DailyActivity    []DailyActivity    `json:"dailyActivity"`
	DailyModelTokens []DailyModelTokens `json:"dailyModelTokens"`
}

// DailyActivity holds per-day message/session/tool counts.
type DailyActivity struct {
	Date          string `json:"date"` // "YYYY-MM-DD"
	MessageCount  int64  `json:"messageCount"`
	SessionCount  int64  `json:"sessionCount"`
	ToolCallCount int64  `json:"toolCallCount"`
}

// DailyModelTokens holds per-day token counts by model.
type DailyModelTokens struct {
	Date          string           `json:"date"`          // "YYYY-MM-DD"
	TokensByModel map[string]int64 `json:"tokensByModel"` // model name -> token count
}

// DailyStat is a convenience view combining activity and tokens for a date.
type DailyStat struct {
	Date          string
	MessageCount  int64
	SessionCount  int64
	ToolCallCount int64
	TokensByModel map[string]int64
}

// SessionMessage represents a single message in a session JSONL file.
type SessionMessage struct {
	Type  string       `json:"type"`
	Usage *TokenUsage  `json:"usage,omitempty"`
}

// TokenUsage holds per-message token counts.
type TokenUsage struct {
	InputTokens              int64 `json:"inputTokens"`
	OutputTokens             int64 `json:"outputTokens"`
	CacheReadInputTokens     int64 `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int64 `json:"cacheCreationInputTokens"`
}

// TotalTokens returns the sum of all token fields.
func (u *TokenUsage) TotalTokens() int64 {
	return u.InputTokens + u.OutputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
}

// Claude wraps the Claude Code CLI as a provider.
type Claude struct {
	dataPath   string      // Path to ~/.claude
	statsCache *StatsCache // Cached stats data
}

// NewClaude creates a Claude Code provider.
func NewClaude() *Claude {
	home, _ := os.UserHomeDir()
	return &Claude{
		dataPath: filepath.Join(home, ".claude"),
	}
}

// NewClaudeWithPath creates a Claude provider with a custom data path.
func NewClaudeWithPath(dataPath string) *Claude {
	return &Claude{
		dataPath: dataPath,
	}
}

// Name returns "claude".
func (c *Claude) Name() string {
	return "claude"
}

// Execute runs a task via Claude Code CLI.
func (c *Claude) Execute(ctx context.Context, task Task) (Result, error) {
	// TODO: Implement - spawn claude CLI process
	return Result{}, nil
}

// Cost returns Claude's token pricing (cents per 1K tokens).
// Based on Claude Opus 4 pricing.
func (c *Claude) Cost() (inputCents, outputCents int64) {
	// Claude Opus 4: $15/M input, $75/M output
	// Per 1K: 1.5 cents input, 7.5 cents output
	return 150, 750 // in hundredths of a cent for precision
}

// ParseStatsCache reads and parses the stats-cache.json file.
func (c *Claude) ParseStatsCache() (*StatsCache, error) {
	return ParseStatsCache(filepath.Join(c.dataPath, "stats-cache.json"))
}

// ParseStatsCache reads stats-cache.json from a specific path.
func ParseStatsCache(path string) (*StatsCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &StatsCache{}, nil
		}
		return nil, fmt.Errorf("reading stats-cache: %w", err)
	}

	var stats StatsCache
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("parsing stats-cache: %w", err)
	}

	return &stats, nil
}

// TokensByDate returns a map of date -> total tokens across all models.
func (s *StatsCache) TokensByDate() map[string]int64 {
	m := make(map[string]int64, len(s.DailyModelTokens))
	for _, entry := range s.DailyModelTokens {
		m[entry.Date] = sumTokensByModel(entry.TokensByModel)
	}
	return m
}

// GetDailyStat builds a combined DailyStat for a given date.
func (s *StatsCache) GetDailyStat(date string) *DailyStat {
	stat := &DailyStat{Date: date}
	found := false

	for _, a := range s.DailyActivity {
		if a.Date == date {
			stat.MessageCount = a.MessageCount
			stat.SessionCount = a.SessionCount
			stat.ToolCallCount = a.ToolCallCount
			found = true
			break
		}
	}

	for _, t := range s.DailyModelTokens {
		if t.Date == date {
			stat.TokensByModel = t.TokensByModel
			found = true
			break
		}
	}

	if !found {
		return nil
	}
	return stat
}

// ParseSessionJSONL reads a session JSONL file and returns token usage.
func ParseSessionJSONL(path string) (*TokenUsage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer func() { _ = file.Close() }()

	total := &TokenUsage{}
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			line = bytes.TrimRight(line, "\r\n")
			if len(line) > 0 {
				var msg SessionMessage
				if jsonErr := json.Unmarshal(line, &msg); jsonErr == nil && msg.Usage != nil {
					total.InputTokens += msg.Usage.InputTokens
					total.OutputTokens += msg.Usage.OutputTokens
					total.CacheReadInputTokens += msg.Usage.CacheReadInputTokens
					total.CacheCreationInputTokens += msg.Usage.CacheCreationInputTokens
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("reading session file: %w", err)
		}
	}

	return total, nil
}

// GetTodayUsage returns today's total token usage from stats-cache.
func (c *Claude) GetTodayUsage() (int64, error) {
	stats, err := c.ParseStatsCache()
	if err != nil {
		return 0, err
	}
	c.statsCache = stats

	today := time.Now().Format("2006-01-02")
	byDate := stats.TokensByDate()
	if tokens, ok := byDate[today]; ok {
		return tokens, nil
	}
	return 0, nil
}

// GetWeeklyUsage returns the last 7 days total token usage.
func (c *Claude) GetWeeklyUsage() (int64, error) {
	stats, err := c.ParseStatsCache()
	if err != nil {
		return 0, err
	}
	c.statsCache = stats

	byDate := stats.TokensByDate()
	var total int64
	now := time.Now()
	for i := range 7 {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		total += byDate[date]
	}
	return total, nil
}

// GetUsedPercent calculates the used percentage based on mode and budget.
// mode: "daily" or "weekly"
// weeklyBudget: total weekly token budget
func (c *Claude) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	if weeklyBudget <= 0 {
		return 0, fmt.Errorf("invalid weekly budget: %d", weeklyBudget)
	}

	switch mode {
	case "daily":
		usage, err := c.GetTodayUsage()
		if err != nil {
			return 0, err
		}
		dailyBudget := weeklyBudget / 7
		if dailyBudget <= 0 {
			return 0, nil
		}
		return float64(usage) / float64(dailyBudget) * 100, nil

	case "weekly":
		usage, err := c.GetWeeklyUsage()
		if err != nil {
			return 0, err
		}
		return float64(usage) / float64(weeklyBudget) * 100, nil

	default:
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// GetDailyStats returns usage stats for a specific date.
func (c *Claude) GetDailyStats(date string) (*DailyStat, error) {
	stats, err := c.ParseStatsCache()
	if err != nil {
		return nil, err
	}

	return stats.GetDailyStat(date), nil
}

// ListSessionFiles finds all session JSONL files under the projects directory.
func (c *Claude) ListSessionFiles() ([]string, error) {
	projectsDir := filepath.Join(c.dataPath, "projects")
	var sessions []string

	err := filepath.WalkDir(projectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't access
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".jsonl") {
			sessions = append(sessions, path)
		}
		return nil
	})

	if os.IsNotExist(err) {
		return nil, nil // No projects directory yet
	}
	return sessions, err
}

// sumTokensByModel sums all token counts across models.
func sumTokensByModel(tokensByModel map[string]int64) int64 {
	var total int64
	for _, count := range tokensByModel {
		total += count
	}
	return total
}

// DataPath returns the configured data path.
func (c *Claude) DataPath() string {
	return c.dataPath
}
