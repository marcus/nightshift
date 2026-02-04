// claude.go implements the Provider interface for Claude Code CLI.
package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StatsCache represents the stats-cache.json structure from Claude Code.
type StatsCache struct {
	DailyStats map[string]DailyStat `json:"dailyStats"` // Key is date "YYYY-MM-DD"
}

// DailyStat holds daily usage aggregates.
type DailyStat struct {
	MessageCount  int64             `json:"messageCount"`
	SessionCount  int64             `json:"sessionCount"`
	ToolCallCount int64             `json:"toolCallCount"`
	TokensByModel map[string]int64  `json:"tokensByModel"` // Model name -> token count
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
			return &StatsCache{DailyStats: make(map[string]DailyStat)}, nil
		}
		return nil, fmt.Errorf("reading stats-cache: %w", err)
	}

	var stats StatsCache
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("parsing stats-cache: %w", err)
	}

	if stats.DailyStats == nil {
		stats.DailyStats = make(map[string]DailyStat)
	}

	return &stats, nil
}

// ParseSessionJSONL reads a session JSONL file and returns token usage.
func ParseSessionJSONL(path string) (*TokenUsage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer file.Close()

	total := &TokenUsage{}
	scanner := bufio.NewScanner(file)
	// Handle large lines (some messages can be very long)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg SessionMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			// Skip malformed lines
			continue
		}

		if msg.Usage != nil {
			total.InputTokens += msg.Usage.InputTokens
			total.OutputTokens += msg.Usage.OutputTokens
			total.CacheReadInputTokens += msg.Usage.CacheReadInputTokens
			total.CacheCreationInputTokens += msg.Usage.CacheCreationInputTokens
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning session file: %w", err)
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
	if stat, ok := stats.DailyStats[today]; ok {
		return sumTokensByModel(stat.TokensByModel), nil
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

	var total int64
	now := time.Now()
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		if stat, ok := stats.DailyStats[date]; ok {
			total += sumTokensByModel(stat.TokensByModel)
		}
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

	if stat, ok := stats.DailyStats[date]; ok {
		return &stat, nil
	}
	return nil, nil
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
