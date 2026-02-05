package trends

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/db"
)

func TestBuildProfileAverages(t *testing.T) {
	database := openTrendDB(t)
	defer func() { _ = database.Close() }()

	base := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)
	insertSnapshot(t, database, "claude", base, 100)
	insertSnapshot(t, database, "claude", base.AddDate(0, 0, -1), 300)
	insertSnapshot(t, database, "claude", base.Add(4*time.Hour), 200)
	insertSnapshot(t, database, "claude", base.AddDate(0, 0, -1).Add(4*time.Hour), 400)

	analyzer := NewAnalyzer(database, 7)
	analyzer.nowFunc = func() time.Time { return base.AddDate(0, 0, 1) }
	profile, err := analyzer.BuildProfile("claude", 7)
	if err != nil {
		t.Fatalf("build profile: %v", err)
	}

	if got := profile.HourlyAverages[10]; got != 200 {
		t.Fatalf("hour 10 avg = %.1f, want 200", got)
	}
	if got := profile.HourlyAverages[14]; got != 300 {
		t.Fatalf("hour 14 avg = %.1f, want 300", got)
	}
	if profile.DailyTotal != 300 {
		t.Fatalf("daily total = %.1f, want 300", profile.DailyTotal)
	}
}

func TestPredictDaytimeUsage(t *testing.T) {
	database := openTrendDB(t)
	defer func() { _ = database.Close() }()

	now := time.Date(2024, 1, 12, 12, 30, 0, 0, time.UTC)
	insertSnapshot(t, database, "codex", now.AddDate(0, 0, -1).Add(-4*time.Hour), 100)
	insertSnapshot(t, database, "codex", now.AddDate(0, 0, -1), 300)
	insertSnapshot(t, database, "codex", now.AddDate(0, 0, -1).Add(11*time.Hour), 500)

	analyzer := NewAnalyzer(database, 7)
	analyzer.nowFunc = func() time.Time { return now }
	predicted, err := analyzer.PredictDaytimeUsage("codex", now, 3500)
	if err != nil {
		t.Fatalf("predict daytime usage: %v", err)
	}
	if predicted != 200 {
		t.Fatalf("predicted usage = %d, want 200", predicted)
	}
}

func TestPredictDaytimeUsageCapsDailyBudget(t *testing.T) {
	database := openTrendDB(t)
	defer func() { _ = database.Close() }()

	now := time.Date(2024, 1, 12, 10, 0, 0, 0, time.UTC)
	insertSnapshot(t, database, "codex", now.AddDate(0, 0, -1).Add(-2*time.Hour), 100)
	insertSnapshot(t, database, "codex", now.AddDate(0, 0, -1).Add(10*time.Hour), 900)

	analyzer := NewAnalyzer(database, 7)
	analyzer.nowFunc = func() time.Time { return now }
	predicted, err := analyzer.PredictDaytimeUsage("codex", now, 2100)
	if err != nil {
		t.Fatalf("predict daytime usage: %v", err)
	}
	if predicted != 300 {
		t.Fatalf("predicted usage = %d, want 300", predicted)
	}
}

func openTrendDB(t *testing.T) *db.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nightshift.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return database
}

func insertSnapshot(t *testing.T, database *db.DB, provider string, timestamp time.Time, daily int64) {
	t.Helper()
	weekStart := startOfWeek(timestamp, time.Monday)
	weekNumber, year := weekStart.ISOWeek()

	if _, err := database.SQL().Exec(
		`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		provider,
		timestamp,
		weekStart,
		daily,
		daily,
		nil,
		nil,
		int(timestamp.Weekday()),
		timestamp.Hour(),
		weekNumber,
		year,
	); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
}

func startOfWeek(now time.Time, weekStartDay time.Weekday) time.Time {
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	delta := (7 + int(now.Weekday()) - int(weekStartDay)) % 7
	return now.AddDate(0, 0, -delta)
}
