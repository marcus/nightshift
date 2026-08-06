package analysis

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SiloResult represents a stored knowledge silo analysis result.
type SiloResult struct {
	ID        int64       `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	RepoPath  string      `json:"repo_path"`
	Depth     int         `json:"depth"`
	Results   []SiloEntry `json:"results"`
	Summary   *SiloReport `json:"summary"`
}

// Store saves a silo analysis result to the database.
func (sr *SiloResult) Store(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	resultsJSON, err := json.Marshal(sr.Results)
	if err != nil {
		return fmt.Errorf("marshaling results: %w", err)
	}

	summaryJSON, err := json.Marshal(sr.Summary)
	if err != nil {
		return fmt.Errorf("marshaling summary: %w", err)
	}

	query := `
		INSERT INTO knowledge_silo_results (timestamp, repo_path, depth, results, summary)
		VALUES (?, ?, ?, ?, ?)
	`

	res, err := db.Exec(query,
		sr.Timestamp,
		sr.RepoPath,
		sr.Depth,
		string(resultsJSON),
		string(summaryJSON),
	)
	if err != nil {
		return fmt.Errorf("inserting silo result: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting insert id: %w", err)
	}
	sr.ID = id

	return nil
}

// LoadLatestSilo loads the most recent silo analysis result for a repo.
func LoadLatestSilo(db *sql.DB, repoPath string) (*SiloResult, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	query := `
		SELECT id, timestamp, repo_path, depth, results, summary
		FROM knowledge_silo_results
		WHERE repo_path = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	row := db.QueryRow(query, repoPath)

	result := &SiloResult{}
	var resultsJSON, summaryJSON string

	err := row.Scan(
		&result.ID,
		&result.Timestamp,
		&result.RepoPath,
		&result.Depth,
		&resultsJSON,
		&summaryJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying silo result: %w", err)
	}

	if err := json.Unmarshal([]byte(resultsJSON), &result.Results); err != nil {
		return nil, fmt.Errorf("unmarshaling results: %w", err)
	}

	if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
		return nil, fmt.Errorf("unmarshaling summary: %w", err)
	}

	return result, nil
}
