// Package budget implements token budget calculation and allocation for nightshift.
// Supports daily and weekly modes with reserve and aggressive end-of-week options.
package budget

import (
	"fmt"
	"math"
	"time"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/providers"
)

// UsageProvider is the interface for getting usage data from a provider.
type UsageProvider interface {
	Name() string
}

// ClaudeUsageProvider extends UsageProvider for Claude-specific usage methods.
type ClaudeUsageProvider interface {
	UsageProvider
	GetUsedPercent(mode string, weeklyBudget int64) (float64, error)
}

// CodexUsageProvider extends UsageProvider for Codex-specific usage methods.
type CodexUsageProvider interface {
	UsageProvider
	GetUsedPercent(mode string, weeklyBudget int64) (float64, error)
	GetResetTime(mode string) (time.Time, error)
}

// BudgetEstimate provides a resolved weekly budget with metadata.
type BudgetEstimate struct {
	WeeklyTokens int64
	Source       string
	Confidence   string
	SampleCount  int
	Variance     float64
}

// BudgetSource provides calibrated or external budget estimates.
type BudgetSource interface {
	GetBudget(provider string) (BudgetEstimate, error)
}

// TrendAnalyzer predicts near-term usage to protect daytime budget.
type TrendAnalyzer interface {
	PredictDaytimeUsage(provider string, now time.Time, weeklyBudget int64) (int64, error)
}

// Option configures a Manager.
type Option func(*Manager)

// Manager calculates and manages token budget allocation across providers.
type Manager struct {
	cfg          *config.Config
	claude       ClaudeUsageProvider
	codex        CodexUsageProvider
	budgetSource BudgetSource
	trend        TrendAnalyzer
	nowFunc      func() time.Time // for testing
}

// NewManager creates a budget manager with the given configuration and providers.
func NewManager(cfg *config.Config, claude ClaudeUsageProvider, codex CodexUsageProvider, opts ...Option) *Manager {
	mgr := &Manager{
		cfg:     cfg,
		claude:  claude,
		codex:   codex,
		nowFunc: time.Now,
	}
	for _, opt := range opts {
		opt(mgr)
	}
	return mgr
}

// WithBudgetSource injects a BudgetSource for calibrated budgets.
func WithBudgetSource(source BudgetSource) Option {
	return func(m *Manager) {
		m.budgetSource = source
	}
}

// WithTrendAnalyzer injects a trend analyzer for predicted daytime usage.
func WithTrendAnalyzer(analyzer TrendAnalyzer) Option {
	return func(m *Manager) {
		m.trend = analyzer
	}
}

// AllowanceResult contains the calculated budget allowance and metadata.
type AllowanceResult struct {
	Allowance         int64   // Final token allowance for this run
	WeeklyBudget      int64   // Weekly token budget used for calculation
	BudgetBase        int64   // Base budget (daily or remaining weekly)
	UsedPercent       float64 // Current used percentage
	ReserveAmount     int64   // Tokens reserved
	PredictedUsage    int64   // Predicted remaining usage today
	Mode              string  // "daily" or "weekly"
	RemainingDays     int     // Days until reset (weekly mode only)
	Multiplier        float64 // End-of-week multiplier (weekly mode only)
	BudgetSource      string  // calibrated, api, config
	BudgetConfidence  string  // none, low, medium, high
	BudgetSampleCount int     // number of samples used
}

// CalculateAllowance determines how many tokens nightshift can use for this run.
func (m *Manager) CalculateAllowance(provider string) (*AllowanceResult, error) {
	estimate, err := m.resolveBudget(provider)
	if err != nil {
		return nil, err
	}
	weeklyBudget := estimate.WeeklyTokens

	usedPercent, err := m.GetUsedPercent(provider)
	if err != nil {
		return nil, fmt.Errorf("getting used percent for %s: %w", provider, err)
	}

	mode := m.cfg.Budget.Mode
	if mode == "" {
		mode = config.DefaultBudgetMode
	}

	maxPercent := m.cfg.Budget.MaxPercent
	if maxPercent <= 0 {
		maxPercent = config.DefaultMaxPercent
	}

	reservePercent := m.cfg.Budget.ReservePercent
	if reservePercent < 0 {
		reservePercent = config.DefaultReservePercent
	}

	var result *AllowanceResult

	switch mode {
	case "daily":
		result = m.calculateDailyAllowance(weeklyBudget, usedPercent, maxPercent)
	case "weekly":
		remainingDays, err := m.DaysUntilWeeklyReset(provider)
		if err != nil {
			return nil, fmt.Errorf("getting days until reset: %w", err)
		}
		result = m.calculateWeeklyAllowance(weeklyBudget, usedPercent, maxPercent, remainingDays)
	default:
		return nil, fmt.Errorf("invalid budget mode: %s", mode)
	}

	// Apply reserve enforcement
	result = m.applyReserve(result, reservePercent)
	if m.trend != nil {
		predicted, err := m.trend.PredictDaytimeUsage(provider, m.nowFunc(), weeklyBudget)
		if err != nil {
			return nil, fmt.Errorf("predict daytime usage: %w", err)
		}
		if predicted > 0 {
			result.PredictedUsage = predicted
			if result.Allowance > predicted {
				result.Allowance -= predicted
			} else {
				result.Allowance = 0
			}
		}
	}
	result.BudgetSource = estimate.Source
	result.BudgetConfidence = estimate.Confidence
	result.BudgetSampleCount = estimate.SampleCount
	result.WeeklyBudget = weeklyBudget

	return result, nil
}

// calculateDailyAllowance implements the daily mode budget algorithm.
// Daily mode: Each night uses up to max_percent of that day's budget (weekly/7).
func (m *Manager) calculateDailyAllowance(weeklyBudget int64, usedPercent float64, maxPercent int) *AllowanceResult {
	dailyBudget := weeklyBudget / 7
	availableToday := float64(dailyBudget) * (1 - usedPercent/100)
	nightshiftAllowance := availableToday * float64(maxPercent) / 100

	// Cap at available (can't use more than available)
	if nightshiftAllowance > availableToday {
		nightshiftAllowance = availableToday
	}

	return &AllowanceResult{
		Allowance:   int64(math.Max(0, nightshiftAllowance)),
		BudgetBase:  dailyBudget,
		UsedPercent: usedPercent,
		Mode:        "daily",
		Multiplier:  1.0,
	}
}

// calculateWeeklyAllowance implements the weekly mode budget algorithm.
// Weekly mode: Each night uses up to max_percent of REMAINING weekly budget.
func (m *Manager) calculateWeeklyAllowance(weeklyBudget int64, usedPercent float64, maxPercent int, remainingDays int) *AllowanceResult {
	if remainingDays <= 0 {
		remainingDays = 1 // Avoid division by zero
	}

	remainingWeekly := float64(weeklyBudget) * (1 - usedPercent/100)

	// Aggressive end-of-week multiplier
	multiplier := 1.0
	if m.cfg.Budget.AggressiveEndOfWeek && remainingDays <= 2 {
		// 2x on day before reset, 3x on last day
		multiplier = float64(3 - remainingDays)
	}

	nightshiftAllowance := (remainingWeekly / float64(remainingDays)) * float64(maxPercent) / 100 * multiplier

	return &AllowanceResult{
		Allowance:     int64(math.Max(0, nightshiftAllowance)),
		BudgetBase:    int64(remainingWeekly),
		UsedPercent:   usedPercent,
		Mode:          "weekly",
		RemainingDays: remainingDays,
		Multiplier:    multiplier,
	}
}

// applyReserve enforces the reserve percentage on the calculated allowance.
func (m *Manager) applyReserve(result *AllowanceResult, reservePercent int) *AllowanceResult {
	reserveAmount := float64(result.BudgetBase) * float64(reservePercent) / 100
	result.ReserveAmount = int64(reserveAmount)
	result.Allowance = int64(math.Max(0, float64(result.Allowance)-reserveAmount))
	return result
}

func (m *Manager) resolveBudget(provider string) (BudgetEstimate, error) {
	estimate := BudgetEstimate{
		WeeklyTokens: int64(m.cfg.GetProviderBudget(provider)),
		Source:       "config",
	}

	if m.budgetSource != nil {
		loaded, err := m.budgetSource.GetBudget(provider)
		if err != nil {
			return estimate, fmt.Errorf("get budget estimate: %w", err)
		}
		if loaded.WeeklyTokens > 0 {
			estimate = loaded
			if estimate.Source == "" {
				estimate.Source = "calibrated"
			}
		}
	}

	if estimate.WeeklyTokens <= 0 {
		return estimate, fmt.Errorf("invalid weekly budget for provider %s: %d", provider, estimate.WeeklyTokens)
	}

	if estimate.Source == "" {
		estimate.Source = "config"
	}

	return estimate, nil
}

// GetUsedPercent retrieves the used percentage from the appropriate provider.
// Uses the resolved (calibrated) budget so percentages match the displayed budget.
func (m *Manager) GetUsedPercent(provider string) (float64, error) {
	estimate, err := m.resolveBudget(provider)
	if err != nil {
		return 0, fmt.Errorf("resolving budget for %s: %w", provider, err)
	}
	weeklyBudget := estimate.WeeklyTokens

	mode := m.cfg.Budget.Mode
	if mode == "" {
		mode = config.DefaultBudgetMode
	}

	switch provider {
	case "claude":
		if m.claude == nil {
			return 0, fmt.Errorf("claude provider not configured")
		}
		return m.claude.GetUsedPercent(mode, weeklyBudget)

	case "codex":
		if m.codex == nil {
			return 0, fmt.Errorf("codex provider not configured")
		}
		return m.codex.GetUsedPercent(mode, weeklyBudget)

	default:
		return 0, fmt.Errorf("unknown provider: %s", provider)
	}
}

// DaysUntilWeeklyReset calculates days remaining until the weekly budget resets.
// For Claude: assumes weekly reset on Sunday (7 - current weekday, or 7 if Sunday).
// For Codex: uses the secondary rate limit's resets_at timestamp.
func (m *Manager) DaysUntilWeeklyReset(provider string) (int, error) {
	now := m.nowFunc()

	switch provider {
	case "claude":
		// Claude resets weekly; assume Sunday reset
		// Weekday: Sunday=0, Monday=1, ..., Saturday=6
		weekday := int(now.Weekday())
		if weekday == 0 {
			return 7, nil // It's Sunday, next reset in 7 days
		}
		return 7 - weekday, nil

	case "codex":
		if m.codex == nil {
			return 7, nil // Default fallback
		}
		resetTime, err := m.codex.GetResetTime("weekly")
		if err != nil {
			return 7, nil // Fallback on error
		}
		if resetTime.IsZero() {
			return 7, nil // No reset time available
		}

		duration := resetTime.Sub(now)
		days := int(math.Ceil(duration.Hours() / 24))
		if days <= 0 {
			return 1, nil // At least 1 day
		}
		return days, nil

	default:
		return 7, nil // Default for unknown providers
	}
}

// Summary returns a human-readable summary of the budget state for a provider.
func (m *Manager) Summary(provider string) (string, error) {
	result, err := m.CalculateAllowance(provider)
	if err != nil {
		return "", err
	}

	estimate, err := m.resolveBudget(provider)
	if err != nil {
		return "", err
	}
	weeklyBudget := estimate.WeeklyTokens

	if result.Mode == "daily" {
		return fmt.Sprintf(
			"%s: %.1f%% used today, %d tokens allowed (daily budget: %d, reserve: %d)",
			provider, result.UsedPercent, result.Allowance, result.BudgetBase, result.ReserveAmount,
		), nil
	}

	return fmt.Sprintf(
		"%s: %.1f%% used this week (%d days left), %d tokens allowed (weekly: %d, remaining: %d, reserve: %d, multiplier: %.1fx)",
		provider, result.UsedPercent, result.RemainingDays, result.Allowance,
		weeklyBudget, result.BudgetBase, result.ReserveAmount, result.Multiplier,
	), nil
}

// CanRun checks if there's enough budget to run a task with the given estimated cost.
func (m *Manager) CanRun(provider string, estimatedTokens int64) (bool, error) {
	result, err := m.CalculateAllowance(provider)
	if err != nil {
		return false, err
	}
	return result.Allowance >= estimatedTokens, nil
}

// Tracker provides backward compatibility for tracking actual spend.
// Deprecated: Use Manager for budget calculations.
type Tracker struct {
	spent map[string]int64
	limit int64
}

// NewTracker creates a budget tracker with the given limit.
// Deprecated: Use NewManager instead.
func NewTracker(limitCents int64) *Tracker {
	return &Tracker{
		spent: make(map[string]int64),
		limit: limitCents,
	}
}

// Record logs spending for a provider.
func (t *Tracker) Record(provider string, tokens int, costCents int64) {
	t.spent[provider] += costCents
}

// Remaining returns cents left in budget.
func (t *Tracker) Remaining() int64 {
	var total int64
	for _, v := range t.spent {
		total += v
	}
	return t.limit - total
}

// NewManagerFromProviders is a convenience constructor that accepts the concrete provider types.
func NewManagerFromProviders(cfg *config.Config, claude *providers.Claude, codex *providers.Codex, opts ...Option) *Manager {
	var claudeProvider ClaudeUsageProvider
	var codexProvider CodexUsageProvider

	if claude != nil {
		claudeProvider = claude
	}
	if codex != nil {
		codexProvider = codex
	}

	return NewManager(cfg, claudeProvider, codexProvider, opts...)
}
