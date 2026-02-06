package stats

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/reporting"
)

// openTestDB creates a temp SQLite database with full migrations applied.
func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

// writeReport marshals a RunResults to a run-*.json file in dir.
func writeReport(t *testing.T, dir string, index int, r *reporting.RunResults) {
	t.Helper()
	name := fmt.Sprintf("run-%s-%d.json", r.StartTime.Format("2006-01-02-150405"), index)
	path := filepath.Join(dir, name)
	if err := reporting.SaveRunResults(r, path); err != nil {
		t.Fatalf("write report: %v", err)
	}
}

// --- Duration type tests ---

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{0, "0"},
		{30 * time.Second, "30"},
		{90 * time.Second, "90"},
		{2*time.Hour + 30*time.Minute, "9000"},
	}
	for _, tt := range tests {
		d := Duration{tt.dur}
		b, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("marshal %v: %v", tt.dur, err)
		}
		if string(b) != tt.want {
			t.Errorf("MarshalJSON(%v) = %s, want %s", tt.dur, b, tt.want)
		}
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"0", 0},
		{"30", 30 * time.Second},
		{"3600", 3600 * time.Second},
	}
	for _, tt := range tests {
		var d Duration
		if err := json.Unmarshal([]byte(tt.input), &d); err != nil {
			t.Fatalf("unmarshal %s: %v", tt.input, err)
		}
		if d.Duration != tt.want {
			t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.input, d.Duration, tt.want)
		}
	}
}

func TestDuration_UnmarshalJSON_Error(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`"not a number"`), &d); err == nil {
		t.Error("expected error for non-numeric input")
	}
}

func TestDuration_String(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{0, "0s"},
		{45 * time.Second, "45s"},
		{2*time.Minute + 15*time.Second, "2m 15s"},
		{1*time.Hour + 30*time.Minute, "1h 30m"},
		{3 * time.Hour, "3h 0m"},
	}
	for _, tt := range tests {
		d := Duration{tt.dur}
		got := d.String()
		if got != tt.want {
			t.Errorf("String(%v) = %q, want %q", tt.dur, got, tt.want)
		}
	}
}

// --- Compute tests ---

func TestCompute_NoData(t *testing.T) {
	dir := t.TempDir() // empty reports dir
	s := New(nil, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.TotalRuns != 0 {
		t.Errorf("TotalRuns = %d, want 0", result.TotalRuns)
	}
	if result.TasksCompleted != 0 {
		t.Errorf("TasksCompleted = %d, want 0", result.TasksCompleted)
	}
	if result.TasksFailed != 0 {
		t.Errorf("TasksFailed = %d, want 0", result.TasksFailed)
	}
	if result.PRsCreated != 0 {
		t.Errorf("PRsCreated = %d, want 0", result.PRsCreated)
	}
	if result.TotalTokensUsed != 0 {
		t.Errorf("TotalTokensUsed = %d, want 0", result.TotalTokensUsed)
	}
	if result.SuccessRate != 0 {
		t.Errorf("SuccessRate = %f, want 0", result.SuccessRate)
	}
	if result.FirstRunAt != nil {
		t.Errorf("FirstRunAt = %v, want nil", result.FirstRunAt)
	}
	if result.LastRunAt != nil {
		t.Errorf("LastRunAt = %v, want nil", result.LastRunAt)
	}
	if result.TotalDuration.Duration != 0 {
		t.Errorf("TotalDuration = %v, want 0", result.TotalDuration)
	}
}

func TestCompute_NoData_NonexistentDir(t *testing.T) {
	s := New(nil, filepath.Join(t.TempDir(), "nonexistent"))
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.TotalRuns != 0 {
		t.Errorf("TotalRuns = %d, want 0", result.TotalRuns)
	}
}

func TestCompute_NoData_EmptyReportsDir(t *testing.T) {
	s := New(nil, "")
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.TotalRuns != 0 {
		t.Errorf("TotalRuns = %d, want 0", result.TotalRuns)
	}
}

func TestCompute_FromReports(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	r1 := &reporting.RunResults{
		StartTime:  now,
		EndTime:    now.Add(30 * time.Minute),
		UsedBudget: 5000,
		Tasks: []reporting.TaskResult{
			{Project: "/home/user/project-a", TaskType: "code-review", Status: "completed", TokensUsed: 2000},
			{Project: "/home/user/project-a", TaskType: "tests", Status: "failed", TokensUsed: 1500},
		},
	}
	r2 := &reporting.RunResults{
		StartTime:  now.Add(24 * time.Hour),
		EndTime:    now.Add(24*time.Hour + 45*time.Minute),
		UsedBudget: 3000,
		Tasks: []reporting.TaskResult{
			{Project: "/home/user/project-b", TaskType: "refactor", Status: "completed", TokensUsed: 1800},
			{Project: "/home/user/project-b", TaskType: "docs", Status: "skipped", SkipReason: "insufficient budget"},
		},
	}

	writeReport(t, dir, 0, r1)
	writeReport(t, dir, 1, r2)

	s := New(nil, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}

	if result.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", result.TotalRuns)
	}
	if result.TasksCompleted != 2 {
		t.Errorf("TasksCompleted = %d, want 2", result.TasksCompleted)
	}
	if result.TasksFailed != 1 {
		t.Errorf("TasksFailed = %d, want 1", result.TasksFailed)
	}
	if result.TasksSkipped != 1 {
		t.Errorf("TasksSkipped = %d, want 1", result.TasksSkipped)
	}
	if result.TotalTokensUsed != 8000 {
		t.Errorf("TotalTokensUsed = %d, want 8000", result.TotalTokensUsed)
	}

	// SuccessRate = 2 completed / 4 total * 100 = 50
	if result.SuccessRate != 50 {
		t.Errorf("SuccessRate = %f, want 50", result.SuccessRate)
	}

	// Duration: 30m + 45m = 75m
	wantDur := 75 * time.Minute
	if result.TotalDuration.Duration != wantDur {
		t.Errorf("TotalDuration = %v, want %v", result.TotalDuration.Duration, wantDur)
	}

	// AvgRunDuration = 75m / 2 = 37m30s
	wantAvg := wantDur / 2
	if result.AvgRunDuration.Duration != wantAvg {
		t.Errorf("AvgRunDuration = %v, want %v", result.AvgRunDuration.Duration, wantAvg)
	}

	// AvgTokensPerRun = 8000 / 2 = 4000
	if result.AvgTokensPerRun != 4000 {
		t.Errorf("AvgTokensPerRun = %d, want 4000", result.AvgTokensPerRun)
	}

	// FirstRunAt / LastRunAt
	if result.FirstRunAt == nil || !result.FirstRunAt.Equal(now) {
		t.Errorf("FirstRunAt = %v, want %v", result.FirstRunAt, now)
	}
	wantLast := now.Add(24 * time.Hour)
	if result.LastRunAt == nil || !result.LastRunAt.Equal(wantLast) {
		t.Errorf("LastRunAt = %v, want %v", result.LastRunAt, wantLast)
	}

	// TaskTypeBreakdown
	if result.TaskTypeBreakdown["code-review"] != 1 {
		t.Errorf("TaskTypeBreakdown[code-review] = %d, want 1", result.TaskTypeBreakdown["code-review"])
	}
	if result.TaskTypeBreakdown["refactor"] != 1 {
		t.Errorf("TaskTypeBreakdown[refactor] = %d, want 1", result.TaskTypeBreakdown["refactor"])
	}
}

func TestCompute_ReportTokensFallback(t *testing.T) {
	// When UsedBudget is 0, tokens should be summed from individual tasks
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	r := &reporting.RunResults{
		StartTime:  now,
		EndTime:    now.Add(10 * time.Minute),
		UsedBudget: 0,
		Tasks: []reporting.TaskResult{
			{Status: "completed", TokensUsed: 1000},
			{Status: "completed", TokensUsed: 2000},
		},
	}
	writeReport(t, dir, 0, r)

	s := New(nil, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.TotalTokensUsed != 3000 {
		t.Errorf("TotalTokensUsed = %d, want 3000", result.TotalTokensUsed)
	}
}

func TestCompute_PRDetection(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	r := &reporting.RunResults{
		StartTime: now,
		EndTime:   now.Add(20 * time.Minute),
		Tasks: []reporting.TaskResult{
			{Status: "completed", OutputType: "PR", OutputRef: "https://github.com/org/repo/pull/1"},
			{Status: "completed", OutputType: "pr", OutputRef: "https://github.com/org/repo/pull/2"},
			{Status: "completed", OutputType: "PR", OutputRef: "https://github.com/org/repo/pull/1"}, // duplicate URL
			{Status: "completed", OutputType: "Report", OutputRef: "/tmp/report.md"},                 // not a PR
			{Status: "completed", OutputType: "PR", OutputRef: ""},                                   // empty ref
		},
	}
	writeReport(t, dir, 0, r)

	s := New(nil, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}

	// PRsCreated counts each task with OutputType=PR and non-empty ref
	if result.PRsCreated != 3 {
		t.Errorf("PRsCreated = %d, want 3", result.PRsCreated)
	}

	// PRURLs are deduplicated (only http refs)
	if len(result.PRURLs) != 2 {
		t.Errorf("len(PRURLs) = %d, want 2", len(result.PRURLs))
	}
}

func TestCompute_ProjectBreakdownFromReports(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	r := &reporting.RunResults{
		StartTime: now,
		EndTime:   now.Add(10 * time.Minute),
		Tasks: []reporting.TaskResult{
			{Project: "/home/user/alpha", Status: "completed"},
			{Project: "/home/user/alpha", Status: "completed"},
			{Project: "/home/user/beta", Status: "completed"},
		},
	}
	writeReport(t, dir, 0, r)

	s := New(nil, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}

	if len(result.ProjectBreakdown) != 2 {
		t.Fatalf("ProjectBreakdown len = %d, want 2", len(result.ProjectBreakdown))
	}
	// Sorted by task count descending: alpha(2), beta(1)
	if result.ProjectBreakdown[0].Name != "alpha" || result.ProjectBreakdown[0].TaskCount != 2 {
		t.Errorf("ProjectBreakdown[0] = %+v, want alpha with 2 tasks", result.ProjectBreakdown[0])
	}
	if result.ProjectBreakdown[1].Name != "beta" || result.ProjectBreakdown[1].TaskCount != 1 {
		t.Errorf("ProjectBreakdown[1] = %+v, want beta with 1 task", result.ProjectBreakdown[1])
	}
}

// --- DB-enriched tests ---

func TestCompute_WithDBRunHistory(t *testing.T) {
	database := openTestDB(t)
	dir := t.TempDir()

	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	// Write 1 report file
	r := &reporting.RunResults{
		StartTime: now,
		EndTime:   now.Add(10 * time.Minute),
		Tasks:     []reporting.TaskResult{{Status: "completed"}},
	}
	writeReport(t, dir, 0, r)

	// Insert 3 run_history rows (more than the 1 report)
	sqlDB := database.SQL()
	for i := 0; i < 3; i++ {
		start := now.Add(time.Duration(i) * 24 * time.Hour)
		_, err := sqlDB.Exec(
			`INSERT INTO run_history (id, start_time, end_time, project, tasks, tokens_used, status) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("run-%d", i), start, start.Add(15*time.Minute), "/proj", "[]", 500, "completed",
		)
		if err != nil {
			t.Fatalf("insert run_history: %v", err)
		}
	}

	s := New(database, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}

	// TotalRuns should be max(1 report, 3 db) = 3
	if result.TotalRuns != 3 {
		t.Errorf("TotalRuns = %d, want 3", result.TotalRuns)
	}

	// FirstRunAt comes from the report (DB time scan has driver limitations with TEXT dates)
	if result.FirstRunAt == nil || !result.FirstRunAt.Equal(now) {
		t.Errorf("FirstRunAt = %v, want %v", result.FirstRunAt, now)
	}
}

func TestCompute_DBTokensFallback(t *testing.T) {
	// When report UsedBudget=0 and task TokensUsed=0, tokens come from DB
	database := openTestDB(t)
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	// Report with no token data
	r := &reporting.RunResults{
		StartTime:  now,
		EndTime:    now.Add(10 * time.Minute),
		UsedBudget: 0,
		Tasks:      []reporting.TaskResult{{Status: "completed", TokensUsed: 0}},
	}
	writeReport(t, dir, 0, r)

	// DB run_history has token data
	sqlDB := database.SQL()
	_, err := sqlDB.Exec(
		`INSERT INTO run_history (id, start_time, end_time, project, tasks, tokens_used, status) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"run-1", now, now.Add(10*time.Minute), "/proj", "[]", 7500, "completed",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	s := New(database, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.TotalTokensUsed != 7500 {
		t.Errorf("TotalTokensUsed = %d, want 7500", result.TotalTokensUsed)
	}
}

func TestCompute_ProjectsFromDB(t *testing.T) {
	database := openTestDB(t)
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	// Report with tasks for project-a
	r := &reporting.RunResults{
		StartTime: now,
		EndTime:   now.Add(10 * time.Minute),
		Tasks: []reporting.TaskResult{
			{Project: "/home/user/project-a", Status: "completed"},
			{Project: "/home/user/project-a", Status: "completed"},
		},
	}
	writeReport(t, dir, 0, r)

	// DB has project-a (with run_count) and project-b (not in reports)
	sqlDB := database.SQL()
	_, err := sqlDB.Exec(`INSERT INTO projects (path, run_count) VALUES (?, ?)`, "/home/user/project-a", 5)
	if err != nil {
		t.Fatalf("insert project-a: %v", err)
	}
	_, err = sqlDB.Exec(`INSERT INTO projects (path, run_count) VALUES (?, ?)`, "/home/user/project-b", 3)
	if err != nil {
		t.Fatalf("insert project-b: %v", err)
	}

	s := New(database, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}

	if result.TotalProjects != 2 {
		t.Errorf("TotalProjects = %d, want 2", result.TotalProjects)
	}

	// ProjectBreakdown: project-a should have TaskCount=2, RunCount=5
	// project-b should have TaskCount=0, RunCount=3
	found := make(map[string]ProjectStats)
	for _, ps := range result.ProjectBreakdown {
		found[ps.Name] = ps
	}

	a := found["project-a"]
	if a.TaskCount != 2 || a.RunCount != 5 {
		t.Errorf("project-a: TaskCount=%d RunCount=%d, want 2/5", a.TaskCount, a.RunCount)
	}
	b := found["project-b"]
	if b.RunCount != 3 {
		t.Errorf("project-b: RunCount=%d, want 3", b.RunCount)
	}
}

func TestCompute_BudgetProjection(t *testing.T) {
	database := openTestDB(t)

	sqlDB := database.SQL()
	now := time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC)

	// Insert snapshots with inferred_budget and local_daily
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * 24 * time.Hour)
		_, err := sqlDB.Exec(
			`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"claude", ts, ts.Format("2006-01-02"), 1000, 2000, 40.0, 500000, int(ts.Weekday()), ts.Hour(), 1, ts.Year(),
		)
		if err != nil {
			t.Fatalf("insert snapshot %d: %v", i, err)
		}
	}

	s := New(database, t.TempDir())
	s.nowFunc = func() time.Time { return now }
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}

	if result.BudgetProjection == nil {
		t.Fatal("BudgetProjection is nil")
	}

	bp := result.BudgetProjection
	if bp.Provider != "claude" {
		t.Errorf("Provider = %s, want claude", bp.Provider)
	}
	if bp.WeeklyBudget != 500000 {
		t.Errorf("WeeklyBudget = %d, want 500000", bp.WeeklyBudget)
	}
	if bp.CurrentUsedPct != 40.0 {
		t.Errorf("CurrentUsedPct = %f, want 40.0", bp.CurrentUsedPct)
	}
	if bp.AvgDailyUsage != 2000 {
		t.Errorf("AvgDailyUsage = %d, want 2000", bp.AvgDailyUsage)
	}
	if bp.AvgHourlyUsage <= 0 {
		t.Errorf("AvgHourlyUsage = %f, want > 0", bp.AvgHourlyUsage)
	}
	if bp.RemainingTokens != 300000 {
		t.Errorf("RemainingTokens = %d, want 300000", bp.RemainingTokens)
	}
	if bp.Source != "calibrated" {
		t.Errorf("Source = %s, want calibrated", bp.Source)
	}
	// EstDaysRemaining = (500000 * 0.60) / 2000 = 150
	if bp.EstDaysRemaining != 150 {
		t.Errorf("EstDaysRemaining = %d, want 150", bp.EstDaysRemaining)
	}
	if bp.ResetAt == nil {
		t.Fatalf("ResetAt is nil")
	}
	if bp.TimeUntilResetSec <= 0 {
		t.Errorf("TimeUntilResetSec = %d, want > 0", bp.TimeUntilResetSec)
	}
	if bp.WillExhaustBeforeReset == nil {
		t.Fatalf("WillExhaustBeforeReset is nil")
	}
	if *bp.WillExhaustBeforeReset {
		t.Errorf("WillExhaustBeforeReset = true, want false")
	}
	if len(result.BudgetProjections) != 1 {
		t.Errorf("len(BudgetProjections) = %d, want 1", len(result.BudgetProjections))
	}
}

func TestCompute_BudgetProjection_NoSnapshots(t *testing.T) {
	database := openTestDB(t)
	s := New(database, t.TempDir())
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.BudgetProjection != nil {
		t.Errorf("BudgetProjection = %+v, want nil", result.BudgetProjection)
	}
	if len(result.BudgetProjections) != 0 {
		t.Errorf("len(BudgetProjections) = %d, want 0", len(result.BudgetProjections))
	}
}

func TestCompute_BudgetProjection_NoLocalDaily(t *testing.T) {
	database := openTestDB(t)
	sqlDB := database.SQL()
	now := time.Now()

	// Snapshot with inferred_budget but local_daily=0
	_, err := sqlDB.Exec(
		`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"claude", now, now.Format("2006-01-02"), 1000, 0, 40.0, 500000, int(now.Weekday()), now.Hour(), 1, now.Year(),
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	s := New(database, t.TempDir())
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.BudgetProjection != nil {
		t.Errorf("BudgetProjection should be nil when no local_daily > 0")
	}
	if len(result.BudgetProjections) != 0 {
		t.Errorf("len(BudgetProjections) = %d, want 0", len(result.BudgetProjections))
	}
}

func TestCompute_BudgetProjection_MultipleProviders(t *testing.T) {
	database := openTestDB(t)
	sqlDB := database.SQL()
	now := time.Date(2026, 2, 6, 12, 0, 0, 0, time.UTC)

	insert := func(provider string, daily int64, pct float64, budget int64) {
		t.Helper()
		for i := 0; i < 3; i++ {
			ts := now.Add(-time.Duration(i) * 24 * time.Hour)
			_, err := sqlDB.Exec(
				`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				provider, ts, ts.Format("2006-01-02"), 1000, daily, pct, budget, int(ts.Weekday()), ts.Hour(), 1, ts.Year(),
			)
			if err != nil {
				t.Fatalf("insert %s snapshot %d: %v", provider, i, err)
			}
		}
	}

	insert("claude", 2000, 40.0, 500000)
	insert("codex", 3000, 25.0, 700000)

	s := New(database, t.TempDir())
	s.nowFunc = func() time.Time { return now }
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}

	if len(result.BudgetProjections) != 2 {
		t.Fatalf("len(BudgetProjections) = %d, want 2", len(result.BudgetProjections))
	}

	found := map[string]BudgetProjection{}
	for _, p := range result.BudgetProjections {
		found[p.Provider] = p
	}
	if _, ok := found["claude"]; !ok {
		t.Fatalf("missing claude projection")
	}
	if _, ok := found["codex"]; !ok {
		t.Fatalf("missing codex projection")
	}
	if result.BudgetProjection == nil {
		t.Fatalf("BudgetProjection legacy field should be set")
	}
}

func TestCompute_IgnoresNonReportFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	// Valid report
	r := &reporting.RunResults{
		StartTime: now,
		EndTime:   now.Add(10 * time.Minute),
		Tasks:     []reporting.TaskResult{{Status: "completed"}},
	}
	writeReport(t, dir, 0, r)

	// Non-matching files (should be ignored)
	nonMatchingFiles := []string{"summary.json", "notes.txt", "run-.json.bak"}
	for _, name := range nonMatchingFiles {
		path := filepath.Join(dir, name)
		_ = reporting.SaveRunResults(r, path)
	}

	s := New(nil, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1 (should ignore non run-*.json files)", result.TotalRuns)
	}
}

func TestCompute_SuccessRateAllCompleted(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	r := &reporting.RunResults{
		StartTime: now,
		EndTime:   now.Add(10 * time.Minute),
		Tasks: []reporting.TaskResult{
			{Status: "completed"},
			{Status: "completed"},
			{Status: "completed"},
		},
	}
	writeReport(t, dir, 0, r)

	s := New(nil, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.SuccessRate != 100 {
		t.Errorf("SuccessRate = %f, want 100", result.SuccessRate)
	}
}

func TestCompute_SuccessRateNoTasks(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	r := &reporting.RunResults{
		StartTime: now,
		EndTime:   now.Add(10 * time.Minute),
		Tasks:     []reporting.TaskResult{},
	}
	writeReport(t, dir, 0, r)

	s := New(nil, dir)
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if result.SuccessRate != 0 {
		t.Errorf("SuccessRate = %f, want 0 (no tasks)", result.SuccessRate)
	}
}

func TestCompute_DBOnlyNoReports(t *testing.T) {
	database := openTestDB(t)
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)

	sqlDB := database.SQL()
	_, err := sqlDB.Exec(
		`INSERT INTO run_history (id, start_time, end_time, project, tasks, tokens_used, status) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"run-1", now, now.Add(10*time.Minute), "/proj", "[]", 3000, "completed",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	s := New(database, t.TempDir())
	result, err := s.Compute()
	if err != nil {
		t.Fatalf("compute: %v", err)
	}

	// With no reports but 1 DB row, TotalRuns should be 1
	if result.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", result.TotalRuns)
	}
	// Tokens from DB fallback
	if result.TotalTokensUsed != 3000 {
		t.Errorf("TotalTokensUsed = %d, want 3000", result.TotalTokensUsed)
	}
}

// --- JSON round-trip test ---

func TestStatsResult_JSONRoundTrip(t *testing.T) {
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)
	original := &StatsResult{
		TotalRuns:       5,
		FirstRunAt:      &now,
		TotalDuration:   Duration{2 * time.Hour},
		AvgRunDuration:  Duration{24 * time.Minute},
		TasksCompleted:  10,
		TasksFailed:     2,
		SuccessRate:     83.33,
		PRsCreated:      3,
		PRURLs:          []string{"https://github.com/org/repo/pull/1"},
		TotalTokensUsed: 50000,
		AvgTokensPerRun: 10000,
		TotalProjects:   2,
		TaskTypeBreakdown: map[string]int{
			"code-review": 5,
			"refactor":    3,
		},
		BudgetProjection: &BudgetProjection{
			Provider:         "claude",
			WeeklyBudget:     500000,
			CurrentUsedPct:   40.0,
			AvgDailyUsage:    2000,
			AvgHourlyUsage:   83.33,
			RemainingTokens:  300000,
			EstDaysRemaining: 150,
			Source:           "calibrated",
		},
		BudgetProjections: []BudgetProjection{
			{
				Provider:         "claude",
				WeeklyBudget:     500000,
				CurrentUsedPct:   40.0,
				AvgDailyUsage:    2000,
				AvgHourlyUsage:   83.33,
				RemainingTokens:  300000,
				EstDaysRemaining: 150,
				Source:           "calibrated",
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded StatsResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.TotalRuns != original.TotalRuns {
		t.Errorf("TotalRuns = %d, want %d", decoded.TotalRuns, original.TotalRuns)
	}
	if decoded.TotalDuration.Duration != original.TotalDuration.Duration {
		t.Errorf("TotalDuration = %v, want %v", decoded.TotalDuration.Duration, original.TotalDuration.Duration)
	}
	if decoded.BudgetProjection == nil {
		t.Fatal("BudgetProjection should not be nil after round-trip")
	}
	if decoded.BudgetProjection.EstDaysRemaining != 150 {
		t.Errorf("EstDaysRemaining = %d, want 150", decoded.BudgetProjection.EstDaysRemaining)
	}
	if len(decoded.BudgetProjections) != 1 {
		t.Errorf("len(BudgetProjections) = %d, want 1", len(decoded.BudgetProjections))
	}
}
