package calibrator

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
)

func TestCalibrateAPIMode(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:  "api",
			WeeklyTokens: 123000,
		},
	}
	cal, _ := newTestCalibrator(t, cfg)

	result, err := cal.Calibrate("claude")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	if result.Source != "api" {
		t.Fatalf("source = %s", result.Source)
	}
	if result.Confidence != "high" {
		t.Fatalf("confidence = %s", result.Confidence)
	}
	if result.InferredBudget != 123000 {
		t.Fatalf("budget = %d", result.InferredBudget)
	}
}

func TestCalibrateDisabled(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:      "subscription",
			CalibrateEnabled: false,
			WeeklyTokens:     456000,
		},
	}
	cal, _ := newTestCalibrator(t, cfg)

	result, err := cal.Calibrate("claude")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	if result.Source != "config" {
		t.Fatalf("source = %s", result.Source)
	}
	if result.Confidence != "none" {
		t.Fatalf("confidence = %s", result.Confidence)
	}
	if result.InferredBudget != 456000 {
		t.Fatalf("budget = %d", result.InferredBudget)
	}
}

func TestCalibrateWithSamples(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:      "subscription",
			CalibrateEnabled: true,
			WeeklyTokens:     700000,
			WeekStartDay:     "monday",
		},
	}
	cal, database := newTestCalibrator(t, cfg)

	now := time.Now()
	insertSnapshot(t, database, "claude", 300000, 30, now)
	insertSnapshot(t, database, "claude", 310000, 30, now.Add(1*time.Hour))
	insertSnapshot(t, database, "claude", 290000, 30, now.Add(2*time.Hour))

	result, err := cal.Calibrate("claude")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	if result.Source != "calibrated" {
		t.Fatalf("source = %s", result.Source)
	}
	if result.Confidence != "medium" {
		t.Fatalf("confidence = %s", result.Confidence)
	}
	if result.InferredBudget != 1000000 {
		t.Fatalf("budget = %d", result.InferredBudget)
	}
}

func TestCalibrateMADOutlier(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:      "subscription",
			CalibrateEnabled: true,
			WeeklyTokens:     700000,
			WeekStartDay:     "monday",
		},
	}
	cal, database := newTestCalibrator(t, cfg)

	now := time.Now()
	insertSnapshot(t, database, "claude", 100000, 10, now)
	insertSnapshot(t, database, "claude", 100000, 10, now.Add(1*time.Hour))
	insertSnapshot(t, database, "claude", 1000000, 10, now.Add(2*time.Hour))

	result, err := cal.Calibrate("claude")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	if result.InferredBudget != 1000000 {
		t.Fatalf("budget = %d", result.InferredBudget)
	}
	if result.SampleCount != 2 {
		t.Fatalf("sample count = %d", result.SampleCount)
	}
}

func newTestCalibrator(t *testing.T, cfg *config.Config) (*Calibrator, *db.DB) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	return New(database, cfg), database
}

func insertSnapshot(t *testing.T, database *db.DB, provider string, localTokens int64, scrapedPct float64, ts time.Time) {
	t.Helper()

	weekStart := startOfWeek(ts, time.Monday)
	weekNumber, year := weekStart.ISOWeek()
	if _, err := database.SQL().Exec(
		`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		provider,
		ts,
		weekStart,
		localTokens,
		0,
		scrapedPct,
		nil,
		int(ts.Weekday()),
		ts.Hour(),
		weekNumber,
		year,
	); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
}

func TestGetBudgetImplementsInterface(t *testing.T) {
	cfg := &config.Config{Budget: config.BudgetConfig{WeeklyTokens: 1000}}
	cal, _ := newTestCalibrator(t, cfg)
	if _, err := cal.GetBudget("claude"); err != nil {
		t.Fatalf("GetBudget error: %v", err)
	}
}

func TestCalibrateCodexWithLocalTokens(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:      "subscription",
			CalibrateEnabled: true,
			WeeklyTokens:     500000,
			WeekStartDay:     "monday",
			PerProvider:      map[string]int{"codex": 1000000},
		},
	}
	cal, database := newTestCalibrator(t, cfg)

	now := time.Now()
	// Codex snapshots with real local token data and scraped percentage.
	// Inferred budget = local_tokens / (scraped_pct / 100)
	insertSnapshot(t, database, "codex", 500000, 50, now)
	insertSnapshot(t, database, "codex", 500000, 50, now.Add(1*time.Hour))
	insertSnapshot(t, database, "codex", 500000, 50, now.Add(2*time.Hour))

	result, err := cal.Calibrate("codex")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	if result.Source != "scraped" {
		t.Fatalf("source = %s, want scraped", result.Source)
	}
	if result.SampleCount != 3 {
		t.Fatalf("sample count = %d, want 3", result.SampleCount)
	}
	// 500000 / 0.50 = 1000000
	if result.InferredBudget != 1000000 {
		t.Fatalf("budget = %d, want 1000000", result.InferredBudget)
	}
}

func TestCalibrateCodexNoLocalTokensFallsBack(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:      "subscription",
			CalibrateEnabled: true,
			WeeklyTokens:     500000,
			WeekStartDay:     "monday",
			PerProvider:      map[string]int{"codex": 800000},
		},
	}
	cal, database := newTestCalibrator(t, cfg)

	now := time.Now()
	// Legacy snapshots with local_tokens=0 (before token parsing was added).
	// No samples will match local_tokens > 0, so calibrator falls back to config.
	insertSnapshot(t, database, "codex", 0, 50, now)
	insertSnapshot(t, database, "codex", 0, 50, now.Add(1*time.Hour))

	result, err := cal.Calibrate("codex")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	if result.Source != "config" {
		t.Fatalf("source = %s, want config", result.Source)
	}
	if result.InferredBudget != 800000 {
		t.Fatalf("budget = %d, want 800000", result.InferredBudget)
	}
}

func TestCalibrateCodexNoSamples(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:      "subscription",
			CalibrateEnabled: true,
			WeeklyTokens:     500000,
			WeekStartDay:     "monday",
			PerProvider:      map[string]int{"codex": 800000},
		},
	}
	cal, _ := newTestCalibrator(t, cfg)

	result, err := cal.Calibrate("codex")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	// No codex snapshots → falls back to config
	if result.Source != "config" {
		t.Fatalf("source = %s, want config", result.Source)
	}
	if result.InferredBudget != 800000 {
		t.Fatalf("budget = %d, want 800000", result.InferredBudget)
	}
}

func TestCalibrateCodexFiltersOutOfRange(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:      "subscription",
			CalibrateEnabled: true,
			WeeklyTokens:     500000,
			WeekStartDay:     "monday",
			PerProvider:      map[string]int{"codex": 1000000},
		},
	}
	cal, database := newTestCalibrator(t, cfg)

	now := time.Now()
	// scraped_pct=5 should be filtered (BETWEEN 10 AND 95)
	insertSnapshot(t, database, "codex", 50000, 5, now)
	// scraped_pct=100 should be filtered
	insertSnapshot(t, database, "codex", 1000000, 100, now.Add(1*time.Hour))
	// scraped_pct=40, local_tokens=400000 should be included
	insertSnapshot(t, database, "codex", 400000, 40, now.Add(2*time.Hour))

	result, err := cal.Calibrate("codex")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	if result.SampleCount != 1 {
		t.Fatalf("sample count = %d, want 1", result.SampleCount)
	}
	// 400000 / 0.40 = 1000000
	if result.InferredBudget != 1000000 {
		t.Fatalf("budget = %d, want 1000000", result.InferredBudget)
	}
}

func TestCalibrateSkipsOutOfRange(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			BillingMode:      "subscription",
			CalibrateEnabled: true,
			WeeklyTokens:     700000,
			WeekStartDay:     "monday",
		},
	}
	cal, database := newTestCalibrator(t, cfg)

	now := time.Now()
	insertSnapshot(t, database, "claude", 100000, 5, now)
	insertSnapshot(t, database, "claude", 100000, 50, now.Add(1*time.Hour))

	result, err := cal.Calibrate("claude")
	if err != nil {
		t.Fatalf("Calibrate error: %v", err)
	}
	if result.SampleCount != 1 {
		t.Fatalf("sample count = %d", result.SampleCount)
	}
}
