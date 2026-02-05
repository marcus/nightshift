package calibrator

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
)

// CalibrationResult represents the inferred budget and confidence.
type CalibrationResult struct {
	InferredBudget int64
	Confidence     string
	SampleCount    int
	Variance       float64
	Source         string
}

// Calibrator infers subscription budgets from snapshots.
type Calibrator struct {
	db           *db.DB
	cfg          *config.Config
	weekStartDay time.Weekday
}

// New creates a new Calibrator.
func New(database *db.DB, cfg *config.Config) *Calibrator {
	weekStart := parseWeekStartDay(cfg)
	return &Calibrator{
		db:           database,
		cfg:          cfg,
		weekStartDay: weekStart,
	}
}

// Calibrate computes an inferred weekly budget for a provider.
func (c *Calibrator) Calibrate(provider string) (CalibrationResult, error) {
	if c == nil || c.db == nil || c.cfg == nil {
		return CalibrationResult{}, errors.New("calibrator not initialized")
	}

	provider = strings.ToLower(provider)

	billingMode := strings.ToLower(c.cfg.Budget.BillingMode)
	if billingMode == "api" {
		budgetTokens := int64(c.cfg.GetProviderBudget(provider))
		return CalibrationResult{
			InferredBudget: budgetTokens,
			Confidence:     "high",
			SampleCount:    0,
			Variance:       0,
			Source:         "api",
		}, nil
	}

	if !c.cfg.Budget.CalibrateEnabled {
		budgetTokens := int64(c.cfg.GetProviderBudget(provider))
		return CalibrationResult{
			InferredBudget: budgetTokens,
			Confidence:     "none",
			SampleCount:    0,
			Variance:       0,
			Source:         "config",
		}, nil
	}

	snapshots, err := c.loadWeeklySamples(provider)
	if err != nil {
		return CalibrationResult{}, err
	}

	if len(snapshots) == 0 {
		budgetTokens := int64(c.cfg.GetProviderBudget(provider))
		return CalibrationResult{
			InferredBudget: budgetTokens,
			Confidence:     "none",
			SampleCount:    0,
			Variance:       0,
			Source:         "config",
		}, nil
	}

	filtered := snapshots
	if len(filtered) >= 3 {
		filtered = filterOutliersMAD(filtered)
	}

	if len(filtered) == 0 {
		budgetTokens := int64(c.cfg.GetProviderBudget(provider))
		return CalibrationResult{
			InferredBudget: budgetTokens,
			Confidence:     "none",
			SampleCount:    0,
			Variance:       0,
			Source:         "config",
		}, nil
	}

	median := median(filtered)
	variance := variance(filtered)
	cv := coefficientOfVariation(median, variance)

	var confidence string
	sampleCount := len(filtered)
	switch {
	case sampleCount == 0:
		confidence = "none"
	case sampleCount <= 2:
		confidence = "low"
	case sampleCount <= 5:
		if cv <= 0.15 {
			confidence = "medium"
		} else {
			confidence = "low"
		}
	default:
		if cv <= 0.10 {
			confidence = "high"
		} else if cv <= 0.15 {
			confidence = "medium"
		} else {
			confidence = "low"
		}
	}

	inferred := roundToNearest(median, 1000)

	source := "calibrated"
	if provider == "codex" {
		source = "scraped"
	}

	return CalibrationResult{
		InferredBudget: int64(inferred),
		Confidence:     confidence,
		SampleCount:    sampleCount,
		Variance:       variance,
		Source:         source,
	}, nil
}

// GetBudget returns a budget estimate for the budget manager.
func (c *Calibrator) GetBudget(provider string) (budget.BudgetEstimate, error) {
	result, err := c.Calibrate(provider)
	if err != nil {
		return budget.BudgetEstimate{}, err
	}
	return budget.BudgetEstimate{
		WeeklyTokens: result.InferredBudget,
		Source:       result.Source,
		Confidence:   result.Confidence,
		SampleCount:  result.SampleCount,
		Variance:     result.Variance,
	}, nil
}

func (c *Calibrator) loadWeeklySamples(provider string) ([]float64, error) {
	if provider == "codex" {
		return c.loadCodexWeeklySamples()
	}
	return c.loadClaudeWeeklySamples(provider)
}

// loadClaudeWeeklySamples infers budget from local_tokens / scraped_pct.
func (c *Calibrator) loadClaudeWeeklySamples(provider string) ([]float64, error) {
	weekStart := startOfWeek(time.Now(), c.weekStartDay)

	rows, err := c.db.SQL().Query(
		`SELECT local_tokens, scraped_pct
		 FROM snapshots
		 WHERE provider = ?
		 AND week_start = ?
		 AND scraped_pct IS NOT NULL
		 AND scraped_pct BETWEEN 10 AND 95
		 AND local_tokens > 0`,
		provider,
		weekStart,
	)
	if err != nil {
		return nil, fmt.Errorf("query snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	values := make([]float64, 0)
	for rows.Next() {
		var localTokens int64
		var scrapedPct float64
		if err := rows.Scan(&localTokens, &scrapedPct); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		if scrapedPct <= 0 {
			continue
		}
		inferred := float64(localTokens) / (scrapedPct / 100)
		values = append(values, inferred)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return values, nil
}

// loadCodexWeeklySamples infers budget from local_tokens / scraped_pct.
// Same approach as Claude: if we know local usage in tokens and the scraped
// percentage, we can infer the total budget. Falls back to config budget
// if no snapshots have local token data.
func (c *Calibrator) loadCodexWeeklySamples() ([]float64, error) {
	weekStart := startOfWeek(time.Now(), c.weekStartDay)

	// Try snapshots with both local_tokens and scraped_pct (preferred)
	rows, err := c.db.SQL().Query(
		`SELECT local_tokens, scraped_pct
		 FROM snapshots
		 WHERE provider = 'codex'
		 AND week_start = ?
		 AND scraped_pct IS NOT NULL
		 AND scraped_pct BETWEEN 10 AND 95
		 AND local_tokens > 0`,
		weekStart,
	)
	if err != nil {
		return nil, fmt.Errorf("query codex snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	values := make([]float64, 0)
	for rows.Next() {
		var localTokens int64
		var scrapedPct float64
		if err := rows.Scan(&localTokens, &scrapedPct); err != nil {
			return nil, fmt.Errorf("scan codex snapshot: %w", err)
		}
		if scrapedPct <= 0 {
			continue
		}
		inferred := float64(localTokens) / (scrapedPct / 100)
		values = append(values, inferred)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate codex snapshots: %w", err)
	}
	return values, nil
}

func parseWeekStartDay(cfg *config.Config) time.Weekday {
	if cfg == nil {
		return time.Monday
	}
	switch strings.ToLower(cfg.Budget.WeekStartDay) {
	case "sunday":
		return time.Sunday
	case "monday", "":
		return time.Monday
	default:
		return time.Monday
	}
}

func startOfWeek(now time.Time, weekStartDay time.Weekday) time.Time {
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	delta := (7 + int(now.Weekday()) - int(weekStartDay)) % 7
	return now.AddDate(0, 0, -delta)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func variance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	var sum float64
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func coefficientOfVariation(medianValue, varianceValue float64) float64 {
	if medianValue == 0 {
		return math.Inf(1)
	}
	return math.Sqrt(varianceValue) / medianValue
}

func filterOutliersMAD(values []float64) []float64 {
	if len(values) < 3 {
		return values
	}

	med := median(values)
	deviations := make([]float64, len(values))
	for i, v := range values {
		deviations[i] = math.Abs(v - med)
	}
	mad := median(deviations)
	filtered := make([]float64, 0, len(values))
	if mad == 0 {
		for _, v := range values {
			if v == med {
				filtered = append(filtered, v)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
		return values
	}

	threshold := 3 * mad
	for _, v := range values {
		if math.Abs(v-med) <= threshold {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func roundToNearest(value float64, step float64) float64 {
	if step <= 0 {
		return value
	}
	return math.Round(value/step) * step
}
