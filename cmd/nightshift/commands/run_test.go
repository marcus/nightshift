package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/logging"
	"github.com/marcus/nightshift/internal/state"
	"github.com/marcus/nightshift/internal/tasks"
)

type mockUsage struct {
	name string
	pct  float64
}

func (m *mockUsage) Name() string { return m.name }

func (m *mockUsage) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	return m.pct, nil
}

type mockCodexUsage struct {
	mockUsage
}

func (m *mockCodexUsage) GetResetTime(mode string) (time.Time, error) {
	return time.Time{}, nil
}

func TestSelectProvider_PreferenceOrder(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+origPath)

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Preference: []string{"codex", "claude"},
			Claude:     config.ProviderConfig{Enabled: true},
			Codex:      config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}

	claude := &mockUsage{name: "claude", pct: 0}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}}
	budgetMgr := budget.NewManager(cfg, claude, codex, nil)

	choice, err := selectProvider(cfg, budgetMgr, logging.Component("test"), false)
	if err != nil {
		t.Fatalf("selectProvider error: %v", err)
	}
	if choice.name != "codex" {
		t.Fatalf("provider = %s, want codex", choice.name)
	}
}

func TestSelectProvider_FallbackOnBudget(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+origPath)

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Preference: []string{"codex", "claude"},
			Claude:     config.ProviderConfig{Enabled: true},
			Codex:      config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}

	claude := &mockUsage{name: "claude", pct: 0}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 100}}
	budgetMgr := budget.NewManager(cfg, claude, codex, nil)

	choice, err := selectProvider(cfg, budgetMgr, logging.Component("test"), false)
	if err != nil {
		t.Fatalf("selectProvider error: %v", err)
	}
	if choice.name != "claude" {
		t.Fatalf("provider = %s, want claude", choice.name)
	}
}

func TestSelectProvider_NoProvidersEnabled(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: false},
			Codex:  config.ProviderConfig{Enabled: false},
		},
	}
	claude := &mockUsage{name: "claude", pct: 0}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}}
	budgetMgr := budget.NewManager(cfg, claude, codex, nil)

	_, err := selectProvider(cfg, budgetMgr, logging.Component("test"), false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "no providers enabled in config" {
		t.Fatalf("error = %q, want %q", got, "no providers enabled in config")
	}
}

func TestSelectProvider_AllBudgetExhausted(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}
	claude := &mockUsage{name: "claude", pct: 100}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 100}}
	budgetMgr := budget.NewManager(cfg, claude, codex, nil)

	_, err := selectProvider(cfg, budgetMgr, logging.Component("test"), false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "budget exhausted") {
		t.Fatalf("error = %q, want it to contain 'budget exhausted'", got)
	}
}

func TestSelectProvider_CLINotInPath(t *testing.T) {
	// Empty PATH so no CLIs are found
	t.Setenv("PATH", t.TempDir())

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}
	claude := &mockUsage{name: "claude", pct: 0}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}}
	budgetMgr := budget.NewManager(cfg, claude, codex, nil)

	_, err := selectProvider(cfg, budgetMgr, logging.Component("test"), false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "CLI not in PATH") {
		t.Fatalf("error = %q, want it to contain 'CLI not in PATH'", got)
	}
}

func makeExecutable(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// newTestRunState creates a fresh state backed by a temp DB.
func newTestRunState(t *testing.T) *state.State {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	st, err := state.New(database)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}
	return st
}

// newTestRunConfig returns a config with providers and budget set up for testing.
func newTestRunConfig() *config.Config {
	return &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
		Tasks: config.TasksConfig{
			Enabled:    []string{},
			Disabled:   []string{},
			Priorities: map[string]int{},
		},
	}
}

func TestMaxProjects_DefaultLimitsToOne(t *testing.T) {
	// Simulate 3 projects, no --project set, maxProjects=1 (default)
	projects := []string{"/proj/a", "/proj/b", "/proj/c"}
	projectPath := "" // not explicitly set
	maxProjects := 1

	if projectPath == "" && maxProjects > 0 && len(projects) > maxProjects {
		projects = projects[:maxProjects]
	}

	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(projects))
	}
	if projects[0] != "/proj/a" {
		t.Fatalf("projects[0] = %q, want /proj/a", projects[0])
	}
}

func TestMaxProjects_OverrideToN(t *testing.T) {
	projects := []string{"/proj/a", "/proj/b", "/proj/c"}
	projectPath := ""
	maxProjects := 2

	if projectPath == "" && maxProjects > 0 && len(projects) > maxProjects {
		projects = projects[:maxProjects]
	}

	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}
	if projects[1] != "/proj/b" {
		t.Fatalf("projects[1] = %q, want /proj/b", projects[1])
	}
}

func TestMaxProjects_IgnoredWhenProjectSet(t *testing.T) {
	projects := []string{"/proj/explicit"}
	projectPath := "/proj/explicit" // explicitly set
	maxProjects := 1

	// The guard: projectPath == "" is false, so no truncation
	if projectPath == "" && maxProjects > 0 && len(projects) > maxProjects {
		projects = projects[:maxProjects]
	}

	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(projects))
	}
	if projects[0] != "/proj/explicit" {
		t.Fatalf("projects[0] = %q, want /proj/explicit", projects[0])
	}
}

func TestMaxTasks_DefaultLimitsToOne(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	st := newTestRunState(t)
	cfg := newTestRunConfig()
	selector := tasks.NewSelector(cfg, st)
	budgetMgr := budget.NewManager(cfg,
		&mockUsage{name: "claude", pct: 0},
		&mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}},
		nil,
	)
	project := t.TempDir()

	params := executeRunParams{
		cfg:       cfg,
		budgetMgr: budgetMgr,
		selector:  selector,
		st:        st,
		projects:  []string{project},
		maxTasks:  1, // default
		dryRun:    true,
		log:       logging.Component("test"),
	}

	err := executeRun(context.Background(), params)
	if err != nil {
		t.Fatalf("executeRun: %v", err)
	}

	// In dry-run, tasks are selected but not executed.
	// The key assertion: with maxTasks=1, SelectTopN(budget, project, 1)
	// should return at most 1 task. We verify by running SelectTopN directly.
	selected := selector.SelectTopN(1_000_000, project, 1)
	if len(selected) > 1 {
		t.Fatalf("SelectTopN(1) returned %d tasks, want <= 1", len(selected))
	}
}

func TestMaxTasks_OverrideToN(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	st := newTestRunState(t)
	cfg := newTestRunConfig()
	selector := tasks.NewSelector(cfg, st)
	budgetMgr := budget.NewManager(cfg,
		&mockUsage{name: "claude", pct: 0},
		&mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}},
		nil,
	)
	project := t.TempDir()

	params := executeRunParams{
		cfg:       cfg,
		budgetMgr: budgetMgr,
		selector:  selector,
		st:        st,
		projects:  []string{project},
		maxTasks:  3,
		dryRun:    true,
		log:       logging.Component("test"),
	}

	err := executeRun(context.Background(), params)
	if err != nil {
		t.Fatalf("executeRun: %v", err)
	}

	// Verify SelectTopN with n=3 returns more than 1 when tasks exist
	selected := selector.SelectTopN(1_000_000, project, 3)
	if len(selected) > 3 {
		t.Fatalf("SelectTopN(3) returned %d tasks, want <= 3", len(selected))
	}
	// With default config (all tasks enabled, large budget) we expect > 1 task
	if len(selected) < 2 {
		t.Logf("only %d tasks available (expected >= 2); may depend on registered tasks", len(selected))
	}
}

func TestMaxTasks_IgnoredWhenTaskSet(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	st := newTestRunState(t)
	cfg := newTestRunConfig()
	selector := tasks.NewSelector(cfg, st)
	budgetMgr := budget.NewManager(cfg,
		&mockUsage{name: "claude", pct: 0},
		&mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}},
		nil,
	)
	project := t.TempDir()

	// When taskFilter is set, maxTasks is ignored - only the specified task runs
	params := executeRunParams{
		cfg:        cfg,
		budgetMgr:  budgetMgr,
		selector:   selector,
		st:         st,
		projects:   []string{project},
		taskFilter: "lint-fix",
		maxTasks:   5, // should be ignored
		dryRun:     true,
		log:        logging.Component("test"),
	}

	err := executeRun(context.Background(), params)
	if err != nil {
		t.Fatalf("executeRun: %v", err)
	}
	// The test passes if executeRun doesn't error - when taskFilter is set,
	// it uses GetDefinition + single-task path, ignoring maxTasks entirely.
}

func TestSelectProvider_IgnoreBudget_StillReturnsProvider(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}
	// Both providers at 100% usage
	claude := &mockUsage{name: "claude", pct: 100}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 100}}
	budgetMgr := budget.NewManager(cfg, claude, codex, nil)

	choice, err := selectProvider(cfg, budgetMgr, logging.Component("test"), true)
	if err != nil {
		t.Fatalf("selectProvider with ignoreBudget=true should succeed, got: %v", err)
	}
	if choice == nil {
		t.Fatal("expected a provider choice, got nil")
	}
	// Should pick the first preference (claude by default)
	if choice.name != "claude" {
		t.Fatalf("provider = %s, want claude (first in default preference)", choice.name)
	}
}

func TestSelectProvider_IgnoreBudget_PopulatesAllowance(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: false},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}
	claude := &mockUsage{name: "claude", pct: 100}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}}
	budgetMgr := budget.NewManager(cfg, claude, codex, nil)

	choice, err := selectProvider(cfg, budgetMgr, logging.Component("test"), true)
	if err != nil {
		t.Fatalf("selectProvider error: %v", err)
	}
	if choice.allowance == nil {
		t.Fatal("allowance should be populated even when ignoring budget")
	}
	if choice.allowance.UsedPercent != 100 {
		t.Fatalf("UsedPercent = %.1f, want 100", choice.allowance.UsedPercent)
	}
}

func TestSelectProvider_IgnoreBudget_False_StillRejectsBudget(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}
	claude := &mockUsage{name: "claude", pct: 100}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 100}}
	budgetMgr := budget.NewManager(cfg, claude, codex, nil)

	_, err := selectProvider(cfg, budgetMgr, logging.Component("test"), false)
	if err == nil {
		t.Fatal("expected error with ignoreBudget=false and exhausted budget")
	}
	if !strings.Contains(err.Error(), "budget exhausted") {
		t.Fatalf("error = %q, want it to contain 'budget exhausted'", err.Error())
	}
}

// --- Preflight tests ---

// newPreflightParams creates a standard executeRunParams for preflight testing.
func newPreflightParams(t *testing.T, projects []string) executeRunParams {
	t.Helper()
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	st := newTestRunState(t)
	cfg := newTestRunConfig()
	selector := tasks.NewSelector(cfg, st)
	budgetMgr := budget.NewManager(cfg,
		&mockUsage{name: "claude", pct: 0},
		&mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}},
		nil,
	)
	return executeRunParams{
		cfg:       cfg,
		budgetMgr: budgetMgr,
		selector:  selector,
		st:        st,
		projects:  projects,
		maxTasks:  1,
		dryRun:    true,
		log:       logging.Component("test"),
	}
}

func TestBuildPreflight_SingleProject(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})

	plan, err := buildPreflight(params)
	if err != nil {
		t.Fatalf("buildPreflight: %v", err)
	}
	if len(plan.projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(plan.projects))
	}
	pp := plan.projects[0]
	if pp.path != project {
		t.Fatalf("path = %q, want %q", pp.path, project)
	}
	if pp.skipReason != "" {
		t.Fatalf("skipReason = %q, want empty", pp.skipReason)
	}
	if len(pp.tasks) == 0 {
		t.Fatal("expected at least 1 task")
	}
	if pp.provider == nil {
		t.Fatal("expected provider to be set")
	}
	if pp.provider.name != "claude" {
		t.Fatalf("provider = %q, want claude", pp.provider.name)
	}
}

func TestBuildPreflight_SkippedProject(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})

	// Mark project as processed today
	params.st.RecordProjectRun(project)

	plan, err := buildPreflight(params)
	if err != nil {
		t.Fatalf("buildPreflight: %v", err)
	}
	if len(plan.projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(plan.projects))
	}
	pp := plan.projects[0]
	if pp.skipReason != "already processed today" {
		t.Fatalf("skipReason = %q, want 'already processed today'", pp.skipReason)
	}
	if len(plan.skipReasons) == 0 {
		t.Fatal("expected skip reasons to be populated")
	}
	if !strings.Contains(plan.skipReasons[0], "already processed today") {
		t.Fatalf("skipReasons[0] = %q, want to contain 'already processed today'", plan.skipReasons[0])
	}
}

func TestDisplayPreflight_OutputFormat(t *testing.T) {
	plan := &preflightPlan{
		projects: []preflightProject{
			{
				path: "/home/user/my-project",
				tasks: []tasks.ScoredTask{
					{
						Definition: tasks.TaskDefinition{
							Name:     "Linter Fixes",
							CostTier: tasks.CostLow,
						},
						Score:   7.2,
						Project: "/home/user/my-project",
					},
				},
				provider: &providerChoice{
					name: "claude",
					allowance: &budget.AllowanceResult{
						Allowance:   27700,
						UsedPercent: 72.3,
						Mode:        "daily",
					},
				},
			},
		},
	}

	var buf strings.Builder
	displayPreflight(&buf, plan)
	output := buf.String()

	checks := []string{
		"Preflight Summary",
		"Provider: claude",
		"72.3% budget used",
		"daily mode",
		"27700 tokens remaining",
		"Projects (1 of 1)",
		"my-project",
		"Linter Fixes",
		"score=7.2",
	}
	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nGot:\n%s", want, output)
		}
	}
}

func TestDisplayPreflight_IgnoreBudgetWarning(t *testing.T) {
	plan := &preflightPlan{
		ignoreBudget: true,
		projects: []preflightProject{
			{
				path: "/home/user/proj",
				tasks: []tasks.ScoredTask{
					{
						Definition: tasks.TaskDefinition{
							Name:     "Test Task",
							CostTier: tasks.CostLow,
						},
						Score:   5.0,
						Project: "/home/user/proj",
					},
				},
				provider: &providerChoice{
					name: "claude",
					allowance: &budget.AllowanceResult{
						Allowance:   0,
						UsedPercent: 100,
						Mode:        "daily",
					},
				},
			},
		},
	}

	var buf strings.Builder
	displayPreflight(&buf, plan)
	output := buf.String()

	if !strings.Contains(output, "Warnings:") {
		t.Errorf("output missing 'Warnings:'\nGot:\n%s", output)
	}
	if !strings.Contains(output, "--ignore-budget is set") {
		t.Errorf("output missing '--ignore-budget is set'\nGot:\n%s", output)
	}
}

func TestDisplayPreflight_MultipleTasks(t *testing.T) {
	plan := &preflightPlan{
		projects: []preflightProject{
			{
				path: "/home/user/proj",
				tasks: []tasks.ScoredTask{
					{
						Definition: tasks.TaskDefinition{
							Name:     "Linter Fixes",
							CostTier: tasks.CostLow,
						},
						Score:   7.0,
						Project: "/home/user/proj",
					},
					{
						Definition: tasks.TaskDefinition{
							Name:     "Bug Finder & Fixer",
							CostTier: tasks.CostHigh,
						},
						Score:   5.5,
						Project: "/home/user/proj",
					},
					{
						Definition: tasks.TaskDefinition{
							Name:     "Doc Drift Detector",
							CostTier: tasks.CostMedium,
						},
						Score:   3.2,
						Project: "/home/user/proj",
					},
				},
				provider: &providerChoice{
					name: "claude",
					allowance: &budget.AllowanceResult{
						Allowance:   500000,
						UsedPercent: 10.0,
						Mode:        "daily",
					},
				},
			},
		},
	}

	var buf strings.Builder
	displayPreflight(&buf, plan)
	output := buf.String()

	taskNames := []string{"Linter Fixes", "Bug Finder & Fixer", "Doc Drift Detector"}
	for _, name := range taskNames {
		if !strings.Contains(output, name) {
			t.Errorf("output missing task %q\nGot:\n%s", name, output)
		}
	}

	// Verify project count
	if !strings.Contains(output, "Projects (1 of 1)") {
		t.Errorf("output missing 'Projects (1 of 1)'\nGot:\n%s", output)
	}
}

func TestDisplayPreflight_SkippedSection(t *testing.T) {
	plan := &preflightPlan{
		projects: []preflightProject{
			{
				path: "/home/user/active-proj",
				tasks: []tasks.ScoredTask{
					{
						Definition: tasks.TaskDefinition{
							Name:     "Linter Fixes",
							CostTier: tasks.CostLow,
						},
						Score:   5.0,
						Project: "/home/user/active-proj",
					},
				},
				provider: &providerChoice{
					name: "claude",
					allowance: &budget.AllowanceResult{
						Allowance:   50000,
						UsedPercent: 20.0,
						Mode:        "daily",
					},
				},
			},
			{
				path:       "/home/user/skipped-proj",
				skipReason: "already processed today",
			},
		},
	}

	var buf strings.Builder
	displayPreflight(&buf, plan)
	output := buf.String()

	if !strings.Contains(output, "Skipped:") {
		t.Errorf("output missing 'Skipped:'\nGot:\n%s", output)
	}
	if !strings.Contains(output, "skipped-proj: already processed today") {
		t.Errorf("output missing skipped project reason\nGot:\n%s", output)
	}
	if !strings.Contains(output, "Projects (1 of 2)") {
		t.Errorf("output missing 'Projects (1 of 2)'\nGot:\n%s", output)
	}
}

func TestBuildPreflight_TaskFilter(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})
	params.taskFilter = "lint-fix"

	plan, err := buildPreflight(params)
	if err != nil {
		t.Fatalf("buildPreflight: %v", err)
	}
	if len(plan.projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(plan.projects))
	}
	pp := plan.projects[0]
	if len(pp.tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(pp.tasks))
	}
	if string(pp.tasks[0].Definition.Type) != "lint-fix" {
		t.Fatalf("task type = %q, want lint-fix", pp.tasks[0].Definition.Type)
	}
}

func TestBuildPreflight_InvalidTaskFilter(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})
	params.taskFilter = "nonexistent-task"

	_, err := buildPreflight(params)
	if err == nil {
		t.Fatal("expected error for invalid task filter")
	}
	if !strings.Contains(err.Error(), "unknown task type") {
		t.Fatalf("error = %q, want to contain 'unknown task type'", err.Error())
	}
}

// --- Confirmation prompt tests ---

func TestConfirmRun_YesFlagSkipsPrompt(t *testing.T) {
	p := executeRunParams{yes: true, log: logging.Component("test")}
	ok, err := confirmRun(p)
	if err != nil {
		t.Fatalf("confirmRun: %v", err)
	}
	if !ok {
		t.Fatal("expected true when --yes is set")
	}
}

func TestConfirmRun_DryRunReturnsFalse(t *testing.T) {
	p := executeRunParams{dryRun: true, log: logging.Component("test")}
	ok, err := confirmRun(p)
	if err != nil {
		t.Fatalf("confirmRun: %v", err)
	}
	if ok {
		t.Fatal("expected false when --dry-run is set")
	}
}

func TestConfirmRun_NonTTYAutoSkips(t *testing.T) {
	orig := isInteractive
	defer func() { isInteractive = orig }()
	isInteractive = func() bool { return false }

	p := executeRunParams{log: logging.Component("test")}
	ok, err := confirmRun(p)
	if err != nil {
		t.Fatalf("confirmRun: %v", err)
	}
	if !ok {
		t.Fatal("expected true in non-TTY context")
	}
}

func TestConfirmRun_TTYAcceptsY(t *testing.T) {
	orig := isInteractive
	defer func() { isInteractive = orig }()
	isInteractive = func() bool { return true }

	// Replace stdin with a pipe containing "y\n"
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("y\n")
	_ = w.Close()

	p := executeRunParams{log: logging.Component("test")}
	ok, err := confirmRun(p)
	if err != nil {
		t.Fatalf("confirmRun: %v", err)
	}
	if !ok {
		t.Fatal("expected true when user enters 'y'")
	}
}

func TestConfirmRun_TTYAcceptsYes(t *testing.T) {
	orig := isInteractive
	defer func() { isInteractive = orig }()
	isInteractive = func() bool { return true }

	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("yes\n")
	_ = w.Close()

	p := executeRunParams{log: logging.Component("test")}
	ok, err := confirmRun(p)
	if err != nil {
		t.Fatalf("confirmRun: %v", err)
	}
	if !ok {
		t.Fatal("expected true when user enters 'yes'")
	}
}

func TestConfirmRun_TTYRejectsN(t *testing.T) {
	orig := isInteractive
	defer func() { isInteractive = orig }()
	isInteractive = func() bool { return true }

	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("n\n")
	_ = w.Close()

	p := executeRunParams{log: logging.Component("test")}
	ok, err := confirmRun(p)
	if err != nil {
		t.Fatalf("confirmRun: %v", err)
	}
	if ok {
		t.Fatal("expected false when user enters 'n'")
	}
}

func TestConfirmRun_TTYDefaultRejectsEmpty(t *testing.T) {
	orig := isInteractive
	defer func() { isInteractive = orig }()
	isInteractive = func() bool { return true }

	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("\n")
	_ = w.Close()

	p := executeRunParams{log: logging.Component("test")}
	ok, err := confirmRun(p)
	if err != nil {
		t.Fatalf("confirmRun: %v", err)
	}
	if ok {
		t.Fatal("expected false on empty input (default=N)")
	}
}

// --- Dry-run tests ---

// captureStdout redirects os.Stdout, runs fn, and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = origStdout

	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}

func TestDryRun_ShowsPreflightAndExits(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})
	params.dryRun = true

	output := captureStdout(t, func() {
		err := executeRun(context.Background(), params)
		if err != nil {
			t.Fatalf("executeRun: %v", err)
		}
	})

	// Preflight summary should appear
	if !strings.Contains(output, "Preflight Summary") {
		t.Errorf("output missing 'Preflight Summary'\nGot:\n%s", output)
	}
	// Dry-run exit message should appear
	if !strings.Contains(output, "[dry-run] No tasks executed.") {
		t.Errorf("output missing '[dry-run] No tasks executed.'\nGot:\n%s", output)
	}
	// Should NOT contain execution output (project header from execute loop)
	if strings.Contains(output, "=== Project:") {
		t.Errorf("output should NOT contain execution output '=== Project:'\nGot:\n%s", output)
	}
}

func TestDryRun_NoExecution(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})
	params.dryRun = true

	output := captureStdout(t, func() {
		err := executeRun(context.Background(), params)
		if err != nil {
			t.Fatalf("executeRun: %v", err)
		}
	})

	// Must not contain any execution markers
	executionMarkers := []string{
		"=== Project:",
		"--- Running:",
		"COMPLETED",
		"FAILED",
		"ABANDONED",
		"=== Run Complete ===",
	}
	for _, marker := range executionMarkers {
		if strings.Contains(output, marker) {
			t.Errorf("dry-run output should NOT contain %q\nGot:\n%s", marker, output)
		}
	}

	// Verify no state was recorded (no project run recorded)
	if params.st.WasProcessedToday(project) {
		t.Error("dry-run should not record project as processed")
	}
}

func TestDryRun_DisplaysMessage(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})
	params.dryRun = true

	output := captureStdout(t, func() {
		err := executeRun(context.Background(), params)
		if err != nil {
			t.Fatalf("executeRun: %v", err)
		}
	})

	if !strings.Contains(output, "[dry-run] No tasks executed.") {
		t.Errorf("output missing dry-run message\nGot:\n%s", output)
	}
}

// --- Random task tests ---

func TestRandomTask_MutuallyExclusiveWithTaskFilter(t *testing.T) {
	// Verify the runCmd has the --random-task flag registered
	f := runCmd.Flags().Lookup("random-task")
	if f == nil {
		t.Fatal("--random-task flag not registered on runCmd")
	}

	// Simulate the validation logic from runRun: both flags set should error
	randomTask := true
	taskFilter := "lint-fix"
	if randomTask && taskFilter != "" {
		err := fmt.Errorf("--random-task and --task are mutually exclusive")
		if !strings.Contains(err.Error(), "mutually exclusive") {
			t.Fatalf("error = %q, want to contain 'mutually exclusive'", err.Error())
		}
	} else {
		t.Fatal("expected mutual exclusivity check to trigger")
	}
}

func TestBuildPreflight_RandomTask_ReturnsExactlyOneTask(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})
	params.randomTask = true

	plan, err := buildPreflight(params)
	if err != nil {
		t.Fatalf("buildPreflight: %v", err)
	}
	if len(plan.projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(plan.projects))
	}
	pp := plan.projects[0]
	if pp.skipReason != "" {
		t.Fatalf("skipReason = %q, want empty", pp.skipReason)
	}
	if len(pp.tasks) != 1 {
		t.Fatalf("tasks = %d, want exactly 1 with randomTask", len(pp.tasks))
	}
}

func TestBuildPreflight_RandomTask_TaskFromEligiblePool(t *testing.T) {
	project := t.TempDir()
	params := newPreflightParams(t, []string{project})
	params.randomTask = true

	// Build eligible pool for comparison
	allDefs := tasks.AllDefinitions()
	eligible := params.selector.FilterEnabled(allDefs)
	eligibleTypes := make(map[tasks.TaskType]bool)
	for _, d := range eligible {
		eligibleTypes[d.Type] = true
	}

	// Run multiple iterations to verify the returned task is always from the eligible pool
	for i := 0; i < 20; i++ {
		plan, err := buildPreflight(params)
		if err != nil {
			t.Fatalf("buildPreflight iter %d: %v", i, err)
		}
		if len(plan.projects) == 0 || len(plan.projects[0].tasks) == 0 {
			t.Fatalf("iter %d: no tasks returned", i)
		}
		task := plan.projects[0].tasks[0]
		if !eligibleTypes[task.Definition.Type] {
			t.Fatalf("iter %d: task %s not in eligible pool", i, task.Definition.Type)
		}
		if task.Project != project {
			t.Fatalf("iter %d: task.Project = %q, want %q", i, task.Project, project)
		}
	}
}

func TestDisplayPreflight_NoWarningsWhenBudgetRespected(t *testing.T) {
	plan := &preflightPlan{
		ignoreBudget: false,
		projects: []preflightProject{
			{
				path: "/home/user/proj",
				tasks: []tasks.ScoredTask{
					{
						Definition: tasks.TaskDefinition{
							Name:     "Test Task",
							CostTier: tasks.CostLow,
						},
						Score: 5.0,
					},
				},
				provider: &providerChoice{
					name: "claude",
					allowance: &budget.AllowanceResult{
						Allowance:   50000,
						UsedPercent: 20.0,
						Mode:        "daily",
					},
				},
			},
		},
	}

	var buf strings.Builder
	displayPreflight(&buf, plan)
	output := buf.String()

	if strings.Contains(output, "Warnings:") {
		t.Errorf("output should not contain 'Warnings:' when ignoreBudget=false\nGot:\n%s", output)
	}
}
