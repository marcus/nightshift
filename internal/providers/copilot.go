// copilot.go implements the Provider interface for GitHub Copilot CLI.
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Copilot wraps the GitHub Copilot CLI as a provider.
//
// Usage tracking approach:
// GitHub Copilot CLI does not expose usage metrics via API or local files like
// Codex or Claude. We track usage by counting requests made through this provider.
// Each request counts as 1 premium request (PRU). Premium requests reset monthly
// on the 1st at 00:00:00 UTC according to GitHub's documented behavior.
//
// PRU cost varies by model (multiplier applied to base request):
//   - GPT-4.1:        0x (free for most tasks)
//   - Claude Sonnet:   1x (standard)
//   - GPT-4o:          1x (standard)
//   - Claude Opus 4:  10x (expensive, use for complex tasks)
//   - o3-pro:         50x (very expensive)
//
// Limitations:
// - No authoritative usage data from GitHub API (not exposed)
// - Request counting only tracks usage through nightshift, not external usage
// - No way to query remaining quota from GitHub servers
// - PRU multiplier estimates may drift as GitHub updates pricing
type Copilot struct {
	dataPath string // Path to ~/.copilot for tracking data
	agent    CopilotExecutor
}

// CopilotExecutor is satisfied by agents.CopilotAgent (avoids import cycle).
type CopilotExecutor interface {
	Execute(ctx context.Context, opts CopilotExecOptions) (stdout string, exitCode int, err error)
}

// CopilotExecOptions mirrors agents.ExecuteOptions for the provider layer.
type CopilotExecOptions struct {
	Prompt  string
	WorkDir string
	Files   []string
}

// CopilotUsageData persists usage tracking between sessions.
type CopilotUsageData struct {
	RequestCount int64     `json:"request_count"`
	LastReset    time.Time `json:"last_reset"` // UTC timestamp of last monthly reset
	Month        string    `json:"month"`      // "YYYY-MM" of current tracking period
}

// PRUMultiplier maps model identifiers to their premium request multipliers.
// Source: GitHub Copilot billing documentation.
var PRUMultiplier = map[string]float64{
	"gpt-4.1":        0,
	"gpt-4.1-mini":   0.25,
	"gpt-4o":         1,
	"gpt-4o-mini":    0.25,
	"claude-sonnet":  1,
	"claude-haiku":   0.25,
	"claude-opus":    10,
	"o1":             10,
	"o3-mini":        0.33,
	"o3-pro":         50,
	"gemini-flash":   0.25,
	"gemini-pro":     1,
}

// EstimatePRUCost returns the PRU cost for a single request using the given model.
// Unknown models default to 1 PRU (standard rate).
func EstimatePRUCost(model string) float64 {
	if mult, ok := PRUMultiplier[model]; ok {
		return mult
	}
	return 1 // conservative default
}

// NewCopilot creates a Copilot provider with default data path.
func NewCopilot() *Copilot {
	home, _ := os.UserHomeDir()
	return &Copilot{
		dataPath: filepath.Join(home, ".copilot"),
	}
}

// NewCopilotWithPath creates a Copilot provider with a custom data path.
func NewCopilotWithPath(dataPath string) *Copilot {
	return &Copilot{
		dataPath: dataPath,
	}
}

// SetAgent attaches an executor (typically agents.CopilotAgent) for task execution.
func (c *Copilot) SetAgent(agent CopilotExecutor) {
	c.agent = agent
}

// Name returns "copilot".
func (c *Copilot) Name() string {
	return "copilot"
}

// Execute runs a task via GitHub Copilot CLI.
// Delegates to the attached CopilotExecutor (agent) and increments the request counter.
func (c *Copilot) Execute(ctx context.Context, task Task) (Result, error) {
	if c.agent == nil {
		return Result{}, fmt.Errorf("copilot agent not configured; call SetAgent first")
	}

	stdout, exitCode, err := c.agent.Execute(ctx, CopilotExecOptions{
		Prompt: task.Prompt,
	})

	// Track the request regardless of outcome
	_ = c.IncrementRequestCount()

	if err != nil {
		return Result{
			Output:   stdout,
			ExitCode: exitCode,
		}, fmt.Errorf("copilot execution: %w", err)
	}

	return Result{
		Output:   stdout,
		ExitCode: exitCode,
	}, nil
}

// Cost returns Copilot's token pricing (cents per 1K tokens).
// GitHub Copilot uses a request-based model, not token-based.
// Premium plans have monthly request limits, not per-token billing.
// We return 0 to indicate no per-token cost.
func (c *Copilot) Cost() (inputCents, outputCents int64) {
	return 0, 0
}

// DataPath returns the configured data path.
func (c *Copilot) DataPath() string {
	return c.dataPath
}

// usageFilePath returns the path to the usage tracking file.
func (c *Copilot) usageFilePath() string {
	return filepath.Join(c.dataPath, "nightshift-usage.json")
}

// LoadUsageData reads the usage tracking file from disk.
func (c *Copilot) LoadUsageData() (*CopilotUsageData, error) {
	path := c.usageFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No tracking file yet, return empty data with current month
			now := time.Now().UTC()
			return &CopilotUsageData{
				RequestCount: 0,
				LastReset:    firstOfMonth(now),
				Month:        now.Format("2006-01"),
			}, nil
		}
		return nil, fmt.Errorf("reading usage data: %w", err)
	}

	var usage CopilotUsageData
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil, fmt.Errorf("parsing usage data: %w", err)
	}

	return &usage, nil
}

// SaveUsageData writes the usage tracking file to disk.
func (c *Copilot) SaveUsageData(data *CopilotUsageData) error {
	// Ensure directory exists
	if err := os.MkdirAll(c.dataPath, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling usage data: %w", err)
	}

	path := c.usageFilePath()
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("writing usage data: %w", err)
	}

	return nil
}

// GetRequestCount returns the current request count for the current month.
// Automatically resets if we've crossed into a new month.
func (c *Copilot) GetRequestCount() (int64, error) {
	data, err := c.LoadUsageData()
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	currentMonth := now.Format("2006-01")

	// Check if we need to reset for a new month
	if data.Month != currentMonth {
		// New month, reset counter
		data.RequestCount = 0
		data.LastReset = firstOfMonth(now)
		data.Month = currentMonth
		if err := c.SaveUsageData(data); err != nil {
			return 0, err
		}
	}

	return data.RequestCount, nil
}

// IncrementRequestCount increments the request counter by 1.
// Should be called after each successful Copilot request.
func (c *Copilot) IncrementRequestCount() error {
	data, err := c.LoadUsageData()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	currentMonth := now.Format("2006-01")

	// Check if we need to reset for a new month
	if data.Month != currentMonth {
		// New month, reset counter
		data.RequestCount = 1
		data.LastReset = firstOfMonth(now)
		data.Month = currentMonth
	} else {
		data.RequestCount++
	}

	return c.SaveUsageData(data)
}

// GetUsedPercent returns the used percentage based on mode and monthly request limit.
// mode: "daily" or "weekly" (Copilot resets monthly, so both modes use the same calculation)
// monthlyLimit: maximum premium requests per month (typically from plan limit)
//
// Note: GitHub Copilot resets monthly on the 1st at 00:00:00 UTC, not daily or weekly.
// For daily mode, we estimate daily usage as (monthly_requests / days_in_month).
// For weekly mode, we estimate weekly usage similarly.
func (c *Copilot) GetUsedPercent(mode string, monthlyLimit int64) (float64, error) {
	if monthlyLimit <= 0 {
		return 0, fmt.Errorf("invalid monthly limit: %d", monthlyLimit)
	}

	requests, err := c.GetRequestCount()
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()

	switch mode {
	case "daily":
		// For daily mode, we calculate what portion of today's allocation has been used
		// Daily allocation = monthly_limit / days_in_month
		daysInMonth := daysInCurrentMonth(now)
		dailyAllocation := float64(monthlyLimit) / float64(daysInMonth)
		if dailyAllocation <= 0 {
			return 0, nil
		}

		// Estimate today's usage by dividing total monthly usage by days elapsed
		daysElapsed := now.Day()
		if daysElapsed == 0 {
			daysElapsed = 1
		}
		todayEstimate := float64(requests) / float64(daysElapsed)

		return (todayEstimate / dailyAllocation) * 100, nil

	case "weekly":
		// For weekly mode, we treat it as a longer period within the month
		// This is an approximation since Copilot resets monthly
		return (float64(requests) / float64(monthlyLimit)) * 100, nil

	default:
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// GetMonthlyResetTime returns the timestamp when the monthly counter resets.
// Copilot resets on the 1st of each month at 00:00:00 UTC.
func (c *Copilot) GetMonthlyResetTime() time.Time {
	now := time.Now().UTC()
	// Next reset is the 1st of next month at 00:00:00 UTC
	year := now.Year()
	month := now.Month() + 1
	if month > 12 {
		month = 1
		year++
	}
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
}

// GetResetTime returns the reset timestamp for the specified mode.
// For Copilot, both daily and weekly modes return the monthly reset time
// since that's how GitHub's system works.
func (c *Copilot) GetResetTime(mode string) (time.Time, error) {
	switch mode {
	case "daily", "weekly":
		return c.GetMonthlyResetTime(), nil
	default:
		return time.Time{}, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// firstOfMonth returns the first day of the month at 00:00:00 UTC for the given time.
func firstOfMonth(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// daysInCurrentMonth returns the number of days in the month of the given time.
func daysInCurrentMonth(t time.Time) int {
	t = t.UTC()
	// Get the first day of next month, then subtract 1 day to get last day of current month
	firstOfNext := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	lastOfCurrent := firstOfNext.Add(-24 * time.Hour)
	return lastOfCurrent.Day()
}

// GetTodayTokens returns today's request count (for snapshot compatibility).
// Note: Copilot tracks requests, not tokens. We return request count as "tokens".
func (c *Copilot) GetTodayTokens() (int64, error) {
	data, err := c.LoadUsageData()
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	currentMonth := now.Format("2006-01")

	// If we've crossed into a new month, return 0
	if data.Month != currentMonth {
		return 0, nil
	}

	// Estimate today's usage: monthly total / days elapsed
	daysElapsed := now.Day()
	if daysElapsed == 0 {
		daysElapsed = 1
	}
	todayEstimate := data.RequestCount / int64(daysElapsed)
	return todayEstimate, nil
}

// GetWeeklyTokens returns this week's request count (for snapshot compatibility).
// Note: Copilot tracks requests monthly, not weekly. We return monthly count.
func (c *Copilot) GetWeeklyTokens() (int64, error) {
	return c.GetRequestCount()
}
