package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportLegacyStateSuccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	legacyPath := filepath.Join(home, ".local", "share", "nightshift", "state", legacyStateFile)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
		t.Fatalf("mkdir legacy dir: %v", err)
	}

	legacy := legacyStateData{
		Version: 1,
		Projects: map[string]*legacyProjectState{
			"/tmp/project": {
				Path:        "/tmp/project",
				LastRun:     time.Now().Add(-2 * time.Hour),
				RunCount:    2,
				TaskHistory: map[string]time.Time{"lint": time.Now().Add(-1 * time.Hour)},
			},
		},
		Assigned: map[string]legacyAssignedTask{
			"task-1": {
				TaskID:     "task-1",
				Project:    "/tmp/project",
				TaskType:   "lint",
				AssignedAt: time.Now().Add(-30 * time.Minute),
			},
		},
		RunHistory: []legacyRunRecord{
			{
				ID:         "run-1",
				StartTime:  time.Now().Add(-90 * time.Minute),
				EndTime:    time.Now().Add(-60 * time.Minute),
				Project:    "/tmp/project",
				Tasks:      []string{"lint"},
				TokensUsed: 1200,
				Status:     "success",
			},
		},
		LastUpdate: time.Now(),
	}

	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	if err := os.WriteFile(legacyPath, payload, 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	if err := importLegacyStateFromPath(database.SQL(), legacyPath); err != nil {
		t.Fatalf("import legacy: %v", err)
	}

	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy file to be renamed, got err=%v", err)
	}
	if _, err := os.Stat(legacyPath + ".migrated"); err != nil {
		t.Fatalf("expected migrated file: %v", err)
	}

	assertRowCount(t, database, "projects", 1)
	assertRowCount(t, database, "task_history", 1)
	assertRowCount(t, database, "assigned_tasks", 1)
	assertRowCount(t, database, "run_history", 1)
}

func TestImportLegacyStateSkipsWhenDataExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	if _, err := database.SQL().Exec(`INSERT INTO projects (path, last_run, run_count) VALUES (?, ?, ?)`, "/tmp/project", time.Now(), 1); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	legacyPath := filepath.Join(home, ".local", "share", "nightshift", "state", legacyStateFile)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
		t.Fatalf("mkdir legacy dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte(`{"version":1}`), 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	if err := importLegacyStateFromPath(database.SQL(), legacyPath); err != nil {
		t.Fatalf("import legacy: %v", err)
	}

	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("expected legacy file to remain, got err=%v", err)
	}
	if _, err := os.Stat(legacyPath + ".migrated"); !os.IsNotExist(err) {
		t.Fatalf("expected no migrated file, got err=%v", err)
	}

	assertRowCount(t, database, "projects", 1)
}

func TestImportLegacyStateParseFailureKeepsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	legacyPath := filepath.Join(home, ".local", "share", "nightshift", "state", legacyStateFile)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
		t.Fatalf("mkdir legacy dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte(`{"version":`), 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	if err := importLegacyStateFromPath(database.SQL(), legacyPath); err == nil {
		t.Fatalf("expected import error")
	}

	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("expected legacy file to remain, got err=%v", err)
	}
}

func assertRowCount(t *testing.T, database *DB, table string, expected int) {
	t.Helper()

	row := database.SQL().QueryRow(`SELECT COUNT(*) FROM ` + table)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %d rows in %s, got %d", expected, table, count)
	}
}
