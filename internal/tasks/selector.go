// Package tasks provides task selection and priority scoring.
package tasks

import (
	"math/rand/v2"
	"sort"
	"time"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/state"
)

// Selector handles task selection based on priority scoring.
type Selector struct {
	cfg                *config.Config
	state              *state.State
	contextMentions    map[string]bool // Tasks mentioned in claude.md/agents.md
	taskSources        map[string]bool // Tasks from td/github issues
	simulatedCooldowns map[string]bool // task:project keys simulated as on cooldown (for preview)
}

// NewSelector creates a new task selector.
func NewSelector(cfg *config.Config, st *state.State) *Selector {
	return &Selector{
		cfg:             cfg,
		state:           st,
		contextMentions: make(map[string]bool),
		taskSources:     make(map[string]bool),
	}
}

// ScoredTask represents a task with its computed score.
type ScoredTask struct {
	Definition TaskDefinition
	Score      float64
	Project    string
}

// SetContextMentions sets tasks mentioned in claude.md/agents.md.
// These tasks get a +2 context bonus.
func (s *Selector) SetContextMentions(mentions []string) {
	s.contextMentions = make(map[string]bool, len(mentions))
	for _, m := range mentions {
		s.contextMentions[m] = true
	}
}

// SetTaskSources sets tasks from td/github issues.
// These tasks get a +3 task source bonus.
func (s *Selector) SetTaskSources(sources []string) {
	s.taskSources = make(map[string]bool, len(sources))
	for _, src := range sources {
		s.taskSources[src] = true
	}
}

// ScoreTask calculates the priority score for a task.
// Formula: base_priority + staleness_bonus + context_bonus + task_source_bonus
func (s *Selector) ScoreTask(taskType TaskType, project string) float64 {
	var score float64

	// Base priority from config
	score += float64(s.cfg.GetTaskPriority(string(taskType)))

	// Staleness bonus: days since last run * 0.1
	score += s.state.StalenessBonus(project, string(taskType))

	// Context bonus: +2 if mentioned in claude.md/agents.md
	if s.contextMentions[string(taskType)] {
		score += 2.0
	}

	// Task source bonus: +3 if from td/github issues
	if s.taskSources[string(taskType)] {
		score += 3.0
	}

	return score
}

// FilterEnabled returns only enabled tasks from the given list.
// Tasks with DisabledByDefault require explicit inclusion in tasks.enabled.
func (s *Selector) FilterEnabled(tasks []TaskDefinition) []TaskDefinition {
	filtered := make([]TaskDefinition, 0, len(tasks))
	for _, t := range tasks {
		if t.DisabledByDefault && !s.cfg.IsTaskExplicitlyEnabled(string(t.Type)) {
			continue
		}
		if s.cfg.IsTaskEnabled(string(t.Type)) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// FilterByBudget returns tasks that fit within the given budget.
// Budget is in tokens.
func (s *Selector) FilterByBudget(tasks []TaskDefinition, budget int64) []TaskDefinition {
	filtered := make([]TaskDefinition, 0, len(tasks))
	for _, t := range tasks {
		_, max := t.EstimatedTokens()
		if int64(max) <= budget {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// IsAssigned returns whether a task ID is currently assigned.
func (s *Selector) IsAssigned(taskID string) bool {
	if s.state == nil {
		return false
	}
	return s.state.IsAssigned(taskID)
}

// FilterUnassigned returns tasks that are not currently assigned.
func (s *Selector) FilterUnassigned(tasks []TaskDefinition, project string) []TaskDefinition {
	filtered := make([]TaskDefinition, 0, len(tasks))
	for _, t := range tasks {
		taskID := makeTaskID(string(t.Type), project)
		if !s.state.IsAssigned(taskID) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// effectiveInterval returns the interval for a task, preferring config override.
func (s *Selector) effectiveInterval(def TaskDefinition) time.Duration {
	if d := s.cfg.GetTaskInterval(string(def.Type)); d > 0 {
		return d
	}
	return def.DefaultInterval
}

// FilterByCooldown returns tasks whose cooldown period has elapsed.
// Tasks that have never run or have no interval (<=0) are always included.
// Also excludes tasks with simulated cooldowns (used by preview).
func (s *Selector) FilterByCooldown(tasks []TaskDefinition, project string) []TaskDefinition {
	filtered := make([]TaskDefinition, 0, len(tasks))
	for _, t := range tasks {
		// Check simulated cooldowns first (preview mode)
		if s.simulatedCooldowns != nil {
			key := makeTaskID(string(t.Type), project)
			if s.simulatedCooldowns[key] {
				continue
			}
		}
		interval := s.effectiveInterval(t)
		if interval <= 0 {
			filtered = append(filtered, t)
			continue
		}
		lastRun := s.state.LastTaskRun(project, string(t.Type))
		if lastRun.IsZero() || time.Since(lastRun) >= interval {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// AddSimulatedCooldown marks a task+project as on cooldown for preview simulation.
// Subsequent calls to FilterByCooldown will exclude this combination.
func (s *Selector) AddSimulatedCooldown(taskType string, project string) {
	if s.simulatedCooldowns == nil {
		s.simulatedCooldowns = make(map[string]bool)
	}
	s.simulatedCooldowns[makeTaskID(taskType, project)] = true
}

// ClearSimulatedCooldowns removes all simulated cooldowns.
func (s *Selector) ClearSimulatedCooldowns() {
	s.simulatedCooldowns = nil
}

// HasSimulatedCooldown returns whether a task+project has a simulated cooldown.
func (s *Selector) HasSimulatedCooldown(taskType string, project string) bool {
	if s.simulatedCooldowns == nil {
		return false
	}
	return s.simulatedCooldowns[makeTaskID(taskType, project)]
}

// IsOnCooldown returns whether a task is on cooldown for a project.
// Returns (onCooldown, remainingTime, totalInterval).
func (s *Selector) IsOnCooldown(taskType TaskType, project string) (bool, time.Duration, time.Duration) {
	def, err := GetDefinition(taskType)
	if err != nil {
		return false, 0, 0
	}
	interval := s.effectiveInterval(def)
	if interval <= 0 {
		return false, 0, 0
	}
	lastRun := s.state.LastTaskRun(project, string(taskType))
	if lastRun.IsZero() {
		return false, 0, interval
	}
	elapsed := time.Since(lastRun)
	if elapsed >= interval {
		return false, 0, interval
	}
	return true, interval - elapsed, interval
}

// SelectNext returns the best task for the given budget and project.
// Returns nil if no suitable task is found.
func (s *Selector) SelectNext(budget int64, project string) *ScoredTask {
	// Start with all task definitions
	tasks := AllDefinitions()

	// Filter: enabled tasks only
	tasks = s.FilterEnabled(tasks)

	// Filter: tasks within budget estimate
	tasks = s.FilterByBudget(tasks, budget)

	// Filter: unassigned tasks
	tasks = s.FilterUnassigned(tasks, project)

	// Filter: tasks not on cooldown
	tasks = s.FilterByCooldown(tasks, project)

	if len(tasks) == 0 {
		return nil
	}

	// Score each task
	scored := make([]ScoredTask, len(tasks))
	for i, t := range tasks {
		scored[i] = ScoredTask{
			Definition: t,
			Score:      s.ScoreTask(t.Type, project),
			Project:    project,
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Select top task that fits remaining budget
	for _, st := range scored {
		_, max := st.Definition.EstimatedTokens()
		if int64(max) <= budget {
			return &st
		}
	}

	return nil
}

// SelectAndAssign selects the best task and marks it as assigned.
// Returns the selected task or nil if none available.
func (s *Selector) SelectAndAssign(budget int64, project string) *ScoredTask {
	task := s.SelectNext(budget, project)
	if task == nil {
		return nil
	}

	// Mark as assigned to prevent duplicate selection
	taskID := makeTaskID(string(task.Definition.Type), project)
	s.state.MarkAssigned(taskID, project, string(task.Definition.Type))

	return task
}

// makeTaskID creates a unique task ID from type and project.
func makeTaskID(taskType, project string) string {
	return taskType + ":" + project
}

// SelectTopN returns the top N tasks by score that fit within budget.
func (s *Selector) SelectTopN(budget int64, project string, n int) []ScoredTask {
	// Start with all task definitions
	tasks := AllDefinitions()

	// Filter: enabled tasks only
	tasks = s.FilterEnabled(tasks)

	// Filter: tasks within budget estimate
	tasks = s.FilterByBudget(tasks, budget)

	// Filter: unassigned tasks
	tasks = s.FilterUnassigned(tasks, project)

	// Filter: tasks not on cooldown
	tasks = s.FilterByCooldown(tasks, project)

	if len(tasks) == 0 {
		return nil
	}

	// Score each task
	scored := make([]ScoredTask, len(tasks))
	for i, t := range tasks {
		scored[i] = ScoredTask{
			Definition: t,
			Score:      s.ScoreTask(t.Type, project),
			Project:    project,
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Return top N
	if n > len(scored) {
		n = len(scored)
	}
	return scored[:n]
}

// SelectRandom returns a random task from the eligible pool.
// It applies the same filter pipeline as SelectNext but picks randomly
// instead of by highest score. The returned ScoredTask still has an
// accurate Score for display purposes. Returns nil if no task is eligible.
func (s *Selector) SelectRandom(budget int64, project string) *ScoredTask {
	// Start with all task definitions
	tasks := AllDefinitions()

	// Filter: enabled tasks only
	tasks = s.FilterEnabled(tasks)

	// Filter: tasks within budget estimate
	tasks = s.FilterByBudget(tasks, budget)

	// Filter: unassigned tasks
	tasks = s.FilterUnassigned(tasks, project)

	// Filter: tasks not on cooldown
	tasks = s.FilterByCooldown(tasks, project)

	if len(tasks) == 0 {
		return nil
	}

	// Score each task
	scored := make([]ScoredTask, len(tasks))
	for i, t := range tasks {
		scored[i] = ScoredTask{
			Definition: t,
			Score:      s.ScoreTask(t.Type, project),
			Project:    project,
		}
	}

	// Pick a random task from the eligible pool
	pick := scored[rand.IntN(len(scored))]
	return &pick
}
