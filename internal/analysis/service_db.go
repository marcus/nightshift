package analysis

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ServiceAdvisorResult represents a stored service boundary analysis result.
type ServiceAdvisorResult struct {
	ID              int64             `json:"id"`
	Component       string            `json:"component"`
	Timestamp       time.Time         `json:"timestamp"`
	Packages        []PackageAnalysis `json:"packages"`
	Metrics         []ServiceMetrics  `json:"metrics"`
	Recommendations []string          `json:"recommendations"`
	ReportPath      string            `json:"report_path,omitempty"`
}

// StoreServiceResult saves a service advisor result to the database.
func (result *ServiceAdvisorResult) Store(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	packagesJSON, err := json.Marshal(result.Packages)
	if err != nil {
		return fmt.Errorf("marshaling packages: %w", err)
	}

	metricsJSON, err := json.Marshal(result.Metrics)
	if err != nil {
		return fmt.Errorf("marshaling metrics: %w", err)
	}

	recsJSON, err := json.Marshal(result.Recommendations)
	if err != nil {
		return fmt.Errorf("marshaling recommendations: %w", err)
	}

	query := `
		INSERT INTO service_advisor_results (component, timestamp, packages, metrics, recommendations, report_path)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	res, err := db.Exec(query,
		result.Component,
		result.Timestamp,
		string(packagesJSON),
		string(metricsJSON),
		string(recsJSON),
		result.ReportPath,
	)
	if err != nil {
		return fmt.Errorf("inserting service advisor result: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting insert id: %w", err)
	}
	result.ID = id

	return nil
}

// LoadLatestServiceResult loads the most recent service advisor result for a component.
func LoadLatestServiceResult(db *sql.DB, component string) (*ServiceAdvisorResult, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	query := `
		SELECT id, component, timestamp, packages, metrics, recommendations, report_path
		FROM service_advisor_results
		WHERE component = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	row := db.QueryRow(query, component)

	result := &ServiceAdvisorResult{}
	var packagesJSON, metricsJSON, recsJSON string

	err := row.Scan(
		&result.ID,
		&result.Component,
		&result.Timestamp,
		&packagesJSON,
		&metricsJSON,
		&recsJSON,
		&result.ReportPath,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying service advisor result: %w", err)
	}

	if err := json.Unmarshal([]byte(packagesJSON), &result.Packages); err != nil {
		return nil, fmt.Errorf("unmarshaling packages: %w", err)
	}

	if err := json.Unmarshal([]byte(metricsJSON), &result.Metrics); err != nil {
		return nil, fmt.Errorf("unmarshaling metrics: %w", err)
	}

	if err := json.Unmarshal([]byte(recsJSON), &result.Recommendations); err != nil {
		return nil, fmt.Errorf("unmarshaling recommendations: %w", err)
	}

	return result, nil
}

// LoadAllServiceResults loads all service advisor results for a component.
func LoadAllServiceResults(db *sql.DB, component string, since time.Time) ([]ServiceAdvisorResult, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	query := `
		SELECT id, component, timestamp, packages, metrics, recommendations, report_path
		FROM service_advisor_results
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
		return nil, fmt.Errorf("querying service advisor results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []ServiceAdvisorResult
	for rows.Next() {
		result := ServiceAdvisorResult{}
		var packagesJSON, metricsJSON, recsJSON string

		err := rows.Scan(
			&result.ID,
			&result.Component,
			&result.Timestamp,
			&packagesJSON,
			&metricsJSON,
			&recsJSON,
			&result.ReportPath,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning service advisor result: %w", err)
		}

		if err := json.Unmarshal([]byte(packagesJSON), &result.Packages); err != nil {
			return nil, fmt.Errorf("unmarshaling packages: %w", err)
		}

		if err := json.Unmarshal([]byte(metricsJSON), &result.Metrics); err != nil {
			return nil, fmt.Errorf("unmarshaling metrics: %w", err)
		}

		if err := json.Unmarshal([]byte(recsJSON), &result.Recommendations); err != nil {
			return nil, fmt.Errorf("unmarshaling recommendations: %w", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating service advisor results: %w", err)
	}

	return results, nil
}
