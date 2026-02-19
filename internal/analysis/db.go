package analysis

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// BusFactorResult represents a stored bus-factor analysis result.
type BusFactorResult struct {
	ID           int64             `json:"id"`
	Component    string            `json:"component"`
	Timestamp    time.Time         `json:"timestamp"`
	Metrics      *OwnershipMetrics `json:"metrics"`
	Contributors []CommitAuthor    `json:"contributors"`
	RiskLevel    string            `json:"risk_level"`
	ReportPath   string            `json:"report_path,omitempty"`
}

// Store saves a bus-factor result to the database.
func (result *BusFactorResult) Store(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	// Serialize metrics and contributors as JSON
	metricsJSON, err := json.Marshal(result.Metrics)
	if err != nil {
		return fmt.Errorf("marshaling metrics: %w", err)
	}

	contributorsJSON, err := json.Marshal(result.Contributors)
	if err != nil {
		return fmt.Errorf("marshaling contributors: %w", err)
	}

	query := `
		INSERT INTO bus_factor_results (component, timestamp, metrics, contributors, risk_level, report_path)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	res, err := db.Exec(query,
		result.Component,
		result.Timestamp,
		string(metricsJSON),
		string(contributorsJSON),
		result.RiskLevel,
		result.ReportPath,
	)
	if err != nil {
		return fmt.Errorf("inserting bus factor result: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting insert id: %w", err)
	}
	result.ID = id

	return nil
}

// LoadLatest loads the most recent bus-factor result for a component.
func LoadLatest(db *sql.DB, component string) (*BusFactorResult, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	query := `
		SELECT id, component, timestamp, metrics, contributors, risk_level, report_path
		FROM bus_factor_results
		WHERE component = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	row := db.QueryRow(query, component)

	result := &BusFactorResult{}
	var metricsJSON, contributorsJSON string

	err := row.Scan(
		&result.ID,
		&result.Component,
		&result.Timestamp,
		&metricsJSON,
		&contributorsJSON,
		&result.RiskLevel,
		&result.ReportPath,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying bus factor result: %w", err)
	}

	// Deserialize metrics and contributors
	if err := json.Unmarshal([]byte(metricsJSON), &result.Metrics); err != nil {
		return nil, fmt.Errorf("unmarshaling metrics: %w", err)
	}

	if err := json.Unmarshal([]byte(contributorsJSON), &result.Contributors); err != nil {
		return nil, fmt.Errorf("unmarshaling contributors: %w", err)
	}

	return result, nil
}

// LoadAll loads all bus-factor results for a component, optionally filtered by date range.
func LoadAll(db *sql.DB, component string, since time.Time) ([]BusFactorResult, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	query := `
		SELECT id, component, timestamp, metrics, contributors, risk_level, report_path
		FROM bus_factor_results
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
		return nil, fmt.Errorf("querying bus factor results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []BusFactorResult
	for rows.Next() {
		result := BusFactorResult{}
		var metricsJSON, contributorsJSON string

		err := rows.Scan(
			&result.ID,
			&result.Component,
			&result.Timestamp,
			&metricsJSON,
			&contributorsJSON,
			&result.RiskLevel,
			&result.ReportPath,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning bus factor result: %w", err)
		}

		if err := json.Unmarshal([]byte(metricsJSON), &result.Metrics); err != nil {
			return nil, fmt.Errorf("unmarshaling metrics: %w", err)
		}

		if err := json.Unmarshal([]byte(contributorsJSON), &result.Contributors); err != nil {
			return nil, fmt.Errorf("unmarshaling contributors: %w", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating bus factor results: %w", err)
	}

	return results, nil
}
