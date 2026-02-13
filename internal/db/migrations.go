package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
)

// Migration represents a single schema change.
type Migration struct {
	Version     int
	Description string
	SQL         string
}

var migrations = []Migration{
	{
		Version:     1,
		Description: "initial schema: projects, task_history, assigned_tasks, run_history, snapshots",
		SQL:         migration001SQL,
	},
	{
		Version:     2,
		Description: "add reset time columns to snapshots",
		SQL:         migration002SQL,
	},
	{
		Version:     3,
		Description: "add provider column to run_history",
		SQL:         migration003SQL,
	},
	{
		Version:     4,
		Description: "add bus_factor_results table for code ownership analysis",
		SQL:         migration004SQL,
	},
	{
		Version:     5,
		Description: "add branch column to run_history",
		SQL:         migration005SQL,
	},
}

const migration002SQL = `
ALTER TABLE snapshots ADD COLUMN session_reset_time TEXT;
ALTER TABLE snapshots ADD COLUMN weekly_reset_time TEXT;
`

const migration003SQL = `
ALTER TABLE run_history ADD COLUMN provider TEXT NOT NULL DEFAULT '';
`

const migration004SQL = `
CREATE TABLE IF NOT EXISTS bus_factor_results (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    component       TEXT NOT NULL,
    timestamp       DATETIME NOT NULL,
    metrics         TEXT NOT NULL,
    contributors    TEXT NOT NULL,
    risk_level      TEXT NOT NULL,
    report_path     TEXT
);

CREATE INDEX IF NOT EXISTS idx_bus_factor_component_time ON bus_factor_results(component, timestamp DESC);
`

const migration001SQL = `
CREATE TABLE projects (
    path        TEXT PRIMARY KEY,
    last_run    DATETIME,
    run_count   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE task_history (
    project_path TEXT NOT NULL,
    task_type    TEXT NOT NULL,
    last_run     DATETIME NOT NULL,
    PRIMARY KEY (project_path, task_type)
);

CREATE TABLE assigned_tasks (
    task_id     TEXT PRIMARY KEY,
    project     TEXT NOT NULL,
    task_type   TEXT NOT NULL,
    assigned_at DATETIME NOT NULL
);

CREATE TABLE run_history (
    id          TEXT PRIMARY KEY,
    start_time  DATETIME NOT NULL,
    end_time    DATETIME,
    project     TEXT NOT NULL,
    tasks       TEXT NOT NULL,
    tokens_used INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL,
    error       TEXT
);

CREATE TABLE snapshots (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    provider        TEXT NOT NULL,
    timestamp       DATETIME NOT NULL,
    week_start      DATE NOT NULL,
    local_tokens    INTEGER NOT NULL,
    local_daily     INTEGER NOT NULL DEFAULT 0,
    scraped_pct     REAL,
    inferred_budget INTEGER,
    day_of_week     INTEGER NOT NULL,
    hour_of_day     INTEGER NOT NULL,
    week_number     INTEGER NOT NULL,
    year            INTEGER NOT NULL
);

CREATE INDEX idx_snapshots_provider_time ON snapshots(provider, timestamp DESC);
CREATE INDEX idx_snapshots_provider_week ON snapshots(provider, week_start);
CREATE INDEX idx_run_history_time ON run_history(start_time DESC);
`

const migration005SQL = `
ALTER TABLE run_history ADD COLUMN branch TEXT NOT NULL DEFAULT '';
`

// Migrate runs all pending migrations inside transactions.
func Migrate(db *sql.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at DATETIME)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	currentVersion, err := CurrentVersion(db)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec(migration.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_version (version, applied_at) VALUES (?, CURRENT_TIMESTAMP)`, migration.Version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", migration.Version, err)
		}

		log.Printf("db: applied migration %d: %s", migration.Version, migration.Description)
		currentVersion = migration.Version
	}

	return nil
}

// CurrentVersion returns the current schema version (0 if no migrations applied).
func CurrentVersion(db *sql.DB) (int, error) {
	if db == nil {
		return 0, errors.New("db is nil")
	}

	row := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	var version int
	if err := row.Scan(&version); err != nil {
		return 0, fmt.Errorf("query schema_version: %w", err)
	}
	return version, nil
}
