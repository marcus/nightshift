// Package state manages persistent state for nightshift runs.
// Tracks run history per project and task to support staleness calculation,
// duplicate run prevention, and task assignment tracking.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// State manages persistent nightshift state.
type State struct {
	mu       sync.RWMutex
	filePath string
	data     *StateData
}

// StateData is the serialized state structure.
type StateData struct {
	Version    int                      `json:"version"`
	Projects   map[string]*ProjectState `json:"projects"`
	Assigned   map[string]AssignedTask  `json:"assigned"`
	RunHistory []RunRecord              `json:"run_history"`
	LastUpdate time.Time                `json:"last_update"`
}

// RunRecord represents a single nightshift run for history tracking.
type RunRecord struct {
	ID         string    `json:"id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Project    string    `json:"project"`
	Tasks      []string  `json:"tasks"`
	TokensUsed int       `json:"tokens_used"`
	Status     string    `json:"status"` // success, failed, partial
	Error      string    `json:"error,omitempty"`
}

// ProjectState tracks state for a single project.
type ProjectState struct {
	Path        string               `json:"path"`
	LastRun     time.Time            `json:"last_run"`
	TaskHistory map[string]time.Time `json:"task_history"` // task type -> last run
	RunCount    int                  `json:"run_count"`
}

// AssignedTask represents a task currently assigned/in-progress.
type AssignedTask struct {
	TaskID     string    `json:"task_id"`
	Project    string    `json:"project"`
	TaskType   string    `json:"task_type"`
	AssignedAt time.Time `json:"assigned_at"`
}

const (
	stateVersion = 1
	stateFile    = "state.json"
)

// DefaultStatePath returns the default state directory path.
func DefaultStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "state")
}

// New creates a new State manager, loading existing state if present.
func New(stateDir string) (*State, error) {
	if stateDir == "" {
		stateDir = DefaultStatePath()
	}
	stateDir = expandPath(stateDir)

	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("creating state dir: %w", err)
	}

	s := &State{
		filePath: filepath.Join(stateDir, stateFile),
		data:     newStateData(),
	}

	// Load existing state
	if err := s.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	return s, nil
}

// newStateData creates an empty StateData.
func newStateData() *StateData {
	return &StateData{
		Version:    stateVersion,
		Projects:   make(map[string]*ProjectState),
		Assigned:   make(map[string]AssignedTask),
		RunHistory: make([]RunRecord, 0),
	}
}

// Load reads state from disk.
func (s *State) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.ErrNotExist
		}
		return err
	}

	var loaded StateData
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parsing state: %w", err)
	}

	// Initialize nil maps/slices
	if loaded.Projects == nil {
		loaded.Projects = make(map[string]*ProjectState)
	}
	if loaded.Assigned == nil {
		loaded.Assigned = make(map[string]AssignedTask)
	}
	if loaded.RunHistory == nil {
		loaded.RunHistory = make([]RunRecord, 0)
	}
	for _, p := range loaded.Projects {
		if p.TaskHistory == nil {
			p.TaskHistory = make(map[string]time.Time)
		}
	}

	s.data = &loaded
	return nil
}

// Save writes state to disk.
func (s *State) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.LastUpdate = time.Now()

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Write atomically via temp file
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}

	if err := os.Rename(tmpFile, s.filePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("renaming state file: %w", err)
	}

	return nil
}

// RecordProjectRun marks a project as having been processed.
func (s *State) RecordProjectRun(projectPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectPath = normalizePath(projectPath)
	ps := s.getOrCreateProject(projectPath)
	ps.LastRun = time.Now()
	ps.RunCount++
}

// RecordTaskRun marks a specific task type as having run for a project.
func (s *State) RecordTaskRun(projectPath, taskType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectPath = normalizePath(projectPath)
	ps := s.getOrCreateProject(projectPath)
	ps.TaskHistory[taskType] = time.Now()
}

// WasProcessedToday returns true if the project was already processed today.
func (s *State) WasProcessedToday(projectPath string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectPath = normalizePath(projectPath)
	ps, ok := s.data.Projects[projectPath]
	if !ok {
		return false
	}

	return isSameDay(ps.LastRun, time.Now())
}

// LastProjectRun returns when a project was last processed.
func (s *State) LastProjectRun(projectPath string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectPath = normalizePath(projectPath)
	ps, ok := s.data.Projects[projectPath]
	if !ok {
		return time.Time{}
	}
	return ps.LastRun
}

// LastTaskRun returns when a task type was last run for a project.
func (s *State) LastTaskRun(projectPath, taskType string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectPath = normalizePath(projectPath)
	ps, ok := s.data.Projects[projectPath]
	if !ok {
		return time.Time{}
	}
	return ps.TaskHistory[taskType]
}

// DaysSinceLastRun returns days since a task type was last run for a project.
// Returns -1 if the task has never run (treated as maximally stale).
func (s *State) DaysSinceLastRun(projectPath, taskType string) int {
	lastRun := s.LastTaskRun(projectPath, taskType)
	if lastRun.IsZero() {
		return -1 // Never run
	}

	return int(time.Since(lastRun).Hours() / 24)
}

// StalenessBonus calculates the staleness bonus for task selection.
// Formula: days since last run * 0.1 (capped at reasonable max).
// Tasks that have never run get a high bonus.
func (s *State) StalenessBonus(projectPath, taskType string) float64 {
	days := s.DaysSinceLastRun(projectPath, taskType)
	if days < 0 {
		// Never run - give high staleness bonus
		return 3.0
	}
	// Cap at 30 days to prevent runaway bonuses
	if days > 30 {
		days = 30
	}
	return float64(days) * 0.1
}

// MarkAssigned marks a task as assigned/in-progress.
func (s *State) MarkAssigned(taskID, project, taskType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Assigned[taskID] = AssignedTask{
		TaskID:     taskID,
		Project:    normalizePath(project),
		TaskType:   taskType,
		AssignedAt: time.Now(),
	}
}

// IsAssigned checks if a task is currently assigned.
func (s *State) IsAssigned(taskID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.data.Assigned[taskID]
	return ok
}

// GetAssigned returns the assigned task info, if any.
func (s *State) GetAssigned(taskID string) (AssignedTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.data.Assigned[taskID]
	return task, ok
}

// ClearAssigned removes a task from the assigned list.
func (s *State) ClearAssigned(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data.Assigned, taskID)
}

// ClearAllAssigned removes all assigned tasks (e.g., on daemon restart).
func (s *State) ClearAllAssigned() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Assigned = make(map[string]AssignedTask)
}

// ClearStaleAssignments removes assignments older than the given duration.
func (s *State) ClearStaleAssignments(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleared := 0

	for id, task := range s.data.Assigned {
		if task.AssignedAt.Before(cutoff) {
			delete(s.data.Assigned, id)
			cleared++
		}
	}
	return cleared
}

// ListAssigned returns all currently assigned tasks.
func (s *State) ListAssigned() []AssignedTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]AssignedTask, 0, len(s.data.Assigned))
	for _, t := range s.data.Assigned {
		tasks = append(tasks, t)
	}
	return tasks
}

// GetProjectState returns the state for a project (or nil if not tracked).
func (s *State) GetProjectState(projectPath string) *ProjectState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectPath = normalizePath(projectPath)
	return s.data.Projects[projectPath]
}

// ProjectCount returns the number of tracked projects.
func (s *State) ProjectCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data.Projects)
}

// getOrCreateProject returns the project state, creating if needed.
// Must be called with lock held.
func (s *State) getOrCreateProject(projectPath string) *ProjectState {
	ps, ok := s.data.Projects[projectPath]
	if !ok {
		ps = &ProjectState{
			Path:        projectPath,
			TaskHistory: make(map[string]time.Time),
		}
		s.data.Projects[projectPath] = ps
	}
	return ps
}

// isSameDay checks if two times are on the same calendar day.
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// normalizePath normalizes a project path for consistent lookups.
func normalizePath(path string) string {
	path = expandPath(path)
	path = filepath.Clean(path)
	return path
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// AddRunRecord adds a run record to history.
func (s *State) AddRunRecord(record RunRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if empty
	if record.ID == "" {
		record.ID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	}

	s.data.RunHistory = append(s.data.RunHistory, record)

	// Keep only the last 100 runs
	if len(s.data.RunHistory) > 100 {
		s.data.RunHistory = s.data.RunHistory[len(s.data.RunHistory)-100:]
	}
}

// GetRunHistory returns the last N run records (most recent first).
func (s *State) GetRunHistory(n int) []RunRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || n > len(s.data.RunHistory) {
		n = len(s.data.RunHistory)
	}

	// Return in reverse order (most recent first)
	result := make([]RunRecord, n)
	for i := 0; i < n; i++ {
		result[i] = s.data.RunHistory[len(s.data.RunHistory)-1-i]
	}
	return result
}

// GetTodayRuns returns all runs from today.
func (s *State) GetTodayRuns() []RunRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	var result []RunRecord

	for i := len(s.data.RunHistory) - 1; i >= 0; i-- {
		run := s.data.RunHistory[i]
		if isSameDay(run.StartTime, now) {
			result = append(result, run)
		}
	}

	return result
}

// TodaySummary returns a summary of today's activity.
type TodaySummary struct {
	TotalRuns      int
	SuccessfulRuns int
	FailedRuns     int
	TotalTokens    int
	TaskCounts     map[string]int
	Projects       []string
}

// GetTodaySummary returns a summary of today's activity.
func (s *State) GetTodaySummary() TodaySummary {
	runs := s.GetTodayRuns()

	summary := TodaySummary{
		TaskCounts: make(map[string]int),
	}

	projectSet := make(map[string]bool)

	for _, run := range runs {
		summary.TotalRuns++
		summary.TotalTokens += run.TokensUsed

		if run.Status == "success" {
			summary.SuccessfulRuns++
		} else if run.Status == "failed" {
			summary.FailedRuns++
		}

		for _, task := range run.Tasks {
			summary.TaskCounts[task]++
		}

		if run.Project != "" && !projectSet[run.Project] {
			projectSet[run.Project] = true
			summary.Projects = append(summary.Projects, run.Project)
		}
	}

	return summary
}
