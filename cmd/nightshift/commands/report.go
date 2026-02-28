package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/reporting"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

type reportOptions struct {
	reportType string
	period     string
	runs       int
	since      string
	until      string
	format     string
	noColor    bool
	showPaths  bool
	maxItems   int
}

type reportRange struct {
	start time.Time
	end   time.Time
	label string
}

type reportRun struct {
	results    *reporting.RunResults
	reportPath string
	source     string
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show what nightshift did",
	Long: `View structured reports from recent nightshift runs.

By default, shows a polished overview of what happened during the last night.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := reportOptions{}
		opts.reportType, _ = cmd.Flags().GetString("report")
		opts.period, _ = cmd.Flags().GetString("period")
		opts.runs, _ = cmd.Flags().GetInt("runs")
		opts.since, _ = cmd.Flags().GetString("since")
		opts.until, _ = cmd.Flags().GetString("until")
		opts.format, _ = cmd.Flags().GetString("format")
		opts.noColor, _ = cmd.Flags().GetBool("no-color")
		opts.showPaths, _ = cmd.Flags().GetBool("paths")
		opts.maxItems, _ = cmd.Flags().GetInt("max-items")

		if opts.noColor || opts.format == "plain" {
			lipgloss.SetColorProfile(termenv.Ascii)
		}

		cfg, _ := config.Load()

		now := time.Now()
		rng, err := resolveReportRange(opts, cfg, now)
		if err != nil {
			return err
		}

		runs, err := loadRunReports(reporting.DefaultReportsDir())
		if err != nil {
			return err
		}

		periodExplicit := cmd.Flags().Changed("period")
		filtered := filterReportRuns(runs, rng, opts)
		if len(filtered) == 0 && !periodExplicit && opts.since == "" && opts.until == "" {
			// Default period returned nothing — fall back to most recent run(s)
			fallbackRange := reportRange{label: "Last run"}
			filtered = filterReportRuns(runs, fallbackRange, reportOptions{period: "last-run", runs: opts.runs})
			if len(filtered) > 0 {
				rng = fallbackRange
			}
		}
		if len(filtered) == 0 {
			fmt.Println("No run reports found for the selected period.")
			if rng.label != "" {
				fmt.Printf("Period: %s\n", rng.label)
			}
			return nil
		}

		if opts.format == "json" {
			return renderReportJSON(filtered, rng)
		}

		if opts.format == "markdown" {
			return renderReportMarkdown(filtered)
		}

		return renderReportFancy(filtered, rng, opts)
	},
}

func init() {
	reportCmd.Flags().StringP("report", "r", "overview", "Report type: overview | tasks | projects | budget | raw")
	reportCmd.Flags().StringP("period", "p", "last-night", "Time period: last-night | last-run | last-24h | last-7d | today | yesterday | all")
	reportCmd.Flags().IntP("runs", "n", 3, "Max runs to include (0 = all)")
	reportCmd.Flags().String("since", "", "Start time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339)")
	reportCmd.Flags().String("until", "", "End time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339)")
	reportCmd.Flags().String("format", "fancy", "Output format: fancy | plain | markdown | json")
	reportCmd.Flags().Bool("no-color", false, "Disable ANSI colors")
	reportCmd.Flags().Bool("paths", false, "Include report/log file paths")
	reportCmd.Flags().Int("max-items", 5, "Max highlights per run")
	rootCmd.AddCommand(reportCmd)
}

func resolveReportRange(opts reportOptions, cfg *config.Config, now time.Time) (reportRange, error) {
	loc := now.Location()
	if cfg != nil && cfg.Schedule.Window != nil && cfg.Schedule.Window.Timezone != "" {
		if tz, err := time.LoadLocation(cfg.Schedule.Window.Timezone); err == nil {
			loc = tz
			now = now.In(loc)
		}
	}

	if opts.since != "" || opts.until != "" {
		var start, end time.Time
		if opts.since != "" {
			parsed, err := parseTimeInput(opts.since, loc)
			if err != nil {
				return reportRange{}, err
			}
			start = parsed
		}
		if opts.until != "" {
			parsed, err := parseTimeInput(opts.until, loc)
			if err != nil {
				return reportRange{}, err
			}
			end = parsed
		} else {
			end = now
		}
		label := fmt.Sprintf("%s → %s", formatTimeShort(start), formatTimeShort(end))
		return reportRange{start: start, end: end, label: label}, nil
	}

	switch strings.ToLower(opts.period) {
	case "last-run":
		return reportRange{label: "Last run"}, nil
	case "last-night":
		start, end, label, err := lastNightRange(cfg, now, loc)
		if err != nil {
			return reportRange{}, err
		}
		return reportRange{start: start, end: end, label: label}, nil
	case "last-24h":
		start := now.Add(-24 * time.Hour)
		return reportRange{start: start, end: now, label: fmt.Sprintf("%s → %s", formatTimeShort(start), formatTimeShort(now))}, nil
	case "last-7d":
		start := now.AddDate(0, 0, -7)
		return reportRange{start: start, end: now, label: fmt.Sprintf("%s → %s", formatTimeShort(start), formatTimeShort(now))}, nil
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return reportRange{start: start, end: now, label: fmt.Sprintf("Today (%s)", formatTimeShort(start))}, nil
	case "yesterday":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -1)
		end := start.Add(24 * time.Hour)
		return reportRange{start: start, end: end, label: fmt.Sprintf("Yesterday (%s)", start.Format("2006-01-02"))}, nil
	case "all":
		return reportRange{label: "All runs"}, nil
	default:
		return reportRange{}, fmt.Errorf("unknown period %q", opts.period)
	}
}

func lastNightRange(cfg *config.Config, now time.Time, loc *time.Location) (time.Time, time.Time, string, error) {
	startClock := "22:00"
	endClock := "06:00"

	if cfg != nil && cfg.Schedule.Window != nil {
		if cfg.Schedule.Window.Start != "" {
			startClock = cfg.Schedule.Window.Start
		}
		if cfg.Schedule.Window.End != "" {
			endClock = cfg.Schedule.Window.End
		}
	}

	startHour, startMin, err := parseClock(startClock)
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("schedule window start: %w", err)
	}
	endHour, endMin, err := parseClock(endClock)
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("schedule window end: %w", err)
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, loc)
	end := time.Date(now.Year(), now.Month(), now.Day(), endHour, endMin, 0, 0, loc)
	if start.After(end) || start.Equal(end) {
		end = end.Add(24 * time.Hour)
	}
	if now.Before(end) {
		start = start.Add(-24 * time.Hour)
		end = end.Add(-24 * time.Hour)
	}

	label := fmt.Sprintf("%s → %s", formatTimeShort(start), formatTimeShort(end))
	return start, end, label, nil
}

func loadRunReports(dir string) ([]reportRun, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading reports dir: %w", err)
	}

	type record struct {
		ts       time.Time
		jsonPath string
		mdPath   string
	}
	records := make(map[string]*record)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "run-") {
			continue
		}
		ext := filepath.Ext(name)
		if ext != ".json" && ext != ".md" {
			continue
		}
		base := strings.TrimSuffix(name, ext)
		ts, err := parseRunTimestamp(base)
		if err != nil {
			continue
		}
		rec, ok := records[base]
		if !ok {
			rec = &record{ts: ts}
			records[base] = rec
		}
		path := filepath.Join(dir, name)
		if ext == ".json" {
			rec.jsonPath = path
		} else {
			rec.mdPath = path
		}
	}

	list := make([]record, 0, len(records))
	for _, rec := range records {
		list = append(list, *rec)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].ts.After(list[j].ts)
	})

	runs := make([]reportRun, 0, len(list))
	for _, rec := range list {
		run := reportRun{}
		run.reportPath = rec.mdPath

		if rec.jsonPath != "" {
			results, err := reporting.LoadRunResults(rec.jsonPath)
			if err != nil {
				return nil, err
			}
			run.results = results
			run.source = "json"
			runs = append(runs, run)
			continue
		}

		if rec.mdPath != "" {
			payload, err := os.ReadFile(rec.mdPath)
			if err != nil {
				return nil, fmt.Errorf("reading report: %w", err)
			}
			results, err := parseRunReportMarkdown(string(payload))
			if err != nil {
				return nil, err
			}
			if results.EndTime.IsZero() {
				results.EndTime = rec.ts
			}
			run.results = results
			run.source = "markdown"
			runs = append(runs, run)
		}
	}

	return runs, nil
}

func parseRunTimestamp(base string) (time.Time, error) {
	ts := strings.TrimPrefix(base, "run-")
	return time.ParseInLocation("2006-01-02-150405", ts, time.Local)
}

func filterReportRuns(runs []reportRun, rng reportRange, opts reportOptions) []reportRun {
	if len(runs) == 0 {
		return runs
	}

	filtered := make([]reportRun, 0, len(runs))
	for _, run := range runs {
		if run.results == nil {
			continue
		}
		if rng.start.IsZero() && rng.end.IsZero() {
			filtered = append(filtered, run)
			continue
		}

		start := run.results.StartTime
		end := run.results.EndTime
		if start.IsZero() && !end.IsZero() {
			start = end
		}
		if end.IsZero() && !start.IsZero() {
			end = start
		}
		if start.IsZero() && end.IsZero() {
			filtered = append(filtered, run)
			continue
		}

		if !rng.start.IsZero() && end.Before(rng.start) {
			continue
		}
		if !rng.end.IsZero() && start.After(rng.end) {
			continue
		}
		filtered = append(filtered, run)
	}

	if strings.ToLower(opts.period) == "last-run" && len(filtered) > 1 {
		return filtered[:1]
	}

	if opts.runs > 0 && len(filtered) > opts.runs {
		return filtered[:opts.runs]
	}

	return filtered
}

func renderReportJSON(runs []reportRun, rng reportRange) error {
	type payload struct {
		Range string                  `json:"range"`
		Runs  []*reporting.RunResults `json:"runs"`
	}
	results := make([]*reporting.RunResults, 0, len(runs))
	for _, run := range runs {
		results = append(results, run.results)
	}
	out := payload{Range: rng.label, Runs: results}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func renderReportMarkdown(runs []reportRun) error {
	for i, run := range runs {
		if run.results == nil {
			continue
		}
		if i > 0 {
			fmt.Print("\n---\n\n")
		}
		content, err := reporting.RenderRunReport(run.results, run.results.LogPath)
		if err != nil {
			return err
		}
		fmt.Print(content)
	}
	return nil
}

func renderReportFancy(runs []reportRun, rng reportRange, opts reportOptions) error {
	styles := newReportStyles()
	var b strings.Builder

	b.WriteString(styles.Title.Render("Nightshift Report"))
	b.WriteString("\n")
	if rng.label != "" {
		b.WriteString(styles.Subtitle.Render(rng.label))
		b.WriteString("\n")
	}
	b.WriteString(styles.Muted.Render(fmt.Sprintf("Runs: %d", len(runs))))
	b.WriteString("\n\n")

	switch strings.ToLower(opts.reportType) {
	case "overview":
		b.WriteString(renderReportOverview(styles, runs, opts))
	case "tasks":
		b.WriteString(renderReportTasks(styles, runs))
	case "projects":
		b.WriteString(renderReportProjects(styles, runs))
	case "budget":
		b.WriteString(renderReportBudget(styles, runs))
	case "raw":
		for _, run := range runs {
			if run.reportPath == "" {
				continue
			}
			payload, err := os.ReadFile(run.reportPath)
			if err != nil {
				return err
			}
			b.WriteString(string(payload))
			b.WriteString("\n")
		}
	default:
		return fmt.Errorf("unknown report type %q", opts.reportType)
	}

	fmt.Print(b.String())
	return nil
}

type reportStyles struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Section  lipgloss.Style
	Label    lipgloss.Style
	Value    lipgloss.Style
	Muted    lipgloss.Style
	Accent   lipgloss.Style
	OK       lipgloss.Style
	Warn     lipgloss.Style
	Error    lipgloss.Style
	Card     lipgloss.Style
	Pill     lipgloss.Style
}

func newReportStyles() reportStyles {
	return reportStyles{
		Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")),
		Subtitle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Section:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		Label:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Value:    lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Accent:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		OK:       lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		Warn:     lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Error:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			BorderForeground(lipgloss.Color("238")),
		Pill: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1),
	}
}

type runSummary struct {
	Start           time.Time
	End             time.Time
	Duration        time.Duration
	Completed       int
	Failed          int
	Skipped         int
	TokensUsed      int
	BudgetStart     int
	BudgetRemaining int
	Projects        map[string]int
	Outputs         []string
	Failures        []string
	Skips           []string
	Tasks           []reporting.TaskResult
}

func summarizeRun(results *reporting.RunResults) runSummary {
	summary := runSummary{
		Start:           results.StartTime,
		End:             results.EndTime,
		TokensUsed:      results.UsedBudget,
		BudgetStart:     results.StartBudget,
		BudgetRemaining: results.RemainingBudget,
		Projects:        make(map[string]int),
		Tasks:           results.Tasks,
	}

	if !results.StartTime.IsZero() && !results.EndTime.IsZero() {
		summary.Duration = results.EndTime.Sub(results.StartTime)
	}

	for _, task := range results.Tasks {
		projectName := projectLabel(task.Project)
		if projectName != "" {
			summary.Projects[projectName]++
		}
		switch task.Status {
		case "completed":
			summary.Completed++
		case "failed":
			summary.Failed++
			summary.Failures = append(summary.Failures, formatTaskDetail(task))
		case "skipped":
			summary.Skipped++
			summary.Skips = append(summary.Skips, formatTaskDetail(task))
		}
		if task.OutputRef != "" {
			if task.OutputType != "" {
				summary.Outputs = append(summary.Outputs, fmt.Sprintf("%s %s", task.OutputType, task.OutputRef))
			} else {
				summary.Outputs = append(summary.Outputs, task.OutputRef)
			}
		}
	}

	if summary.TokensUsed == 0 {
		for _, task := range results.Tasks {
			summary.TokensUsed += task.TokensUsed
		}
	}

	return summary
}

func renderReportOverview(styles reportStyles, runs []reportRun, opts reportOptions) string {
	var b strings.Builder

	agg := aggregateRuns(runs)

	// Tasks line: "Tasks: N completed · X% success" with failed/skipped only when > 0
	total := agg.completed + agg.failed + agg.skipped
	tasksLine := fmt.Sprintf("%s %d completed", styles.Label.Render("Tasks:"), agg.completed)
	if agg.failed > 0 {
		tasksLine += fmt.Sprintf(" · %s", styles.Error.Render(fmt.Sprintf("%d failed", agg.failed)))
	}
	if agg.skipped > 0 {
		tasksLine += fmt.Sprintf(" · %s", styles.Warn.Render(fmt.Sprintf("%d skipped", agg.skipped)))
	}
	if total > 0 {
		rate := float64(agg.completed) / float64(total) * 100
		tasksLine += fmt.Sprintf(" · %.0f%% success", rate)
	}

	summaryLines := []string{tasksLine}

	// Duration line with project count
	if agg.totalDuration > 0 {
		durationLine := fmt.Sprintf("%s %s", styles.Label.Render("Duration:"), formatDuration(agg.totalDuration))
		if agg.projectCount > 0 {
			projectWord := "project"
			if agg.projectCount > 1 {
				projectWord = "projects"
			}
			durationLine += fmt.Sprintf(" across %d %s", agg.projectCount, projectWord)
		}
		summaryLines = append(summaryLines, durationLine)
	}

	if agg.hasBudget {
		summaryLines = append(summaryLines, fmt.Sprintf("%s %s used / %s start",
			styles.Label.Render("Budget:"),
			formatTokensCompact(agg.tokensUsed),
			formatTokensCompact(agg.budgetStart),
		))
	}
	if agg.prCount > 0 {
		prLabel := "PR created"
		if agg.prCount > 1 {
			prLabel = "PRs created"
		}
		prStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
		summaryLines = append(summaryLines, prStyle.Render(fmt.Sprintf("\u2192 %d %s", agg.prCount, prLabel)))
	}
	if len(agg.outputs) > 0 {
		summaryLines = append(summaryLines, fmt.Sprintf("%s %s", styles.Label.Render("Outputs:"), strings.Join(agg.outputs, ", ")))
	}
	b.WriteString(styles.Section.Render("Summary"))
	b.WriteString("\n")
	b.WriteString(styles.Card.Render(strings.Join(summaryLines, "\n")))
	b.WriteString("\n\n")

	for i, run := range runs {
		if run.results == nil {
			continue
		}
		summary := summarizeRun(run.results)
		header := fmt.Sprintf("Run %d · %s", i+1, formatRunWindow(summary))
		b.WriteString(styles.Section.Render(header))
		b.WriteString("\n")

		runLines := []string{
			fmt.Sprintf("%s %d completed, %d failed, %d skipped",
				styles.Label.Render("Tasks:"), summary.Completed, summary.Failed, summary.Skipped),
		}
		if summary.BudgetStart > 0 {
			runLines = append(runLines, fmt.Sprintf("%s %s used / %s start (%s remaining)",
				styles.Label.Render("Budget:"),
				formatTokensCompact(summary.TokensUsed),
				formatTokensCompact(summary.BudgetStart),
				formatTokensCompact(summary.BudgetRemaining),
			))
		} else if summary.TokensUsed > 0 {
			runLines = append(runLines, fmt.Sprintf("%s %s", styles.Label.Render("Tokens:"), formatTokensCompact(summary.TokensUsed)))
		}

		if len(summary.Projects) > 0 {
			runLines = append(runLines, fmt.Sprintf("%s %s", styles.Label.Render("Projects:"), formatProjectSummary(summary.Projects)))
		}

		b.WriteString(styles.Card.Render(strings.Join(runLines, "\n")))
		b.WriteString("\n")

		// Build task list grouped by status: completed, failed, skipped
		var ordered []reporting.TaskResult
		for _, t := range summary.Tasks {
			if t.Status == "completed" {
				ordered = append(ordered, t)
			}
		}
		for _, t := range summary.Tasks {
			if t.Status == "failed" {
				ordered = append(ordered, t)
			}
		}
		for _, t := range summary.Tasks {
			if t.Status != "completed" && t.Status != "failed" {
				ordered = append(ordered, t)
			}
		}

		shown := len(ordered)
		if opts.maxItems > 0 && shown > opts.maxItems {
			shown = opts.maxItems
		}
		for idx, task := range ordered {
			if idx >= shown {
				break
			}
			var icon string
			switch task.Status {
			case "completed":
				icon = styles.OK.Render("\u2713")
			case "failed":
				icon = styles.Error.Render("\u2717")
			default:
				icon = styles.Warn.Render("~")
			}
			project := projectLabel(task.Project)
			line := fmt.Sprintf(" %s  %-30s", icon, task.Title)
			if project != "" {
				line += fmt.Sprintf("  %s", styles.Muted.Render(project))
			}
			line += fmt.Sprintf("  %s", styles.Muted.Render("("+task.TaskType+")"))
			if task.Duration > 0 {
				line += fmt.Sprintf("  %s", formatDuration(task.Duration))
			}
			if task.TokensUsed > 0 {
				line += fmt.Sprintf("  %s", styles.Muted.Render(formatTokensCompact(task.TokensUsed)+" tok"))
			}
			if task.OutputRef != "" {
				line += fmt.Sprintf("  %s", formatOutputRef(styles, task))
			}
			if task.SkipReason != "" && task.Status != "completed" {
				line += fmt.Sprintf("  %s", styles.Warn.Render(task.SkipReason))
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		if shown < len(ordered) {
			b.WriteString(styles.Muted.Render(fmt.Sprintf("  ...and %d more", len(ordered)-shown)))
			b.WriteString("\n")
		}

		if opts.showPaths {
			if run.reportPath != "" {
				b.WriteString(styles.Muted.Render(fmt.Sprintf("Report file: %s", run.reportPath)))
				b.WriteString("\n")
			}
			if run.results.LogPath != "" {
				b.WriteString(styles.Muted.Render(fmt.Sprintf("Log file: %s", run.results.LogPath)))
				b.WriteString("\n")
			}
		}

		if i < len(runs)-1 {
			b.WriteString("\n")
		}
	}

	// What's Next section
	b.WriteString("\n")
	b.WriteString(renderWhatsNext(styles, runs))

	return b.String()
}

// renderWhatsNext generates context-aware action items based on run results.
func renderWhatsNext(styles reportStyles, runs []reportRun) string {
	var b strings.Builder
	var items []string

	totalCompleted := 0
	totalTasks := 0
	var totalBudgetStart, totalBudgetRemaining int

	for _, run := range runs {
		if run.results == nil {
			continue
		}
		for _, task := range run.results.Tasks {
			totalTasks++
			if task.Status == "completed" {
				totalCompleted++
			}

			// PRs to review
			if strings.EqualFold(task.OutputType, "pr") && task.OutputRef != "" {
				ref := task.OutputRef
				if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
					ref = termenv.Hyperlink(ref, ref)
				}
				items = append(items, styles.Accent.Render(fmt.Sprintf("\u2192 Review PR: %s", ref)))
			}

			// Failed tasks
			if task.Status == "failed" {
				project := projectLabel(task.Project)
				detail := task.Title
				if project != "" {
					detail += " (" + project + ")"
				}
				items = append(items, styles.Error.Render(fmt.Sprintf("\u2192 Investigate failed: %s", detail)))
			}
		}

		totalBudgetStart += run.results.StartBudget
		totalBudgetRemaining += run.results.RemainingBudget
	}

	// Budget warning: remaining < 20% of start
	if totalBudgetStart > 0 {
		threshold := totalBudgetStart / 5
		if totalBudgetRemaining < threshold {
			items = append(items, styles.Warn.Render(fmt.Sprintf("\u2192 Budget low: %s remaining of %s start",
				formatTokensCompact(totalBudgetRemaining),
				formatTokensCompact(totalBudgetStart))))
		}
	}

	b.WriteString(styles.Section.Render("What's Next"))
	b.WriteString("\n")

	if len(items) == 0 {
		taskWord := "tasks"
		if totalCompleted == 1 {
			taskWord = "task"
		}
		b.WriteString(styles.OK.Render(fmt.Sprintf("  \u2713 All %d %s completed successfully", totalCompleted, taskWord)))
		b.WriteString("\n")
	} else {
		for _, item := range items {
			b.WriteString("  " + item + "\n")
		}
	}

	return b.String()
}

func renderReportTasks(styles reportStyles, runs []reportRun) string {
	var b strings.Builder
	for i, run := range runs {
		if run.results == nil {
			continue
		}
		summary := summarizeRun(run.results)
		header := fmt.Sprintf("Run %d · %s", i+1, formatRunWindow(summary))
		b.WriteString(styles.Section.Render(header))
		b.WriteString("\n")

		for _, task := range summary.Tasks {
			status := formatTaskStatus(styles, task.Status)
			project := projectLabel(task.Project)
			line := fmt.Sprintf("%s %s (%s)", status, task.Title, task.TaskType)
			if project != "" {
				line += fmt.Sprintf(" · %s", project)
			}
			if task.TokensUsed > 0 {
				line += fmt.Sprintf(" · %s tokens", formatTokensCompact(task.TokensUsed))
			}
			if task.Duration > 0 {
				line += fmt.Sprintf(" · %s", formatDuration(task.Duration))
			}
			if task.SkipReason != "" && task.Status != "completed" {
				line += fmt.Sprintf(" · %s", task.SkipReason)
			}
			b.WriteString("  " + line + "\n")
		}

		if i < len(runs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderReportProjects(styles reportStyles, runs []reportRun) string {
	projectTotals := make(map[string]struct {
		completed int
		failed    int
		skipped   int
	})

	for _, run := range runs {
		if run.results == nil {
			continue
		}
		for _, task := range run.results.Tasks {
			project := projectLabel(task.Project)
			if project == "" {
				project = "unknown"
			}
			entry := projectTotals[project]
			switch task.Status {
			case "completed":
				entry.completed++
			case "failed":
				entry.failed++
			case "skipped":
				entry.skipped++
			}
			projectTotals[project] = entry
		}
	}

	type projectRow struct {
		name      string
		completed int
		failed    int
		skipped   int
		total     int
	}
	rows := make([]projectRow, 0, len(projectTotals))
	for name, entry := range projectTotals {
		rows = append(rows, projectRow{
			name:      name,
			completed: entry.completed,
			failed:    entry.failed,
			skipped:   entry.skipped,
			total:     entry.completed + entry.failed + entry.skipped,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].total > rows[j].total
	})

	var b strings.Builder
	b.WriteString(styles.Section.Render("Projects"))
	b.WriteString("\n")
	for _, row := range rows {
		line := fmt.Sprintf("%s %d total (%d completed, %d failed, %d skipped)",
			styles.Accent.Render(row.name),
			row.total, row.completed, row.failed, row.skipped,
		)
		b.WriteString("  " + line + "\n")
	}
	return b.String()
}

func renderReportBudget(styles reportStyles, runs []reportRun) string {
	var b strings.Builder
	b.WriteString(styles.Section.Render("Budget"))
	b.WriteString("\n")
	for i, run := range runs {
		if run.results == nil {
			continue
		}
		summary := summarizeRun(run.results)
		header := fmt.Sprintf("Run %d · %s", i+1, formatRunWindow(summary))
		b.WriteString(styles.Accent.Render(header))
		b.WriteString("\n")

		if summary.BudgetStart > 0 {
			b.WriteString(fmt.Sprintf("  %s %s used / %s start (%s remaining)\n",
				styles.Label.Render("Budget:"),
				formatTokensCompact(summary.TokensUsed),
				formatTokensCompact(summary.BudgetStart),
				formatTokensCompact(summary.BudgetRemaining),
			))
		} else if summary.TokensUsed > 0 {
			b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Tokens:"), formatTokensCompact(summary.TokensUsed)))
		} else {
			b.WriteString("  No budget data recorded\n")
		}

		if i < len(runs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

type aggregateSummary struct {
	completed     int
	failed        int
	skipped       int
	tokensUsed    int
	budgetStart   int
	outputCounts  map[string]int
	hasBudget     bool
	outputs       []string
	prCount       int
	totalDuration time.Duration
	projectCount  int
}

func aggregateRuns(runs []reportRun) aggregateSummary {
	agg := aggregateSummary{
		outputCounts: make(map[string]int),
	}
	allProjects := make(map[string]struct{})
	for _, run := range runs {
		if run.results == nil {
			continue
		}
		summary := summarizeRun(run.results)
		agg.completed += summary.Completed
		agg.failed += summary.Failed
		agg.skipped += summary.Skipped
		agg.tokensUsed += summary.TokensUsed
		agg.totalDuration += summary.Duration
		if summary.BudgetStart > 0 {
			agg.budgetStart += summary.BudgetStart
			agg.hasBudget = true
		}
		for _, task := range run.results.Tasks {
			if strings.EqualFold(task.OutputType, "pr") && task.OutputRef != "" {
				agg.prCount++
			}
		}
		for name := range summary.Projects {
			allProjects[name] = struct{}{}
		}
		for _, output := range summary.Outputs {
			agg.outputCounts[output]++
		}
	}
	agg.projectCount = len(allProjects)

	if len(agg.outputCounts) > 0 {
		outputs := make([]string, 0, len(agg.outputCounts))
		for name, count := range agg.outputCounts {
			outputs = append(outputs, fmt.Sprintf("%s (%d)", name, count))
		}
		sort.Strings(outputs)
		agg.outputs = outputs
	}

	return agg
}

func formatRunWindow(summary runSummary) string {
	if summary.Start.IsZero() && summary.End.IsZero() {
		return "time unknown"
	}
	start := summary.Start
	end := summary.End
	if start.IsZero() {
		start = end
	}
	if end.IsZero() {
		end = start
	}
	if summary.Duration > 0 {
		return fmt.Sprintf("%s → %s (%s)", start.Format("2006-01-02 15:04"), end.Format("15:04"), formatDuration(summary.Duration))
	}
	if start.Equal(end) {
		return start.Format("2006-01-02 15:04")
	}
	return fmt.Sprintf("%s → %s", start.Format("2006-01-02 15:04"), end.Format("15:04"))
}

func formatTaskStatus(styles reportStyles, status string) string {
	switch status {
	case "completed":
		return styles.OK.Render("OK")
	case "failed":
		return styles.Error.Render("FAIL")
	case "skipped":
		return styles.Warn.Render("SKIP")
	default:
		return styles.Muted.Render(strings.ToUpper(status))
	}
}

func projectLabel(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func formatProjectSummary(projects map[string]int) string {
	type pair struct {
		name  string
		count int
	}
	list := make([]pair, 0, len(projects))
	for name, count := range projects {
		list = append(list, pair{name: name, count: count})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].count > list[j].count
	})
	parts := make([]string, 0, len(list))
	for _, item := range list {
		parts = append(parts, fmt.Sprintf("%s (%d)", item.name, item.count))
	}
	return strings.Join(parts, ", ")
}

// formatOutputRef formats a task's OutputRef for display in the report.
// PR-type outputs get bold accent styling and a "-> PR:" prefix.
// Other typed outputs show "-> {Type}: {Ref}".
// Untyped outputs show "-> {Ref}".
func formatOutputRef(styles reportStyles, task reporting.TaskResult) string {
	if task.OutputRef == "" {
		return ""
	}

	ref := task.OutputRef
	outputType := strings.TrimSpace(task.OutputType)
	isPR := strings.EqualFold(outputType, "pr")

	// For PR links, use bold accent and wrap as terminal hyperlink if it's a URL
	if isPR {
		display := ref
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			display = termenv.Hyperlink(ref, ref)
		}
		prStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
		return prStyle.Render("\u2192 PR: " + display)
	}

	// Typed non-PR output
	if outputType != "" {
		return styles.Accent.Render(fmt.Sprintf("\u2192 %s: %s", outputType, ref))
	}

	// Untyped output
	return styles.Accent.Render("\u2192 " + ref)
}

func formatTaskDetail(task reporting.TaskResult) string {
	if task.SkipReason != "" {
		return fmt.Sprintf("%s (%s)", task.Title, task.SkipReason)
	}
	return task.Title
}

func formatTokensCompact(tokens int) string {
	switch {
	case tokens >= 1_000_000:
		return fmt.Sprintf("%.1fm", float64(tokens)/1_000_000)
	case tokens >= 10_000:
		return fmt.Sprintf("%.0fk", float64(tokens)/1_000)
	case tokens >= 1_000:
		return fmt.Sprintf("%.1fk", float64(tokens)/1_000)
	default:
		return fmt.Sprintf("%d", tokens)
	}
}

func formatTimeShort(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Format("2006-01-02 15:04")
}

func parseRunReportMarkdown(content string) (*reporting.RunResults, error) {
	results := &reporting.RunResults{
		Tasks: []reporting.TaskResult{},
	}
	lines := strings.Split(content, "\n")
	section := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# Nightshift Run - ") {
			raw := strings.TrimPrefix(line, "# Nightshift Run - ")
			if ts, err := time.ParseInLocation("2006-01-02 15:04", raw, time.Local); err == nil {
				results.StartTime = ts
				results.Date = ts
			}
			continue
		}
		if strings.HasPrefix(line, "- Duration: ") {
			duration := strings.TrimPrefix(line, "- Duration: ")
			if parsed, err := parseDurationShort(duration); err == nil {
				if !results.StartTime.IsZero() {
					results.EndTime = results.StartTime.Add(parsed)
				}
			}
			continue
		}
		if strings.HasPrefix(line, "- Budget: ") {
			budget := strings.TrimPrefix(line, "- Budget: ")
			parseBudgetLine(results, budget)
			continue
		}
		if strings.HasPrefix(line, "- Logs: ") {
			results.LogPath = strings.TrimPrefix(line, "- Logs: ")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			switch strings.TrimPrefix(line, "## ") {
			case "Tasks Completed":
				section = "completed"
			case "Tasks Failed":
				section = "failed"
			case "Tasks Skipped":
				section = "skipped"
			default:
				section = ""
			}
			continue
		}
		if strings.HasPrefix(line, "- ") && section != "" {
			task := parseTaskLine(strings.TrimPrefix(line, "- "), section)
			results.Tasks = append(results.Tasks, task)
		}
	}
	return results, nil
}

func parseBudgetLine(results *reporting.RunResults, budget string) {
	parts := strings.Split(budget, ",")
	if len(parts) < 3 {
		return
	}
	start := parseTokenString(strings.TrimSpace(strings.TrimSuffix(parts[0], " start")))
	used := parseTokenString(strings.TrimSpace(strings.TrimSuffix(parts[1], " used")))
	remaining := parseTokenString(strings.TrimSpace(strings.TrimSuffix(parts[2], " remaining")))
	results.StartBudget = start
	results.UsedBudget = used
	results.RemainingBudget = remaining
}

func parseTaskLine(line string, status string) reporting.TaskResult {
	task := reporting.TaskResult{Status: status}
	parts := strings.Split(line, " — ")
	base := parts[0]

	if idx := strings.LastIndex(base, " ("); idx != -1 && strings.HasSuffix(base, ")") {
		task.TaskType = strings.TrimSuffix(base[idx+2:], ")")
		base = base[:idx]
	}

	if split := strings.SplitN(base, ": ", 2); len(split) == 2 {
		task.Project = split[0]
		task.Title = split[1]
	} else {
		task.Title = base
	}

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasSuffix(part, " tokens"):
			task.TokensUsed = parseTokenString(strings.TrimSuffix(part, " tokens"))
		case strings.HasPrefix(part, "output: "):
			task.OutputRef = strings.TrimPrefix(part, "output: ")
		case strings.HasPrefix(part, "Skip reason: "):
			task.SkipReason = strings.TrimPrefix(part, "Skip reason: ")
		default:
			if d, err := parseDurationShort(part); err == nil {
				task.Duration = d
			}
		}
	}
	return task
}

func parseTokenString(value string) int {
	clean := strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	var n int
	_, _ = fmt.Sscanf(clean, "%d", &n)
	return n
}

func parseDurationShort(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty duration")
	}
	parts := strings.Fields(value)
	var total time.Duration
	for _, part := range parts {
		switch {
		case strings.HasSuffix(part, "h"):
			var n int
			if _, err := fmt.Sscanf(part, "%dh", &n); err != nil {
				return 0, err
			}
			total += time.Duration(n) * time.Hour
		case strings.HasSuffix(part, "m"):
			var n int
			if _, err := fmt.Sscanf(part, "%dm", &n); err != nil {
				return 0, err
			}
			total += time.Duration(n) * time.Minute
		case strings.HasSuffix(part, "s"):
			var n int
			if _, err := fmt.Sscanf(part, "%ds", &n); err != nil {
				return 0, err
			}
			total += time.Duration(n) * time.Second
		default:
			return 0, fmt.Errorf("invalid duration %q", value)
		}
	}
	return total, nil
}
