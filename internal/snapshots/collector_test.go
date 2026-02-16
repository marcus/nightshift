package snapshots

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/tmux"
)

type fakeClaude struct {
	weekly int64
	daily  int64
	err    error
}

func (f fakeClaude) GetWeeklyUsage() (int64, error) { return f.weekly, f.err }
func (f fakeClaude) GetTodayUsage() (int64, error)  { return f.daily, f.err }

type fakeScraper struct {
	claudePct        float64
	codexPct         float64
	sessionResetTime string
	weeklyResetTime  string
}

func (f fakeScraper) ScrapeClaudeUsage(ctx context.Context) (tmux.UsageResult, error) {
	return tmux.UsageResult{
		Provider:         "claude",
		WeeklyPct:        f.claudePct,
		SessionResetTime: f.sessionResetTime,
		WeeklyResetTime:  f.weeklyResetTime,
		ScrapedAt:        time.Now(),
	}, nil
}

func (f fakeScraper) ScrapeCodexUsage(ctx context.Context) (tmux.UsageResult, error) {
	return tmux.UsageResult{
		Provider:         "codex",
		WeeklyPct:        f.codexPct,
		SessionResetTime: f.sessionResetTime,
		WeeklyResetTime:  f.weeklyResetTime,
		ScrapedAt:        time.Now(),
	}, nil
}

type fakeCodex struct {
	files        []string
	dailyTokens  int64
	weeklyTokens int64
	err          error
}

func (f fakeCodex) ListSessionFiles() ([]string, error) { return f.files, f.err }
func (f fakeCodex) GetTodayTokens() (int64, error)      { return f.dailyTokens, f.err }
func (f fakeCodex) GetWeeklyTokens() (int64, error)     { return f.weeklyTokens, f.err }

func TestTakeSnapshotInsertsClaude(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, fakeClaude{weekly: 700, daily: 120}, nil, nil, fakeScraper{claudePct: 50}, time.Monday)

	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}

	latest, err := collector.GetLatest("claude", 1)
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(latest))
	}

	snap := latest[0]
	if snap.LocalTokens != 700 {
		t.Fatalf("local tokens = %d", snap.LocalTokens)
	}
	if snap.LocalDaily != 120 {
		t.Fatalf("local daily = %d", snap.LocalDaily)
	}
	if snap.ScrapedPct == nil || *snap.ScrapedPct != 50 {
		t.Fatalf("scraped pct = %v", snap.ScrapedPct)
	}
	if snap.InferredBudget == nil || *snap.InferredBudget != 1400 {
		t.Fatalf("inferred budget = %v", snap.InferredBudget)
	}

	weekStart := startOfWeek(snap.Timestamp, time.Monday)
	if !snap.WeekStart.Equal(weekStart) {
		t.Fatalf("week_start = %v, want %v", snap.WeekStart, weekStart)
	}
}

func TestTakeSnapshotCodexWithTokenData(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	codex := fakeCodex{weeklyTokens: 35000, dailyTokens: 5000}
	collector := NewCollector(database, nil, codex, nil, fakeScraper{codexPct: 42}, time.Monday)

	snap, err := collector.TakeSnapshot(context.Background(), "codex")
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}

	if snap.LocalTokens != 35000 {
		t.Fatalf("local tokens = %d, want 35000", snap.LocalTokens)
	}
	if snap.LocalDaily != 5000 {
		t.Fatalf("local daily = %d, want 5000", snap.LocalDaily)
	}
	if snap.ScrapedPct == nil || *snap.ScrapedPct != 42 {
		t.Fatalf("scraped pct = %v, want 42", snap.ScrapedPct)
	}
	// With token data + scraped pct, inferred budget = 35000 / (42/100) ≈ 83333
	if snap.InferredBudget == nil {
		t.Fatalf("inferred budget = nil, want computed value")
	}
	if *snap.InferredBudget != 83333 {
		t.Fatalf("inferred budget = %d, want 83333", *snap.InferredBudget)
	}
}

func TestTakeSnapshotCodexNoTokenData(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, nil, fakeCodex{}, nil, fakeScraper{codexPct: 42}, time.Monday)

	snap, err := collector.TakeSnapshot(context.Background(), "codex")
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}

	if snap.LocalTokens != 0 {
		t.Fatalf("local tokens = %d, want 0", snap.LocalTokens)
	}
	if snap.ScrapedPct == nil || *snap.ScrapedPct != 42 {
		t.Fatalf("scraped pct = %v, want 42", snap.ScrapedPct)
	}
	// No local tokens, so inferred budget must be nil
	if snap.InferredBudget != nil {
		t.Fatalf("inferred budget = %v, want nil", snap.InferredBudget)
	}
}

func TestCodexTokenTotalsReturnsTokenData(t *testing.T) {
	weekly, daily, err := codexTokenTotals(fakeCodex{
		files:        []string{"/some/path.jsonl"},
		weeklyTokens: 50000,
		dailyTokens:  8000,
	})
	if err != nil {
		t.Fatalf("codexTokenTotals: %v", err)
	}
	if weekly != 50000 {
		t.Fatalf("weekly tokens = %d, want 50000", weekly)
	}
	if daily != 8000 {
		t.Fatalf("daily tokens = %d, want 8000", daily)
	}
}

func TestCodexTokenTotalsNoData(t *testing.T) {
	weekly, daily, err := codexTokenTotals(fakeCodex{})
	if err != nil {
		t.Fatalf("codexTokenTotals: %v", err)
	}
	if weekly != 0 {
		t.Fatalf("weekly tokens = %d, want 0", weekly)
	}
	if daily != 0 {
		t.Fatalf("daily tokens = %d, want 0", daily)
	}
}

func TestCodexTokenTotalsPropagatesErrors(t *testing.T) {
	_, _, err := codexTokenTotals(fakeCodex{err: context.DeadlineExceeded})
	if err == nil {
		t.Fatal("expected error from codexTokenTotals")
	}
}

func TestTakeSnapshotStoresResetTimes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	scraper := fakeScraper{
		claudePct:        50,
		sessionResetTime: "9pm (America/Los_Angeles)",
		weeklyResetTime:  "Feb 8 at 10am (America/Los_Angeles)",
	}
	collector := NewCollector(database, fakeClaude{weekly: 700, daily: 120}, nil, nil, scraper, time.Monday)

	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}

	latest, err := collector.GetLatest("claude", 1)
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(latest))
	}

	snap := latest[0]
	if snap.SessionResetTime != "9pm (America/Los_Angeles)" {
		t.Fatalf("session reset = %q, want %q", snap.SessionResetTime, "9pm (America/Los_Angeles)")
	}
	if snap.WeeklyResetTime != "Feb 8 at 10am (America/Los_Angeles)" {
		t.Fatalf("weekly reset = %q, want %q", snap.WeeklyResetTime, "Feb 8 at 10am (America/Los_Angeles)")
	}
}

func TestTakeSnapshotCodexResetTimes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	scraper := fakeScraper{
		codexPct:         42,
		sessionResetTime: "20:15",
		weeklyResetTime:  "20:08 on 9 Feb",
	}
	collector := NewCollector(database, nil, fakeCodex{}, nil, scraper, time.Monday)

	snap, err := collector.TakeSnapshot(context.Background(), "codex")
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}

	if snap.SessionResetTime != "20:15" {
		t.Fatalf("session reset = %q, want %q", snap.SessionResetTime, "20:15")
	}
	if snap.WeeklyResetTime != "20:08 on 9 Feb" {
		t.Fatalf("weekly reset = %q, want %q", snap.WeeklyResetTime, "20:08 on 9 Feb")
	}
}

func TestTakeSnapshotEmptyResetTimes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// scraper with no reset times
	collector := NewCollector(database, fakeClaude{weekly: 100, daily: 10}, nil, nil, fakeScraper{claudePct: 25}, time.Monday)

	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}

	latest, err := collector.GetLatest("claude", 1)
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}

	snap := latest[0]
	if snap.SessionResetTime != "" {
		t.Fatalf("session reset = %q, want empty", snap.SessionResetTime)
	}
	if snap.WeeklyResetTime != "" {
		t.Fatalf("weekly reset = %q, want empty", snap.WeeklyResetTime)
	}
}

func TestPruneSnapshots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, fakeClaude{}, nil, nil, nil, time.Monday)

	oldTime := time.Now().AddDate(0, 0, -3)
	weekStart := startOfWeek(oldTime, time.Monday)
	weekNumber, year := weekStart.ISOWeek()
	if _, err := database.SQL().Exec(
		`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"claude",
		oldTime,
		weekStart,
		10,
		2,
		nil,
		nil,
		int(oldTime.Weekday()),
		oldTime.Hour(),
		weekNumber,
		year,
	); err != nil {
		t.Fatalf("insert old snapshot: %v", err)
	}

	deleted, err := collector.Prune(1)
	if err != nil {
		t.Fatalf("prune snapshots: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 row deleted, got %d", deleted)
	}
}
