package deps

import (
	"database/sql"
	"fmt"
	"time"
)

// Store handles SQLite persistence for dependency scan results.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store backed by the given database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// InitTables creates the dep_scans and dep_findings tables if they don't exist.
func (s *Store) InitTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS dep_scans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project TEXT NOT NULL,
			scanned_at DATETIME NOT NULL,
			total_deps INTEGER,
			critical_count INTEGER,
			high_count INTEGER,
			medium_count INTEGER,
			low_count INTEGER
		);

		CREATE TABLE IF NOT EXISTS dep_findings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL REFERENCES dep_scans(id),
			module TEXT NOT NULL,
			version TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			category TEXT NOT NULL,
			detail TEXT NOT NULL,
			cve_id TEXT,
			cvss_score REAL
		);

		CREATE INDEX IF NOT EXISTS idx_dep_scans_project ON dep_scans(project, scanned_at DESC);
		CREATE INDEX IF NOT EXISTS idx_dep_findings_scan ON dep_findings(scan_id);
	`)
	if err != nil {
		return fmt.Errorf("creating dep tables: %w", err)
	}
	return nil
}

// SaveScan persists a scan result and its findings in a transaction.
func (s *Store) SaveScan(result *ScanResult) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	critical, high, medium, low := SummarizeResults(result)

	res, err := tx.Exec(
		`INSERT INTO dep_scans (project, scanned_at, total_deps, critical_count, high_count, medium_count, low_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		result.Project, result.ScannedAt, len(result.Deps), critical, high, medium, low,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting scan: %w", err)
	}

	scanID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting scan id: %w", err)
	}

	for _, f := range result.Findings {
		_, err := tx.Exec(
			`INSERT INTO dep_findings (scan_id, module, version, risk_level, category, detail, cve_id, cvss_score)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			scanID, f.Module, f.Version, f.RiskLevel, f.Category, f.Detail, f.CVEID, f.CVSSScore,
		)
		if err != nil {
			return 0, fmt.Errorf("inserting finding: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing scan: %w", err)
	}

	result.ID = scanID
	return scanID, nil
}

// LatestScan retrieves the most recent scan for a project.
func (s *Store) LatestScan(project string) (*ScanResult, error) {
	row := s.db.QueryRow(
		`SELECT id, project, scanned_at FROM dep_scans WHERE project = ? ORDER BY scanned_at DESC LIMIT 1`,
		project,
	)

	var result ScanResult
	var scannedAt string
	if err := row.Scan(&result.ID, &result.Project, &scannedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying scan: %w", err)
	}
	result.ScannedAt, _ = time.Parse(time.RFC3339, scannedAt)

	findings, err := s.loadFindings(result.ID)
	if err != nil {
		return nil, err
	}
	result.Findings = findings

	return &result, nil
}

// AllLatestScans retrieves the most recent scan for each project.
func (s *Store) AllLatestScans() ([]*ScanResult, error) {
	rows, err := s.db.Query(
		`SELECT id, project, scanned_at FROM dep_scans ds
		 WHERE scanned_at = (SELECT MAX(scanned_at) FROM dep_scans WHERE project = ds.project)
		 ORDER BY project`,
	)
	if err != nil {
		return nil, fmt.Errorf("querying scans: %w", err)
	}
	defer rows.Close()

	var results []*ScanResult
	for rows.Next() {
		var r ScanResult
		var scannedAt string
		if err := rows.Scan(&r.ID, &r.Project, &scannedAt); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		r.ScannedAt, _ = time.Parse(time.RFC3339, scannedAt)

		findings, err := s.loadFindings(r.ID)
		if err != nil {
			return nil, err
		}
		r.Findings = findings
		results = append(results, &r)
	}

	return results, rows.Err()
}

func (s *Store) loadFindings(scanID int64) ([]Finding, error) {
	rows, err := s.db.Query(
		`SELECT module, version, risk_level, category, detail, cve_id, cvss_score
		 FROM dep_findings WHERE scan_id = ? ORDER BY id`,
		scanID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying findings: %w", err)
	}
	defer rows.Close()

	var findings []Finding
	for rows.Next() {
		var f Finding
		var cveID sql.NullString
		var cvss sql.NullFloat64
		if err := rows.Scan(&f.Module, &f.Version, &f.RiskLevel, &f.Category, &f.Detail, &cveID, &cvss); err != nil {
			return nil, fmt.Errorf("scanning finding: %w", err)
		}
		f.CVEID = cveID.String
		f.CVSSScore = cvss.Float64
		findings = append(findings, f)
	}

	return findings, rows.Err()
}
