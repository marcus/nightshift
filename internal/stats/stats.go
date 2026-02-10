// Package stats computes aggregate statistics from nightshift run data.
// It reads from existing report JSONs, run_history, snapshots, and projects tables.
package stats

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/reporting"
)

// Duration wraps time.Duration for clean JSON serialization as seconds.
type Duration struct {
	time.Duration
}

// MarshalJSON serializes Duration as integer seconds.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(d.Seconds()))
}

// UnmarshalJSON deserializes Duration from integer seconds.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var secs int64
	if err := json.Unmarshal(b, &secs); err != nil {
		return err
	}
	d.Duration = time.Duration(secs) * time.Second
	return nil
}

// String returns a human-readable duration string.
func (d Duration) String() string {
	dur := d.Duration
	if dur < time.Minute {
		return fmt.Sprintf("%ds", int(dur.Seconds()))
	}
	if dur < time.Hour {
		return fmt.Sprintf("%dm %ds", int(dur.Minutes()), int(dur.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(dur.Hours()), int(dur.Minutes())%60)
}

// StatsResult holds all computed statistics, JSON-serializable.
type StatsResult struct {
	// Run overview
	TotalRuns      int        `json:"total_runs"`
	FirstRunAt     *time.Time `json:"first_run_at,omitempty"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	TotalDuration  Duration   `json:"total_duration"`
	AvgRunDuration Duration   `json:"avg_run_duration"`

	// Task outcomes
	TasksCompleted int     `json:"tasks_completed"`
	TasksFailed    int     `json:"tasks_failed"`
	TasksSkipped   int     `json:"tasks_skipped"`
	SuccessRate    float64 `json:"success_rate"`

	// PR output
	PRsCreated int      `json:"prs_created"`
	PRURLs     []string `json:"pr_urls,omitempty"`

	// Token usage
	TotalTokensUsed int `json:"total_tokens_used"`
	AvgTokensPerRun int `json:"avg_tokens_per_run"`

	// Budget
	BudgetProjection  *BudgetProjection  `json:"budget_projection,omitempty"` // Deprecated: use BudgetProjections.
	BudgetProjections []BudgetProjection `json:"budget_projections,omitempty"`

	// Projects
	TotalProjects    int            `json:"total_projects"`
	ProjectBreakdown []ProjectStats `json:"project_breakdown,omitempty"`

	// Task types
	TaskTypeBreakdown map[string]int `json:"task_type_breakdown,omitempty"`
}

// BudgetProjection estimates remaining budget days from snapshot data.
type BudgetProjection struct {
	Provider               string     `json:"provider"`
	WeeklyBudget           int64      `json:"weekly_budget"`
	CurrentUsedPct         float64    `json:"current_used_pct"`
	AvgDailyUsage          int64      `json:"avg_daily_usage"`
	AvgHourlyUsage         float64    `json:"avg_hourly_usage"`
	RemainingTokens        int64      `json:"remaining_tokens"`
	EstDaysRemaining       int        `json:"est_days_remaining"`
	EstHoursRemaining      float64    `json:"est_hours_remaining,omitempty"`
	EstExhaustAt           *time.Time `json:"est_exhaust_at,omitempty"`
	ResetAt                *time.Time `json:"reset_at,omitempty"`
	TimeUntilResetSec      int64      `json:"time_until_reset_sec,omitempty"`
	ResetHint              string     `json:"reset_hint,omitempty"`
	WillExhaustBeforeReset *bool      `json:"will_exhaust_before_reset,omitempty"`
	Source                 string     `json:"source"`
}

// ProjectStats summarizes activity for a single project.
type ProjectStats struct {
	Name      string `json:"name"`
	RunCount  int    `json:"run_count"`
	TaskCount int    `json:"task_count"`
}

// Stats computes aggregate statistics from nightshift data sources.
type Stats struct {
	db           *db.DB
	reportsDir   string
	nowFunc      func() time.Time
	budgetSource budget.BudgetSource
}

// New creates a Stats instance.
func New(database *db.DB, reportsDir string) *Stats {
	return &Stats{
		db:         database,
		reportsDir: reportsDir,
		nowFunc:    time.Now,
	}
}

// NewWithBudgetSource creates a Stats instance with a calibrated budget source.
func NewWithBudgetSource(database *db.DB, reportsDir string, source budget.BudgetSource) *Stats {
	return &Stats{
		db:           database,
		reportsDir:   reportsDir,
		nowFunc:      time.Now,
		budgetSource: source,
	}
}

// Compute aggregates all available data into a StatsResult.
func (s *Stats) Compute() (*StatsResult, error) {
	result := &StatsResult{
		TaskTypeBreakdown: make(map[string]int),
	}

	// Load report JSONs for task-level stats
	reports := s.loadReports()
	s.computeFromReports(result, reports)

	// Enrich from run_history DB (run count, date range, tokens)
	if s.db != nil {
		s.computeFromRunHistory(result)
		s.computeFromProjects(result)
		s.computeBudgetProjections(result)
	}

	// Compute averages
	if result.TotalRuns > 0 {
		result.AvgRunDuration = Duration{time.Duration(int64(result.TotalDuration.Duration) / int64(result.TotalRuns))}
		if result.TotalTokensUsed > 0 {
			result.AvgTokensPerRun = result.TotalTokensUsed / result.TotalRuns
		}
	}

	// Success rate
	totalTasks := result.TasksCompleted + result.TasksFailed + result.TasksSkipped
	if totalTasks > 0 {
		result.SuccessRate = float64(result.TasksCompleted) / float64(totalTasks) * 100
	}

	return result, nil
}

// loadReports reads all run-*.json files from the reports directory.
func (s *Stats) loadReports() []*reporting.RunResults {
	if s.reportsDir == "" {
		return nil
	}

	entries, err := os.ReadDir(s.reportsDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("stats: read reports dir: %v", err)
		}
		return nil
	}

	var results []*reporting.RunResults
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "run-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(s.reportsDir, name)
		r, err := reporting.LoadRunResults(path)
		if err != nil {
			log.Printf("stats: load report %s: %v", name, err)
			continue
		}
		results = append(results, r)
	}

	// Sort by start time ascending
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartTime.Before(results[j].StartTime)
	})

	return results
}

// computeFromReports extracts task-level stats from report JSON files.
func (s *Stats) computeFromReports(result *StatsResult, reports []*reporting.RunResults) {
	if len(reports) == 0 {
		return
	}

	result.TotalRuns = len(reports)

	projectTaskCounts := make(map[string]int)
	prURLSet := make(map[string]struct{})

	for _, r := range reports {
		// Date range from reports
		if !r.StartTime.IsZero() {
			if result.FirstRunAt == nil || r.StartTime.Before(*result.FirstRunAt) {
				t := r.StartTime
				result.FirstRunAt = &t
			}
			if result.LastRunAt == nil || r.StartTime.After(*result.LastRunAt) {
				t := r.StartTime
				result.LastRunAt = &t
			}
		}

		// Duration
		if !r.StartTime.IsZero() && !r.EndTime.IsZero() {
			result.TotalDuration.Duration += r.EndTime.Sub(r.StartTime)
		}

		// Token usage from report-level budget data
		if r.UsedBudget > 0 {
			result.TotalTokensUsed += r.UsedBudget
		}

		for _, task := range r.Tasks {
			switch task.Status {
			case "completed":
				result.TasksCompleted++
			case "failed":
				result.TasksFailed++
			case "skipped":
				result.TasksSkipped++
			}

			// Task type breakdown
			if task.TaskType != "" {
				result.TaskTypeBreakdown[task.TaskType]++
			}

			// PR detection
			if strings.EqualFold(task.OutputType, "pr") && task.OutputRef != "" {
				result.PRsCreated++
				if strings.HasPrefix(task.OutputRef, "http") {
					prURLSet[task.OutputRef] = struct{}{}
				}
			}

			// Per-project task counts
			if task.Project != "" {
				projectTaskCounts[filepath.Base(task.Project)]++
			}

			// Accumulate tokens from tasks if report-level budget is missing
			if r.UsedBudget == 0 && task.TokensUsed > 0 {
				result.TotalTokensUsed += task.TokensUsed
			}
		}
	}

	// Collect unique PR URLs (most recent first)
	if len(prURLSet) > 0 {
		urls := make([]string, 0, len(prURLSet))
		for url := range prURLSet {
			urls = append(urls, url)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(urls)))
		result.PRURLs = urls
	}

	// Build project breakdown from reports
	for name, count := range projectTaskCounts {
		result.ProjectBreakdown = append(result.ProjectBreakdown, ProjectStats{
			Name:      name,
			TaskCount: count,
		})
	}
	sort.Slice(result.ProjectBreakdown, func(i, j int) bool {
		return result.ProjectBreakdown[i].TaskCount > result.ProjectBreakdown[j].TaskCount
	})
}

// computeFromRunHistory queries the run_history table for run-level stats.
func (s *Stats) computeFromRunHistory(result *StatsResult) {
	sqlDB := s.db.SQL()
	if sqlDB == nil {
		return
	}

	// Total run count — use max of DB and report-derived count
	row := sqlDB.QueryRow(`SELECT COUNT(*) FROM run_history`)
	var count int
	if err := row.Scan(&count); err != nil {
		log.Printf("stats: count run_history: %v", err)
		return
	}
	if count > result.TotalRuns {
		result.TotalRuns = count
	}

	if count == 0 {
		return
	}

	// First and last run times — scan as strings because the modernc SQLite
	// driver returns aggregated DATETIME columns as strings, not time.Time.
	row = sqlDB.QueryRow(`SELECT CAST(MIN(start_time) AS TEXT), CAST(MAX(start_time) AS TEXT) FROM run_history`)
	var firstRaw, lastRaw sql.NullString
	if err := row.Scan(&firstRaw, &lastRaw); err != nil {
		log.Printf("stats: run_history min/max: %v", err)
	} else {
		if firstRaw.Valid {
			if t, ok := parseDBTimestamp(firstRaw.String); ok && (result.FirstRunAt == nil || t.Before(*result.FirstRunAt)) {
				result.FirstRunAt = &t
			}
		}
		if lastRaw.Valid {
			if t, ok := parseDBTimestamp(lastRaw.String); ok && (result.LastRunAt == nil || t.After(*result.LastRunAt)) {
				result.LastRunAt = &t
			}
		}
	}

	// Sum tokens from run_history if reports gave us nothing
	if result.TotalTokensUsed == 0 {
		row = sqlDB.QueryRow(`SELECT COALESCE(SUM(tokens_used), 0) FROM run_history`)
		var totalTokens int
		if err := row.Scan(&totalTokens); err != nil {
			log.Printf("stats: sum tokens: %v", err)
		} else {
			result.TotalTokensUsed = totalTokens
		}
	}
}

// computeFromProjects queries the projects table for project count and run counts.
func (s *Stats) computeFromProjects(result *StatsResult) {
	sqlDB := s.db.SQL()
	if sqlDB == nil {
		return
	}

	rows, err := sqlDB.Query(`SELECT path, run_count FROM projects ORDER BY run_count DESC`)
	if err != nil {
		log.Printf("stats: query projects: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	projectRunCounts := make(map[string]int)
	for rows.Next() {
		var path string
		var runCount int
		if err := rows.Scan(&path, &runCount); err != nil {
			log.Printf("stats: scan project: %v", err)
			continue
		}
		name := filepath.Base(path)
		projectRunCounts[name] = runCount
	}
	if err := rows.Err(); err != nil {
		log.Printf("stats: projects rows: %v", err)
	}

	result.TotalProjects = len(projectRunCounts)

	// Merge run_count from projects into existing project breakdown.
	// Use indices (not pointers) so appends don't invalidate references.
	existing := make(map[string]int) // name → index in ProjectBreakdown
	for i := range result.ProjectBreakdown {
		existing[result.ProjectBreakdown[i].Name] = i
	}
	for name, runCount := range projectRunCounts {
		if idx, ok := existing[name]; ok {
			result.ProjectBreakdown[idx].RunCount = runCount
		} else {
			result.ProjectBreakdown = append(result.ProjectBreakdown, ProjectStats{
				Name:     name,
				RunCount: runCount,
			})
		}
	}

	// Re-sort by task count descending, then run count
	sort.Slice(result.ProjectBreakdown, func(i, j int) bool {
		if result.ProjectBreakdown[i].TaskCount != result.ProjectBreakdown[j].TaskCount {
			return result.ProjectBreakdown[i].TaskCount > result.ProjectBreakdown[j].TaskCount
		}
		return result.ProjectBreakdown[i].RunCount > result.ProjectBreakdown[j].RunCount
	})
}

// computeBudgetProjections estimates projection windows for available providers.
func (s *Stats) computeBudgetProjections(result *StatsResult) {
	sqlDB := s.db.SQL()
	if sqlDB == nil {
		return
	}

	now := time.Now()
	if s.nowFunc != nil {
		now = s.nowFunc()
	}

	// Keep output stable and always include both providers when possible.
	for _, provider := range []string{"codex", "claude"} {
		proj, ok := s.computeProviderBudgetProjection(sqlDB, provider, now)
		if !ok {
			continue
		}
		result.BudgetProjections = append(result.BudgetProjections, proj)
	}

	if len(result.BudgetProjections) == 0 {
		return
	}

	// Back-compat single projection field.
	legacy := result.BudgetProjections[0]
	result.BudgetProjection = &legacy
}

func (s *Stats) computeProviderBudgetProjection(sqlDB *sql.DB, provider string, now time.Time) (BudgetProjection, bool) {
	if sqlDB == nil {
		return BudgetProjection{}, false
	}

	// Latest calibrated snapshot for this provider.
	row := sqlDB.QueryRow(
		`SELECT CAST(timestamp AS TEXT), CAST(week_start AS TEXT), local_tokens, scraped_pct, inferred_budget, COALESCE(weekly_reset_time, '')
		 FROM snapshots
		 WHERE provider = ? AND inferred_budget IS NOT NULL AND inferred_budget > 0
		 ORDER BY timestamp DESC
		 LIMIT 1`,
		provider,
	)
	var (
		tsRaw          string
		weekStartRaw   string
		localTokens    int64
		scrapedPct     sql.NullFloat64
		inferredBudget sql.NullInt64
		weeklyResetRaw string
	)
	if err := row.Scan(&tsRaw, &weekStartRaw, &localTokens, &scrapedPct, &inferredBudget, &weeklyResetRaw); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("stats: latest %s snapshot: %v", provider, err)
		}
		return BudgetProjection{}, false
	}

	if !inferredBudget.Valid || inferredBudget.Int64 <= 0 {
		return BudgetProjection{}, false
	}

	// Override budget from calibrator when available.
	budgetValue := inferredBudget.Int64
	budgetSourceLabel := "calibrated"
	if s.budgetSource != nil {
		if est, err := s.budgetSource.GetBudget(provider); err == nil && est.WeeklyTokens > 0 {
			budgetValue = est.WeeklyTokens
			if est.Source != "" {
				budgetSourceLabel = est.Source
			}
		}
	}

	// Average each day's max local_daily, preferring the current billing week
	// to avoid spanning week boundaries. Falls back to rolling 7-day window
	// when the current week has fewer than 2 days of data.
	var avgDaily sql.NullFloat64
	useWeekWindow := false

	if weekStart, ok := parseDBTimestamp(weekStartRaw); ok {
		var dayCount int
		// Format as ISO string for consistent SQLite text comparison.
		weekStartStr := weekStart.Format(time.RFC3339)
		row = sqlDB.QueryRow(
			`SELECT AVG(day_max), COUNT(*) FROM (
			   SELECT SUBSTR(timestamp, 1, 10) AS day, MAX(local_daily) AS day_max
			   FROM snapshots
			   WHERE provider = ? AND timestamp >= ? AND local_daily > 0
			   GROUP BY SUBSTR(timestamp, 1, 10)
			 )`,
			provider,
			weekStartStr,
		)
		if err := row.Scan(&avgDaily, &dayCount); err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("stats: week avg daily for %s: %v", provider, err)
		}
		if avgDaily.Valid && avgDaily.Float64 > 0 && dayCount >= 2 {
			useWeekWindow = true
		}
	}

	if !useWeekWindow {
		cutoff := now.AddDate(0, 0, -7)
		row = sqlDB.QueryRow(
			`SELECT AVG(day_max)
			 FROM (
			   SELECT SUBSTR(timestamp, 1, 10) AS day, MAX(local_daily) AS day_max
			   FROM snapshots
			   WHERE provider = ? AND timestamp >= ? AND local_daily > 0
			   GROUP BY SUBSTR(timestamp, 1, 10)
			 )`,
			provider,
			cutoff,
		)
		avgDaily = sql.NullFloat64{}
		if err := row.Scan(&avgDaily); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				log.Printf("stats: avg daily usage for %s: %v", provider, err)
			}
			return BudgetProjection{}, false
		}
	}

	if !avgDaily.Valid || avgDaily.Float64 <= 0 {
		return BudgetProjection{}, false
	}

	usedPct := 0.0
	if scrapedPct.Valid {
		usedPct = scrapedPct.Float64
	} else if budgetValue > 0 && localTokens > 0 {
		usedPct = (float64(localTokens) / float64(budgetValue)) * 100
	}
	if usedPct < 0 {
		usedPct = 0
	}
	if usedPct > 100 {
		usedPct = 100
	}

	remaining := int64(math.Round(float64(budgetValue) * (1 - usedPct/100)))
	if remaining < 0 {
		remaining = 0
	}

	proj := BudgetProjection{
		Provider:        provider,
		WeeklyBudget:    budgetValue,
		CurrentUsedPct:  usedPct,
		AvgDailyUsage:   int64(math.Round(avgDaily.Float64)),
		AvgHourlyUsage:  avgDaily.Float64 / 24.0,
		RemainingTokens: remaining,
		Source:          budgetSourceLabel,
	}

	if proj.AvgDailyUsage > 0 && remaining > 0 {
		proj.EstDaysRemaining = int(float64(remaining) / float64(proj.AvgDailyUsage))
		proj.EstHoursRemaining = (float64(remaining) / float64(proj.AvgDailyUsage)) * 24.0
		exhaustAt := now.Add(time.Duration(proj.EstHoursRemaining * float64(time.Hour)))
		proj.EstExhaustAt = &exhaustAt
	}

	snapshotAt := now
	if parsed, ok := parseDBTimestamp(tsRaw); ok {
		snapshotAt = parsed
	}
	if resetAt, ok := resolveResetAt(provider, weeklyResetRaw, weekStartRaw, snapshotAt, now); ok {
		proj.ResetAt = &resetAt
		proj.TimeUntilResetSec = int64(resetAt.Sub(now).Seconds())
		if proj.EstExhaustAt != nil {
			will := proj.EstExhaustAt.Before(resetAt)
			proj.WillExhaustBeforeReset = &will
		}
	} else if strings.TrimSpace(weeklyResetRaw) != "" {
		proj.ResetHint = strings.TrimSpace(weeklyResetRaw)
	}

	return proj, true
}

var resetZoneSuffixRe = regexp.MustCompile(`\s+\(([^)]+)\)\s*$`)

func resolveResetAt(provider, weeklyResetRaw, weekStartRaw string, snapshotAt, now time.Time) (time.Time, bool) {
	if at, ok := parseWeeklyResetTime(weeklyResetRaw, snapshotAt, now); ok {
		return at, true
	}

	if weekStart, ok := parseDBTimestamp(weekStartRaw); ok {
		resetAt := weekStart.Add(7 * 24 * time.Hour)
		for resetAt.Before(now) {
			resetAt = resetAt.Add(7 * 24 * time.Hour)
		}
		return resetAt, true
	}

	_ = provider
	return time.Time{}, false
}

func parseWeeklyResetTime(raw string, snapshotAt, now time.Time) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}

	loc := snapshotAt.Location()
	if m := resetZoneSuffixRe.FindStringSubmatch(raw); len(m) == 2 {
		if parsedLoc, err := time.LoadLocation(strings.TrimSpace(m[1])); err == nil {
			loc = parsedLoc
		}
		raw = strings.TrimSpace(resetZoneSuffixRe.ReplaceAllString(raw, ""))
	}

	layouts := []string{
		"Jan 2 at 3:04pm",
		"Jan 2 at 3pm",
		"15:04 on 2 Jan",
	}
	ref := snapshotAt.In(loc)
	current := now.In(loc)

	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, raw, loc)
		if err != nil {
			continue
		}

		candidate := time.Date(ref.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
		// Year rollover handling (e.g., Jan dates observed from late Dec snapshots).
		if candidate.Before(ref.Add(-31 * 24 * time.Hour)) {
			candidate = candidate.AddDate(1, 0, 0)
		}
		for candidate.Before(current) {
			candidate = candidate.Add(7 * 24 * time.Hour)
		}
		return candidate, true
	}

	if parsed, ok := parseDBTimestamp(raw); ok {
		for parsed.Before(now) {
			parsed = parsed.Add(7 * 24 * time.Hour)
		}
		return parsed, true
	}

	return time.Time{}, false
}

func parseDBTimestamp(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if idx := strings.Index(raw, " m=+"); idx >= 0 {
		raw = strings.TrimSpace(raw[:idx])
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
