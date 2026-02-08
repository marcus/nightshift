package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/nightshift/internal/agents"
	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/calibrator"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/logging"
	"github.com/marcus/nightshift/internal/orchestrator"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/marcus/nightshift/internal/reporting"
	"github.com/marcus/nightshift/internal/state"
	"github.com/marcus/nightshift/internal/tasks"
	"github.com/marcus/nightshift/internal/trends"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// isInteractive reports whether stdout is a terminal. Override in tests.
var isInteractive = func() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// confirmRun prompts the user for confirmation unless bypassed by flags or
// non-TTY context. Returns true if execution should proceed.
func confirmRun(p executeRunParams) (bool, error) {
	if p.yes {
		return true, nil
	}
	if p.dryRun {
		return false, nil
	}
	if !isInteractive() {
		p.log.Info("non-TTY: auto-confirming")
		return true, nil
	}
	fmt.Print("Proceed? [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		ans := strings.TrimSpace(scanner.Text())
		if strings.EqualFold(ans, "y") || strings.EqualFold(ans, "yes") {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read stdin: %w", err)
	}
	return false, nil
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute tasks",
	Long: `Execute configured tasks immediately.

Before executing, nightshift displays a preflight summary showing the
selected provider, budget status, projects, and tasks. In interactive
terminals a confirmation prompt is shown; use --yes to skip it. In
non-TTY environments (cron, daemon, CI) confirmation is auto-skipped.

Use --dry-run to display the preflight summary and exit without
executing anything.

Flags:
  --max-projects N   Limit how many projects are processed (default 1).
                     Ignored when --project is set.
  --max-tasks N      Limit how many tasks run per project (default 1).
                     Ignored when --task is set.
  --random-task      Pick a random task from eligible tasks (exactly 1).
                     Mutually exclusive with --task.
  --ignore-budget    Bypass budget checks (use with caution).
  --yes / -y         Skip the confirmation prompt.
  --dry-run          Show preflight summary and exit without executing.

Examples:
  nightshift run                              # Interactive: preflight + prompt
  nightshift run --yes                        # Skip confirmation
  nightshift run --dry-run                    # Preview only, no execution
  nightshift run --max-projects 3             # Process up to 3 projects
  nightshift run --max-tasks 3                # Up to 3 tasks per project
  nightshift run --random-task                # Pick a random eligible task
  nightshift run --ignore-budget              # Run even if budget exhausted
  nightshift run -p ./my-project -t lint-fix  # Specific project + task`,
	RunE: runRun,
}

func init() {
	runCmd.Flags().Bool("dry-run", false, "Simulate execution without making changes")
	runCmd.Flags().StringP("project", "p", "", "Path to project directory")
	runCmd.Flags().StringP("task", "t", "", "Run specific task by name")
	runCmd.Flags().Int("max-projects", 1, "Max projects to process per run (ignored when --project is set)")
	runCmd.Flags().Int("max-tasks", 1, "Max tasks to run per project (ignored when --task is set)")
	runCmd.Flags().Bool("ignore-budget", false, "Bypass budget checks (use with caution)")
	runCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	runCmd.Flags().Bool("random-task", false, "Pick a random task from eligible tasks")
	runCmd.Flags().Bool("no-color", false, "Disable colored output")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	projectPath, _ := cmd.Flags().GetString("project")
	taskFilter, _ := cmd.Flags().GetString("task")
	maxProjects, _ := cmd.Flags().GetInt("max-projects")
	maxTasks, _ := cmd.Flags().GetInt("max-tasks")
	ignoreBudget, _ := cmd.Flags().GetBool("ignore-budget")
	yes, _ := cmd.Flags().GetBool("yes")
	randomTask, _ := cmd.Flags().GetBool("random-task")

	if randomTask && taskFilter != "" {
		return fmt.Errorf("--random-task and --task are mutually exclusive")
	}

	noColor, _ := cmd.Flags().GetBool("no-color")
	if noColor || os.Getenv("NO_COLOR") != "" {
		lipgloss.SetColorProfile(termenv.Ascii)
	}

	// Augment PATH so provider CLIs are discoverable when launched
	// from launchd/systemd/cron which have a minimal PATH.
	ensurePATH()

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\ninterrupt received, shutting down...")
		cancel()
	}()

	// Load configuration
	cfg, err := loadConfig(projectPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize logging
	if err := initLogging(cfg); err != nil {
		return fmt.Errorf("init logging: %w", err)
	}
	log := logging.Component("run")
	log.Info("starting nightshift run")

	// Initialize state manager
	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() { _ = database.Close() }()

	st, err := state.New(database)
	if err != nil {
		return fmt.Errorf("init state: %w", err)
	}

	// Clear stale assignments older than 2 hours
	cleared := st.ClearStaleAssignments(2 * time.Hour)
	if cleared > 0 {
		log.Infof("cleared %d stale assignments", cleared)
	}

	// Initialize providers
	claudeProvider := providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude"))
	codexProvider := providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex"))

	// Initialize budget manager
	cal := calibrator.New(database, cfg)
	trend := trends.NewAnalyzer(database, cfg.Budget.SnapshotRetentionDays)
	budgetMgr := budget.NewManagerFromProviders(cfg, claudeProvider, codexProvider, budget.WithBudgetSource(cal), budget.WithTrendAnalyzer(trend))

	// Determine projects to run
	projects, err := resolveProjects(cfg, projectPath)
	if err != nil {
		return fmt.Errorf("resolve projects: %w", err)
	}

	// Limit projects when --project was not explicitly set
	if projectPath == "" && maxProjects > 0 && len(projects) > maxProjects {
		projects = projects[:maxProjects]
	}

	if len(projects) == 0 {
		fmt.Println("no projects configured")
		return nil
	}

	// Register custom tasks from config
	tasks.ClearCustom()
	if err := tasks.RegisterCustomTasksFromConfig(cfg.Tasks.Custom); err != nil {
		return fmt.Errorf("register custom tasks: %w", err)
	}

	// Create task selector
	selector := tasks.NewSelector(cfg, st)

	// Run execution
	if ignoreBudget {
		fmt.Println("WARNING: --ignore-budget is set, budget checks will be bypassed")
		log.Warn("--ignore-budget active, bypassing budget checks")
	}

	params := executeRunParams{
		cfg:          cfg,
		budgetMgr:    budgetMgr,
		selector:     selector,
		st:           st,
		projects:     projects,
		taskFilter:   taskFilter,
		maxTasks:     maxTasks,
		randomTask:   randomTask,
		ignoreBudget: ignoreBudget,
		dryRun:       dryRun,
		yes:          yes,
		log:          log,
	}
	if !dryRun {
		params.report = newRunReport(time.Now(), calculateRunBudgetStart(cfg, budgetMgr, log))
	}
	return executeRun(ctx, params)
}

type executeRunParams struct {
	cfg          *config.Config
	budgetMgr    *budget.Manager
	selector     *tasks.Selector
	st           *state.State
	projects     []string
	taskFilter   string
	maxTasks     int
	randomTask   bool
	ignoreBudget bool
	dryRun       bool
	yes          bool
	report       *runReport
	log          *logging.Logger
}

// providerChoice holds a selected provider's agent and name.
type providerChoice struct {
	agent     agents.Agent
	name      string
	allowance *budget.AllowanceResult
}

// selectProvider picks the best available provider with budget remaining.
// Order is determined by providers.preference (default: claude, codex).
// When ignoreBudget is true, budget-exhausted providers are still selected.
func selectProvider(cfg *config.Config, budgetMgr *budget.Manager, log *logging.Logger, ignoreBudget bool) (*providerChoice, error) {
	type candidate struct {
		name      string
		binary    string
		makeAgent func() agents.Agent
	}

	var candidates []candidate
	for _, name := range providerPreference(cfg) {
		switch name {
		case "claude":
			if cfg.Providers.Claude.Enabled {
				candidates = append(candidates, candidate{
					name:      "claude",
					binary:    "claude",
					makeAgent: func() agents.Agent { return newClaudeAgentFromConfig(cfg) },
				})
			}
		case "codex":
			if cfg.Providers.Codex.Enabled {
				candidates = append(candidates, candidate{
					name:      "codex",
					binary:    "codex",
					makeAgent: func() agents.Agent { return newCodexAgentFromConfig(cfg) },
				})
			}
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no providers enabled in config")
	}

	var notInPath, budgetExhausted []string
	for _, c := range candidates {
		if _, err := exec.LookPath(c.binary); err != nil {
			log.Infof("provider %s: CLI not in PATH, skipping", c.name)
			notInPath = append(notInPath, c.name)
			continue
		}
		allowance, err := budgetMgr.CalculateAllowance(c.name)
		if err != nil {
			log.Warnf("provider %s: budget error: %v", c.name, err)
			continue
		}
		if allowance.Allowance <= 0 {
			log.Infof("provider %s: budget exhausted (%.1f%% used)", c.name, allowance.UsedPercent)
			if ignoreBudget {
				log.Warnf("provider %s: ignoring exhausted budget per --ignore-budget", c.name)
				return &providerChoice{
					agent:     c.makeAgent(),
					name:      c.name,
					allowance: allowance,
				}, nil
			}
			budgetExhausted = append(budgetExhausted, fmt.Sprintf("%s (%.0f%% used)", c.name, allowance.UsedPercent))
			continue
		}
		return &providerChoice{
			agent:     c.makeAgent(),
			name:      c.name,
			allowance: allowance,
		}, nil
	}

	if len(notInPath) > 0 && len(budgetExhausted) == 0 {
		return nil, fmt.Errorf("CLI not in PATH: %s", strings.Join(notInPath, ", "))
	}
	if len(budgetExhausted) > 0 && len(notInPath) == 0 {
		return nil, fmt.Errorf("budget exhausted: %s", strings.Join(budgetExhausted, ", "))
	}
	if len(budgetExhausted) > 0 && len(notInPath) > 0 {
		return nil, fmt.Errorf("budget exhausted: %s; CLI not in PATH: %s",
			strings.Join(budgetExhausted, ", "), strings.Join(notInPath, ", "))
	}
	return nil, fmt.Errorf("no providers available")
}

func providerPreference(cfg *config.Config) []string {
	defaults := []string{"claude", "codex"}
	if cfg == nil || len(cfg.Providers.Preference) == 0 {
		return defaults
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(cfg.Providers.Preference))
	for _, pref := range cfg.Providers.Preference {
		name := strings.ToLower(strings.TrimSpace(pref))
		if name == "" || seen[name] {
			continue
		}
		if name != "claude" && name != "codex" {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	if len(out) == 0 {
		return defaults
	}
	return out
}

// preflightProject holds the planned tasks for a single project.
type preflightProject struct {
	path       string
	tasks      []tasks.ScoredTask
	provider   *providerChoice
	skipReason string // non-empty if project was skipped
}

// preflightPlan collects all planned work before execution.
type preflightPlan struct {
	projects     []preflightProject
	skipReasons  []string // global skip reasons (e.g., no provider)
	ignoreBudget bool
}

// buildPreflight performs the planning phase: resolve provider, select tasks
// per project, but does NOT execute anything.
func buildPreflight(p executeRunParams) (*preflightPlan, error) {
	plan := &preflightPlan{
		ignoreBudget: p.ignoreBudget,
	}

	for _, projectPath := range p.projects {
		// Skip if already processed today (unless task filter specified)
		if p.taskFilter == "" && p.st.WasProcessedToday(projectPath) {
			p.log.Infof("skip %s (processed today)", projectPath)
			reason := fmt.Sprintf("%s: already processed today", filepath.Base(projectPath))
			plan.projects = append(plan.projects, preflightProject{
				path:       projectPath,
				skipReason: "already processed today",
			})
			plan.skipReasons = append(plan.skipReasons, reason)
			continue
		}

		// Select the best available provider with remaining budget
		choice, err := selectProvider(p.cfg, p.budgetMgr, p.log, p.ignoreBudget)
		if err != nil {
			p.log.Infof("no provider available: %v", err)
			plan.skipReasons = append(plan.skipReasons, fmt.Sprintf("no provider: %v", err))
			break
		}

		// Select tasks
		var selectedTasks []tasks.ScoredTask

		if p.taskFilter != "" {
			def, err := tasks.GetDefinition(tasks.TaskType(p.taskFilter))
			if err != nil {
				return nil, fmt.Errorf("unknown task type: %s", p.taskFilter)
			}
			selectedTasks = []tasks.ScoredTask{{
				Definition: def,
				Score:      p.selector.ScoreTask(def.Type, projectPath),
				Project:    projectPath,
			}}
		} else if p.randomTask {
			taskBudget := choice.allowance.Allowance
			if p.ignoreBudget {
				taskBudget = math.MaxInt64
			}
			if picked := p.selector.SelectRandom(taskBudget, projectPath); picked != nil {
				selectedTasks = []tasks.ScoredTask{*picked}
			}
		} else {
			n := p.maxTasks
			if n <= 0 {
				n = 1
			}
			taskBudget := choice.allowance.Allowance
			if p.ignoreBudget {
				taskBudget = math.MaxInt64
			}
			selectedTasks = p.selector.SelectTopN(taskBudget, projectPath, n)
		}

		pp := preflightProject{
			path:     projectPath,
			tasks:    selectedTasks,
			provider: choice,
		}

		if len(selectedTasks) == 0 {
			skipReason := "no tasks available within budget"
			allEnabled := p.selector.FilterEnabled(tasks.AllDefinitions())
			inBudget := p.selector.FilterByBudget(allEnabled, choice.allowance.Allowance)
			unassigned := p.selector.FilterUnassigned(inBudget, projectPath)
			afterCooldown := p.selector.FilterByCooldown(unassigned, projectPath)
			cooledDown := len(unassigned) - len(afterCooldown)
			if cooledDown > 0 {
				skipReason = fmt.Sprintf("%d task(s) on cooldown", cooledDown)
			}
			pp.skipReason = skipReason
			plan.skipReasons = append(plan.skipReasons, fmt.Sprintf("%s: %s", filepath.Base(projectPath), skipReason))
		}

		plan.projects = append(plan.projects, pp)
	}

	return plan, nil
}

// displayPreflight renders the preflight summary to the given writer.
func displayPreflight(w io.Writer, plan *preflightPlan) {
	fmt.Fprintf(w, "\n=== Preflight Summary ===\n")

	// Show provider info from first project that has one
	for _, pp := range plan.projects {
		if pp.provider != nil {
			fmt.Fprintf(w, "Provider: %s (%.1f%% budget used, %s mode)\n",
				pp.provider.name, pp.provider.allowance.UsedPercent, pp.provider.allowance.Mode)
			fmt.Fprintf(w, "Budget: %d tokens remaining\n", pp.provider.allowance.Allowance)
			break
		}
	}

	// Count active projects (those with tasks)
	active := 0
	for _, pp := range plan.projects {
		if len(pp.tasks) > 0 {
			active++
		}
	}
	fmt.Fprintf(w, "\nProjects (%d of %d):\n", active, len(plan.projects))

	idx := 0
	for _, pp := range plan.projects {
		if pp.skipReason != "" || len(pp.tasks) == 0 {
			continue
		}
		idx++
		fmt.Fprintf(w, "  %d. %s\n", idx, filepath.Base(pp.path))
		for _, st := range pp.tasks {
			minTok, maxTok := st.Definition.EstimatedTokens()
			fmt.Fprintf(w, "     - %s (score=%.1f, cost=%s, ~%dk-%dk tokens)\n",
				st.Definition.Name, st.Score, st.Definition.CostTier, minTok/1000, maxTok/1000)
		}
	}

	// Skipped projects
	var skipped []preflightProject
	for _, pp := range plan.projects {
		if pp.skipReason != "" {
			skipped = append(skipped, pp)
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintf(w, "\nSkipped:\n")
		for _, pp := range skipped {
			fmt.Fprintf(w, "  - %s: %s\n", filepath.Base(pp.path), pp.skipReason)
		}
	}

	// Warnings
	if plan.ignoreBudget {
		fmt.Fprintf(w, "\nWarnings:\n")
		fmt.Fprintf(w, "  - --ignore-budget is set: budget limits bypassed\n")
	}

	fmt.Fprintln(w)
}

func executeRun(ctx context.Context, p executeRunParams) error {
	start := time.Now()

	// Build preflight plan
	plan, err := buildPreflight(p)
	if err != nil {
		return err
	}

	// Display preflight summary
	if isInteractive() {
		displayPreflightColored(plan)
	} else {
		displayPreflight(os.Stdout, plan)
	}

	// Dry-run: show preflight and exit without executing
	if p.dryRun {
		fmt.Println("[dry-run] No tasks executed.")
		return nil
	}

	// Confirm before proceeding
	proceed, err := confirmRun(p)
	if err != nil {
		return err
	}
	if !proceed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Execute based on the plan
	var tasksRun, tasksCompleted, tasksFailed int
	var skipReasons []string
	skipReasons = append(skipReasons, plan.skipReasons...)

	for _, pp := range plan.projects {
		select {
		case <-ctx.Done():
			p.log.Info("run cancelled")
			return ctx.Err()
		default:
		}

		if pp.skipReason != "" {
			if pp.skipReason == "already processed today" {
				fmt.Printf("Skipping %s: already processed today\n", filepath.Base(pp.path))
			}
			if pp.provider == nil && len(pp.tasks) == 0 {
				// Skip reason already in plan.skipReasons
				if p.report != nil {
					p.report.addTask(reporting.TaskResult{
						Project:    pp.path,
						TaskType:   "",
						Title:      "No tasks selected",
						Status:     "skipped",
						SkipReason: pp.skipReason,
					})
				}
			}
			continue
		}

		if len(pp.tasks) == 0 {
			continue
		}

		choice := pp.provider
		projectPath := pp.path

		if isInteractive() {
			displayProjectHeaderColored(projectPath, choice.name, choice.allowance, len(pp.tasks), pp.tasks)
		} else {
			fmt.Printf("\n=== Project: %s ===\n", projectPath)
			fmt.Printf("Provider: %s\n", choice.name)
			fmt.Printf("Budget: %d tokens available (%.1f%% used, mode=%s)\n",
				choice.allowance.Allowance, choice.allowance.UsedPercent, choice.allowance.Mode)

			fmt.Printf("Selected %d task(s):\n", len(pp.tasks))
			for i, st := range pp.tasks {
				minTok, maxTok := st.Definition.EstimatedTokens()
				fmt.Printf("  %d. %s (score=%.1f, cost=%s, tokens=%d-%d)\n",
					i+1, st.Definition.Name, st.Score, st.Definition.CostTier, minTok, maxTok)
			}
		}

		// Create orchestrator with the selected agent
		var renderer *liveRenderer
		if isInteractive() {
			renderer = newLiveRenderer()
			defer renderer.cleanup()
		}

		orchOpts := []orchestrator.Option{
			orchestrator.WithAgent(choice.agent),
			orchestrator.WithConfig(orchestrator.Config{
				MaxIterations: 3,
				AgentTimeout:  30 * time.Minute,
			}),
			orchestrator.WithLogger(logging.Component("orchestrator")),
		}
		if renderer != nil {
			orchOpts = append(orchOpts, orchestrator.WithEventHandler(renderer.HandleEvent))
		}
		orch := orchestrator.New(orchOpts...)

		projectStart := time.Now()
		projectTaskTypes := make([]string, 0, len(pp.tasks))
		projectTokensUsed := 0
		projectCompleted := 0
		projectFailed := 0

		// Execute each selected task
		for _, scoredTask := range pp.tasks {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			tasksRun++
			if !isInteractive() {
				fmt.Printf("\n--- Running: %s (via %s) ---\n", scoredTask.Definition.Name, choice.name)
			}
			projectTaskTypes = append(projectTaskTypes, string(scoredTask.Definition.Type))

			// Create task instance
			taskInstance := &tasks.Task{
				ID:          fmt.Sprintf("%s:%s", scoredTask.Definition.Type, projectPath),
				Title:       scoredTask.Definition.Name,
				Description: scoredTask.Definition.Description,
				Priority:    int(scoredTask.Score),
				Type:        scoredTask.Definition.Type,
			}

			// Mark as assigned
			p.st.MarkAssigned(taskInstance.ID, projectPath, string(scoredTask.Definition.Type))

			// Inject run metadata for PR traceability
			orch.SetRunMetadata(&orchestrator.RunMetadata{
				Provider:  choice.name,
				TaskType:  string(scoredTask.Definition.Type),
				TaskScore: scoredTask.Score,
				CostTier:  scoredTask.Definition.CostTier.String(),
				RunStart:  projectStart,
			})

			// Execute via orchestrator
			result, err := orch.RunTask(ctx, taskInstance, projectPath)

			// Clear assignment
			p.st.ClearAssigned(taskInstance.ID)

			if err != nil {
				tasksFailed++
				projectFailed++
				if !isInteractive() {
					fmt.Printf("  FAILED: %v\n", err)
				}
				p.log.Errorf("task %s failed: %v", taskInstance.ID, err)
				if p.report != nil {
					p.report.addTask(reporting.TaskResult{
						Project:    projectPath,
						TaskType:   string(scoredTask.Definition.Type),
						Title:      scoredTask.Definition.Name,
						Status:     "failed",
						TokensUsed: 0,
						Duration:   result.Duration,
					})
				}
				continue
			}

			// Record result
			switch result.Status {
			case orchestrator.StatusCompleted:
				tasksCompleted++
				projectCompleted++
				if !isInteractive() {
					fmt.Printf("  COMPLETED in %d iteration(s) (%s)\n", result.Iterations, result.Duration)
				}
				p.st.RecordTaskRun(projectPath, string(scoredTask.Definition.Type))
				_, maxTok := scoredTask.Definition.EstimatedTokens()
				projectTokensUsed += maxTok
				if p.report != nil {
					p.report.addTask(reporting.TaskResult{
						Project:    projectPath,
						TaskType:   string(scoredTask.Definition.Type),
						Title:      scoredTask.Definition.Name,
						Status:     "completed",
						OutputType: result.OutputType,
						OutputRef:  result.OutputRef,
						TokensUsed: maxTok,
						Duration:   result.Duration,
					})
				}
			case orchestrator.StatusAbandoned:
				tasksFailed++
				projectFailed++
				if !isInteractive() {
					fmt.Printf("  ABANDONED after %d iteration(s): %s\n", result.Iterations, result.Error)
				}
				if p.report != nil {
					p.report.addTask(reporting.TaskResult{
						Project:    projectPath,
						TaskType:   string(scoredTask.Definition.Type),
						Title:      scoredTask.Definition.Name,
						Status:     "failed",
						SkipReason: result.Error,
						Duration:   result.Duration,
					})
				}
			default:
				tasksFailed++
				projectFailed++
				if !isInteractive() {
					fmt.Printf("  FAILED: %s\n", result.Error)
				}
				if p.report != nil {
					p.report.addTask(reporting.TaskResult{
						Project:    projectPath,
						TaskType:   string(scoredTask.Definition.Type),
						Title:      scoredTask.Definition.Name,
						Status:     "failed",
						SkipReason: result.Error,
						Duration:   result.Duration,
					})
				}
			}
		}

		// Record project run
		p.st.RecordProjectRun(projectPath)
		projectStatus := "partial"
		if projectFailed == 0 && projectCompleted > 0 {
			projectStatus = "success"
		}
		if projectCompleted == 0 && projectFailed > 0 {
			projectStatus = "failed"
		}
		p.st.AddRunRecord(state.RunRecord{
			StartTime:  projectStart,
			EndTime:    time.Now(),
			Provider:   choice.name,
			Project:    projectPath,
			Tasks:      projectTaskTypes,
			TokensUsed: projectTokensUsed,
			Status:     projectStatus,
		})
	}

	// Summary
	duration := time.Since(start)
	if isInteractive() {
		displayRunSummaryColored(duration, tasksRun, tasksCompleted, tasksFailed, skipReasons)
	} else {
		fmt.Printf("\n=== Run Complete ===\n")
		fmt.Printf("Duration: %s\n", duration.Round(time.Second))
		fmt.Printf("Tasks: %d run, %d completed, %d failed\n", tasksRun, tasksCompleted, tasksFailed)

		if tasksRun == 0 && len(skipReasons) > 0 {
			fmt.Println("\nNothing ran because:")
			for _, reason := range skipReasons {
				fmt.Printf("  - %s\n", reason)
			}
		}
	}

	p.log.InfoCtx("run complete", map[string]any{
		"duration":  duration.String(),
		"tasks_run": tasksRun,
		"completed": tasksCompleted,
		"failed":    tasksFailed,
		"projects":  len(p.projects),
	})

	if p.report != nil {
		p.report.finalize(p.cfg, p.log)
	}

	return nil
}

// loadConfig loads configuration from the appropriate paths.
func loadConfig(projectPath string) (*config.Config, error) {
	if projectPath == "" {
		return config.Load()
	}
	return config.LoadFromPaths(projectPath, "")
}

// initLogging initializes the logging subsystem.
func initLogging(cfg *config.Config) error {
	return logging.Init(logging.Config{
		Level:  cfg.Logging.Level,
		Path:   cfg.ExpandedLogPath(),
		Format: cfg.Logging.Format,
	})
}

// resolveProjects determines which projects to process.
func resolveProjects(cfg *config.Config, projectPath string) ([]string, error) {
	// If explicit project path specified, use only that
	if projectPath != "" {
		abs, err := filepath.Abs(projectPath)
		if err != nil {
			return nil, fmt.Errorf("invalid project path: %w", err)
		}
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			return nil, fmt.Errorf("project path does not exist: %s", abs)
		}
		return []string{abs}, nil
	}

	// Use projects from config
	if len(cfg.Projects) > 0 {
		var projects []string
		for _, p := range cfg.Projects {
			path := expandPath(p.Path)
			if _, err := os.Stat(path); err == nil {
				projects = append(projects, path)
			}
		}
		return projects, nil
	}

	// Default to current directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return []string{cwd}, nil
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

// ensurePATH augments the current PATH with common bin directories so that
// provider CLIs (claude, codex) are discoverable even when nightshift is
// launched from launchd, systemd, or cron which provide a minimal PATH.
func ensurePATH() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	extra := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "go", "bin"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, ".npm-global", "bin"),
		"/usr/local/bin",
		"/opt/homebrew/bin",
	}

	// Also include $GOPATH/bin if set
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		extra = append(extra, filepath.Join(gopath, "bin"))
	}

	current := os.Getenv("PATH")
	existing := make(map[string]bool)
	for _, p := range strings.Split(current, string(os.PathListSeparator)) {
		existing[p] = true
	}

	var added []string
	for _, dir := range extra {
		if existing[dir] {
			continue
		}
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			added = append(added, dir)
		}
	}

	if len(added) > 0 {
		newPath := current + string(os.PathListSeparator) + strings.Join(added, string(os.PathListSeparator))
		os.Setenv("PATH", newPath)
	}
}
