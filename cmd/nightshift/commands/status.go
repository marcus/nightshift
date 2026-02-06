package commands

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/state"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show run history",
	Long: `Display nightshift run history and activity.

Shows the last N runs (default: 5) or today's activity summary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		last, _ := cmd.Flags().GetInt("last")
		today, _ := cmd.Flags().GetBool("today")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		database, err := db.Open(cfg.ExpandedDBPath())
		if err != nil {
			return fmt.Errorf("opening db: %w", err)
		}
		defer func() { _ = database.Close() }()

		st, err := state.New(database)
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		if today {
			return showTodaySummary(st)
		}
		return showLastRuns(st, last)
	},
}

func init() {
	statusCmd.Flags().IntP("last", "n", 5, "Show last N runs")
	statusCmd.Flags().Bool("today", false, "Show today's activity summary")
	rootCmd.AddCommand(statusCmd)
}

func showLastRuns(st *state.State, n int) error {
	runs := st.GetRunHistory(n)

	if len(runs) == 0 {
		fmt.Println("No run history found.")
		return nil
	}

	fmt.Printf("Last %d runs:\n\n", len(runs))

	for _, run := range runs {
		printRunRecord(run)
		fmt.Println()
	}

	return nil
}

func showTodaySummary(st *state.State) error {
	summary := st.GetTodaySummary()

	if summary.TotalRuns == 0 {
		fmt.Println("No runs today.")
		return nil
	}

	fmt.Printf("Today's Activity Summary\n")
	fmt.Printf("========================\n\n")

	fmt.Printf("Runs:    %d total (%d success, %d failed)\n",
		summary.TotalRuns, summary.SuccessfulRuns, summary.FailedRuns)
	fmt.Printf("Tokens:  %s\n", formatTokens(summary.TotalTokens))

	if len(summary.Projects) > 0 {
		fmt.Printf("\nProjects (%d):\n", len(summary.Projects))
		for _, p := range summary.Projects {
			fmt.Printf("  - %s\n", filepath.Base(p))
		}
	}

	if len(summary.TaskCounts) > 0 {
		fmt.Printf("\nTasks executed:\n")
		for task, count := range summary.TaskCounts {
			fmt.Printf("  - %s: %d\n", task, count)
		}
	}

	return nil
}

func printRunRecord(run state.RunRecord) {
	status := formatStatus(run.Status)
	duration := run.EndTime.Sub(run.StartTime)

	fmt.Printf("[%s] %s\n", run.StartTime.Format("2006-01-02 15:04"), status)

	if run.Project != "" {
		fmt.Printf("  Project: %s\n", filepath.Base(run.Project))
	}
	if run.Provider != "" {
		fmt.Printf("  Provider: %s\n", run.Provider)
	}

	if len(run.Tasks) > 0 {
		fmt.Printf("  Tasks:   %s\n", strings.Join(run.Tasks, ", "))
	}

	if run.TokensUsed > 0 {
		fmt.Printf("  Tokens:  %s\n", formatTokens(run.TokensUsed))
	}

	if duration > 0 {
		fmt.Printf("  Duration: %s\n", formatDuration(duration))
	}

	if run.Error != "" {
		fmt.Printf("  Error:   %s\n", run.Error)
	}
}

func formatStatus(status string) string {
	switch status {
	case "success":
		return "SUCCESS"
	case "failed":
		return "FAILED"
	case "partial":
		return "PARTIAL"
	default:
		return strings.ToUpper(status)
	}
}

func formatTokens(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}

func formatDuration(d time.Duration) string {
	if d >= time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	if d >= time.Minute {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
