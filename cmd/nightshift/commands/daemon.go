package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/calibrator"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/logging"
	"github.com/marcus/nightshift/internal/orchestrator"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/marcus/nightshift/internal/reporting"
	"github.com/marcus/nightshift/internal/scheduler"
	"github.com/marcus/nightshift/internal/snapshots"
	"github.com/marcus/nightshift/internal/state"
	"github.com/marcus/nightshift/internal/tasks"
	"github.com/marcus/nightshift/internal/tmux"
	"github.com/marcus/nightshift/internal/trends"
	"github.com/spf13/cobra"
)

const (
	pidFileName = "nightshift.pid"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage background daemon",
	Long:  `Start, stop, or check status of the nightshift background daemon.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start background daemon",
	Long: `Start the nightshift daemon as a background process.

The daemon runs the scheduler loop, executing tasks according to
the configured schedule (cron or interval) and respecting time windows.`,
	RunE: runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop background daemon",
	Long:  `Stop the running nightshift daemon by sending SIGTERM.`,
	RunE:  runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Long:  `Check if the nightshift daemon is running and show status information.`,
	RunE:  runDaemonStatus,
}

var daemonForegroundFlag bool

func init() {
	daemonStartCmd.Flags().BoolVarP(&daemonForegroundFlag, "foreground", "f", false, "Run in foreground (don't daemonize)")
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)
}

// pidFilePath returns the path to the PID file.
func pidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", pidFileName)
}

// ensurePidDir ensures the PID file directory exists.
func ensurePidDir() error {
	dir := filepath.Dir(pidFilePath())
	return os.MkdirAll(dir, 0755)
}

// writePidFile writes the current process PID to the PID file.
func writePidFile() error {
	if err := ensurePidDir(); err != nil {
		return fmt.Errorf("creating pid dir: %w", err)
	}
	return os.WriteFile(pidFilePath(), []byte(strconv.Itoa(os.Getpid())), 0644)
}

// readPidFile reads the PID from the PID file.
func readPidFile() (int, error) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// removePidFile removes the PID file.
func removePidFile() error {
	return os.Remove(pidFilePath())
}

// isProcessRunning checks if a process with the given PID is running.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; send signal 0 to check if alive
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// isDaemonRunning checks if the daemon is currently running.
func isDaemonRunning() (bool, int) {
	pid, err := readPidFile()
	if err != nil {
		return false, 0
	}
	return isProcessRunning(pid), pid
}

func runDaemonStart(cmd *cobra.Command, args []string) error {
	// Check if already running
	if running, pid := isDaemonRunning(); running {
		return fmt.Errorf("daemon already running (pid %d)", pid)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Validate schedule is configured
	if cfg.Schedule.Cron == "" && cfg.Schedule.Interval == "" {
		return fmt.Errorf("no schedule configured (set cron or interval in config)")
	}

	if daemonForegroundFlag {
		// Run in foreground
		return runDaemonLoop(cfg)
	}

	// Daemonize: start a new process with --foreground flag
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable: %w", err)
	}

	daemonCmd := exec.Command(executable, "daemon", "start", "--foreground")
	daemonCmd.Stdout = nil
	daemonCmd.Stderr = nil
	daemonCmd.Stdin = nil
	// Detach from parent process group
	daemonCmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := daemonCmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	fmt.Printf("daemon started (pid %d)\n", daemonCmd.Process.Pid)
	return nil
}

func runDaemonLoop(cfg *config.Config) error {
	// Initialize logging
	if err := initLogging(cfg); err != nil {
		return fmt.Errorf("init logging: %w", err)
	}
	log := logging.Component("daemon")

	// Write PID file
	if err := writePidFile(); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}
	defer func() { _ = removePidFile() }()

	log.Info("daemon starting")

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Set up context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Infof("received signal %v, shutting down", sig)
		cancel()
	}()

	// Initialize scheduler from config
	sched, err := scheduler.NewFromConfig(&cfg.Schedule)
	if err != nil {
		return fmt.Errorf("init scheduler: %w", err)
	}

	// Add the main run job
	sched.AddJob(func(jobCtx context.Context) error {
		return runScheduledTasks(jobCtx, cfg, database, log)
	})

	startSnapshotLoop(ctx, cfg, database, log)
	startSnapshotPruneLoop(ctx, cfg, database, log)

	// Start scheduler
	if err := sched.Start(ctx); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}

	log.InfoCtx("daemon running", map[string]any{
		"next_run": sched.NextRun().Format(time.RFC3339),
	})

	// Wait for context cancellation
	<-ctx.Done()

	// Stop scheduler gracefully
	if err := sched.Stop(); err != nil && err != scheduler.ErrNotRunning {
		log.Errorf("stopping scheduler: %v", err)
	}

	log.Info("daemon stopped")
	return nil
}

// runScheduledTasks executes the scheduled nightshift tasks.
func runScheduledTasks(ctx context.Context, cfg *config.Config, database *db.DB, log *logging.Logger) error {
	log.Info("scheduled run starting")
	start := time.Now()

	// Initialize state manager
	st, err := state.New(database)
	if err != nil {
		log.Errorf("init state: %v", err)
		return err
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

	report := newRunReport(time.Now(), calculateRunBudgetStart(cfg, budgetMgr, log))

	// Resolve projects
	projects, err := resolveProjects(cfg, "")
	if err != nil {
		log.Errorf("resolve projects: %v", err)
		return err
	}

	if len(projects) == 0 {
		log.Info("no projects configured")
		return nil
	}

	// Create task selector
	selector := tasks.NewSelector(cfg, st)

	var tasksRun, tasksCompleted, tasksFailed int

	// Process each project
	for _, projectPath := range projects {
		select {
		case <-ctx.Done():
			log.Info("run cancelled")
			return ctx.Err()
		default:
		}

		// Skip if already processed today
		if st.WasProcessedToday(projectPath) {
			log.Debugf("skip %s (processed today)", projectPath)
			continue
		}

		// Select the best available provider with remaining budget
		choice, err := selectProvider(cfg, budgetMgr, log)
		if err != nil {
			log.Infof("no provider available: %v", err)
			break
		}

		allowance := choice.allowance
		if allowance.Allowance <= 0 {
			log.Info("budget exhausted")
			break
		}

		orch := orchestrator.New(
			orchestrator.WithAgent(choice.agent),
			orchestrator.WithConfig(orchestrator.Config{
				MaxIterations: 3,
				AgentTimeout:  30 * time.Minute,
			}),
			orchestrator.WithLogger(logging.Component("orchestrator")),
		)

		// Select tasks
		selectedTasks := selector.SelectTopN(allowance.Allowance, projectPath, 5)
		if len(selectedTasks) == 0 {
			if report != nil {
				report.addTask(reporting.TaskResult{
					Project:    projectPath,
					TaskType:   "",
					Title:      "No tasks selected",
					Status:     "skipped",
					SkipReason: "no tasks available within budget",
				})
			}
			continue
		}

		log.InfoCtx("processing project", map[string]any{
			"project":  projectPath,
			"tasks":    len(selectedTasks),
			"budget":   allowance.Allowance,
			"provider": choice.name,
		})

		// Execute each selected task
		for _, scoredTask := range selectedTasks {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			tasksRun++

			// Create task instance
			taskInstance := &tasks.Task{
				ID:          fmt.Sprintf("%s:%s", scoredTask.Definition.Type, projectPath),
				Title:       scoredTask.Definition.Name,
				Description: scoredTask.Definition.Description,
				Priority:    int(scoredTask.Score),
				Type:        scoredTask.Definition.Type,
			}

			// Mark as assigned
			st.MarkAssigned(taskInstance.ID, projectPath, string(scoredTask.Definition.Type))

			// Execute via orchestrator
			result, err := orch.RunTask(ctx, taskInstance, projectPath)

			// Clear assignment
			st.ClearAssigned(taskInstance.ID)

			if err != nil {
				tasksFailed++
				log.Errorf("task %s failed: %v", taskInstance.ID, err)
				if report != nil {
					report.addTask(reporting.TaskResult{
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
				st.RecordTaskRun(projectPath, string(scoredTask.Definition.Type))
				log.InfoCtx("task completed", map[string]any{
					"task":       taskInstance.ID,
					"iterations": result.Iterations,
					"duration":   result.Duration.String(),
				})
				if report != nil {
					_, maxTok := scoredTask.Definition.EstimatedTokens()
					report.addTask(reporting.TaskResult{
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
				log.Warnf("task %s abandoned: %s", taskInstance.ID, result.Error)
				if report != nil {
					report.addTask(reporting.TaskResult{
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
				log.Errorf("task %s failed: %s", taskInstance.ID, result.Error)
				if report != nil {
					report.addTask(reporting.TaskResult{
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
		st.RecordProjectRun(projectPath)
	}

	// Summary
	duration := time.Since(start)
	log.InfoCtx("scheduled run complete", map[string]any{
		"duration":  duration.String(),
		"tasks_run": tasksRun,
		"completed": tasksCompleted,
		"failed":    tasksFailed,
		"projects":  len(projects),
	})

	if report != nil {
		report.finalize(cfg, log)
	}

	return nil
}

type tmuxScraper struct{}

// ScrapeClaudeUsage delegates to tmux.ScrapeClaudeUsage.
func (tmuxScraper) ScrapeClaudeUsage(ctx context.Context) (tmux.UsageResult, error) {
	return tmux.ScrapeClaudeUsage(ctx)
}

// ScrapeCodexUsage delegates to tmux.ScrapeCodexUsage.
func (tmuxScraper) ScrapeCodexUsage(ctx context.Context) (tmux.UsageResult, error) {
	return tmux.ScrapeCodexUsage(ctx)
}

func startSnapshotLoop(ctx context.Context, cfg *config.Config, database *db.DB, log *logging.Logger) {
	interval, err := time.ParseDuration(cfg.Budget.SnapshotInterval)
	if err != nil || interval <= 0 {
		if err != nil {
			log.Warnf("invalid snapshot interval %q: %v", cfg.Budget.SnapshotInterval, err)
		}
		return
	}

	go func() {
		takeSnapshot(ctx, cfg, database, log)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				takeSnapshot(ctx, cfg, database, log)
			}
		}
	}()
}

func startSnapshotPruneLoop(ctx context.Context, cfg *config.Config, database *db.DB, log *logging.Logger) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pruneSnapshots(ctx, cfg, database, log)
			}
		}
	}()
}

func takeSnapshot(ctx context.Context, cfg *config.Config, database *db.DB, log *logging.Logger) {
	scraper := snapshots.UsageScraper(nil)
	if cfg.Budget.CalibrateEnabled && strings.ToLower(cfg.Budget.BillingMode) != "api" {
		scraper = tmuxScraper{}
	}

	collector := snapshots.NewCollector(
		database,
		providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude")),
		providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex")),
		scraper,
		weekStartDayFromConfig(cfg),
	)

	if cfg.Providers.Claude.Enabled {
		snapshot, err := collector.TakeSnapshot(ctx, "claude")
		if err != nil {
			log.Warnf("snapshot claude: %v", err)
		} else if snapshot.ScrapedPct != nil {
			log.Infof("snapshot claude: %.1f%%", *snapshot.ScrapedPct)
		} else {
			log.Info("snapshot claude: local-only")
		}
	}

	if cfg.Providers.Codex.Enabled {
		snapshot, err := collector.TakeSnapshot(ctx, "codex")
		if err != nil {
			log.Warnf("snapshot codex: %v", err)
		} else if snapshot.ScrapedPct != nil {
			log.Infof("snapshot codex: %.1f%%", *snapshot.ScrapedPct)
		} else {
			log.Info("snapshot codex: local-only")
		}
	}
}

func pruneSnapshots(ctx context.Context, cfg *config.Config, database *db.DB, log *logging.Logger) {
	collector := snapshots.NewCollector(database, nil, nil, nil, weekStartDayFromConfig(cfg))
	deleted, err := collector.Prune(cfg.Budget.SnapshotRetentionDays)
	if err != nil {
		log.Warnf("snapshot prune: %v", err)
		return
	}
	if deleted > 0 {
		log.Infof("snapshot prune: deleted %d rows", deleted)
	}
}

func weekStartDayFromConfig(cfg *config.Config) time.Weekday {
	if cfg == nil {
		return time.Monday
	}
	switch strings.ToLower(cfg.Budget.WeekStartDay) {
	case "sunday":
		return time.Sunday
	default:
		return time.Monday
	}
}

func runDaemonStop(cmd *cobra.Command, args []string) error {
	running, pid := isDaemonRunning()
	if !running {
		// Check if PID file exists but process is dead
		if _, err := readPidFile(); err == nil {
			_ = removePidFile()
			fmt.Println("daemon not running (stale pid file removed)")
			return nil
		}
		fmt.Println("daemon not running")
		return nil
	}

	// Send SIGTERM to the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM: %w", err)
	}

	fmt.Printf("stopping daemon (pid %d)...\n", pid)

	// Wait for process to exit (with timeout)
	timeout := time.After(10 * time.Second)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-timeout:
			// Force kill if still running
			fmt.Println("daemon did not stop, sending SIGKILL")
			_ = process.Signal(syscall.SIGKILL)
			_ = removePidFile()
			return nil
		case <-tick.C:
			if !isProcessRunning(pid) {
				fmt.Println("daemon stopped")
				_ = removePidFile()
				return nil
			}
		}
	}
}

func runDaemonStatus(cmd *cobra.Command, args []string) error {
	running, pid := isDaemonRunning()

	if !running {
		fmt.Println("Status: not running")
		return nil
	}

	fmt.Printf("Status: running\n")
	fmt.Printf("PID: %d\n", pid)

	// Try to load config and show next run time
	cfg, err := config.Load()
	if err == nil && (cfg.Schedule.Cron != "" || cfg.Schedule.Interval != "") {
		sched, err := scheduler.NewFromConfig(&cfg.Schedule)
		if err == nil {
			// We need to start the scheduler briefly to calculate next run
			// Instead, just show the schedule config
			if cfg.Schedule.Cron != "" {
				fmt.Printf("Schedule: cron %s\n", cfg.Schedule.Cron)
			} else if cfg.Schedule.Interval != "" {
				fmt.Printf("Schedule: every %s\n", cfg.Schedule.Interval)
			}
			if cfg.Schedule.Window != nil {
				fmt.Printf("Window: %s - %s", cfg.Schedule.Window.Start, cfg.Schedule.Window.End)
				if cfg.Schedule.Window.Timezone != "" {
					fmt.Printf(" (%s)", cfg.Schedule.Window.Timezone)
				}
				fmt.Println()
			}
			_ = sched // satisfy compiler
		}
	}

	// Show PID file path for reference
	fmt.Printf("PID file: %s\n", pidFilePath())

	return nil
}
