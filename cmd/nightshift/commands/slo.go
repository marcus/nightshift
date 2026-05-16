package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/reporting"
	"github.com/marcus/nightshift/internal/slo"
)

var sloCmd = &cobra.Command{
	Use:   "slo",
	Short: "Suggest SLO/SLA candidates from nightshift run history",
	Long: `Analyze existing nightshift run history and suggest concrete SLO/SLA
targets with rationale and confidence labels.

Suggestions cover reliability (task success rate), latency (p95 task
duration), throughput (PRs per run), cost (p90 tokens per run), and
per-project availability. The command is read-only: it does not persist
suggestions or modify configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		windowStr, _ := cmd.Flags().GetString("window")
		project, _ := cmd.Flags().GetString("project")
		format, _ := cmd.Flags().GetString("format")
		minConf, _ := cmd.Flags().GetString("min-confidence")
		reportsDir, _ := cmd.Flags().GetString("reports-dir")

		window, err := parseSLOWindow(windowStr)
		if err != nil {
			return err
		}
		conf, err := parseSLOConfidence(minConf)
		if err != nil {
			return err
		}
		if reportsDir == "" {
			reportsDir = reporting.DefaultReportsDir()
		}

		runs, err := loadRunResultsFromDir(reportsDir)
		if err != nil {
			return fmt.Errorf("loading run reports: %w", err)
		}

		candidates := slo.Suggest(runs, slo.Options{
			Window:        window,
			Project:       project,
			MinConfidence: conf,
		})

		switch strings.ToLower(format) {
		case "", "text":
			return renderSLOText(os.Stdout, candidates, windowStr, project)
		case "json":
			return renderSLOJSON(os.Stdout, candidates)
		default:
			return fmt.Errorf("unsupported --format %q (use text or json)", format)
		}
	},
}

func init() {
	sloCmd.Flags().String("window", "30d", "Lookback window: e.g. 7d, 30d, 90d, or 'all'")
	sloCmd.Flags().String("project", "", "Filter to a single project (basename of project path)")
	sloCmd.Flags().String("format", "text", "Output format: text or json")
	sloCmd.Flags().String("min-confidence", "", "Minimum confidence to include: low, medium, high")
	sloCmd.Flags().String("reports-dir", "", "Override reports directory (defaults to nightshift's reports dir)")
	rootCmd.AddCommand(sloCmd)
}

func parseSLOWindow(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "all" {
		return 0, nil
	}
	if strings.HasSuffix(s, "d") {
		days, err := parseUint(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid --window %q: %w", s, err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --window %q: %w", s, err)
	}
	return d, nil
}

func parseUint(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a non-negative integer: %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func parseSLOConfidence(s string) (slo.Confidence, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return "", nil
	case "low":
		return slo.ConfidenceLow, nil
	case "medium", "med":
		return slo.ConfidenceMedium, nil
	case "high":
		return slo.ConfidenceHigh, nil
	default:
		return "", fmt.Errorf("invalid --min-confidence %q (use low, medium, high)", s)
	}
}

func loadRunResultsFromDir(dir string) ([]*reporting.RunResults, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var runs []*reporting.RunResults
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "run-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		r, err := reporting.LoadRunResults(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		runs = append(runs, r)
	}
	return runs, nil
}

func renderSLOJSON(w io.Writer, candidates []slo.Candidate) error {
	payload := struct {
		Generated  time.Time       `json:"generated_at"`
		Candidates []slo.Candidate `json:"candidates"`
	}{
		Generated:  time.Now().UTC(),
		Candidates: candidates,
	}
	if payload.Candidates == nil {
		payload.Candidates = []slo.Candidate{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderSLOText(w io.Writer, candidates []slo.Candidate, window, project string) error {
	fmt.Fprintln(w, "SLO/SLA Candidate Suggestions")
	fmt.Fprintln(w, "================================")
	fmt.Fprintf(w, "  Window:  %s\n", strings.TrimSpace(window))
	if project != "" {
		fmt.Fprintf(w, "  Project: %s\n", project)
	}
	fmt.Fprintln(w)

	if len(candidates) == 0 {
		fmt.Fprintln(w, "No SLO candidates: insufficient run history in the selected window.")
		fmt.Fprintln(w, "Try widening --window or running more nightshift jobs first.")
		return nil
	}

	groups := map[slo.Category][]slo.Candidate{}
	for _, c := range candidates {
		groups[c.Category] = append(groups[c.Category], c)
	}

	order := []struct {
		cat   slo.Category
		title string
	}{
		{slo.CategoryReliability, "Reliability"},
		{slo.CategoryLatency, "Latency"},
		{slo.CategoryThroughput, "Throughput"},
		{slo.CategoryCost, "Cost"},
		{slo.CategoryAvailability, "Per-Project Availability"},
	}

	for _, sec := range order {
		items := groups[sec.cat]
		if len(items) == 0 {
			continue
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].Project != items[j].Project {
				return items[i].Project < items[j].Project
			}
			return items[i].Name < items[j].Name
		})
		fmt.Fprintf(w, "%s\n", sec.title)
		for _, c := range items {
			head := c.Name
			if c.Project != "" {
				head = fmt.Sprintf("%s (%s)", c.Name, c.Project)
			}
			fmt.Fprintf(w, "  • %s\n", head)
			fmt.Fprintf(w, "      target:     %s\n", c.Target)
			fmt.Fprintf(w, "      metric:     %s\n", c.Metric)
			fmt.Fprintf(w, "      window:     %s\n", c.Window)
			fmt.Fprintf(w, "      sample:     n=%d (%s confidence)\n", c.SampleSize, c.Confidence)
			fmt.Fprintf(w, "      rationale:  %s\n", c.Rationale)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Suggestions are informational and based purely on observed history.")
	return nil
}
