package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/nightshift/internal/budget"
)

type previewTextOptions struct {
	LongPrompt bool
	Explain    bool
}

type previewPagerOptions struct {
	Plain bool
}

type previewStyles struct {
	Title   lipgloss.Style
	Section lipgloss.Style
	Label   lipgloss.Style
	Value   lipgloss.Style
	Muted   lipgloss.Style
	Warn    lipgloss.Style
	Error   lipgloss.Style
	Accent  lipgloss.Style
}

func newPreviewStyles() previewStyles {
	return previewStyles{
		Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")),
		Section: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		Label:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Value:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Muted:   lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Warn:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Error:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")),
		Accent:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
	}
}

func renderPreviewText(result *previewResult, opts previewTextOptions) string {
	styles := newPreviewStyles()
	var b strings.Builder

	b.WriteString(styles.Title.Render("Nightshift Preview"))
	b.WriteString("\n")
	b.WriteString(styles.Muted.Render(fmt.Sprintf("Previewing next %d run(s). Assumes current state and usage; no tasks are executed.", len(result.Runs))))
	b.WriteString("\n\n")

	b.WriteString(styles.Section.Render("Summary"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Provider: %s (preview picks first enabled: claude -> codex)\n", result.Provider))
	b.WriteString(fmt.Sprintf("  Budget mode: %s (max %d%%, reserve %d%%)\n", result.BudgetMode, result.MaxPercent, result.ReservePercent))
	if result.ConfigSources != nil {
		b.WriteString(fmt.Sprintf("  Config global: %s (%s)\n", result.ConfigSources.GlobalPath, configLoadedLabel(result.ConfigSources.GlobalExists)))
		b.WriteString(fmt.Sprintf("  Config project: %s (%s)\n", result.ConfigSources.ProjectPath, configLoadedLabel(result.ConfigSources.ProjectExists)))
		b.WriteString("  Config order: global -> project -> env overrides\n")
	}
	if len(result.Providers) > 0 {
		b.WriteString("  Provider budgets:\n")
		for _, summary := range result.Providers {
			if summary.err != nil {
				b.WriteString(fmt.Sprintf("    - %s: budget error: %v\n", summary.name, summary.err))
				continue
			}
			b.WriteString(fmt.Sprintf("    - %s: %s available (%.1f%% used, weekly=%s, source=%s)\n",
				summary.name,
				formatTokens64(summary.allowance.Allowance),
				summary.allowance.UsedPercent,
				formatTokens64(summary.allowance.WeeklyBudget),
				summary.allowance.BudgetSource))
		}
	}
	if result.TaskFilter != "" {
		b.WriteString(fmt.Sprintf("  Task filter: %s\n", result.TaskFilter))
	} else if len(result.EnabledTasks) == 0 {
		b.WriteString("  Task filter: all enabled tasks (none explicitly enabled)\n")
	} else {
		b.WriteString(fmt.Sprintf("  Task filter: enabled list (%d) [%s]\n", len(result.EnabledTasks), strings.Join(result.EnabledTasks, ", ")))
	}
	if opts.Explain && result.ProjectCount > 1 {
		b.WriteString("  Note: budget is not split per project during preview/run\n")
	}

	for _, run := range result.Runs {
		b.WriteString("\n")
		b.WriteString(styles.Section.Render(fmt.Sprintf("Run %d · %s", run.Index, run.RunAt.Format("2006-01-02 15:04"))))
		b.WriteString("\n")

		for _, project := range run.Projects {
			b.WriteString(styles.Label.Render("  " + project.Path))
			b.WriteString("\n")

			switch project.Status {
			case previewProjectSkipped:
				b.WriteString("    ")
				b.WriteString(styles.Muted.Render(fmt.Sprintf("skipped: %s", project.Detail)))
				b.WriteString("\n")
				continue
			case previewProjectBudgetExhausted:
				b.WriteString("    ")
				b.WriteString(styles.Warn.Render(project.Detail))
				b.WriteString("\n")
			case previewProjectNoTasks:
				b.WriteString("    ")
				b.WriteString(styles.Warn.Render(project.Detail))
				b.WriteString("\n")
			case previewProjectError:
				b.WriteString("    ")
				b.WriteString(styles.Error.Render(project.Detail))
				b.WriteString("\n")
			}

			if opts.Explain && project.Diagnostics != nil {
				renderDiagnosticsText(&b, styles, project.Diagnostics, "    ")
			}

			if project.Status != previewProjectReady {
				continue
			}

			if opts.Explain {
				renderBudgetText(&b, project.Budget, "    ")
			}

			for _, task := range project.Tasks {
				b.WriteString("    ")
				b.WriteString(styles.Accent.Render(fmt.Sprintf("%d. %s", task.Index, task.Name)))
				b.WriteString(fmt.Sprintf(" (%s)\n", task.Type))
				b.WriteString("       ")
				b.WriteString(styles.Muted.Render(fmt.Sprintf("score=%.1f, cost=%s (%d-%d)\n", task.Score, task.CostTier, task.MinTokens, task.MaxTokens)))
				b.WriteString("       Prompt:\n")
				preview := renderPromptPreview(task.Prompt, opts.LongPrompt)
				b.WriteString(indentLines(preview, "       "))
				b.WriteString("\n")
				if task.PromptFileError != "" {
					b.WriteString("       ")
					b.WriteString(styles.Warn.Render(fmt.Sprintf("Prompt file: error writing (%s)", task.PromptFileError)))
					b.WriteString("\n")
				} else if task.PromptFile != "" {
					b.WriteString(fmt.Sprintf("       Prompt file: %s\n", task.PromptFile))
				}
				b.WriteString("\n")
			}
		}
	}

	if result.Note != "" {
		b.WriteString(styles.Muted.Render(result.Note))
		b.WriteString("\n")
	}

	return b.String()
}

func renderBudgetText(b *strings.Builder, allowance *budget.AllowanceResult, indent string) {
	if allowance == nil {
		return
	}
	b.WriteString(indent)
	b.WriteString(fmt.Sprintf("Budget: %s available (%.1f%% used)\n", formatTokens64(allowance.Allowance), allowance.UsedPercent))
	b.WriteString(indent)
	b.WriteString(fmt.Sprintf("Budget calc: weekly=%s, base=%s, reserve=%s, predicted=%s\n",
		formatTokens64(allowance.WeeklyBudget),
		formatTokens64(allowance.BudgetBase),
		formatTokens64(allowance.ReserveAmount),
		formatTokens64(allowance.PredictedUsage)))
	if allowance.Mode == "weekly" {
		b.WriteString(indent)
		b.WriteString(fmt.Sprintf("Budget window: %d day(s) remaining, multiplier %.2f\n", allowance.RemainingDays, allowance.Multiplier))
	}
	b.WriteString(indent)
	b.WriteString(fmt.Sprintf("Budget source: %s (confidence=%s, samples=%d)\n",
		allowance.BudgetSource,
		allowance.BudgetConfidence,
		allowance.BudgetSampleCount))
}

func renderDiagnosticsText(b *strings.Builder, styles previewStyles, diagnostics *previewDiagnostics, indent string) {
	b.WriteString(indent)
	b.WriteString(styles.Muted.Render("Diagnostics:"))
	b.WriteString("\n")
	if diagnostics.FilteredTask != nil {
		if diagnostics.FilteredTask.Error != "" {
			b.WriteString(indent)
			b.WriteString("  - ")
			b.WriteString(styles.Warn.Render(fmt.Sprintf("Task filter unknown: %s", diagnostics.FilteredTask.Type)))
			b.WriteString("\n")
			return
		}
		b.WriteString(indent)
		b.WriteString(fmt.Sprintf("  - Filtered to %s (%s), cost %s (%d-%d)\n",
			diagnostics.FilteredTask.Type,
			diagnostics.FilteredTask.Name,
			diagnostics.FilteredTask.CostTier,
			diagnostics.FilteredTask.MinTokens,
			diagnostics.FilteredTask.MaxTokens))
		if diagnostics.FilteredTask.BudgetTooLow {
			b.WriteString(indent)
			b.WriteString(fmt.Sprintf("  - Budget too low: need %s, have %s\n",
				formatTokens64(int64(diagnostics.FilteredTask.MaxTokens)),
				formatTokens64(diagnostics.FilteredTask.Budget)))
		}
		if diagnostics.FilteredTask.Disabled {
			b.WriteString(indent)
			b.WriteString("  - Task disabled by config\n")
		}
		return
	}
	if diagnostics.Aggregate == nil {
		return
	}

	agg := diagnostics.Aggregate
	b.WriteString(indent)
	b.WriteString(fmt.Sprintf("  - Enabled tasks: %d (disabled: %d)\n", agg.Enabled, agg.Disabled))
	b.WriteString(indent)
	b.WriteString(fmt.Sprintf("  - Over budget: %d (budget=%s)\n", agg.OverBudget, formatTokens64(agg.Budget)))
	if agg.Assigned > 0 {
		b.WriteString(indent)
		b.WriteString(fmt.Sprintf("  - Already assigned: %d\n", agg.Assigned))
	}
	b.WriteString(indent)
	b.WriteString(fmt.Sprintf("  - Candidates after filters: %d\n", agg.Candidates))
	if len(agg.UnknownEnabled) > 0 {
		b.WriteString(indent)
		b.WriteString(fmt.Sprintf("  - Unknown enabled task types: %s\n", strings.Join(agg.UnknownEnabled, ", ")))
	}
	if agg.NoEnabledTasks {
		b.WriteString(indent)
		b.WriteString("  - No enabled tasks in config\n")
	}
}

func configLoadedLabel(loaded bool) string {
	if loaded {
		return "loaded"
	}
	return "missing"
}

func indentLines(text, prefix string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func writePreviewText(w io.Writer, text string, options previewPagerOptions) error {
	if canUseGumPager(w, options) {
		if gumPath, ok := ensureGum(); ok {
			if err := runGumPager(w, gumPath, text); err == nil {
				return nil
			}
		}
	}

	_, err := io.WriteString(w, text)
	return err
}

func canUseGumPager(w io.Writer, options previewPagerOptions) bool {
	if options.Plain {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return file.Fd() == os.Stdout.Fd()
}

func ensureGum() (string, bool) {
	if path, err := exec.LookPath("gum"); err == nil {
		return path, true
	}
	if _, err := exec.LookPath("brew"); err == nil {
		cmd := exec.Command("brew", "install", "gum")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		_ = cmd.Run()
		if path, err := exec.LookPath("gum"); err == nil {
			return path, true
		}
	}
	return "", false
}

func runGumPager(w io.Writer, gumPath, text string) error {
	cmd := exec.Command(gumPath, "pager")
	cmd.Stdin = strings.NewReader(text)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type previewJSON struct {
	GeneratedAt     string                      `json:"generated_at"`
	Provider        string                      `json:"provider"`
	TaskFilter      string                      `json:"task_filter,omitempty"`
	EnabledTasks    []string                    `json:"enabled_tasks,omitempty"`
	Budget          previewJSONBudgetConfig     `json:"budget"`
	Config          previewJSONConfigSources    `json:"config"`
	ProviderBudgets []previewJSONProviderBudget `json:"provider_budgets,omitempty"`
	Runs            []previewJSONRun            `json:"runs"`
	Notes           []string                    `json:"notes,omitempty"`
}

type previewJSONBudgetConfig struct {
	Mode           string `json:"mode"`
	MaxPercent     int    `json:"max_percent"`
	ReservePercent int    `json:"reserve_percent"`
}

type previewJSONConfigSources struct {
	Global  previewJSONConfigSource `json:"global"`
	Project previewJSONConfigSource `json:"project"`
	Order   string                  `json:"order"`
}

type previewJSONConfigSource struct {
	Path   string `json:"path"`
	Loaded bool   `json:"loaded"`
}

type previewJSONProviderBudget struct {
	Provider     string  `json:"provider"`
	Allowance    int64   `json:"allowance"`
	UsedPercent  float64 `json:"used_percent"`
	WeeklyBudget int64   `json:"weekly_budget"`
	Source       string  `json:"source"`
	Confidence   string  `json:"confidence,omitempty"`
	Samples      int     `json:"samples,omitempty"`
	Error        string  `json:"error,omitempty"`
}

type previewJSONRun struct {
	Index    int                  `json:"index"`
	RunAt    string               `json:"run_at"`
	Projects []previewJSONProject `json:"projects"`
}

type previewJSONProject struct {
	Path        string              `json:"path"`
	Status      string              `json:"status"`
	Detail      string              `json:"detail,omitempty"`
	Budget      *previewJSONBudget  `json:"budget,omitempty"`
	Tasks       []previewJSONTask   `json:"tasks,omitempty"`
	Diagnostics *previewDiagnostics `json:"diagnostics,omitempty"`
}

type previewJSONBudget struct {
	Allowance      int64   `json:"allowance"`
	WeeklyBudget   int64   `json:"weekly_budget"`
	BudgetBase     int64   `json:"budget_base"`
	UsedPercent    float64 `json:"used_percent"`
	ReserveAmount  int64   `json:"reserve_amount"`
	PredictedUsage int64   `json:"predicted_usage"`
	Mode           string  `json:"mode"`
	RemainingDays  int     `json:"remaining_days,omitempty"`
	Multiplier     float64 `json:"multiplier,omitempty"`
	Source         string  `json:"source"`
	Confidence     string  `json:"confidence,omitempty"`
	Samples        int     `json:"samples,omitempty"`
}

type previewJSONTask struct {
	Index           int     `json:"index"`
	Type            string  `json:"type"`
	Name            string  `json:"name"`
	Description     string  `json:"description,omitempty"`
	Score           float64 `json:"score"`
	CostTier        string  `json:"cost_tier"`
	MinTokens       int     `json:"min_tokens"`
	MaxTokens       int     `json:"max_tokens"`
	Prompt          string  `json:"prompt"`
	PromptFile      string  `json:"prompt_file,omitempty"`
	PromptFileError string  `json:"prompt_file_error,omitempty"`
}

func writePreviewJSON(w io.Writer, result *previewResult) error {
	payload := buildPreviewJSON(result)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func buildPreviewJSON(result *previewResult) previewJSON {
	configSources := previewJSONConfigSources{
		Order: "global -> project -> env overrides",
	}
	if result.ConfigSources != nil {
		configSources.Global = previewJSONConfigSource{Path: result.ConfigSources.GlobalPath, Loaded: result.ConfigSources.GlobalExists}
		configSources.Project = previewJSONConfigSource{Path: result.ConfigSources.ProjectPath, Loaded: result.ConfigSources.ProjectExists}
	}

	budgets := make([]previewJSONProviderBudget, 0, len(result.Providers))
	for _, summary := range result.Providers {
		entry := previewJSONProviderBudget{Provider: summary.name}
		if summary.err != nil {
			entry.Error = summary.err.Error()
			budgets = append(budgets, entry)
			continue
		}
		entry.Allowance = summary.allowance.Allowance
		entry.UsedPercent = summary.allowance.UsedPercent
		entry.WeeklyBudget = summary.allowance.WeeklyBudget
		entry.Source = summary.allowance.BudgetSource
		entry.Confidence = summary.allowance.BudgetConfidence
		entry.Samples = summary.allowance.BudgetSampleCount
		budgets = append(budgets, entry)
	}

	runs := make([]previewJSONRun, 0, len(result.Runs))
	for _, run := range result.Runs {
		projects := make([]previewJSONProject, 0, len(run.Projects))
		for _, project := range run.Projects {
			var budgetPayload *previewJSONBudget
			if project.Budget != nil {
				budgetPayload = &previewJSONBudget{
					Allowance:      project.Budget.Allowance,
					WeeklyBudget:   project.Budget.WeeklyBudget,
					BudgetBase:     project.Budget.BudgetBase,
					UsedPercent:    project.Budget.UsedPercent,
					ReserveAmount:  project.Budget.ReserveAmount,
					PredictedUsage: project.Budget.PredictedUsage,
					Mode:           project.Budget.Mode,
					RemainingDays:  project.Budget.RemainingDays,
					Multiplier:     project.Budget.Multiplier,
					Source:         project.Budget.BudgetSource,
					Confidence:     project.Budget.BudgetConfidence,
					Samples:        project.Budget.BudgetSampleCount,
				}
			}

			tasksPayload := make([]previewJSONTask, 0, len(project.Tasks))
			for _, task := range project.Tasks {
				tasksPayload = append(tasksPayload, previewJSONTask{
					Index:           task.Index,
					Type:            task.Type,
					Name:            task.Name,
					Description:     task.Description,
					Score:           task.Score,
					CostTier:        task.CostTier,
					MinTokens:       task.MinTokens,
					MaxTokens:       task.MaxTokens,
					Prompt:          task.Prompt,
					PromptFile:      task.PromptFile,
					PromptFileError: task.PromptFileError,
				})
			}

			projects = append(projects, previewJSONProject{
				Path:        project.Path,
				Status:      string(project.Status),
				Detail:      project.Detail,
				Budget:      budgetPayload,
				Tasks:       tasksPayload,
				Diagnostics: project.Diagnostics,
			})
		}

		runs = append(runs, previewJSONRun{
			Index:    run.Index,
			RunAt:    run.RunAt.Format(time.RFC3339),
			Projects: projects,
		})
	}

	payload := previewJSON{
		GeneratedAt:  result.GeneratedAt.Format(time.RFC3339),
		Provider:     result.Provider,
		TaskFilter:   result.TaskFilter,
		EnabledTasks: result.EnabledTasks,
		Budget: previewJSONBudgetConfig{
			Mode:           result.BudgetMode,
			MaxPercent:     result.MaxPercent,
			ReservePercent: result.ReservePercent,
		},
		Config:          configSources,
		ProviderBudgets: budgets,
		Runs:            runs,
	}
	if result.Note != "" {
		payload.Notes = []string{result.Note}
	}

	return payload
}
