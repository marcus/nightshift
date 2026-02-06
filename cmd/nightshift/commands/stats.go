package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/reporting"
	"github.com/marcus/nightshift/internal/stats"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show aggregate statistics",
	Long: `Display aggregate statistics from all nightshift runs.

Shows run counts, task outcomes, token usage, budget projections,
and per-project breakdowns. Use --json for machine-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		period, _ := cmd.Flags().GetString("period")
		return runStats(jsonOutput, period)
	},
}

func init() {
	statsCmd.Flags().Bool("json", false, "Output as JSON")
	statsCmd.Flags().StringP("period", "p", "all", "Time period: all, last-7d, last-30d, last-night")
	rootCmd.AddCommand(statsCmd)
}

func runStats(jsonOutput bool, period string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer func() { _ = database.Close() }()

	reportsDir := reporting.DefaultReportsDir()
	s := stats.New(database, reportsDir)
	result, err := s.Compute()
	if err != nil {
		return fmt.Errorf("computing stats: %w", err)
	}

	// Apply period filter if not "all"
	if period != "all" {
		result = filterStatsByPeriod(result, s, reportsDir, period)
	}

	if jsonOutput {
		return renderStatsJSON(result)
	}
	return renderStatsHuman(result)
}

// filterStatsByPeriod recomputes stats from reports filtered by the given period.
// For period filtering we reload reports, filter by date, and recompute.
func filterStatsByPeriod(original *stats.StatsResult, s *stats.Stats, reportsDir string, period string) *stats.StatsResult {
	_ = s // stats.Stats doesn't expose period filtering; we do it here

	runs, err := loadRunReports(reportsDir)
	if err != nil || len(runs) == 0 {
		return original
	}

	now := time.Now()
	var filtered []reportRun
	switch strings.ToLower(period) {
	case "last-7d":
		cutoff := now.AddDate(0, 0, -7)
		for _, run := range runs {
			if run.results != nil && !run.results.StartTime.Before(cutoff) {
				filtered = append(filtered, run)
			}
		}
	case "last-30d":
		cutoff := now.AddDate(0, 0, -30)
		for _, run := range runs {
			if run.results != nil && !run.results.StartTime.Before(cutoff) {
				filtered = append(filtered, run)
			}
		}
	case "last-night":
		cfg, _ := config.Load()
		rng, err := resolveReportRange(reportOptions{period: "last-night"}, cfg, now)
		if err != nil {
			return original
		}
		filtered = filterReportRuns(runs, rng, reportOptions{})
	default:
		return original
	}

	if len(filtered) == 0 {
		return &stats.StatsResult{TaskTypeBreakdown: make(map[string]int)}
	}

	// Recompute stats from filtered runs
	return computeStatsFromRuns(filtered)
}

// computeStatsFromRuns builds a StatsResult from a set of report runs.
func computeStatsFromRuns(runs []reportRun) *stats.StatsResult {
	result := &stats.StatsResult{
		TotalRuns:         len(runs),
		TaskTypeBreakdown: make(map[string]int),
	}

	prURLSet := make(map[string]struct{})
	projectTasks := make(map[string]int)
	projectRuns := make(map[string]int)

	for _, run := range runs {
		r := run.results
		if r == nil {
			continue
		}

		// Date range
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

		// Tokens
		if r.UsedBudget > 0 {
			result.TotalTokensUsed += r.UsedBudget
		}

		// Track projects per run
		runProjects := make(map[string]struct{})

		for _, task := range r.Tasks {
			switch task.Status {
			case "completed":
				result.TasksCompleted++
			case "failed":
				result.TasksFailed++
			case "skipped":
				result.TasksSkipped++
			}

			if task.TaskType != "" {
				result.TaskTypeBreakdown[task.TaskType]++
			}

			if strings.EqualFold(task.OutputType, "pr") && task.OutputRef != "" {
				result.PRsCreated++
				if strings.HasPrefix(task.OutputRef, "http") {
					prURLSet[task.OutputRef] = struct{}{}
				}
			}

			if task.Project != "" {
				name := projectLabel(task.Project)
				projectTasks[name]++
				runProjects[name] = struct{}{}
			}

			// Accumulate tokens from tasks if report-level budget is missing
			if r.UsedBudget == 0 && task.TokensUsed > 0 {
				result.TotalTokensUsed += task.TokensUsed
			}
		}

		for name := range runProjects {
			projectRuns[name]++
		}
	}

	// PR URLs
	if len(prURLSet) > 0 {
		urls := make([]string, 0, len(prURLSet))
		for url := range prURLSet {
			urls = append(urls, url)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(urls)))
		result.PRURLs = urls
	}

	// Projects
	allProjects := make(map[string]struct{})
	for name := range projectTasks {
		allProjects[name] = struct{}{}
	}
	for name := range projectRuns {
		allProjects[name] = struct{}{}
	}
	result.TotalProjects = len(allProjects)

	for name := range allProjects {
		result.ProjectBreakdown = append(result.ProjectBreakdown, stats.ProjectStats{
			Name:      name,
			RunCount:  projectRuns[name],
			TaskCount: projectTasks[name],
		})
	}
	sort.Slice(result.ProjectBreakdown, func(i, j int) bool {
		if result.ProjectBreakdown[i].TaskCount != result.ProjectBreakdown[j].TaskCount {
			return result.ProjectBreakdown[i].TaskCount > result.ProjectBreakdown[j].TaskCount
		}
		return result.ProjectBreakdown[i].RunCount > result.ProjectBreakdown[j].RunCount
	})

	// Averages
	if result.TotalRuns > 0 {
		result.AvgRunDuration = stats.Duration{Duration: result.TotalDuration.Duration / time.Duration(result.TotalRuns)}
		if result.TotalTokensUsed > 0 {
			result.AvgTokensPerRun = result.TotalTokensUsed / result.TotalRuns
		}
	}

	// Success rate
	totalTasks := result.TasksCompleted + result.TasksFailed + result.TasksSkipped
	if totalTasks > 0 {
		result.SuccessRate = float64(result.TasksCompleted) / float64(totalTasks) * 100
	}

	return result
}

func renderStatsJSON(result *stats.StatsResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func renderStatsHuman(result *stats.StatsResult) error {
	fmt.Println("Nightshift Stats")
	fmt.Println("================================")
	fmt.Println()

	// Runs section
	fmt.Println("Runs")
	fmt.Printf("  Total:        %d runs\n", result.TotalRuns)
	if result.FirstRunAt != nil {
		fmt.Printf("  First run:    %s\n", result.FirstRunAt.Format("Jan 2, 2006"))
	}
	if result.LastRunAt != nil {
		fmt.Printf("  Last run:     %s\n", result.LastRunAt.Format("Jan 2, 2006"))
	}
	if result.TotalDuration.Duration > 0 {
		fmt.Printf("  Total time:   %s across all runs\n", result.TotalDuration.String())
	}
	if result.TotalRuns > 0 && result.AvgRunDuration.Duration > 0 {
		fmt.Printf("  Avg duration: %s per run\n", result.AvgRunDuration.String())
	}
	fmt.Println()

	// Tasks section
	totalTasks := result.TasksCompleted + result.TasksFailed + result.TasksSkipped
	fmt.Println("Tasks")
	fmt.Printf("  Completed:    %d", result.TasksCompleted)
	if totalTasks > 0 {
		fmt.Printf(" (%.0f%% success rate)", result.SuccessRate)
	}
	fmt.Println()
	fmt.Printf("  Failed:       %d\n", result.TasksFailed)
	fmt.Printf("  Skipped:      %d\n", result.TasksSkipped)
	fmt.Printf("  PRs created:  %d\n", result.PRsCreated)
	fmt.Println()

	// Tokens section
	fmt.Println("Tokens")
	fmt.Printf("  Total used:   %s tokens\n", formatTokens64(int64(result.TotalTokensUsed)))
	if result.TotalRuns > 0 && result.AvgTokensPerRun > 0 {
		fmt.Printf("  Avg per run:  %s tokens\n", formatTokens64(int64(result.AvgTokensPerRun)))
	}
	fmt.Println()

	// Budget Projection section
	if result.BudgetProjection != nil {
		bp := result.BudgetProjection
		fmt.Println("Budget Projection")
		fmt.Printf("  [%s]\n", bp.Provider)
		fmt.Printf("    Weekly:     %s tokens (%s)\n", formatTokens64(bp.WeeklyBudget), bp.Source)
		fmt.Printf("    Avg daily:  %s tokens\n", formatTokens64(bp.AvgDailyUsage))
		if bp.EstDaysRemaining > 0 {
			fmt.Printf("    At current rate: ~%d days until budget exhausted\n", bp.EstDaysRemaining)
		} else {
			fmt.Printf("    At current rate: budget may be exhausted\n")
		}
		fmt.Println()
	}

	// Projects section
	if len(result.ProjectBreakdown) > 0 {
		fmt.Printf("Projects (%d)\n", result.TotalProjects)
		for _, p := range result.ProjectBreakdown {
			parts := []string{}
			if p.RunCount > 0 {
				parts = append(parts, fmt.Sprintf("%d runs", p.RunCount))
			}
			if p.TaskCount > 0 {
				parts = append(parts, fmt.Sprintf("%d tasks", p.TaskCount))
			}
			fmt.Printf("  %-14s%s\n", p.Name, strings.Join(parts, ", "))
		}
		fmt.Println()
	}

	// Task Types section
	if len(result.TaskTypeBreakdown) > 0 {
		fmt.Println("Task Types")

		// Sort by count descending
		type typeCount struct {
			name  string
			count int
		}
		types := make([]typeCount, 0, len(result.TaskTypeBreakdown))
		for name, count := range result.TaskTypeBreakdown {
			types = append(types, typeCount{name, count})
		}
		sort.Slice(types, func(i, j int) bool {
			return types[i].count > types[j].count
		})

		parts := make([]string, 0, len(types))
		for _, t := range types {
			parts = append(parts, fmt.Sprintf("%s: %d", t.name, t.count))
		}
		fmt.Printf("  %s\n", strings.Join(parts, "  "))
	}

	return nil
}
