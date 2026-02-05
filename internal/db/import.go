package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const legacyStateFile = "state.json"

// importLegacyState loads legacy state.json into SQLite if present.
func importLegacyState(db *sql.DB) error {
	return importLegacyStateFromPath(db, legacyStatePath())
}

func legacyStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "state", legacyStateFile)
}

func importLegacyStateFromPath(db *sql.DB, path string) error {
	if db == nil {
		return errors.New("db is nil")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat legacy state: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("legacy state path is directory: %s", path)
	}

	hasRows, err := dbHasStateRows(db)
	if err != nil {
		return err
	}
	if hasRows {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read legacy state: %w", err)
	}

	var legacy legacyStateData
	if err := json.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("parse legacy state: %w", err)
	}

	projectCount, runCount, err := importLegacyStateData(db, legacy)
	if err != nil {
		return err
	}

	if err := os.Rename(path, path+".migrated"); err != nil {
		return fmt.Errorf("rename legacy state: %w", err)
	}

	log.Printf("migrated %d projects, %d run records from state.json", projectCount, runCount)
	return nil
}

func importLegacyStateData(db *sql.DB, legacy legacyStateData) (int, int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("begin legacy import: %w", err)
	}

	projectStmt, err := tx.Prepare(`INSERT INTO projects (path, last_run, run_count) VALUES (?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return 0, 0, fmt.Errorf("prepare projects insert: %w", err)
	}
	defer func() { _ = projectStmt.Close() }()

	taskStmt, err := tx.Prepare(`INSERT INTO task_history (project_path, task_type, last_run) VALUES (?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return 0, 0, fmt.Errorf("prepare task_history insert: %w", err)
	}
	defer func() { _ = taskStmt.Close() }()

	assignedStmt, err := tx.Prepare(`INSERT INTO assigned_tasks (task_id, project, task_type, assigned_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return 0, 0, fmt.Errorf("prepare assigned_tasks insert: %w", err)
	}
	defer func() { _ = assignedStmt.Close() }()

	runStmt, err := tx.Prepare(`INSERT INTO run_history (id, start_time, end_time, project, tasks, tokens_used, status, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return 0, 0, fmt.Errorf("prepare run_history insert: %w", err)
	}
	defer func() { _ = runStmt.Close() }()

	projectCount := 0
	runCount := 0

	for path, project := range legacy.Projects {
		projectPath := path
		if project != nil && project.Path != "" {
			projectPath = project.Path
		}
		lastRun := time.Time{}
		runCountValue := 0
		if project != nil {
			lastRun = project.LastRun
			runCountValue = project.RunCount
		}

		if _, err := projectStmt.Exec(projectPath, lastRun, runCountValue); err != nil {
			_ = tx.Rollback()
			return 0, 0, fmt.Errorf("insert project %s: %w", projectPath, err)
		}
		projectCount++

		if project == nil || project.TaskHistory == nil {
			continue
		}
		for taskType, lastRun := range project.TaskHistory {
			if _, err := taskStmt.Exec(projectPath, taskType, lastRun); err != nil {
				_ = tx.Rollback()
				return 0, 0, fmt.Errorf("insert task_history %s/%s: %w", projectPath, taskType, err)
			}
		}
	}

	for taskID, assigned := range legacy.Assigned {
		resolvedID := assigned.TaskID
		if resolvedID == "" {
			resolvedID = taskID
		}

		if _, err := assignedStmt.Exec(resolvedID, assigned.Project, assigned.TaskType, assigned.AssignedAt); err != nil {
			_ = tx.Rollback()
			return 0, 0, fmt.Errorf("insert assigned task %s: %w", resolvedID, err)
		}
	}

	for _, run := range legacy.RunHistory {
		tasks := run.Tasks
		if tasks == nil {
			tasks = []string{}
		}

		tasksJSON, err := json.Marshal(tasks)
		if err != nil {
			_ = tx.Rollback()
			return 0, 0, fmt.Errorf("marshal tasks for run %s: %w", run.ID, err)
		}

		var endTime sql.NullTime
		if !run.EndTime.IsZero() {
			endTime = sql.NullTime{Time: run.EndTime, Valid: true}
		}

		if _, err := runStmt.Exec(run.ID, run.StartTime, endTime, run.Project, string(tasksJSON), run.TokensUsed, run.Status, run.Error); err != nil {
			_ = tx.Rollback()
			return 0, 0, fmt.Errorf("insert run_history %s: %w", run.ID, err)
		}
		runCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit legacy import: %w", err)
	}

	return projectCount, runCount, nil
}

func dbHasStateRows(db *sql.DB) (bool, error) {
	tables := []string{"projects", "task_history", "assigned_tasks", "run_history"}
	for _, table := range tables {
		row := db.QueryRow(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", table))
		var one int
		err := row.Scan(&one)
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("check %s rows: %w", table, err)
		}
	}
	return false, nil
}

type legacyStateData struct {
	Version    int                            `json:"version"`
	Projects   map[string]*legacyProjectState `json:"projects"`
	Assigned   map[string]legacyAssignedTask  `json:"assigned"`
	RunHistory []legacyRunRecord              `json:"run_history"`
	LastUpdate time.Time                      `json:"last_update"`
}

type legacyRunRecord struct {
	ID         string    `json:"id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Project    string    `json:"project"`
	Tasks      []string  `json:"tasks"`
	TokensUsed int       `json:"tokens_used"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
}

type legacyProjectState struct {
	Path        string               `json:"path"`
	LastRun     time.Time            `json:"last_run"`
	TaskHistory map[string]time.Time `json:"task_history"`
	RunCount    int                  `json:"run_count"`
}

type legacyAssignedTask struct {
	TaskID     string    `json:"task_id"`
	Project    string    `json:"project"`
	TaskType   string    `json:"task_type"`
	AssignedAt time.Time `json:"assigned_at"`
}
