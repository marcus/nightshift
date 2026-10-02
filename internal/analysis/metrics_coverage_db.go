package analysis

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// MetricsCoverageResult represents a stored metrics coverage analysis result.
type MetricsCoverageResult struct {
	ID          int64             `json:"id"`
	Component   string            `json:"component"`
	Timestamp   time.Time         `json:"timestamp"`
	Summary     *CoverageSummary  `json:"summary"`
	Packages    []PackageCoverage `json:"packages"`
	CoveragePct float64           `json:"coverage_pct"`
	GapCount    int               `json:"gap_count"`
	ReportPath  string            `json:"report_path,omitempty"`
}

// Store saves a metrics coverage result to the database.
func (r *MetricsCoverageResult) Store(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	summaryJSON, err := json.Marshal(r.Summary)
	if err != nil {
		return fmt.Errorf("marshaling summary: %w", err)
	}

	packagesJSON, err := json.Marshal(r.Packages)
	if err != nil {
		return fmt.Errorf("marshaling packages: %w", err)
	}

	query := `
		INSERT INTO metrics_coverage_results (component, timestamp, summary, packages, coverage_pct, gap_count, report_path)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	res, err := db.Exec(query,
		r.Component,
		r.Timestamp,
		string(summaryJSON),
		string(packagesJSON),
		r.CoveragePct,
		r.GapCount,
		r.ReportPath,
	)
	if err != nil {
		return fmt.Errorf("inserting metrics coverage result: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting insert id: %w", err)
	}
	r.ID = id

	return nil
}

// LoadLatestCoverage loads the most recent metrics coverage result for a component.
func LoadLatestCoverage(db *sql.DB, component string) (*MetricsCoverageResult, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	query := `
		SELECT id, component, timestamp, summary, packages, coverage_pct, gap_count, report_path
		FROM metrics_coverage_results
		WHERE component = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	row := db.QueryRow(query, component)

	result := &MetricsCoverageResult{}
	var summaryJSON, packagesJSON string

	err := row.Scan(
		&result.ID,
		&result.Component,
		&result.Timestamp,
		&summaryJSON,
		&packagesJSON,
		&result.CoveragePct,
		&result.GapCount,
		&result.ReportPath,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying metrics coverage result: %w", err)
	}

	if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
		return nil, fmt.Errorf("unmarshaling summary: %w", err)
	}

	if err := json.Unmarshal([]byte(packagesJSON), &result.Packages); err != nil {
		return nil, fmt.Errorf("unmarshaling packages: %w", err)
	}

	return result, nil
}

// LoadAllCoverage loads all metrics coverage results for a component, optionally filtered by date.
func LoadAllCoverage(db *sql.DB, component string, since time.Time) ([]MetricsCoverageResult, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	query := `
		SELECT id, component, timestamp, summary, packages, coverage_pct, gap_count, report_path
		FROM metrics_coverage_results
		WHERE component = ?
	`
	args := []any{component}

	if !since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, since)
	}

	query += " ORDER BY timestamp DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying metrics coverage results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []MetricsCoverageResult
	for rows.Next() {
		result := MetricsCoverageResult{}
		var summaryJSON, packagesJSON string

		err := rows.Scan(
			&result.ID,
			&result.Component,
			&result.Timestamp,
			&summaryJSON,
			&packagesJSON,
			&result.CoveragePct,
			&result.GapCount,
			&result.ReportPath,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning metrics coverage result: %w", err)
		}

		if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
			return nil, fmt.Errorf("unmarshaling summary: %w", err)
		}

		if err := json.Unmarshal([]byte(packagesJSON), &result.Packages); err != nil {
			return nil, fmt.Errorf("unmarshaling packages: %w", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating metrics coverage results: %w", err)
	}

	return results, nil
}
