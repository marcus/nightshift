package analysis

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SiloEntry represents a directory's knowledge silo analysis.
type SiloEntry struct {
	Directory        string         `json:"directory"`
	TopContributors  []CommitAuthor `json:"top_contributors"`
	TotalCommits     int            `json:"total_commits"`
	ContributorCount int            `json:"contributor_count"`
	SiloScore        float64        `json:"silo_score"` // 0-1, 1 = max silo risk
	RiskLevel        string         `json:"risk_level"` // critical, high, medium, low
}

// SiloReport holds the full knowledge silo analysis results.
type SiloReport struct {
	Timestamp       time.Time   `json:"timestamp"`
	RepoPath        string      `json:"repo_path"`
	Depth           int         `json:"depth"`
	Entries         []SiloEntry `json:"entries"`
	TotalDirs       int         `json:"total_dirs"`
	CriticalCount   int         `json:"critical_count"`
	HighCount       int         `json:"high_count"`
	Recommendations []string    `json:"recommendations"`
	ReportedAt      string      `json:"reported_at"`
}

// SiloParseOptions defines filtering options for silo analysis.
type SiloParseOptions struct {
	Since      time.Time
	Until      time.Time
	Depth      int // directory depth to analyze (default 2)
	MinCommits int // minimum commits to include a directory (default 5)
}

// ParseAuthorsByDirectory extracts per-directory author contributions from git history.
// It runs 'git log --format=%an|%ae --name-only' and groups files by directory at the
// configured depth, returning a map of directory path to author contributions.
func (gp *GitParser) ParseAuthorsByDirectory(opts SiloParseOptions) (map[string][]CommitAuthor, error) {
	if opts.Depth <= 0 {
		opts.Depth = 2
	}

	args := []string{"log", "--format=COMMIT:%an|%ae", "--name-only"}

	if !opts.Since.IsZero() {
		args = append(args, fmt.Sprintf("--since=%s", opts.Since.Format(time.RFC3339)))
	}
	if !opts.Until.IsZero() {
		args = append(args, fmt.Sprintf("--until=%s", opts.Until.Format(time.RFC3339)))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = gp.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running git log: %w", err)
	}

	// Parse output: lines alternate between "COMMIT:name|email" headers and file paths
	// dirAuthors maps directory -> email -> CommitAuthor
	dirAuthors := make(map[string]map[string]*CommitAuthor)
	var currentName, currentEmail string

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "COMMIT:") {
			parts := strings.SplitN(line[7:], "|", 2)
			if len(parts) == 2 {
				currentName = parts[0]
				currentEmail = parts[1]
			}
			continue
		}

		// This is a file path — extract directory at configured depth
		if currentEmail == "" {
			continue
		}

		dir := truncateToDepth(line, opts.Depth)
		if dir == "" {
			continue
		}

		if dirAuthors[dir] == nil {
			dirAuthors[dir] = make(map[string]*CommitAuthor)
		}

		key := strings.ToLower(currentEmail)
		if author, exists := dirAuthors[dir][key]; exists {
			author.Commits++
		} else {
			dirAuthors[dir][key] = &CommitAuthor{
				Name:    currentName,
				Email:   currentEmail,
				Commits: 1,
			}
		}
	}

	// Convert nested maps to map[string][]CommitAuthor
	result := make(map[string][]CommitAuthor, len(dirAuthors))
	for dir, authorMap := range dirAuthors {
		authors := make([]CommitAuthor, 0, len(authorMap))
		for _, author := range authorMap {
			authors = append(authors, *author)
		}
		// Sort by commits descending
		sort.Slice(authors, func(i, j int) bool {
			return authors[i].Commits > authors[j].Commits
		})
		result[dir] = authors
	}

	return result, nil
}

// truncateToDepth returns the directory path truncated to the given depth.
// For depth=2, "internal/analysis/silo.go" returns "internal/analysis".
// Files at the root level return "." for depth >= 1.
func truncateToDepth(filePath string, depth int) string {
	dir := filepath.Dir(filePath)
	if dir == "." {
		return "."
	}

	parts := strings.Split(filepath.ToSlash(dir), "/")
	if len(parts) > depth {
		parts = parts[:depth]
	}

	return strings.Join(parts, "/")
}

// CalculateSilos computes silo scores for each directory based on author distributions.
func CalculateSilos(dirAuthors map[string][]CommitAuthor, minCommits int) []SiloEntry {
	if minCommits <= 0 {
		minCommits = 5
	}

	var entries []SiloEntry

	for dir, authors := range dirAuthors {
		totalCommits := 0
		for _, a := range authors {
			totalCommits += a.Commits
		}

		// Skip directories with too few commits
		if totalCommits < minCommits {
			continue
		}

		entry := SiloEntry{
			Directory:        dir,
			TotalCommits:     totalCommits,
			ContributorCount: len(authors),
		}

		// Keep top 3 contributors for display
		topN := 3
		if len(authors) < topN {
			topN = len(authors)
		}
		entry.TopContributors = authors[:topN]

		// Calculate silo score: combine contributor count and commit concentration
		entry.SiloScore = calculateSiloScore(authors, totalCommits)
		entry.RiskLevel = assessSiloRisk(authors, totalCommits)

		entries = append(entries, entry)
	}

	// Sort by silo score descending (worst silos first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SiloScore > entries[j].SiloScore
	})

	return entries
}

// calculateSiloScore computes a 0-1 score where 1 = maximum silo risk.
// Uses inverse normalized contributor count weighted by commit concentration (Herfindahl).
func calculateSiloScore(authors []CommitAuthor, totalCommits int) float64 {
	if len(authors) == 0 || totalCommits == 0 {
		return 0
	}

	// Herfindahl index for commit concentration
	hhi := 0.0
	for _, a := range authors {
		share := float64(a.Commits) / float64(totalCommits)
		hhi += share * share
	}

	// For a single contributor, HHI = 1.0 which is the max silo
	if len(authors) == 1 {
		return 1.0
	}

	// Normalize HHI: remove baseline for n contributors
	n := float64(len(authors))
	minHHI := 1.0 / n
	normalizedHHI := (hhi - minHHI) / (1.0 - minHHI)
	if normalizedHHI < 0 {
		normalizedHHI = 0
	}
	if normalizedHHI > 1 {
		normalizedHHI = 1
	}

	// Weight: 60% concentration, 40% inverse contributor count
	// Fewer contributors = higher silo risk
	contributorFactor := 1.0 / n // 1 person = 1.0, 10 people = 0.1
	if contributorFactor > 1 {
		contributorFactor = 1
	}

	score := 0.6*normalizedHHI + 0.4*contributorFactor
	if score > 1 {
		score = 1
	}

	return score
}

// assessSiloRisk determines the risk level for a directory.
func assessSiloRisk(authors []CommitAuthor, totalCommits int) string {
	if len(authors) == 0 || totalCommits == 0 {
		return "unknown"
	}

	// Single contributor = critical
	if len(authors) <= 1 {
		return "critical"
	}

	// Top contributor percentage
	top1Pct := float64(authors[0].Commits) / float64(totalCommits)

	// Critical: top contributor owns > 80%
	if top1Pct > 0.8 {
		return "critical"
	}

	// High: top contributor owns > 60% or only 2 contributors
	if top1Pct > 0.6 || len(authors) <= 2 {
		return "high"
	}

	// Medium: top contributor owns > 40% or 3 or fewer contributors
	if top1Pct > 0.4 || len(authors) <= 3 {
		return "medium"
	}

	return "low"
}
