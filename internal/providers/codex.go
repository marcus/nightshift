// codex.go implements the Provider interface for OpenAI Codex CLI.
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
	"sort"
	"strings"
	"time"
)

// CodexRateLimits represents the rate_limits object in Codex session JSONL.
type CodexRateLimits struct {
	Primary   *CodexRateLimit `json:"primary"`
	Secondary *CodexRateLimit `json:"secondary"`
}

// CodexRateLimit represents a single rate limit entry.
type CodexRateLimit struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowMinutes int64   `json:"window_minutes"`
	ResetsAt      int64   `json:"resets_at"` // Unix timestamp
}

// CodexTokenUsage represents token usage counters from a Codex session.
type CodexTokenUsage struct {
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
}

// CodexTokenCountInfo holds per-event and cumulative token usage.
type CodexTokenCountInfo struct {
	TotalTokenUsage *CodexTokenUsage `json:"total_token_usage"`
	LastTokenUsage  *CodexTokenUsage `json:"last_token_usage"`
}

// CodexSessionPayload represents the payload object in a Codex JSONL entry.
type CodexSessionPayload struct {
	Type       string               `json:"type"`
	Info       *CodexTokenCountInfo `json:"info,omitempty"`
	RateLimits *CodexRateLimits     `json:"rate_limits,omitempty"`
}

// CodexSessionEntry represents a line in Codex session JSONL.
// Codex wraps data in {"type":"event_msg","payload":{"type":"token_count","rate_limits":{...}}}.
type CodexSessionEntry struct {
	Type    string               `json:"type"`
	Payload *CodexSessionPayload `json:"payload,omitempty"`
}

// Codex wraps the Codex CLI as a provider.
type Codex struct {
	dataPath   string           // Path to ~/.codex
	rateLimits *CodexRateLimits // Cached rate limits
}

// NewCodex creates a Codex provider.
func NewCodex() *Codex {
	home, _ := os.UserHomeDir()
	return &Codex{
		dataPath: filepath.Join(home, ".codex"),
	}
}

// NewCodexWithPath creates a Codex provider with a custom data path.
func NewCodexWithPath(dataPath string) *Codex {
	return &Codex{
		dataPath: dataPath,
	}
}

// Name returns "codex".
func (c *Codex) Name() string {
	return "codex"
}

// Execute runs a task via Codex CLI.
func (c *Codex) Execute(ctx context.Context, task Task) (Result, error) {
	// TODO: Implement - spawn codex CLI process
	return Result{}, nil
}

// Cost returns Codex's token pricing (cents per 1K tokens).
// Based on GPT-4 pricing estimates.
func (c *Codex) Cost() (inputCents, outputCents int64) {
	// GPT-4: ~$10/M input, ~$30/M output (estimates)
	// Per 1K: 1 cent input, 3 cents output
	return 100, 300 // in hundredths of a cent for precision
}

// DataPath returns the configured data path.
func (c *Codex) DataPath() string {
	return c.dataPath
}

// ParseSessionJSONL reads a Codex session JSONL file and extracts rate limits.
// Returns the most recent rate_limits entry found in the file.
func (c *Codex) ParseSessionJSONL(path string) (*CodexRateLimits, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening codex session: %w", err)
	}
	defer file.Close()

	var latest *CodexRateLimits
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			line = bytes.TrimRight(line, "\r\n")
			if len(line) > 0 {
				var entry CodexSessionEntry
				if jsonErr := json.Unmarshal(line, &entry); jsonErr == nil {
					if entry.Payload != nil && entry.Payload.RateLimits != nil {
						latest = entry.Payload.RateLimits
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("reading codex session: %w", err)
		}
	}

	return latest, nil
}

// ListSessionFiles finds all session JSONL files under sessions/<year>/<month>/<day>/.
func (c *Codex) ListSessionFiles() ([]string, error) {
	sessionsDir := filepath.Join(c.dataPath, "sessions")
	var sessions []string

	err := filepath.WalkDir(sessionsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
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
		return nil, nil
	}
	return sessions, err
}

// FindMostRecentSession finds the most recent session file by modification time.
func (c *Codex) FindMostRecentSession() (string, error) {
	sessions, err := c.ListSessionFiles()
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", nil
	}

	// Sort by modification time, most recent first
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var files []fileInfo

	for _, s := range sessions {
		info, err := os.Stat(s)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: s, modTime: info.ModTime()})
	}

	if len(files) == 0 {
		return "", nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	return files[0].path, nil
}

// GetRateLimits retrieves the latest rate limits from the most recent session.
func (c *Codex) GetRateLimits() (*CodexRateLimits, error) {
	if c.rateLimits != nil {
		return c.rateLimits, nil
	}

	sessionPath, err := c.FindMostRecentSession()
	if err != nil {
		return nil, fmt.Errorf("finding session: %w", err)
	}
	if sessionPath == "" {
		return nil, nil // No sessions found
	}

	limits, err := c.ParseSessionJSONL(sessionPath)
	if err != nil {
		return nil, err
	}

	c.rateLimits = limits
	return limits, nil
}

// GetUsedPercent returns the used percentage based on mode.
// Uses Codex's own rate limit percentages as the authoritative source since our
// local billable token calculation may not match Codex's internal accounting
// (prompt caching, model launch bonuses, etc.).
// For daily mode: primary rate limit used_percent (5h window).
// For weekly mode: secondary rate limit used_percent.
// Falls back to token-based calculation only if rate limits are unavailable.
func (c *Codex) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	switch mode {
	case "daily":
		pct, err := c.GetPrimaryUsedPercent()
		if err == nil && pct > 0 {
			return pct, nil
		}
		// Fall back to token-based if no rate limit data
		if weeklyBudget > 0 {
			usage, err := c.GetTodayTokenUsage()
			if err == nil && usage != nil && usage.TotalTokens > 0 {
				dailyBudget := weeklyBudget / 7
				if dailyBudget > 0 {
					return float64(usage.TotalTokens) / float64(dailyBudget) * 100, nil
				}
			}
		}
		return 0, nil
	case "weekly":
		pct, err := c.GetSecondaryUsedPercent()
		if err == nil && pct > 0 {
			return pct, nil
		}
		// Fall back to token-based if no rate limit data
		if weeklyBudget > 0 {
			usage, err := c.GetWeeklyTokenUsage()
			if err == nil && usage != nil && usage.TotalTokens > 0 {
				return float64(usage.TotalTokens) / float64(weeklyBudget) * 100, nil
			}
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// UsageBreakdown contains both rate-limit and local token data for display.
type UsageBreakdown struct {
	PrimaryPct   float64        // 5h window used_percent from rate limit
	WeeklyPct    float64        // weekly used_percent from rate limit
	TodayTokens  *CodexTokenUsage // local billable tokens today
	WeeklyTokens *CodexTokenUsage // local billable tokens this week
}

// GetUsageBreakdown returns both rate limit percentages and local token counts
// so the budget display can show both authoritative and measured data.
func (c *Codex) GetUsageBreakdown() UsageBreakdown {
	var bd UsageBreakdown
	if pct, err := c.GetPrimaryUsedPercent(); err == nil {
		bd.PrimaryPct = pct
	}
	if pct, err := c.GetSecondaryUsedPercent(); err == nil {
		bd.WeeklyPct = pct
	}
	if usage, err := c.GetTodayTokenUsage(); err == nil {
		bd.TodayTokens = usage
	}
	if usage, err := c.GetWeeklyTokenUsage(); err == nil {
		bd.WeeklyTokens = usage
	}
	return bd
}

// GetPrimaryUsedPercent returns the primary (daily) used_percent.
func (c *Codex) GetPrimaryUsedPercent() (float64, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return 0, err
	}
	if limits == nil || limits.Primary == nil {
		return 0, nil
	}
	return limits.Primary.UsedPercent, nil
}

// GetSecondaryUsedPercent returns the secondary (weekly) used_percent.
func (c *Codex) GetSecondaryUsedPercent() (float64, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return 0, err
	}
	if limits == nil || limits.Secondary == nil {
		return 0, nil
	}
	return limits.Secondary.UsedPercent, nil
}

// GetResetTime returns the reset timestamp for the specified mode.
// mode: "daily" returns primary.resets_at, "weekly" returns secondary.resets_at.
func (c *Codex) GetResetTime(mode string) (time.Time, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return time.Time{}, err
	}
	if limits == nil {
		return time.Time{}, nil
	}

	switch mode {
	case "daily":
		if limits.Primary == nil {
			return time.Time{}, nil
		}
		return time.Unix(limits.Primary.ResetsAt, 0), nil
	case "weekly":
		if limits.Secondary == nil {
			return time.Time{}, nil
		}
		return time.Unix(limits.Secondary.ResetsAt, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// GetWindowMinutes returns the window duration for the specified mode.
func (c *Codex) GetWindowMinutes(mode string) (int64, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return 0, err
	}
	if limits == nil {
		return 0, nil
	}

	switch mode {
	case "daily":
		if limits.Primary == nil {
			return 0, nil
		}
		return limits.Primary.WindowMinutes, nil
	case "weekly":
		if limits.Secondary == nil {
			return 0, nil
		}
		return limits.Secondary.WindowMinutes, nil
	default:
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// RefreshRateLimits clears cached rate limits and re-reads from disk.
func (c *Codex) RefreshRateLimits() (*CodexRateLimits, error) {
	c.rateLimits = nil
	return c.GetRateLimits()
}

// ParseSessionTokenUsage reads a Codex session JSONL file and computes the
// session's billable token usage. Codex total_token_usage is cumulative within
// a session, so we take the delta between the first and last events. The result
// excludes cached input tokens since those don't count against rate limits.
// Returns nil if no token usage data is found (e.g. stub sessions).
func (c *Codex) ParseSessionTokenUsage(path string) (*CodexTokenUsage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening codex session: %w", err)
	}
	defer file.Close()

	var first, latest *CodexTokenUsage
	eventCount := 0
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			line = bytes.TrimRight(line, "\r\n")
			if len(line) > 0 {
				var entry CodexSessionEntry
				if jsonErr := json.Unmarshal(line, &entry); jsonErr == nil {
					if entry.Payload != nil && entry.Payload.Type == "token_count" &&
						entry.Payload.Info != nil && entry.Payload.Info.TotalTokenUsage != nil {
						eventCount++
						if first == nil {
							// Copy the first value
							v := *entry.Payload.Info.TotalTokenUsage
							first = &v
						}
						latest = entry.Payload.Info.TotalTokenUsage
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("reading codex session: %w", err)
		}
	}

	if latest == nil {
		return nil, nil
	}

	// For a single event, use its values directly as the session usage
	src := latest
	if eventCount > 1 {
		// Multiple events: compute delta (last - first = session usage)
		src = &CodexTokenUsage{
			InputTokens:           latest.InputTokens - first.InputTokens,
			CachedInputTokens:     latest.CachedInputTokens - first.CachedInputTokens,
			OutputTokens:          latest.OutputTokens - first.OutputTokens,
			ReasoningOutputTokens: latest.ReasoningOutputTokens - first.ReasoningOutputTokens,
		}
	}

	// Compute billable TotalTokens = non-cached input + output + reasoning
	nonCachedInput := src.InputTokens - src.CachedInputTokens
	return &CodexTokenUsage{
		InputTokens:           src.InputTokens,
		CachedInputTokens:     src.CachedInputTokens,
		OutputTokens:          src.OutputTokens,
		ReasoningOutputTokens: src.ReasoningOutputTokens,
		TotalTokens:           nonCachedInput + src.OutputTokens + src.ReasoningOutputTokens,
	}, nil
}

// FindMostRecentSessionWithData finds the most recent session file by
// modification time that actually contains token_count events with data.
// This avoids returning stub sessions (started then exited, no data).
func (c *Codex) FindMostRecentSessionWithData() (string, error) {
	sessions, err := c.ListSessionFiles()
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", nil
	}

	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var files []fileInfo

	for _, s := range sessions {
		info, err := os.Stat(s)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: s, modTime: info.ModTime()})
	}

	if len(files) == 0 {
		return "", nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	for _, f := range files {
		usage, err := c.ParseSessionTokenUsage(f.path)
		if err != nil {
			continue
		}
		if usage != nil {
			return f.path, nil
		}
	}

	return "", nil
}

// ListTodaySessionFiles returns session files for today's date.
// Codex stores sessions at sessions/YYYY/MM/DD/*.jsonl.
func (c *Codex) ListTodaySessionFiles() ([]string, error) {
	now := time.Now()
	todayDir := filepath.Join(
		c.dataPath, "sessions",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)

	entries, err := os.ReadDir(todayDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading today's session dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, filepath.Join(todayDir, e.Name()))
		}
	}
	return files, nil
}

// GetTodayTokens returns total tokens used across all sessions today.
// Satisfies the snapshots.CodexUsage interface.
func (c *Codex) GetTodayTokens() (int64, error) {
	usage, err := c.GetTodayTokenUsage()
	if err != nil {
		return 0, err
	}
	if usage == nil {
		return 0, nil
	}
	return usage.TotalTokens, nil
}

// GetWeeklyTokens returns total tokens used across all sessions in the last 7 days.
// Satisfies the snapshots.CodexUsage interface.
func (c *Codex) GetWeeklyTokens() (int64, error) {
	usage, err := c.GetWeeklyTokenUsage()
	if err != nil {
		return 0, err
	}
	if usage == nil {
		return 0, nil
	}
	return usage.TotalTokens, nil
}

// ListSessionFilesForDate returns session files for a specific date.
func (c *Codex) ListSessionFilesForDate(t time.Time) ([]string, error) {
	dateDir := filepath.Join(
		c.dataPath, "sessions",
		fmt.Sprintf("%04d", t.Year()),
		fmt.Sprintf("%02d", int(t.Month())),
		fmt.Sprintf("%02d", t.Day()),
	)

	entries, err := os.ReadDir(dateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading session dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, filepath.Join(dateDir, e.Name()))
		}
	}
	return files, nil
}

// GetWeeklyTokenUsage sums token usage across all sessions from the last 7 days.
func (c *Codex) GetWeeklyTokenUsage() (*CodexTokenUsage, error) {
	now := time.Now()
	var sum CodexTokenUsage
	found := false

	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i)
		files, err := c.ListSessionFilesForDate(date)
		if err != nil {
			continue
		}
		for _, f := range files {
			usage, err := c.ParseSessionTokenUsage(f)
			if err != nil || usage == nil {
				continue
			}
			found = true
			sum.InputTokens += usage.InputTokens
			sum.CachedInputTokens += usage.CachedInputTokens
			sum.OutputTokens += usage.OutputTokens
			sum.ReasoningOutputTokens += usage.ReasoningOutputTokens
			sum.TotalTokens += usage.TotalTokens
		}
	}

	if !found {
		return nil, nil
	}
	return &sum, nil
}

// GetTodayTokenUsage sums token usage across ALL sessions for today's date.
// Each session's cumulative total_token_usage is taken from the last
// token_count event in that file (since total_token_usage is cumulative
// within a session, the last event gives the session total).
func (c *Codex) GetTodayTokenUsage() (*CodexTokenUsage, error) {
	files, err := c.ListTodaySessionFiles()
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	var sum CodexTokenUsage
	found := false

	for _, f := range files {
		usage, err := c.ParseSessionTokenUsage(f)
		if err != nil {
			continue
		}
		if usage == nil {
			continue
		}
		found = true
		sum.InputTokens += usage.InputTokens
		sum.CachedInputTokens += usage.CachedInputTokens
		sum.OutputTokens += usage.OutputTokens
		sum.ReasoningOutputTokens += usage.ReasoningOutputTokens
		sum.TotalTokens += usage.TotalTokens
	}

	if !found {
		return nil, nil
	}
	return &sum, nil
}
