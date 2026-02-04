// codex.go implements the Provider interface for OpenAI Codex CLI.
package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CodexRateLimits represents the rate_limits object in Codex session JSONL.
type CodexRateLimits struct {
	Primary   *CodexRateLimit `json:"primary"`
	Secondary *CodexRateLimit `json:"secondary"`
}

// CodexRateLimit represents a single rate limit entry.
type CodexRateLimit struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowMinutes int64   `json:"window_minutes"`
	ResetsAt      int64   `json:"resets_at"` // Unix timestamp
}

// CodexSessionEntry represents a line in Codex session JSONL.
type CodexSessionEntry struct {
	RateLimits *CodexRateLimits `json:"rate_limits,omitempty"`
	TokenCount *int64           `json:"token_count,omitempty"`
}

// Codex wraps the Codex CLI as a provider.
type Codex struct {
	dataPath   string           // Path to ~/.codex
	rateLimits *CodexRateLimits // Cached rate limits
}

// NewCodex creates a Codex provider.
func NewCodex() *Codex {
	home, _ := os.UserHomeDir()
	return &Codex{
		dataPath: filepath.Join(home, ".codex"),
	}
}

// NewCodexWithPath creates a Codex provider with a custom data path.
func NewCodexWithPath(dataPath string) *Codex {
	return &Codex{
		dataPath: dataPath,
	}
}

// Name returns "codex".
func (c *Codex) Name() string {
	return "codex"
}

// Execute runs a task via Codex CLI.
func (c *Codex) Execute(ctx context.Context, task Task) (Result, error) {
	// TODO: Implement - spawn codex CLI process
	return Result{}, nil
}

// Cost returns Codex's token pricing (cents per 1K tokens).
// Based on GPT-4 pricing estimates.
func (c *Codex) Cost() (inputCents, outputCents int64) {
	// GPT-4: ~$10/M input, ~$30/M output (estimates)
	// Per 1K: 1 cent input, 3 cents output
	return 100, 300 // in hundredths of a cent for precision
}

// DataPath returns the configured data path.
func (c *Codex) DataPath() string {
	return c.dataPath
}

// ParseSessionJSONL reads a Codex session JSONL file and extracts rate limits.
// Returns the most recent rate_limits entry found in the file.
func (c *Codex) ParseSessionJSONL(path string) (*CodexRateLimits, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening codex session: %w", err)
	}
	defer file.Close()

	var latest *CodexRateLimits
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry CodexSessionEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}

		if entry.RateLimits != nil {
			latest = entry.RateLimits
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning codex session: %w", err)
	}

	return latest, nil
}

// ListSessionFiles finds all session JSONL files under sessions/<year>/<month>/<day>/.
func (c *Codex) ListSessionFiles() ([]string, error) {
	sessionsDir := filepath.Join(c.dataPath, "sessions")
	var sessions []string

	err := filepath.WalkDir(sessionsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".jsonl") {
			sessions = append(sessions, path)
		}
		return nil
	})

	if os.IsNotExist(err) {
		return nil, nil
	}
	return sessions, err
}

// FindMostRecentSession finds the most recent session file by modification time.
func (c *Codex) FindMostRecentSession() (string, error) {
	sessions, err := c.ListSessionFiles()
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", nil
	}

	// Sort by modification time, most recent first
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var files []fileInfo

	for _, s := range sessions {
		info, err := os.Stat(s)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: s, modTime: info.ModTime()})
	}

	if len(files) == 0 {
		return "", nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	return files[0].path, nil
}

// GetRateLimits retrieves the latest rate limits from the most recent session.
func (c *Codex) GetRateLimits() (*CodexRateLimits, error) {
	if c.rateLimits != nil {
		return c.rateLimits, nil
	}

	sessionPath, err := c.FindMostRecentSession()
	if err != nil {
		return nil, fmt.Errorf("finding session: %w", err)
	}
	if sessionPath == "" {
		return nil, nil // No sessions found
	}

	limits, err := c.ParseSessionJSONL(sessionPath)
	if err != nil {
		return nil, err
	}

	c.rateLimits = limits
	return limits, nil
}

// GetUsedPercent returns the used_percent based on mode.
// mode: "daily" uses primary rate limit, "weekly" uses secondary.
func (c *Codex) GetUsedPercent(mode string) (float64, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return 0, err
	}
	if limits == nil {
		return 0, nil // No data available
	}

	switch mode {
	case "daily":
		return c.GetPrimaryUsedPercent()
	case "weekly":
		return c.GetSecondaryUsedPercent()
	default:
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// GetPrimaryUsedPercent returns the primary (daily) used_percent.
func (c *Codex) GetPrimaryUsedPercent() (float64, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return 0, err
	}
	if limits == nil || limits.Primary == nil {
		return 0, nil
	}
	return limits.Primary.UsedPercent, nil
}

// GetSecondaryUsedPercent returns the secondary (weekly) used_percent.
func (c *Codex) GetSecondaryUsedPercent() (float64, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return 0, err
	}
	if limits == nil || limits.Secondary == nil {
		return 0, nil
	}
	return limits.Secondary.UsedPercent, nil
}

// GetResetTime returns the reset timestamp for the specified mode.
// mode: "daily" returns primary.resets_at, "weekly" returns secondary.resets_at.
func (c *Codex) GetResetTime(mode string) (time.Time, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return time.Time{}, err
	}
	if limits == nil {
		return time.Time{}, nil
	}

	switch mode {
	case "daily":
		if limits.Primary == nil {
			return time.Time{}, nil
		}
		return time.Unix(limits.Primary.ResetsAt, 0), nil
	case "weekly":
		if limits.Secondary == nil {
			return time.Time{}, nil
		}
		return time.Unix(limits.Secondary.ResetsAt, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// GetWindowMinutes returns the window duration for the specified mode.
func (c *Codex) GetWindowMinutes(mode string) (int64, error) {
	limits, err := c.GetRateLimits()
	if err != nil {
		return 0, err
	}
	if limits == nil {
		return 0, nil
	}

	switch mode {
	case "daily":
		if limits.Primary == nil {
			return 0, nil
		}
		return limits.Primary.WindowMinutes, nil
	case "weekly":
		if limits.Secondary == nil {
			return 0, nil
		}
		return limits.Secondary.WindowMinutes, nil
	default:
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// RefreshRateLimits clears cached rate limits and re-reads from disk.
func (c *Codex) RefreshRateLimits() (*CodexRateLimits, error) {
	c.rateLimits = nil
	return c.GetRateLimits()
}
