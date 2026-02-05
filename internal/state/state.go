// Package state manages persistent state for nightshift runs.
// Tracks run history per project and task to support staleness calculation,
// duplicate run prevention, and task assignment tracking.
package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/marcus/nightshift/internal/db"
)

// State manages persistent nightshift state.
type State struct {
	mu sync.RWMutex
	db *db.DB
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

// New creates a new State manager backed by the provided database.
func New(database *db.DB) (*State, error) {
	if database == nil {
		return nil, errors.New("db is nil")
	}
	return &State{db: database}, nil
}

// RecordProjectRun marks a project as having been processed.
func (s *State) RecordProjectRun(projectPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectPath = normalizePath(projectPath)
	now := time.Now()
	_, err := s.db.SQL().Exec(
		`INSERT INTO projects (path, last_run, run_count) VALUES (?, ?, 1)
		 ON CONFLICT(path) DO UPDATE SET last_run = excluded.last_run, run_count = projects.run_count + 1`,
		projectPath,
		now,
	)
	if err != nil {
		log.Printf("state: record project run: %v", err)
	}
}

// RecordTaskRun marks a specific task type as having run for a project.
func (s *State) RecordTaskRun(projectPath, taskType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectPath = normalizePath(projectPath)
	now := time.Now()
	_, err := s.db.SQL().Exec(
		`INSERT INTO task_history (project_path, task_type, last_run) VALUES (?, ?, ?)
		 ON CONFLICT(project_path, task_type) DO UPDATE SET last_run = excluded.last_run`,
		projectPath,
		taskType,
		now,
	)
	if err != nil {
		log.Printf("state: record task run: %v", err)
	}
}

// WasProcessedToday returns true if the project was already processed today.
func (s *State) WasProcessedToday(projectPath string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectPath = normalizePath(projectPath)
	row := s.db.SQL().QueryRow(`SELECT last_run FROM projects WHERE path = ?`, projectPath)
	var lastRun sql.NullTime
	if err := row.Scan(&lastRun); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("state: query last_run: %v", err)
		}
		return false
	}
	if !lastRun.Valid {
		return false
	}
	return isSameDay(lastRun.Time, time.Now())
}

// LastProjectRun returns when a project was last processed.
func (s *State) LastProjectRun(projectPath string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectPath = normalizePath(projectPath)
	row := s.db.SQL().QueryRow(`SELECT last_run FROM projects WHERE path = ?`, projectPath)
	var lastRun sql.NullTime
	if err := row.Scan(&lastRun); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("state: query last_run: %v", err)
		}
		return time.Time{}
	}
	if !lastRun.Valid {
		return time.Time{}
	}
	return lastRun.Time
}

// LastTaskRun returns when a task type was last run for a project.
func (s *State) LastTaskRun(projectPath, taskType string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectPath = normalizePath(projectPath)
	row := s.db.SQL().QueryRow(`SELECT last_run FROM task_history WHERE project_path = ? AND task_type = ?`, projectPath, taskType)
	var lastRun time.Time
	if err := row.Scan(&lastRun); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("state: query task last_run: %v", err)
		}
		return time.Time{}
	}
	return lastRun
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

	_, err := s.db.SQL().Exec(
		`INSERT INTO assigned_tasks (task_id, project, task_type, assigned_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(task_id) DO UPDATE SET project = excluded.project, task_type = excluded.task_type, assigned_at = excluded.assigned_at`,
		taskID,
		normalizePath(project),
		taskType,
		time.Now(),
	)
	if err != nil {
		log.Printf("state: mark assigned: %v", err)
	}
}

// IsAssigned checks if a task is currently assigned.
func (s *State) IsAssigned(taskID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.SQL().QueryRow(`SELECT 1 FROM assigned_tasks WHERE task_id = ?`, taskID)
	var one int
	if err := row.Scan(&one); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("state: check assigned: %v", err)
		}
		return false
	}
	return true
}

// GetAssigned returns the assigned task info, if any.
func (s *State) GetAssigned(taskID string) (AssignedTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.SQL().QueryRow(`SELECT task_id, project, task_type, assigned_at FROM assigned_tasks WHERE task_id = ?`, taskID)
	var task AssignedTask
	if err := row.Scan(&task.TaskID, &task.Project, &task.TaskType, &task.AssignedAt); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("state: get assigned: %v", err)
		}
		return AssignedTask{}, false
	}
	return task, true
}

// ClearAssigned removes a task from the assigned list.
func (s *State) ClearAssigned(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.SQL().Exec(`DELETE FROM assigned_tasks WHERE task_id = ?`, taskID); err != nil {
		log.Printf("state: clear assigned: %v", err)
	}
}

// ClearAllAssigned removes all assigned tasks (e.g., on daemon restart).
func (s *State) ClearAllAssigned() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.SQL().Exec(`DELETE FROM assigned_tasks`); err != nil {
		log.Printf("state: clear all assigned: %v", err)
	}
}

// ClearStaleAssignments removes assignments older than the given duration.
func (s *State) ClearStaleAssignments(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	result, err := s.db.SQL().Exec(`DELETE FROM assigned_tasks WHERE assigned_at < ?`, cutoff)
	if err != nil {
		log.Printf("state: clear stale assignments: %v", err)
		return 0
	}
	cleared, err := result.RowsAffected()
	if err != nil {
		log.Printf("state: rows affected: %v", err)
		return 0
	}
	return int(cleared)
}

// ListAssigned returns all currently assigned tasks.
func (s *State) ListAssigned() []AssignedTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.SQL().Query(`SELECT task_id, project, task_type, assigned_at FROM assigned_tasks ORDER BY assigned_at ASC`)
	if err != nil {
		log.Printf("state: list assigned: %v", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	tasks := make([]AssignedTask, 0)
	for rows.Next() {
		var task AssignedTask
		if err := rows.Scan(&task.TaskID, &task.Project, &task.TaskType, &task.AssignedAt); err != nil {
			log.Printf("state: scan assigned: %v", err)
			return tasks
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		log.Printf("state: list assigned rows: %v", err)
	}
	return tasks
}

// GetProjectState returns the state for a project (or nil if not tracked).
func (s *State) GetProjectState(projectPath string) *ProjectState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectPath = normalizePath(projectPath)
	row := s.db.SQL().QueryRow(`SELECT path, last_run, run_count FROM projects WHERE path = ?`, projectPath)
	var path string
	var lastRun sql.NullTime
	var runCount int
	if err := row.Scan(&path, &lastRun, &runCount); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("state: get project state: %v", err)
		}
		return nil
	}

	state := &ProjectState{
		Path:        path,
		LastRun:     time.Time{},
		TaskHistory: make(map[string]time.Time),
		RunCount:    runCount,
	}
	if lastRun.Valid {
		state.LastRun = lastRun.Time
	}

	rows, err := s.db.SQL().Query(`SELECT task_type, last_run FROM task_history WHERE project_path = ?`, projectPath)
	if err != nil {
		log.Printf("state: load task history: %v", err)
		return state
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var taskType string
		var last time.Time
		if err := rows.Scan(&taskType, &last); err != nil {
			log.Printf("state: scan task history: %v", err)
			return state
		}
		state.TaskHistory[taskType] = last
	}
	if err := rows.Err(); err != nil {
		log.Printf("state: task history rows: %v", err)
	}

	return state
}

// ProjectCount returns the number of tracked projects.
func (s *State) ProjectCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.SQL().QueryRow(`SELECT COUNT(*) FROM projects`)
	var count int
	if err := row.Scan(&count); err != nil {
		log.Printf("state: project count: %v", err)
		return 0
	}
	return count
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

	if record.ID == "" {
		record.ID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	if record.StartTime.IsZero() {
		record.StartTime = time.Now()
	}

	tasks := record.Tasks
	if tasks == nil {
		tasks = []string{}
	}
	tasksJSON, err := json.Marshal(tasks)
	if err != nil {
		log.Printf("state: marshal tasks: %v", err)
		return
	}

	var endTime sql.NullTime
	if !record.EndTime.IsZero() {
		endTime = sql.NullTime{Time: record.EndTime, Valid: true}
	}

	tx, err := s.db.SQL().Begin()
	if err != nil {
		log.Printf("state: begin run insert: %v", err)
		return
	}

	_, err = tx.Exec(
		`INSERT INTO run_history (id, start_time, end_time, project, tasks, tokens_used, status, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.StartTime,
		endTime,
		record.Project,
		string(tasksJSON),
		record.TokensUsed,
		record.Status,
		record.Error,
	)
	if err != nil {
		_ = tx.Rollback()
		log.Printf("state: insert run_history: %v", err)
		return
	}

	if _, err := tx.Exec(`DELETE FROM run_history WHERE id NOT IN (SELECT id FROM run_history ORDER BY start_time DESC LIMIT 100)`); err != nil {
		_ = tx.Rollback()
		log.Printf("state: prune run_history: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("state: commit run_history: %v", err)
	}
}

// GetRunHistory returns the last N run records (most recent first).
func (s *State) GetRunHistory(n int) []RunRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := n
	if limit <= 0 {
		limit = -1
	}

	rows, err := s.db.SQL().Query(
		`SELECT id, start_time, end_time, project, tasks, tokens_used, status, error
		 FROM run_history
		 ORDER BY start_time DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		log.Printf("state: get run history: %v", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	result := make([]RunRecord, 0)
	for rows.Next() {
		var record RunRecord
		var tasksJSON string
		var endTime sql.NullTime
		if err := rows.Scan(&record.ID, &record.StartTime, &endTime, &record.Project, &tasksJSON, &record.TokensUsed, &record.Status, &record.Error); err != nil {
			log.Printf("state: scan run history: %v", err)
			return result
		}
		if endTime.Valid {
			record.EndTime = endTime.Time
		}
		if tasksJSON != "" {
			if err := json.Unmarshal([]byte(tasksJSON), &record.Tasks); err != nil {
				log.Printf("state: unmarshal tasks: %v", err)
				return result
			}
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		log.Printf("state: run history rows: %v", err)
	}
	return result
}

// GetTodayRuns returns all runs from today.
func (s *State) GetTodayRuns() []RunRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	rows, err := s.db.SQL().Query(
		`SELECT id, start_time, end_time, project, tasks, tokens_used, status, error
		 FROM run_history
		 WHERE start_time >= ? AND start_time < ?
		 ORDER BY start_time DESC`,
		startOfDay,
		endOfDay,
	)
	if err != nil {
		log.Printf("state: get today runs: %v", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	result := make([]RunRecord, 0)
	for rows.Next() {
		var record RunRecord
		var tasksJSON string
		var endTime sql.NullTime
		if err := rows.Scan(&record.ID, &record.StartTime, &endTime, &record.Project, &tasksJSON, &record.TokensUsed, &record.Status, &record.Error); err != nil {
			log.Printf("state: scan today runs: %v", err)
			return result
		}
		if endTime.Valid {
			record.EndTime = endTime.Time
		}
		if tasksJSON != "" {
			if err := json.Unmarshal([]byte(tasksJSON), &record.Tasks); err != nil {
				log.Printf("state: unmarshal tasks: %v", err)
				return result
			}
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		log.Printf("state: today runs rows: %v", err)
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

		switch run.Status {
		case "success":
			summary.SuccessfulRuns++
		case "failed":
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
