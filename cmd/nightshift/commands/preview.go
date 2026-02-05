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
	rootCmd.AddCommand(previewCmd)
}

func runPreview(cmd *cobra.Command, args []string) error {
	runs, _ := cmd.Flags().GetInt("runs")
	projectPath, _ := cmd.Flags().GetString("project")
	taskFilter, _ := cmd.Flags().GetString("task")
	longPrompt, _ := cmd.Flags().GetBool("long")
	writeDir, _ := cmd.Flags().GetString("write")

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

	return renderPreview(cmd.OutOrStdout(), cfg, database, projects, taskFilter, runs, longPrompt, writeDir)
}

func renderPreview(w io.Writer, cfg *config.Config, database *db.DB, projects []string, taskFilter string, runs int, longPrompt bool, writeDir string) error {
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
				fmt.Fprintf(w, "  %s: no tasks available within budget\n", project)
				continue
			}

			fmt.Fprintf(w, "  %s:\n", project)
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
				fmt.Fprintf(w, "       Prompt (plan): %s\n", renderPromptPreview(prompt, longPrompt))

				if writeDir != "" {
					filename := fmt.Sprintf("run-%02d-%s-%s-plan.txt", i+1, sanitizeFileName(filepath.Base(project)), scored.Definition.Type)
					fullPath := filepath.Join(writeDir, filename)
					if err := os.WriteFile(fullPath, []byte(prompt), 0644); err != nil {
						fmt.Fprintf(w, "       Prompt file: error writing (%v)\n", err)
					} else {
						fmt.Fprintf(w, "       Prompt file: %s\n", fullPath)
					}
				}
			}
		}

		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Note: Only the plan prompt is deterministic. Implement/review prompts are generated after plan output.")
	return nil
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
