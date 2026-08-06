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
