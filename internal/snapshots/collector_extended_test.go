package snapshots

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/tmux"
)

type mockClaude struct {
	weekly int64
	daily  int64
}

func (m mockClaude) GetWeeklyUsage() (int64, error) { return m.weekly, nil }
func (m mockClaude) GetTodayUsage() (int64, error)  { return m.daily, nil }

type mockScraper struct {
	pct  float64
	hour int
}

func (m mockScraper) ScrapeClaudeUsage(ctx context.Context) (tmux.UsageResult, error) {
	return tmux.UsageResult{
		Provider:  "claude",
		WeeklyPct: m.pct,
		ScrapedAt: time.Now(),
	}, nil
}

func (m mockScraper) ScrapeCodexUsage(ctx context.Context) (tmux.UsageResult, error) {
	return tmux.UsageResult{
		Provider:  "codex",
		WeeklyPct: m.pct,
		ScrapedAt: time.Now(),
	}, nil
}

func TestGetSinceWeekStart(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, mockClaude{}, nil, mockScraper{pct: 50}, time.Monday)

	// Take multiple snapshots
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := collector.TakeSnapshot(ctx, "claude")
		if err != nil {
			t.Fatalf("TakeSnapshot: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Get snapshots since week start
	snapshots, err := collector.GetSinceWeekStart("claude")
	if err != nil {
		t.Fatalf("GetSinceWeekStart: %v", err)
	}

	if len(snapshots) < 1 {
		t.Fatalf("expected at least 1 snapshot, got %d", len(snapshots))
	}

	// All snapshots should be for the same week
	for _, snap := range snapshots {
		if snap.Provider != "claude" {
			t.Fatalf("expected provider 'claude', got %s", snap.Provider)
		}
	}
}

func TestGetSinceWeekStartEmpty(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, mockClaude{}, nil, mockScraper{pct: 50}, time.Monday)

	snapshots, err := collector.GetSinceWeekStart("codex")
	if err != nil {
		t.Fatalf("GetSinceWeekStart: %v", err)
	}

	if len(snapshots) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestGetHourlyAverages(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, mockClaude{daily: 100}, nil, mockScraper{pct: 50}, time.Monday)

	// Take a snapshot
	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}

	// Get hourly averages
	averages, err := collector.GetHourlyAverages("claude", 7)
	if err != nil {
		t.Fatalf("GetHourlyAverages: %v", err)
	}

	if len(averages) < 1 {
		t.Fatalf("expected at least 1 hourly average, got %d", len(averages))
	}

	for _, avg := range averages {
		if avg.Hour < 0 || avg.Hour > 23 {
			t.Fatalf("invalid hour: %d", avg.Hour)
		}
		if avg.AvgDailyTokens < 0 {
			t.Fatalf("invalid average tokens: %f", avg.AvgDailyTokens)
		}
	}
}

func TestGetHourlyAveragesZeroLookback(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, mockClaude{}, nil, mockScraper{pct: 50}, time.Monday)

	// Get with zero lookback should return empty
	averages, err := collector.GetHourlyAverages("claude", 0)
	if err != nil {
		t.Fatalf("GetHourlyAverages: %v", err)
	}

	if len(averages) != 0 {
		t.Fatalf("expected 0 averages for zero lookback, got %d", len(averages))
	}
}

func TestGetHourlyAveragesNegativeLookback(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, mockClaude{}, nil, mockScraper{pct: 50}, time.Monday)

	// Get with negative lookback should return empty
	averages, err := collector.GetHourlyAverages("claude", -5)
	if err != nil {
		t.Fatalf("GetHourlyAverages: %v", err)
	}

	if len(averages) != 0 {
		t.Fatalf("expected 0 averages for negative lookback, got %d", len(averages))
	}
}

func TestPruneZeroDays(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, mockClaude{weekly: 100}, nil, mockScraper{pct: 50}, time.Monday)

	// Take a snapshot
	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}

	// Prune with zero days should delete nothing
	deleted, err := collector.Prune(0)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if deleted != 0 {
		t.Fatalf("expected 0 deleted for 0 days, got %d", deleted)
	}
}

func TestPruneNegativeDays(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, mockClaude{weekly: 100}, nil, mockScraper{pct: 50}, time.Monday)

	// Take a snapshot
	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}

	// Prune with negative days should delete nothing
	deleted, err := collector.Prune(-1)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if deleted != 0 {
		t.Fatalf("expected 0 deleted for -1 days, got %d", deleted)
	}
}

func TestPruneOldSnapshots(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, mockClaude{weekly: 100, daily: 50}, nil, mockScraper{pct: 50}, time.Monday)

	// Manually insert old snapshot
	oldTime := time.Now().AddDate(0, 0, -10)
	weekStart := startOfWeek(oldTime, time.Monday)
	weekNumber, year := weekStart.ISOWeek()
	_, err = database.SQL().Exec(
		`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, day_of_week, hour_of_day, week_number, year)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"claude",
		oldTime,
		weekStart,
		50,
		10,
		int(oldTime.Weekday()),
		oldTime.Hour(),
		weekNumber,
		year,
	)
	if err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	// Take a current snapshot
	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}

	// Prune with 5 days retention
	deleted, err := collector.Prune(5)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	// Should have deleted the old snapshot
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	// Verify only current snapshot remains
	latest, err := collector.GetLatest("claude", 100)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}

	if len(latest) != 1 {
		t.Fatalf("expected 1 remaining snapshot, got %d", len(latest))
	}
}
