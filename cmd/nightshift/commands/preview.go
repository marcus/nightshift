package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/calibrator"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/orchestrator"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/marcus/nightshift/internal/scheduler"
	"github.com/marcus/nightshift/internal/state"
	"github.com/marcus/nightshift/internal/tasks"
	"github.com/marcus/nightshift/internal/trends"
)

const defaultPromptPreviewChars = 400

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview the next scheduled runs",
	Long: `Preview the next scheduled runs using current state and budget.

This does not execute tasks or modify state.`,
	RunE: runPreview,
}

func init() {
	previewCmd.Flags().IntP("runs", "n", 3, "Number of upcoming runs to preview")
	previewCmd.Flags().StringP("project", "p", "", "Preview only a specific project path")
	previewCmd.Flags().StringP("task", "t", "", "Preview only a specific task type")
	previewCmd.Flags().Bool("long", false, "Show full prompts (default shows a truncated preview)")
	previewCmd.Flags().String("write", "", "Write full prompts to a directory")
	previewCmd.Flags().Bool("explain", false, "Show budget and task-filter explanations")
	rootCmd.AddCommand(previewCmd)
}

func runPreview(cmd *cobra.Command, args []string) error {
	runs, _ := cmd.Flags().GetInt("runs")
	projectPath, _ := cmd.Flags().GetString("project")
	taskFilter, _ := cmd.Flags().GetString("task")
	longPrompt, _ := cmd.Flags().GetBool("long")
	writeDir, _ := cmd.Flags().GetString("write")
	explain, _ := cmd.Flags().GetBool("explain")

	if runs <= 0 {
		return fmt.Errorf("runs must be positive")
	}

	cfg, err := loadConfig(projectPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer database.Close()

	projects, err := resolveProjects(cfg, projectPath)
	if err != nil {
		return fmt.Errorf("resolve projects: %w", err)
	}

	return renderPreview(cmd.OutOrStdout(), cfg, database, projects, taskFilter, runs, longPrompt, writeDir, explain)
}

func renderPreview(w io.Writer, cfg *config.Config, database *db.DB, projects []string, taskFilter string, runs int, longPrompt bool, writeDir string, explain bool) error {
	if runs <= 0 {
		return fmt.Errorf("runs must be positive")
	}
	if len(projects) == 0 {
		return fmt.Errorf("no projects configured")
	}

	st, err := state.New(database)
	if err != nil {
		return fmt.Errorf("init state: %w", err)
	}

	sched, err := scheduler.NewFromConfig(&cfg.Schedule)
	if err != nil {
		return fmt.Errorf("schedule config: %w", err)
	}

	nextRuns, err := sched.NextRuns(runs)
	if err != nil {
		return fmt.Errorf("compute next runs: %w", err)
	}

	claudeProvider := providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude"))
	codexProvider := providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex"))
	cal := calibrator.New(database, cfg)
	trend := trends.NewAnalyzer(database, cfg.Budget.SnapshotRetentionDays)
	budgetMgr := budget.NewManagerFromProviders(cfg, claudeProvider, codexProvider, budget.WithBudgetSource(cal), budget.WithTrendAnalyzer(trend))

	selector := tasks.NewSelector(cfg, st)
	orch := orchestrator.New()

	if writeDir != "" {
		if err := os.MkdirAll(writeDir, 0755); err != nil {
			return fmt.Errorf("create write dir: %w", err)
		}
	}

	provider, err := previewProvider(cfg)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Previewing next %d run(s). Assumes current state and usage; no tasks are executed.\n\n", runs)

	if explain {
		providers := collectProviderBudgets(cfg, budgetMgr)
		printPreviewContext(w, cfg, provider, taskFilter, providers)
		fmt.Fprintln(w)
	}

	for i, runAt := range nextRuns {
		fmt.Fprintf(w, "Run %d: %s\n", i+1, runAt.Format("2006-01-02 15:04"))

		for _, project := range projects {
			if taskFilter == "" && st.WasProcessedToday(project) {
				fmt.Fprintf(w, "  %s: skipped (already processed today)\n", project)
				continue
			}

			allowance, err := budgetMgr.CalculateAllowance(provider)
			if err != nil {
				fmt.Fprintf(w, "  %s: budget error: %v\n", project, err)
				continue
			}
			if allowance.Allowance <= 0 {
				fmt.Fprintf(w, "  %s: budget exhausted\n", project)
				continue
			}

			selected, err := previewSelectTasks(selector, project, taskFilter, allowance.Allowance)
			if err != nil {
				fmt.Fprintf(w, "  %s: %v\n", project, err)
				continue
			}
			if len(selected) == 0 {
				if explain {
					fmt.Fprintf(w, "  %s: no tasks available within budget\n", project)
					printTaskFilterDiagnostics(w, cfg, st, project, taskFilter, allowance.Allowance)
				} else {
					fmt.Fprintf(w, "  %s: no tasks available within budget\n", project)
				}
				continue
			}

			fmt.Fprintf(w, "  %s:\n", project)
			if explain {
				printProjectBudget(w, cfg, allowance)
			}
			for idx, scored := range selected {
				taskInstance := &tasks.Task{
					ID:          fmt.Sprintf("%s:%s", scored.Definition.Type, project),
					Title:       scored.Definition.Name,
					Description: scored.Definition.Description,
					Priority:    int(scored.Score),
					Type:        scored.Definition.Type,
				}
				prompt := orch.PlanPrompt(taskInstance)

				fmt.Fprintf(w, "    %d. %s (%s)\n", idx+1, scored.Definition.Name, scored.Definition.Type)
				fmt.Fprintf(w, "       Prompt preview:\n")
				fmt.Fprintf(w, "       %s\n", renderPromptPreview(prompt, longPrompt))

				if writeDir != "" {
					filename := fmt.Sprintf("run-%02d-%s-%s-plan.txt", i+1, sanitizeFileName(filepath.Base(project)), scored.Definition.Type)
					fullPath := filepath.Join(writeDir, filename)
					if err := os.WriteFile(fullPath, []byte(prompt), 0644); err != nil {
						fmt.Fprintf(w, "       Prompt file: error writing (%v)\n", err)
					} else {
						fmt.Fprintf(w, "       Prompt file: %s\n", fullPath)
					}
				}
				fmt.Fprintln(w)
			}
		}

		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Note: Only the plan prompt is deterministic. Implement/review prompts are generated after plan output.")
	return nil
}

type providerBudgetSummary struct {
	name      string
	allowance *budget.AllowanceResult
	err       error
}

func collectProviderBudgets(cfg *config.Config, budgetMgr *budget.Manager) []providerBudgetSummary {
	var summaries []providerBudgetSummary
	if cfg.Providers.Claude.Enabled {
		allowance, err := budgetMgr.CalculateAllowance("claude")
		summaries = append(summaries, providerBudgetSummary{
			name:      "claude",
			allowance: allowance,
			err:       err,
		})
	}
	if cfg.Providers.Codex.Enabled {
		allowance, err := budgetMgr.CalculateAllowance("codex")
		summaries = append(summaries, providerBudgetSummary{
			name:      "codex",
			allowance: allowance,
			err:       err,
		})
	}
	return summaries
}

func printPreviewContext(w io.Writer, cfg *config.Config, provider, taskFilter string, providers []providerBudgetSummary) {
	mode := cfg.Budget.Mode
	if mode == "" {
		mode = config.DefaultBudgetMode
	}
	maxPercent := cfg.Budget.MaxPercent
	if maxPercent <= 0 {
		maxPercent = config.DefaultMaxPercent
	}
	reservePercent := cfg.Budget.ReservePercent
	if reservePercent < 0 {
		reservePercent = config.DefaultReservePercent
	}

	fmt.Fprintln(w, "Context:")
	fmt.Fprintf(w, "  Provider: %s (preview picks first enabled: claude -> codex)\n", provider)
	fmt.Fprintf(w, "  Budget mode: %s (max %d%%, reserve %d%%)\n", mode, maxPercent, reservePercent)
	if len(providers) > 0 {
		fmt.Fprintln(w, "  Provider budgets:")
		for _, summary := range providers {
			if summary.err != nil {
				fmt.Fprintf(w, "    - %s: budget error: %v\n", summary.name, summary.err)
				continue
			}
			fmt.Fprintf(w, "    - %s: %s available (%.1f%% used, weekly=%s, source=%s)\n",
				summary.name,
				formatTokens64(summary.allowance.Allowance),
				summary.allowance.UsedPercent,
				formatTokens64(summary.allowance.WeeklyBudget),
				summary.allowance.BudgetSource)
		}
	}
	if taskFilter != "" {
		fmt.Fprintf(w, "  Task filter: %s\n", taskFilter)
	} else {
		enabled := cfg.Tasks.Enabled
		if len(enabled) == 0 {
			fmt.Fprintln(w, "  Task filter: all enabled tasks (none explicitly enabled)")
		} else {
			fmt.Fprintf(w, "  Task filter: enabled list (%d) [%s]\n", len(enabled), strings.Join(enabled, ", "))
		}
	}
}

func printProjectBudget(w io.Writer, cfg *config.Config, allowance *budget.AllowanceResult) {
	if allowance == nil {
		return
	}
	available := allowance.Allowance
	fmt.Fprintf(w, "    Budget: %s available (%.1f%% used)\n", formatTokens64(available), allowance.UsedPercent)
	fmt.Fprintf(w, "    Budget calc: weekly=%s, base=%s, reserve=%s, predicted=%s\n",
		formatTokens64(allowance.WeeklyBudget),
		formatTokens64(allowance.BudgetBase),
		formatTokens64(allowance.ReserveAmount),
		formatTokens64(allowance.PredictedUsage))
	if allowance.Mode == "weekly" {
		fmt.Fprintf(w, "    Budget window: %d day(s) remaining, multiplier %.2f\n", allowance.RemainingDays, allowance.Multiplier)
	}
	fmt.Fprintf(w, "    Budget source: %s (confidence=%s, samples=%d)\n",
		allowance.BudgetSource, allowance.BudgetConfidence, allowance.BudgetSampleCount)
	if len(cfg.Projects) > 1 {
		fmt.Fprintf(w, "    Note: budget is not split per project during preview/run\n")
	}
}

func printTaskFilterDiagnostics(w io.Writer, cfg *config.Config, st *state.State, project, taskFilter string, allowance int64) {
	fmt.Fprintf(w, "    Diagnostics:\n")
	if taskFilter != "" {
		def, err := tasks.GetDefinition(tasks.TaskType(taskFilter))
		if err != nil {
			fmt.Fprintf(w, "      - Task filter unknown: %s\n", taskFilter)
			return
		}
		minTok, maxTok := def.EstimatedTokens()
		fmt.Fprintf(w, "      - Filtered to %s (%s), cost %s (%d-%d)\n",
			def.Type, def.Name, def.CostTier, minTok, maxTok)
		if int64(maxTok) > allowance {
			fmt.Fprintf(w, "      - Budget too low for %s: need %d, have %d\n", def.Type, maxTok, allowance)
		}
		if !cfg.IsTaskEnabled(string(def.Type)) {
			fmt.Fprintf(w, "      - Task disabled by config\n")
		}
		return
	}

	defs := tasks.AllDefinitions()
	known := make(map[string]bool, len(defs))
	for _, def := range defs {
		known[string(def.Type)] = true
	}
	enabledCount := 0
	disabledCount := 0
	overBudgetCount := 0
	assignedCount := 0
	candidateCount := 0
	for _, def := range defs {
		if !cfg.IsTaskEnabled(string(def.Type)) {
			disabledCount++
			continue
		}
		enabledCount++
		_, maxTok := def.EstimatedTokens()
		if int64(maxTok) > allowance {
			overBudgetCount++
			continue
		}
		taskID := fmt.Sprintf("%s:%s", def.Type, project)
		if st != nil && st.IsAssigned(taskID) {
			assignedCount++
			continue
		}
		candidateCount++
	}

	fmt.Fprintf(w, "      - Enabled tasks: %d (disabled: %d)\n", enabledCount, disabledCount)
	fmt.Fprintf(w, "      - Over budget: %d (budget=%d)\n", overBudgetCount, allowance)
	if assignedCount > 0 {
		fmt.Fprintf(w, "      - Already assigned: %d\n", assignedCount)
	}
	fmt.Fprintf(w, "      - Candidates after filters: %d\n", candidateCount)
	if len(cfg.Tasks.Enabled) > 0 {
		var unknown []string
		for _, taskName := range cfg.Tasks.Enabled {
			if !known[taskName] {
				unknown = append(unknown, taskName)
			}
		}
		if len(unknown) > 0 {
			fmt.Fprintf(w, "      - Unknown enabled task types: %s\n", strings.Join(unknown, ", "))
		}
	}
	if enabledCount == 0 {
		fmt.Fprintf(w, "      - No enabled tasks in config\n")
	}
}

func previewProvider(cfg *config.Config) (string, error) {
	if cfg.Providers.Claude.Enabled {
		return "claude", nil
	}
	if cfg.Providers.Codex.Enabled {
		return "codex", nil
	}
	return "", fmt.Errorf("no providers enabled for preview")
}

func previewSelectTasks(selector *tasks.Selector, projectPath, taskFilter string, allowance int64) ([]tasks.ScoredTask, error) {
	if taskFilter != "" {
		def, err := tasks.GetDefinition(tasks.TaskType(taskFilter))
		if err != nil {
			return nil, fmt.Errorf("unknown task type: %s", taskFilter)
		}
		return []tasks.ScoredTask{{
			Definition: def,
			Score:      selector.ScoreTask(def.Type, projectPath),
			Project:    projectPath,
		}}, nil
	}
	return selector.SelectTopN(allowance, projectPath, 5), nil
}

func renderPromptPreview(prompt string, full bool) string {
	prompt = strings.TrimSpace(prompt)
	if full || len(prompt) <= defaultPromptPreviewChars {
		return prompt
	}
	return fmt.Sprintf("%s… (truncated, use --long or --write)", prompt[:defaultPromptPreviewChars])
}

func sanitizeFileName(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('-')
	}
	return strings.Trim(b.String(), "-")
}
