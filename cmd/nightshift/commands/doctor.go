package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/calibrator"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/marcus/nightshift/internal/scheduler"
	"github.com/marcus/nightshift/internal/snapshots"
	"github.com/marcus/nightshift/internal/state"
	"github.com/marcus/nightshift/internal/trends"
)

type checkStatus string

const (
	statusOK   checkStatus = "OK"
	statusWarn checkStatus = "WARN"
	statusFail checkStatus = "FAIL"
)

type checkResult struct {
	name   string
	status checkStatus
	detail string
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check Nightshift configuration and environment",
	Long: `Run diagnostics to detect configuration and environment issues.

Checks config, scheduling, providers, database health, and budget readiness.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	// Augment PATH the same way 'run' does so CLI checks are accurate.
	ensurePATH()

	results := make([]checkResult, 0)
	hasFail := false

	add := func(name string, status checkStatus, detail string) {
		if status == statusFail {
			hasFail = true
		}
		results = append(results, checkResult{name: name, status: status, detail: detail})
	}

	cfg, err := config.Load()
	if err != nil {
		add("config", statusFail, err.Error())
		printDoctorResults(results)
		return fmt.Errorf("config load failed")
	}
	add("config", statusOK, "loaded")

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		add("db", statusFail, err.Error())
		printDoctorResults(results)
		return fmt.Errorf("db open failed")
	}
	defer func() { _ = database.Close() }()
	add("db", statusOK, cfg.ExpandedDBPath())

	if _, err := state.New(database); err != nil {
		add("state", statusFail, err.Error())
	} else {
		add("state", statusOK, "ready")
	}

	checkSchedule(cfg, add)
	checkService(add)
	checkDaemon(add)

	checkCLIs(cfg, add)
	claudeProvider, codexProvider, copilotProvider := checkProviders(cfg, add)
	checkBudget(cfg, database, claudeProvider, codexProvider, copilotProvider, add)
	checkSnapshots(cfg, database, add)
	checkTmux(cfg, add)

	printDoctorResults(results)

	if hasFail {
		return fmt.Errorf("doctor found failures")
	}
	return nil
}

func checkSchedule(cfg *config.Config, add func(string, checkStatus, string)) {
	sched, err := scheduler.NewFromConfig(&cfg.Schedule)
	if err != nil {
		if errors.Is(err, scheduler.ErrNoSchedule) {
			add("schedule", statusWarn, "no schedule configured (cron or interval)")
			return
		}
		add("schedule", statusFail, err.Error())
		return
	}
	nextRuns, err := sched.NextRuns(1)
	if err != nil || len(nextRuns) == 0 {
		add("schedule", statusWarn, "unable to compute next run")
		return
	}
	add("schedule", statusOK, fmt.Sprintf("next run %s", nextRuns[0].Format("2006-01-02 15:04")))
}

func checkService(add func(string, checkStatus, string)) {
	service := detectServiceType()
	switch service {
	case ServiceLaunchd:
		home, _ := os.UserHomeDir()
		plistPath := filepath.Join(home, "Library", "LaunchAgents", launchdPlistName)
		if _, err := os.Stat(plistPath); err == nil {
			add("service", statusOK, fmt.Sprintf("launchd installed (%s)", plistPath))
			return
		}
		add("service", statusWarn, "launchd service not installed")
	case ServiceSystemd:
		home, _ := os.UserHomeDir()
		servicePath := filepath.Join(home, ".config", "systemd", "user", systemdServiceName)
		timerPath := filepath.Join(home, ".config", "systemd", "user", systemdTimerName)
		if _, err := os.Stat(servicePath); err == nil {
			add("service", statusOK, fmt.Sprintf("systemd service present (%s)", servicePath))
		} else {
			add("service", statusWarn, "systemd service not installed")
			return
		}
		if _, err := os.Stat(timerPath); err == nil {
			add("service.timer", statusOK, fmt.Sprintf("systemd timer present (%s)", timerPath))
		} else {
			add("service.timer", statusWarn, "systemd timer missing")
		}
	case ServiceCron:
		out, err := exec.Command("crontab", "-l").CombinedOutput()
		if err != nil {
			add("service", statusWarn, "cron not accessible")
			return
		}
		if strings.Contains(string(out), cronMarker) {
			add("service", statusOK, "cron entry installed")
		} else {
			add("service", statusWarn, "cron entry not installed")
		}
	default:
		add("service", statusWarn, fmt.Sprintf("unknown service type (%s)", runtime.GOOS))
	}
}

func checkDaemon(add func(string, checkStatus, string)) {
	pid, err := readPidFile()
	if err != nil {
		add("daemon", statusWarn, "not running (pid file missing)")
		return
	}
	if isProcessRunning(pid) {
		add("daemon", statusOK, fmt.Sprintf("running (pid %d)", pid))
	} else {
		add("daemon", statusWarn, "pid file present but process not running")
	}
}

func checkCLIs(cfg *config.Config, add func(string, checkStatus, string)) {
	if cfg.Providers.Claude.Enabled {
		if path, err := exec.LookPath("claude"); err != nil {
			add("claude.cli", statusFail, "claude not found in PATH")
		} else {
			add("claude.cli", statusOK, path)
		}
	}
	if cfg.Providers.Codex.Enabled {
		if path, err := exec.LookPath("codex"); err != nil {
			add("codex.cli", statusFail, "codex not found in PATH")
		} else {
			add("codex.cli", statusOK, path)
		}
	}
}

func checkProviders(cfg *config.Config, add func(string, checkStatus, string)) (*providers.Claude, *providers.Codex, *providers.Copilot) {
	var claudeProvider *providers.Claude
	var codexProvider *providers.Codex
	var copilotProvider *providers.Copilot

	mode := cfg.Budget.Mode
	if mode == "" {
		mode = config.DefaultBudgetMode
	}

	if cfg.Providers.Claude.Enabled {
		path := cfg.ExpandedProviderPath("claude")
		if _, err := os.Stat(path); err != nil {
			add("claude.data_path", statusFail, fmt.Sprintf("missing %s", path))
		} else {
			add("claude.data_path", statusOK, path)
		}
		claudeProvider = providers.NewClaudeWithPath(path)
		if usage, err := claudeProvider.GetWeeklyUsage(); err == nil {
			add("claude.weekly_tokens", statusOK, fmt.Sprintf("%d tokens", usage))
		}
		if pct, err := claudeProvider.GetUsedPercent(mode, int64(cfg.GetProviderBudget("claude"))); err != nil {
			add("claude.usage", statusFail, err.Error())
		} else {
			add("claude.usage", statusOK, fmt.Sprintf("%.1f%% used (%s, vs config budget)", pct, mode))
		}
	}

	if cfg.Providers.Codex.Enabled {
		path := cfg.ExpandedProviderPath("codex")
		if _, err := os.Stat(path); err != nil {
			add("codex.data_path", statusFail, fmt.Sprintf("missing %s", path))
		} else {
			add("codex.data_path", statusOK, path)
		}
		codexProvider = providers.NewCodexWithPath(path)
		if pct, err := codexProvider.GetUsedPercent(mode, int64(cfg.GetProviderBudget("codex"))); err != nil {
			add("codex.usage", statusFail, err.Error())
		} else {
			add("codex.usage", statusOK, fmt.Sprintf("%.1f%% used (%s)", pct, mode))
		}
	}

	if cfg.Providers.Copilot.Enabled {
		path := cfg.ExpandedProviderPath("copilot")
		if _, err := os.Stat(path); err != nil {
			add("copilot.data_path", statusFail, fmt.Sprintf("missing %s", path))
		} else {
			add("copilot.data_path", statusOK, path)
		}
		copilotProvider = providers.NewCopilotWithPath(path)
		monthlyLimit := int64(cfg.GetProviderBudget("copilot"))
		if pct, err := copilotProvider.GetUsedPercent(mode, monthlyLimit); err != nil {
			add("copilot.usage", statusFail, err.Error())
		} else {
			add("copilot.usage", statusOK, fmt.Sprintf("%.1f%% used (%s)", pct, mode))
		}
	}

	return claudeProvider, codexProvider, copilotProvider
}

func checkBudget(cfg *config.Config, database *db.DB, claudeProvider *providers.Claude, codexProvider *providers.Codex, copilotProvider *providers.Copilot, add func(string, checkStatus, string)) {
	cal := calibrator.New(database, cfg)
	trend := trends.NewAnalyzer(database, cfg.Budget.SnapshotRetentionDays)
	budgetMgr := budget.NewManagerFromProviders(cfg, claudeProvider, codexProvider, copilotProvider, budget.WithBudgetSource(cal), budget.WithTrendAnalyzer(trend))

	if cfg.Providers.Claude.Enabled {
		if allowance, err := budgetMgr.CalculateAllowance("claude"); err != nil {
			add("budget.claude", statusFail, err.Error())
		} else {
			add("budget.claude", statusOK, fmt.Sprintf("%.1f%% used, %d tokens available", allowance.UsedPercent, allowance.Allowance))
		}
	}

	if cfg.Providers.Codex.Enabled {
		if allowance, err := budgetMgr.CalculateAllowance("codex"); err != nil {
			add("budget.codex", statusFail, err.Error())
		} else {
			add("budget.codex", statusOK, fmt.Sprintf("%.1f%% used, %d tokens available", allowance.UsedPercent, allowance.Allowance))
		}
	}
}

func checkSnapshots(cfg *config.Config, database *db.DB, add func(string, checkStatus, string)) {
	collector := snapshots.NewCollector(database, nil, nil, nil, nil, weekStartDayFromConfig(cfg))

	for _, provider := range []string{"claude", "codex", "copilot"} {
		if provider == "claude" && !cfg.Providers.Claude.Enabled {
			continue
		}
		if provider == "codex" && !cfg.Providers.Codex.Enabled {
			continue
		}
		if provider == "copilot" && !cfg.Providers.Copilot.Enabled {
			continue
		}
		latest, err := collector.GetLatest(provider, 1)
		if err != nil {
			add(fmt.Sprintf("snapshots.%s", provider), statusWarn, err.Error())
			continue
		}
		if len(latest) == 0 {
			add(fmt.Sprintf("snapshots.%s", provider), statusWarn, "no snapshots yet")
			continue
		}
		age := time.Since(latest[0].Timestamp)
		msg := fmt.Sprintf("last snapshot %s ago", age.Truncate(time.Minute))
		if latest[0].ScrapedPct == nil && cfg.Budget.CalibrateEnabled && strings.ToLower(cfg.Budget.BillingMode) != "api" {
			add(fmt.Sprintf("snapshots.%s", provider), statusWarn, msg+" (local-only)")
			continue
		}
		add(fmt.Sprintf("snapshots.%s", provider), statusOK, msg)
	}
}

func checkTmux(cfg *config.Config, add func(string, checkStatus, string)) {
	if !cfg.Budget.CalibrateEnabled || strings.EqualFold(cfg.Budget.BillingMode, "api") {
		add("tmux", statusOK, "not required")
		return
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		add("tmux", statusWarn, "tmux not found; calibration will be local-only")
		return
	}
	add("tmux", statusOK, "available")
}

func printDoctorResults(results []checkResult) {
	fmt.Println("Nightshift doctor")
	fmt.Println("=================")
	for _, result := range results {
		fmt.Printf("[%s] %-20s %s\n", result.status, result.name, result.detail)
	}
	fmt.Println()
}
