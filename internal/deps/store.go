package deps

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Migration006SQL creates dep_scans and dep_findings tables.
const Migration006SQL = `
CREATE TABLE IF NOT EXISTS dep_scans (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    project     TEXT NOT NULL,
    timestamp   DATETIME NOT NULL,
    duration_ms INTEGER NOT NULL,
    total_deps  INTEGER NOT NULL,
    direct_deps INTEGER NOT NULL,
    summary     TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dep_findings (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    scan_id  INTEGER NOT NULL REFERENCES dep_scans(id),
    module   TEXT NOT NULL,
    version  TEXT NOT NULL,
    category TEXT NOT NULL,
    risk     TEXT NOT NULL,
    title    TEXT NOT NULL,
    description TEXT NOT NULL,
    reference   TEXT
);

CREATE INDEX IF NOT EXISTS idx_dep_scans_project ON dep_scans(project, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_dep_findings_scan ON dep_findings(scan_id);
CREATE INDEX IF NOT EXISTS idx_dep_findings_risk ON dep_findings(risk, module);
`

// Store persists scan results to SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// SaveScanResult persists a complete scan result, returning the scan ID.
func (s *Store) SaveScanResult(result *ScanResult) (int64, error) {
	summaryJSON, err := json.Marshal(result.Summary)
	if err != nil {
		return 0, fmt.Errorf("marshaling summary: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		`INSERT INTO dep_scans (project, timestamp, duration_ms, total_deps, direct_deps, summary)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		result.Project,
		result.Timestamp,
		parseDurationMs(result.Duration),
		result.TotalDeps,
		result.DirectDeps,
		string(summaryJSON),
	)
	if err != nil {
		return 0, fmt.Errorf("inserting scan: %w", err)
	}

	scanID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting scan ID: %w", err)
	}

	for _, f := range result.Findings {
		_, err := tx.Exec(
			`INSERT INTO dep_findings (scan_id, module, version, category, risk, title, description, reference)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			scanID, f.Module, f.Version, f.Category, f.Risk, f.Title, f.Description, f.Reference,
		)
		if err != nil {
			return 0, fmt.Errorf("inserting finding: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return scanID, nil
}

func parseDurationMs(d string) int64 {
	dur, err := time.ParseDuration(d)
	if err != nil {
		return 0
	}
	return dur.Milliseconds()
}
