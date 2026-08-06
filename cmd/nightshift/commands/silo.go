package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/analysis"
	"github.com/marcus/nightshift/internal/analysis/silo"
)

var siloCmd = &cobra.Command{
	Use:   "silo [path]",
	Short: "Detect knowledge silos in git history",
	Long: `Analyze git history to find files or directories where one author
dominates recent commits, surfacing knowledge silos and bus-factor risk.`,
	RunE: runSilo,
}

func init() {
	siloCmd.Flags().String("since", "180d", "Window to analyze (e.g. 180d, 30d, 2024-01-01)")
	siloCmd.Flags().Float64("threshold", 0.8, "Dominance ratio above which a path is flagged a silo")
	siloCmd.Flags().String("path", "", "Restrict analysis to a path prefix")
	siloCmd.Flags().String("format", "table", "Output format: table|json")
	siloCmd.Flags().Bool("dirs", false, "Aggregate by top-level directory instead of files")
	siloCmd.Flags().Int("limit", 25, "Max rows to display in table output")
	rootCmd.AddCommand(siloCmd)
}

func runSilo(cmd *cobra.Command, args []string) error {
	repo, err := os.Getwd()
	if err != nil {
		return err
	}
	if len(args) > 0 {
		repo = args[0]
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	if !analysis.RepositoryExists(abs) {
		return fmt.Errorf("not a git repository: %s", abs)
	}

	sinceStr, _ := cmd.Flags().GetString("since")
	threshold, _ := cmd.Flags().GetFloat64("threshold")
	pathFilter, _ := cmd.Flags().GetString("path")
	format, _ := cmd.Flags().GetString("format")
	dirs, _ := cmd.Flags().GetBool("dirs")
	limit, _ := cmd.Flags().GetInt("limit")

	since, err := parseSince(sinceStr)
	if err != nil {
		return err
	}

	src := silo.NewGitCommitSource(abs, since)
	results, err := silo.Analyze(src, silo.Options{
		Since:              since,
		PathFilter:         pathFilter,
		DominanceThreshold: threshold,
		GroupByDir:         dirs,
	})
	if err != nil {
		return err
	}

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "RISK\tSILO\tCOMMITS\tAUTHORS\tDOMINANCE\tOWNER\tPATH")
	for i, r := range results {
		if i >= limit {
			break
		}
		flag := " "
		if r.IsSilo {
			flag = "*"
		}
		fmt.Fprintf(tw, "%.2f\t%s\t%d\t%d\t%.0f%%\t%s\t%s\n",
			r.Risk, flag, r.Commits, r.Authors, r.Dominance*100, r.TopAuthor, r.Path)
	}
	return tw.Flush()
}

func parseSince(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	// support "180d", "30d"
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Now().AddDate(0, 0, -days), nil
		}
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("invalid --since: %s", s)
}
