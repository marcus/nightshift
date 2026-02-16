package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	previewCmd.Flags().Bool("plain", false, "Disable gum pager output")
	previewCmd.Flags().Bool("json", false, "Output JSON (includes full prompts)")
	rootCmd.AddCommand(previewCmd)
}

func runPreview(cmd *cobra.Command, args []string) error {
	runs, _ := cmd.Flags().GetInt("runs")
	projectPath, _ := cmd.Flags().GetString("project")
	taskFilter, _ := cmd.Flags().GetString("task")
	longPrompt, _ := cmd.Flags().GetBool("long")
	writeDir, _ := cmd.Flags().GetString("write")
	explain, _ := cmd.Flags().GetBool("explain")
	plainOutput, _ := cmd.Flags().GetBool("plain")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	sources, err := detectPreviewConfigSources(projectPath)
	if err != nil {
		return err
	}

	if runs <= 0 {
		return fmt.Errorf("runs must be positive")
	}

	cfg, err := loadConfig(projectPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Register custom tasks from config
	tasks.ClearCustom()
	if err := tasks.RegisterCustomTasksFromConfig(cfg.Tasks.Custom); err != nil {
		return fmt.Errorf("register custom tasks: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() { _ = database.Close() }()

	projects, err := resolveProjects(cfg, projectPath)
	if err != nil {
		return fmt.Errorf("resolve projects: %w", err)
	}

	result, err := buildPreviewResult(cfg, database, projects, taskFilter, runs, writeDir, sources, explain || jsonOutput)
	if err != nil {
		return err
	}

	if jsonOutput {
		return writePreviewJSON(cmd.OutOrStdout(), result)
	}

	text := renderPreviewText(result, previewTextOptions{
		LongPrompt: longPrompt,
		Explain:    explain,
	})
	return writePreviewText(cmd.OutOrStdout(), text, previewPagerOptions{
		Plain: plainOutput,
	})
}

type previewConfigSources struct {
	GlobalPath    string
	GlobalExists  bool
	ProjectPath   string
	ProjectExists bool
}

func detectPreviewConfigSources(projectPath string) (*previewConfigSources, error) {
	globalPath := config.GlobalConfigPath()
	globalExists := false
	if _, err := os.Stat(globalPath); err == nil {
		globalExists = true
	}

	base := projectPath
	if base == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get cwd: %w", err)
		}
		base = cwd
	}
	projectConfigPath := filepath.Join(base, config.ProjectConfigName)
	projectExists := false
	if _, err := os.Stat(projectConfigPath); err == nil {
		projectExists = true
	}

	return &previewConfigSources{
		GlobalPath:    globalPath,
		GlobalExists:  globalExists,
		ProjectPath:   projectConfigPath,
		ProjectExists: projectExists,
	}, nil
}

type previewResult struct {
	GeneratedAt    time.Time
	Provider       string
	TaskFilter     string
	BudgetMode     string
	MaxPercent     int
	ReservePercent int
	EnabledTasks   []string
	ProjectCount   int
	Runs           []previewRun
	Providers      []providerBudgetSummary
	ConfigSources  *previewConfigSources
	Note           string
}

type previewRun struct {
	Index    int
	RunAt    time.Time
	Projects []previewProject
}

type previewProjectStatus string

const (
	previewProjectReady           previewProjectStatus = "ready"
	previewProjectSkipped         previewProjectStatus = "skipped"
	previewProjectBudgetExhausted previewProjectStatus = "budget_exhausted"
	previewProjectNoTasks         previewProjectStatus = "no_tasks"
	previewProjectError           previewProjectStatus = "error"
)

type previewProject struct {
	Path        string
	Status      previewProjectStatus
	Detail      string
	Budget      *budget.AllowanceResult
	Tasks       []previewTask
	Diagnostics *previewDiagnostics
}

type previewTask struct {
	Index           int
	Name            string
	Type            string
	Description     string
	Score           float64
	CostTier        string
	MinTokens       int
	MaxTokens       int
	Prompt          string
	PromptFile      string
	PromptFileError string
}

type previewDiagnostics struct {
	FilteredTask *previewFilteredTaskDiagnostic `json:"filtered_task,omitempty"`
	Aggregate    *previewAggregateDiagnostic    `json:"aggregate,omitempty"`
	Cooldowns    []previewCooldownEntry         `json:"cooldowns,omitempty"`
}

type previewFilteredTaskDiagnostic struct {
	Type         string `json:"type"`
	Name         string `json:"name,omitempty"`
	CostTier     string `json:"cost_tier,omitempty"`
	MinTokens    int    `json:"min_tokens,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	Budget       int64  `json:"budget,omitempty"`
	BudgetTooLow bool   `json:"budget_too_low,omitempty"`
	Disabled     bool   `json:"disabled,omitempty"`
	Error        string `json:"error,omitempty"`
}

type previewAggregateDiagnostic struct {
	Enabled        int      `json:"enabled"`
	Disabled       int      `json:"disabled"`
	OverBudget     int      `json:"over_budget"`
	Assigned       int      `json:"assigned"`
	OnCooldown     int      `json:"on_cooldown"`
	Candidates     int      `json:"candidates"`
	Budget         int64    `json:"budget"`
	UnknownEnabled []string `json:"unknown_enabled,omitempty"`
	NoEnabledTasks bool     `json:"no_enabled_tasks,omitempty"`
}

type previewCooldownEntry struct {
	TaskType      string `json:"task_type"`
	TaskName      string `json:"task_name"`
	Remaining     string `json:"remaining"`
	TotalInterval string `json:"total_interval"`
	Simulated     bool   `json:"simulated,omitempty"`
}

func buildPreviewResult(cfg *config.Config, database *db.DB, projects []string, taskFilter string, runs int, writeDir string, sources *previewConfigSources, includeDiagnostics bool) (*previewResult, error) {
	if runs <= 0 {
		return nil, fmt.Errorf("runs must be positive")
	}
	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects configured")
	}

	st, err := state.New(database)
	if err != nil {
		return nil, fmt.Errorf("init state: %w", err)
	}

	sched, err := scheduler.NewFromConfig(&cfg.Schedule)
	if err != nil {
		return nil, fmt.Errorf("schedule config: %w", err)
	}

	nextRuns, err := sched.NextRuns(runs)
	if err != nil {
		return nil, fmt.Errorf("compute next runs: %w", err)
	}

	claudeProvider := providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude"))
	codexProvider := providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex"))
	copilotProvider := providers.NewCopilotWithPath(cfg.ExpandedProviderPath("copilot"))
	cal := calibrator.New(database, cfg)
	trend := trends.NewAnalyzer(database, cfg.Budget.SnapshotRetentionDays)
	budgetMgr := budget.NewManagerFromProviders(cfg, claudeProvider, codexProvider, copilotProvider, budget.WithBudgetSource(cal), budget.WithTrendAnalyzer(trend))

	selector := tasks.NewSelector(cfg, st)
	orch := orchestrator.New()

	if writeDir != "" {
		if err := os.MkdirAll(writeDir, 0755); err != nil {
			return nil, fmt.Errorf("create write dir: %w", err)
		}
	}

	provider, err := previewProvider(cfg)
	if err != nil {
		return nil, err
	}

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

	result := &previewResult{
		GeneratedAt:    time.Now(),
		Provider:       provider,
		TaskFilter:     taskFilter,
		BudgetMode:     mode,
		MaxPercent:     maxPercent,
		ReservePercent: reservePercent,
		EnabledTasks:   append([]string(nil), cfg.Tasks.Enabled...),
		ProjectCount:   len(projects),
		Providers:      collectProviderBudgets(cfg, budgetMgr),
		ConfigSources:  sources,
		Note:           "Only the plan prompt is deterministic. Implement/review prompts are generated after plan output.",
	}

	for i, runAt := range nextRuns {
		run := previewRun{Index: i + 1, RunAt: runAt}
		for _, project := range projects {
			projectResult := previewProject{Path: project}

			if taskFilter == "" && st.WasProcessedToday(project) {
				projectResult.Status = previewProjectSkipped
				projectResult.Detail = "already processed today"
				run.Projects = append(run.Projects, projectResult)
				continue
			}

			allowance, err := budgetMgr.CalculateAllowance(provider)
			if err != nil {
				projectResult.Status = previewProjectError
				projectResult.Detail = fmt.Sprintf("budget error: %v", err)
				run.Projects = append(run.Projects, projectResult)
				continue
			}

			projectResult.Budget = allowance
			if allowance.Allowance <= 0 {
				projectResult.Status = previewProjectBudgetExhausted
				projectResult.Detail = "budget exhausted"
				if includeDiagnostics {
					projectResult.Diagnostics = computePreviewDiagnostics(cfg, selector, project, taskFilter, allowance.Allowance)
				}
				run.Projects = append(run.Projects, projectResult)
				continue
			}

			selected, err := previewSelectTasks(selector, project, taskFilter, allowance.Allowance)
			if err != nil {
				projectResult.Status = previewProjectError
				projectResult.Detail = err.Error()
				if includeDiagnostics {
					projectResult.Diagnostics = computePreviewDiagnostics(cfg, selector, project, taskFilter, allowance.Allowance)
				}
				run.Projects = append(run.Projects, projectResult)
				continue
			}
			if len(selected) == 0 {
				projectResult.Status = previewProjectNoTasks
				projectResult.Detail = "no tasks available within budget"
				if includeDiagnostics {
					projectResult.Diagnostics = computePreviewDiagnostics(cfg, selector, project, taskFilter, allowance.Allowance)
				}
				run.Projects = append(run.Projects, projectResult)
				continue
			}

			projectResult.Status = previewProjectReady
			if includeDiagnostics {
				projectResult.Diagnostics = computePreviewDiagnostics(cfg, selector, project, taskFilter, allowance.Allowance)
			}
			projectResult.Tasks = make([]previewTask, 0, len(selected))
			for idx, scored := range selected {
				taskInstance := &tasks.Task{
					ID:          fmt.Sprintf("%s:%s", scored.Definition.Type, project),
					Title:       scored.Definition.Name,
					Description: scored.Definition.Description,
					Priority:    int(scored.Score),
					Type:        scored.Definition.Type,
				}
				prompt := orch.PlanPrompt(taskInstance)
				minTokens, maxTokens := scored.Definition.EstimatedTokens()

				taskPreview := previewTask{
					Index:       idx + 1,
					Name:        scored.Definition.Name,
					Type:        string(scored.Definition.Type),
					Description: scored.Definition.Description,
					Score:       scored.Score,
					CostTier:    scored.Definition.CostTier.String(),
					MinTokens:   minTokens,
					MaxTokens:   maxTokens,
					Prompt:      prompt,
				}

				if writeDir != "" {
					filename := fmt.Sprintf("run-%02d-%s-%s-plan.txt", i+1, sanitizeFileName(filepath.Base(project)), scored.Definition.Type)
					fullPath := filepath.Join(writeDir, filename)
					if err := os.WriteFile(fullPath, []byte(prompt), 0644); err != nil {
						taskPreview.PromptFileError = err.Error()
					} else {
						taskPreview.PromptFile = fullPath
					}
				}

				projectResult.Tasks = append(projectResult.Tasks, taskPreview)
			}

			// Simulate cooldown: mark selected tasks so subsequent runs pick different tasks
			for _, scored := range selected {
				selector.AddSimulatedCooldown(string(scored.Definition.Type), project)
			}

			run.Projects = append(run.Projects, projectResult)
		}

		result.Runs = append(result.Runs, run)
	}

	return result, nil
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
	if cfg.Providers.Copilot.Enabled {
		allowance, err := budgetMgr.CalculateAllowance("copilot")
		summaries = append(summaries, providerBudgetSummary{
			name:      "copilot",
			allowance: allowance,
			err:       err,
		})
	}
	return summaries
}

func computePreviewDiagnostics(cfg *config.Config, selector *tasks.Selector, project, taskFilter string, allowance int64) *previewDiagnostics {
	diagnostics := &previewDiagnostics{}
	if taskFilter != "" {
		def, err := tasks.GetDefinition(tasks.TaskType(taskFilter))
		if err != nil {
			diagnostics.FilteredTask = &previewFilteredTaskDiagnostic{
				Type:  taskFilter,
				Error: "unknown task type",
			}
			return diagnostics
		}
		minTok, maxTok := def.EstimatedTokens()
		diagnostics.FilteredTask = &previewFilteredTaskDiagnostic{
			Type:         string(def.Type),
			Name:         def.Name,
			CostTier:     def.CostTier.String(),
			MinTokens:    minTok,
			MaxTokens:    maxTok,
			Budget:       allowance,
			BudgetTooLow: int64(maxTok) > allowance,
			Disabled:     !cfg.IsTaskEnabled(string(def.Type)),
		}
		// Check cooldown for the filtered task
		onCooldown, remaining, interval := selector.IsOnCooldown(tasks.TaskType(taskFilter), project)
		simulated := selector.HasSimulatedCooldown(taskFilter, project)
		if onCooldown || simulated {
			entry := previewCooldownEntry{
				TaskType:      taskFilter,
				TaskName:      def.Name,
				TotalInterval: formatCooldownDuration(interval),
				Simulated:     simulated,
			}
			if onCooldown {
				entry.Remaining = formatCooldownDuration(remaining)
			} else {
				entry.Remaining = "simulated"
			}
			diagnostics.Cooldowns = append(diagnostics.Cooldowns, entry)
		}
		return diagnostics
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
	cooldownCount := 0
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
		if selector.IsAssigned(taskID) {
			assignedCount++
			continue
		}
		// Check real cooldown and simulated cooldown
		onCooldown, remaining, interval := selector.IsOnCooldown(def.Type, project)
		simulated := selector.HasSimulatedCooldown(string(def.Type), project)
		if onCooldown || simulated {
			cooldownCount++
			entry := previewCooldownEntry{
				TaskType:      string(def.Type),
				TaskName:      def.Name,
				TotalInterval: formatCooldownDuration(interval),
				Simulated:     simulated,
			}
			if onCooldown {
				entry.Remaining = formatCooldownDuration(remaining)
			} else {
				entry.Remaining = "simulated"
			}
			diagnostics.Cooldowns = append(diagnostics.Cooldowns, entry)
			continue
		}
		candidateCount++
	}

	var unknown []string
	if len(cfg.Tasks.Enabled) > 0 {
		for _, taskName := range cfg.Tasks.Enabled {
			if !known[taskName] {
				unknown = append(unknown, taskName)
			}
		}
	}

	diagnostics.Aggregate = &previewAggregateDiagnostic{
		Enabled:        enabledCount,
		Disabled:       disabledCount,
		OverBudget:     overBudgetCount,
		Assigned:       assignedCount,
		OnCooldown:     cooldownCount,
		Candidates:     candidateCount,
		Budget:         allowance,
		UnknownEnabled: unknown,
		NoEnabledTasks: enabledCount == 0,
	}

	return diagnostics
}

func previewProvider(cfg *config.Config) (string, error) {
	if cfg.Providers.Claude.Enabled {
		return "claude", nil
	}
	if cfg.Providers.Codex.Enabled {
		return "codex", nil
	}
	if cfg.Providers.Copilot.Enabled {
		return "copilot", nil
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
	task := selector.SelectNext(allowance, projectPath)
	if task == nil {
		return nil, nil
	}
	return []tasks.ScoredTask{*task}, nil
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

func formatCooldownDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	hours := int(d.Hours())
	if hours >= 24 {
		days := hours / 24
		remaining := hours % 24
		if remaining > 0 {
			return fmt.Sprintf("%dd%dh", days, remaining)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		mins := int(d.Minutes()) % 60
		if mins > 0 {
			return fmt.Sprintf("%dh%dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}
	mins := int(d.Minutes())
	if mins > 0 {
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
