package tasks

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/state"
)

func setupTestSelector(t *testing.T) (*Selector, *state.State) {
	t.Helper()

	st := newTestState(t)

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled:    []string{}, // Empty means all enabled
			Disabled:   []string{},
			Priorities: map[string]int{},
		},
	}

	return NewSelector(cfg, st), st
}

func newTestState(t *testing.T) *state.State {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	st, err := state.New(database)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	return st
}

func TestScoreTask(t *testing.T) {
	sel, st := setupTestSelector(t)

	project := "/test/project"

	// Base case: no bonuses
	score := sel.ScoreTask(TaskLintFix, project)
	// Staleness bonus for never-run task is 3.0 (from state.StalenessBonus)
	if score < 2.9 || score > 3.1 {
		t.Errorf("expected score ~3.0 for never-run task, got %f", score)
	}

	// Record a recent run to reduce staleness bonus
	st.RecordTaskRun(project, string(TaskLintFix))
	score = sel.ScoreTask(TaskLintFix, project)
	if score > 0.1 {
		t.Errorf("expected score ~0 for just-run task, got %f", score)
	}

	// Add context mention bonus
	sel.SetContextMentions([]string{string(TaskLintFix)})
	score = sel.ScoreTask(TaskLintFix, project)
	if score < 1.9 || score > 2.1 {
		t.Errorf("expected score ~2.0 with context bonus, got %f", score)
	}

	// Add task source bonus
	sel.SetTaskSources([]string{string(TaskLintFix)})
	score = sel.ScoreTask(TaskLintFix, project)
	if score < 4.9 || score > 5.1 {
		t.Errorf("expected score ~5.0 with context+source bonus, got %f", score)
	}
}

func TestScoreTaskWithConfigPriority(t *testing.T) {
	st := newTestState(t)

	// Set priority in config
	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Priorities: map[string]int{
				string(TaskLintFix): 5,
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"
	st.RecordTaskRun(project, string(TaskLintFix)) // Remove staleness bonus

	score := sel.ScoreTask(TaskLintFix, project)
	if score < 4.9 || score > 5.1 {
		t.Errorf("expected score ~5.0 with config priority, got %f", score)
	}
}

func TestFilterEnabled(t *testing.T) {
	st := newTestState(t)

	tests := []struct {
		name     string
		enabled  []string
		disabled []string
		tasks    []TaskDefinition
		wantLen  int
	}{
		{
			name:    "all enabled by default",
			tasks:   []TaskDefinition{{Type: TaskLintFix}, {Type: TaskBugFinder}},
			wantLen: 2,
		},
		{
			name:    "explicit enabled list",
			enabled: []string{string(TaskLintFix)},
			tasks:   []TaskDefinition{{Type: TaskLintFix}, {Type: TaskBugFinder}},
			wantLen: 1,
		},
		{
			name:     "disabled takes precedence",
			disabled: []string{string(TaskLintFix)},
			tasks:    []TaskDefinition{{Type: TaskLintFix}, {Type: TaskBugFinder}},
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Tasks: config.TasksConfig{
					Enabled:  tt.enabled,
					Disabled: tt.disabled,
				},
			}
			sel := NewSelector(cfg, st)
			got := sel.FilterEnabled(tt.tasks)
			if len(got) != tt.wantLen {
				t.Errorf("FilterEnabled() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestFilterByBudget(t *testing.T) {
	sel, _ := setupTestSelector(t)

	tasks := []TaskDefinition{
		{Type: TaskLintFix, CostTier: CostLow},                 // 10-50k
		{Type: TaskBugFinder, CostTier: CostHigh},              // 150-500k
		{Type: TaskMigrationRehearsal, CostTier: CostVeryHigh}, // 500k+
	}

	tests := []struct {
		budget  int64
		wantLen int
	}{
		{budget: 100_000, wantLen: 1},   // Only low cost fits
		{budget: 500_000, wantLen: 2},   // Low and high fit
		{budget: 1_000_000, wantLen: 3}, // All fit
		{budget: 10_000, wantLen: 0},    // None fit
	}

	for _, tt := range tests {
		got := sel.FilterByBudget(tasks, tt.budget)
		if len(got) != tt.wantLen {
			t.Errorf("FilterByBudget(%d) len = %d, want %d", tt.budget, len(got), tt.wantLen)
		}
	}
}

func TestFilterUnassigned(t *testing.T) {
	sel, st := setupTestSelector(t)

	project := "/test/project"
	tasks := []TaskDefinition{
		{Type: TaskLintFix},
		{Type: TaskBugFinder},
		{Type: TaskDeadCode},
	}

	// Mark one as assigned
	taskID := makeTaskID(string(TaskLintFix), project)
	st.MarkAssigned(taskID, project, string(TaskLintFix))

	got := sel.FilterUnassigned(tasks, project)
	if len(got) != 2 {
		t.Errorf("FilterUnassigned() len = %d, want 2", len(got))
	}

	// Verify the assigned one is filtered out
	for _, task := range got {
		if task.Type == TaskLintFix {
			t.Error("FilterUnassigned() did not filter out assigned task")
		}
	}
}

func TestSelectNext(t *testing.T) {
	st := newTestState(t)

	// Enable only specific tasks for predictable testing
	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{
				string(TaskLintFix),
				string(TaskDocsBackfill),
			},
			Priorities: map[string]int{
				string(TaskLintFix):      5,
				string(TaskDocsBackfill): 1,
			},
			Intervals: map[string]string{
				string(TaskLintFix):      "1ns",
				string(TaskDocsBackfill): "1ns",
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"
	// Run both tasks recently to remove staleness bonus
	st.RecordTaskRun(project, string(TaskLintFix))
	st.RecordTaskRun(project, string(TaskDocsBackfill))

	// Select should return highest priority task
	task := sel.SelectNext(100_000, project)
	if task == nil {
		t.Fatal("SelectNext() returned nil")
		return
	}
	if task.Definition.Type != TaskLintFix {
		t.Errorf("SelectNext() = %s, want %s", task.Definition.Type, TaskLintFix)
	}
}

func TestSelectNextNoBudget(t *testing.T) {
	sel, _ := setupTestSelector(t)

	// Budget too low for any task
	task := sel.SelectNext(1000, "/test/project")
	if task != nil {
		t.Errorf("SelectNext() with tiny budget should return nil, got %v", task)
	}
}

func TestSelectAndAssign(t *testing.T) {
	st := newTestState(t)

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{string(TaskLintFix)},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"

	// First selection should work
	task1 := sel.SelectAndAssign(100_000, project)
	if task1 == nil {
		t.Fatal("First SelectAndAssign() returned nil")
		return
	}

	// Verify task is now assigned
	taskID := makeTaskID(string(task1.Definition.Type), project)
	if !st.IsAssigned(taskID) {
		t.Error("Task should be marked as assigned")
	}

	// Second selection should return nil (only task is assigned)
	task2 := sel.SelectAndAssign(100_000, project)
	if task2 != nil {
		t.Errorf("Second SelectAndAssign() should return nil, got %v", task2)
	}
}

func TestSelectTopN(t *testing.T) {
	st := newTestState(t)

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{
				string(TaskLintFix),
				string(TaskDocsBackfill),
				string(TaskDeadCode),
			},
			Priorities: map[string]int{
				string(TaskLintFix):      10,
				string(TaskDocsBackfill): 5,
				string(TaskDeadCode):     1,
			},
			Intervals: map[string]string{
				string(TaskLintFix):      "1ns",
				string(TaskDocsBackfill): "1ns",
				string(TaskDeadCode):     "1ns",
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"
	// Run all tasks recently
	st.RecordTaskRun(project, string(TaskLintFix))
	st.RecordTaskRun(project, string(TaskDocsBackfill))
	st.RecordTaskRun(project, string(TaskDeadCode))

	// Get top 2
	tasks := sel.SelectTopN(1_000_000, project, 2)
	if len(tasks) != 2 {
		t.Fatalf("SelectTopN(2) len = %d, want 2", len(tasks))
	}

	// Verify ordering
	if tasks[0].Definition.Type != TaskLintFix {
		t.Errorf("First task should be lint-fix, got %s", tasks[0].Definition.Type)
	}
	if tasks[1].Definition.Type != TaskDocsBackfill {
		t.Errorf("Second task should be docs-backfill, got %s", tasks[1].Definition.Type)
	}

	// Verify scores are descending
	if tasks[0].Score < tasks[1].Score {
		t.Error("Tasks should be sorted by score descending")
	}
}

func TestStalenessAffectsSelection(t *testing.T) {
	st := newTestState(t)

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{
				string(TaskLintFix),
				string(TaskDocsBackfill),
			},
			Priorities: map[string]int{
				string(TaskLintFix):      1, // Lower base priority
				string(TaskDocsBackfill): 1, // Same base priority
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"

	// Run lint-fix recently, never run docs-backfill
	st.RecordTaskRun(project, string(TaskLintFix))
	// docs-backfill never run -> higher staleness bonus

	task := sel.SelectNext(100_000, project)
	if task == nil {
		t.Fatal("SelectNext() returned nil")
		return
	}
	// docs-backfill should win due to staleness bonus (never run = +3.0)
	if task.Definition.Type != TaskDocsBackfill {
		t.Errorf("Stale task should be selected, got %s", task.Definition.Type)
	}
}

func TestMakeTaskID(t *testing.T) {
	id := makeTaskID("lint-fix", "/test/project")
	want := "lint-fix:/test/project"
	if id != want {
		t.Errorf("makeTaskID() = %s, want %s", id, want)
	}
}

func TestSetContextMentions(t *testing.T) {
	sel, st := setupTestSelector(t)

	project := "/test/project"
	st.RecordTaskRun(project, string(TaskLintFix))

	// Without context mentions
	score1 := sel.ScoreTask(TaskLintFix, project)

	// With context mentions
	sel.SetContextMentions([]string{string(TaskLintFix)})
	score2 := sel.ScoreTask(TaskLintFix, project)

	if score2-score1 < 1.9 || score2-score1 > 2.1 {
		t.Errorf("Context mention should add ~2.0 to score, got diff %f", score2-score1)
	}
}

func TestSetTaskSources(t *testing.T) {
	sel, st := setupTestSelector(t)

	project := "/test/project"
	st.RecordTaskRun(project, string(TaskLintFix))

	// Without task sources
	score1 := sel.ScoreTask(TaskLintFix, project)

	// With task sources
	sel.SetTaskSources([]string{string(TaskLintFix)})
	score2 := sel.ScoreTask(TaskLintFix, project)

	if score2-score1 < 2.9 || score2-score1 > 3.1 {
		t.Errorf("Task source should add ~3.0 to score, got diff %f", score2-score1)
	}
}

func TestFilterByCooldown_OnCooldown(t *testing.T) {
	sel, st := setupTestSelector(t)

	project := "/test/project"

	// Record a recent run - task has 24h default interval so it's on cooldown
	st.RecordTaskRun(project, string(TaskLintFix))

	tasks := []TaskDefinition{
		{Type: TaskLintFix, DefaultInterval: 24 * time.Hour},
		{Type: TaskDocsBackfill, DefaultInterval: 168 * time.Hour},
	}

	got := sel.FilterByCooldown(tasks, project)
	// Only DocsBackfill should pass (never run), LintFix just ran
	if len(got) != 1 {
		t.Fatalf("FilterByCooldown() len = %d, want 1", len(got))
	}
	if got[0].Type != TaskDocsBackfill {
		t.Errorf("FilterByCooldown() kept %s, want %s", got[0].Type, TaskDocsBackfill)
	}
}

func TestFilterByCooldown_NeverRunIncluded(t *testing.T) {
	sel, _ := setupTestSelector(t)

	project := "/test/project"

	tasks := []TaskDefinition{
		{Type: TaskLintFix, DefaultInterval: 24 * time.Hour},
		{Type: TaskBugFinder, DefaultInterval: 72 * time.Hour},
	}

	// Neither task has ever run - both should be included
	got := sel.FilterByCooldown(tasks, project)
	if len(got) != 2 {
		t.Errorf("FilterByCooldown() len = %d, want 2 (never-run tasks included)", len(got))
	}
}

func TestFilterByCooldown_ZeroIntervalIncluded(t *testing.T) {
	sel, st := setupTestSelector(t)

	project := "/test/project"
	st.RecordTaskRun(project, string(TaskLintFix))

	tasks := []TaskDefinition{
		{Type: TaskLintFix, DefaultInterval: 0}, // No interval = always included
	}

	got := sel.FilterByCooldown(tasks, project)
	if len(got) != 1 {
		t.Errorf("FilterByCooldown() len = %d, want 1 (zero interval always included)", len(got))
	}
}

func TestFilterByCooldown_ConfigOverride(t *testing.T) {
	st := newTestState(t)

	// Override lint-fix interval to 1 nanosecond (effectively always off cooldown)
	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Intervals: map[string]string{
				string(TaskLintFix): "1ns",
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"
	st.RecordTaskRun(project, string(TaskLintFix))

	// Small sleep to ensure 1ns has passed
	time.Sleep(time.Microsecond)

	tasks := []TaskDefinition{
		{Type: TaskLintFix, DefaultInterval: 24 * time.Hour},
	}

	got := sel.FilterByCooldown(tasks, project)
	if len(got) != 1 {
		t.Errorf("FilterByCooldown() with 1ns config override: len = %d, want 1", len(got))
	}
}

func TestIsOnCooldown(t *testing.T) {
	sel, st := setupTestSelector(t)
	project := "/test/project"

	// Never run - not on cooldown
	onCooldown, remaining, interval := sel.IsOnCooldown(TaskLintFix, project)
	if onCooldown {
		t.Error("IsOnCooldown() = true for never-run task, want false")
	}
	if remaining != 0 {
		t.Errorf("IsOnCooldown() remaining = %v, want 0 for never-run", remaining)
	}
	if interval != 24*time.Hour {
		t.Errorf("IsOnCooldown() interval = %v, want 24h", interval)
	}

	// Record run - should be on cooldown now
	st.RecordTaskRun(project, string(TaskLintFix))
	onCooldown, remaining, interval = sel.IsOnCooldown(TaskLintFix, project)
	if !onCooldown {
		t.Error("IsOnCooldown() = false for just-run task, want true")
	}
	if remaining <= 0 || remaining > 24*time.Hour {
		t.Errorf("IsOnCooldown() remaining = %v, want >0 and <=24h", remaining)
	}
	if interval != 24*time.Hour {
		t.Errorf("IsOnCooldown() interval = %v, want 24h", interval)
	}
}

func TestIsOnCooldown_UnknownTask(t *testing.T) {
	sel, _ := setupTestSelector(t)
	project := "/test/project"

	onCooldown, remaining, interval := sel.IsOnCooldown("nonexistent-task", project)
	if onCooldown {
		t.Error("IsOnCooldown() = true for unknown task, want false")
	}
	if remaining != 0 || interval != 0 {
		t.Errorf("IsOnCooldown() = (_, %v, %v) for unknown task, want (_, 0, 0)", remaining, interval)
	}
}

func TestSelectNextRespectssCooldown(t *testing.T) {
	st := newTestState(t)

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{
				string(TaskLintFix),
				string(TaskDocsBackfill),
			},
			Priorities: map[string]int{
				string(TaskLintFix):      10,
				string(TaskDocsBackfill): 1,
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"

	// Record lint-fix run (puts it on 24h cooldown)
	st.RecordTaskRun(project, string(TaskLintFix))

	// SelectNext should skip lint-fix (on cooldown) and return docs-backfill
	task := sel.SelectNext(100_000, project)
	if task == nil {
		t.Fatal("SelectNext() returned nil")
		return
	}
	if task.Definition.Type != TaskDocsBackfill {
		t.Errorf("SelectNext() = %s, want %s (lint-fix on cooldown)", task.Definition.Type, TaskDocsBackfill)
	}
}
