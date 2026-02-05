package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultReportsDir returns the default directory for run reports.
func DefaultReportsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "reports")
}

// DefaultRunResultsPath returns the default path for a run results JSON file.
func DefaultRunResultsPath(ts time.Time) string {
	return filepath.Join(DefaultReportsDir(),
		fmt.Sprintf("run-%s.json", ts.Format("2006-01-02-150405")))
}

// SaveRunResults writes structured run results to disk as JSON.
func SaveRunResults(results *RunResults, path string) error {
	if results == nil {
		return fmt.Errorf("results cannot be nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating results dir: %w", err)
	}
	payload, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding results: %w", err)
	}
	if err := os.WriteFile(path, payload, 0644); err != nil {
		return fmt.Errorf("writing results: %w", err)
	}
	return nil
}

// LoadRunResults reads structured run results from disk.
func LoadRunResults(path string) (*RunResults, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading results: %w", err)
	}
	var results RunResults
	if err := json.Unmarshal(payload, &results); err != nil {
		return nil, fmt.Errorf("decoding results: %w", err)
	}
	return &results, nil
}
