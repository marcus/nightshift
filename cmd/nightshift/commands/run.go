package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/marcusvorwaller/nightshift/internal/agents"
	"github.com/marcusvorwaller/nightshift/internal/budget"
	"github.com/marcusvorwaller/nightshift/internal/config"
	"github.com/marcusvorwaller/nightshift/internal/logging"
	"github.com/marcusvorwaller/nightshift/internal/orchestrator"
	"github.com/marcusvorwaller/nightshift/internal/providers"
	"github.com/marcusvorwaller/nightshift/internal/state"
	"github.com/marcusvorwaller/nightshift/internal/tasks"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute tasks",
	Long: `Execute configured tasks immediately.

By default, runs all enabled tasks. Use --task to run a specific task.
Use --dry-run to simulate execution without making changes.`,
	RunE: runRun,
}

func init() {
	runCmd.Flags().Bool("dry-run", false, "Simulate execution without making changes")
	runCmd.Flags().StringP("project", "p", "", "Path to project directory")
	runCmd.Flags().StringP("task", "t", "", "Run specific task by name")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	projectPath, _ := cmd.Flags().GetString("project")
	taskFilter, _ := cmd.Flags().GetString("task")

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
	st, err := state.New("")
	if err != nil {
		return fmt.Errorf("init state: %w", err)
	}
	defer st.Save()

	// Clear stale assignments older than 2 hours
	cleared := st.ClearStaleAssignments(2 * time.Hour)
	if cleared > 0 {
		log.Infof("cleared %d stale assignments", cleared)
	}

	// Initialize providers
	claudeProvider := providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude"))
	codexProvider := providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex"))

	// Initialize budget manager
	budgetMgr := budget.NewManagerFromProviders(cfg, claudeProvider, codexProvider)

	// Determine projects to run
	projects, err := resolveProjects(cfg, projectPath)
	if err != nil {
		return fmt.Errorf("resolve projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("no projects configured")
		return nil
	}

	// Create task selector
	selector := tasks.NewSelector(cfg, st)

	// Initialize agent
	agent := agents.NewClaudeAgent()
	if !agent.Available() {
		return fmt.Errorf("claude CLI not found in PATH")
	}

	// Create orchestrator
	orch := orchestrator.New(
		orchestrator.WithAgent(agent),
		orchestrator.WithConfig(orchestrator.Config{
			MaxIterations: 3,
			AgentTimeout:  30 * time.Minute,
		}),
		orchestrator.WithLogger(logging.Component("orchestrator")),
	)

	// Run execution
	return executeRun(ctx, executeRunParams{
		cfg:       cfg,
		budgetMgr: budgetMgr,
		selector:  selector,
		orch:      orch,
		st:        st,
		projects:  projects,
		taskFilter: taskFilter,
		dryRun:    dryRun,
		log:       log,
	})
}

type executeRunParams struct {
	cfg        *config.Config
	budgetMgr  *budget.Manager
	selector   *tasks.Selector
	orch       *orchestrator.Orchestrator
	st         *state.State
	projects   []string
	taskFilter string
	dryRun     bool
	log        *logging.Logger
}

func executeRun(ctx context.Context, p executeRunParams) error {
	start := time.Now()
	var tasksRun, tasksCompleted, tasksFailed int

	// Process each project
	for _, projectPath := range p.projects {
		select {
		case <-ctx.Done():
			p.log.Info("run cancelled")
			return ctx.Err()
		default:
		}

		// Skip if already processed today (unless task filter specified)
		if p.taskFilter == "" && p.st.WasProcessedToday(projectPath) {
			p.log.Infof("skip %s (processed today)", projectPath)
			continue
		}

		// Calculate budget allowance for primary provider (claude)
		allowance, err := p.budgetMgr.CalculateAllowance("claude")
		if err != nil {
			p.log.Warnf("budget calc error: %v", err)
			continue
		}

		if allowance.Allowance <= 0 {
			p.log.Info("budget exhausted")
			break
		}

		fmt.Printf("\n=== Project: %s ===\n", projectPath)
		fmt.Printf("Budget: %d tokens available (%.1f%% used, mode=%s)\n",
			allowance.Allowance, allowance.UsedPercent, allowance.Mode)

		// Select tasks
		var selectedTasks []tasks.ScoredTask

		if p.taskFilter != "" {
			// Filter to specific task type
			def, err := tasks.GetDefinition(tasks.TaskType(p.taskFilter))
			if err != nil {
				return fmt.Errorf("unknown task type: %s", p.taskFilter)
			}
			selectedTasks = []tasks.ScoredTask{{
				Definition: def,
				Score:      p.selector.ScoreTask(def.Type, projectPath),
				Project:    projectPath,
			}}
		} else {
			// Select top tasks that fit budget
			selectedTasks = p.selector.SelectTopN(allowance.Allowance, projectPath, 5)
		}

		if len(selectedTasks) == 0 {
			fmt.Println("No tasks available within budget")
			continue
		}

		fmt.Printf("Selected %d task(s):\n", len(selectedTasks))
		for i, st := range selectedTasks {
			minTok, maxTok := st.Definition.EstimatedTokens()
			fmt.Printf("  %d. %s (score=%.1f, cost=%s, tokens=%d-%d)\n",
				i+1, st.Definition.Name, st.Score, st.Definition.CostTier, minTok, maxTok)
		}

		if p.dryRun {
			fmt.Println("[dry-run] would execute tasks above")
			continue
		}

		// Execute each selected task
		for _, scoredTask := range selectedTasks {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			tasksRun++
			fmt.Printf("\n--- Running: %s ---\n", scoredTask.Definition.Name)

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

			// Execute via orchestrator
			result, err := p.orch.RunTask(ctx, taskInstance, projectPath)

			// Clear assignment
			p.st.ClearAssigned(taskInstance.ID)

			if err != nil {
				tasksFailed++
				fmt.Printf("  FAILED: %v\n", err)
				p.log.Errorf("task %s failed: %v", taskInstance.ID, err)
				continue
			}

			// Record result
			switch result.Status {
			case orchestrator.StatusCompleted:
				tasksCompleted++
				fmt.Printf("  COMPLETED in %d iteration(s) (%s)\n", result.Iterations, result.Duration)
				p.st.RecordTaskRun(projectPath, string(scoredTask.Definition.Type))
			case orchestrator.StatusAbandoned:
				tasksFailed++
				fmt.Printf("  ABANDONED after %d iteration(s): %s\n", result.Iterations, result.Error)
			default:
				tasksFailed++
				fmt.Printf("  FAILED: %s\n", result.Error)
			}
		}

		// Record project run
		p.st.RecordProjectRun(projectPath)
	}

	// Summary
	duration := time.Since(start)
	fmt.Printf("\n=== Run Complete ===\n")
	fmt.Printf("Duration: %s\n", duration.Round(time.Second))
	fmt.Printf("Tasks: %d run, %d completed, %d failed\n", tasksRun, tasksCompleted, tasksFailed)

	p.log.InfoCtx("run complete", map[string]any{
		"duration":   duration.String(),
		"tasks_run":  tasksRun,
		"completed":  tasksCompleted,
		"failed":     tasksFailed,
		"projects":   len(p.projects),
	})

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
