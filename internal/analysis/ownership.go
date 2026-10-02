package analysis

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DirectoryOwnership represents ownership statistics for a single directory.
type DirectoryOwnership struct {
	Dir           string                  `json:"dir"`
	Authors       map[string]*AuthorStats `json:"authors"`
	DominantOwner string                  `json:"dominant_owner"`
	Confidence    float64                 `json:"confidence"` // 0-1, how dominant the top owner is
	TotalCommits  int                     `json:"total_commits"`
}

// AuthorStats tracks an author's contribution to a directory.
type AuthorStats struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Commits int    `json:"commits"`
}

// OwnershipBoundary groups directories by their dominant owner.
type OwnershipBoundary struct {
	Owner       string   `json:"owner"`
	Email       string   `json:"email"`
	Directories []string `json:"directories"`
	TotalCommit int      `json:"total_commits"`
}

// CohesionPair represents two directories frequently co-changed.
type CohesionPair struct {
	DirA      string  `json:"dir_a"`
	DirB      string  `json:"dir_b"`
	CoChanges int     `json:"co_changes"`
	Strength  float64 `json:"strength"` // 0-1, ratio of co-changes to total changes
}

// OwnershipReport is the complete output of an ownership boundary analysis.
type OwnershipReport struct {
	Timestamp   time.Time            `json:"timestamp"`
	RepoPath    string               `json:"repo_path"`
	Directories []DirectoryOwnership `json:"directories"`
	Boundaries  []OwnershipBoundary  `json:"boundaries"`
	Cohesion    []CohesionPair       `json:"cohesion"`
	Unclear     []DirectoryOwnership `json:"unclear"` // directories with no clear dominant owner
}

// ParseDirectoryOwnership runs git log to aggregate per-directory author commit counts.
func (gp *GitParser) ParseDirectoryOwnership(opts ParseOptions, depth int, minCommits int) ([]DirectoryOwnership, error) {
	args := []string{"log", "--format=%an|%ae", "--name-only"}

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

	// Parse the output. git log --format=%an|%ae --name-only produces:
	//   author|email
	//                    <-- blank line between format and filenames
	//   file1
	//   file2
	//   author|email
	//                    <-- blank
	//   file3
	//
	// Author lines contain "|" with an email (containing "@").
	// Blank lines are ignored. File lines are attributed to the most recent author.
	dirAuthors := make(map[string]map[string]*AuthorStats) // dir -> email -> stats

	lines := strings.Split(string(output), "\n")
	var currentAuthor, currentEmail string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect author lines: contain "|" and the second part has "@"
		if parts := strings.SplitN(line, "|", 2); len(parts) == 2 && strings.Contains(parts[1], "@") {
			currentAuthor = parts[0]
			currentEmail = parts[1]
			continue
		}

		if currentAuthor == "" {
			continue
		}

		// This is a file path — extract the directory at the given depth
		dir := directoryAtDepth(line, depth)
		if dir == "" {
			continue
		}

		emailKey := strings.ToLower(currentEmail)
		if dirAuthors[dir] == nil {
			dirAuthors[dir] = make(map[string]*AuthorStats)
		}
		if dirAuthors[dir][emailKey] == nil {
			dirAuthors[dir][emailKey] = &AuthorStats{
				Name:  currentAuthor,
				Email: currentEmail,
			}
		}
		dirAuthors[dir][emailKey].Commits++
	}

	// Build DirectoryOwnership entries
	var results []DirectoryOwnership
	for dir, authors := range dirAuthors {
		total := 0
		for _, a := range authors {
			total += a.Commits
		}
		if total < minCommits {
			continue
		}

		// Find dominant owner
		var dominant *AuthorStats
		for _, a := range authors {
			if dominant == nil || a.Commits > dominant.Commits {
				dominant = a
			}
		}

		confidence := 0.0
		if dominant != nil && total > 0 {
			confidence = float64(dominant.Commits) / float64(total)
		}

		dominantName := ""
		if dominant != nil {
			dominantName = dominant.Name
		}

		results = append(results, DirectoryOwnership{
			Dir:           dir,
			Authors:       authors,
			DominantOwner: dominantName,
			Confidence:    confidence,
			TotalCommits:  total,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Dir < results[j].Dir
	})

	return results, nil
}

// directoryAtDepth returns the directory path truncated to the given depth.
// depth=1 returns just the top-level dir, depth=2 returns two levels, etc.
func directoryAtDepth(filePath string, depth int) string {
	dir := filepath.Dir(filePath)
	if dir == "." {
		return ""
	}

	parts := strings.Split(filepath.ToSlash(dir), "/")
	if len(parts) > depth {
		parts = parts[:depth]
	}
	return strings.Join(parts, "/")
}

// DetectCohesion analyzes git history to find directories frequently co-changed in the same commit.
func (gp *GitParser) DetectCohesion(opts ParseOptions, depth int, minCoChanges int) ([]CohesionPair, error) {
	args := []string{"log", "--name-only", "--pretty=format:COMMIT_SEP"}

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

	// Build co-change matrix
	type pairKey struct{ a, b string }
	coChangeCount := make(map[pairKey]int)
	dirChangeCount := make(map[string]int)

	commits := strings.Split(string(output), "COMMIT_SEP")
	for _, commit := range commits {
		// Collect unique directories in this commit
		dirSet := make(map[string]bool)
		for _, line := range strings.Split(commit, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			dir := directoryAtDepth(line, depth)
			if dir != "" {
				dirSet[dir] = true
			}
		}

		dirs := make([]string, 0, len(dirSet))
		for d := range dirSet {
			dirs = append(dirs, d)
		}
		sort.Strings(dirs)

		for _, d := range dirs {
			dirChangeCount[d]++
		}

		// Count pairwise co-changes
		for i := 0; i < len(dirs); i++ {
			for j := i + 1; j < len(dirs); j++ {
				key := pairKey{dirs[i], dirs[j]}
				coChangeCount[key]++
			}
		}
	}

	// Build result pairs
	var pairs []CohesionPair
	for key, count := range coChangeCount {
		if count < minCoChanges {
			continue
		}
		// Strength = co-changes / min(changes_a, changes_b)
		minChanges := dirChangeCount[key.a]
		if dirChangeCount[key.b] < minChanges {
			minChanges = dirChangeCount[key.b]
		}
		strength := 0.0
		if minChanges > 0 {
			strength = float64(count) / float64(minChanges)
		}

		pairs = append(pairs, CohesionPair{
			DirA:      key.a,
			DirB:      key.b,
			CoChanges: count,
			Strength:  strength,
		})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].CoChanges > pairs[j].CoChanges
	})

	return pairs, nil
}

// SuggestBoundaries groups directories by their dominant owner to suggest ownership boundaries.
func SuggestBoundaries(dirs []DirectoryOwnership) []OwnershipBoundary {
	ownerDirs := make(map[string]*OwnershipBoundary) // email -> boundary

	for _, d := range dirs {
		if d.DominantOwner == "" {
			continue
		}

		// Find the email for the dominant owner
		var email string
		for _, a := range d.Authors {
			if a.Name == d.DominantOwner {
				email = strings.ToLower(a.Email)
				break
			}
		}
		if email == "" {
			continue
		}

		if ownerDirs[email] == nil {
			ownerDirs[email] = &OwnershipBoundary{
				Owner: d.DominantOwner,
				Email: email,
			}
		}
		ownerDirs[email].Directories = append(ownerDirs[email].Directories, d.Dir)
		ownerDirs[email].TotalCommit += d.TotalCommits
	}

	boundaries := make([]OwnershipBoundary, 0, len(ownerDirs))
	for _, b := range ownerDirs {
		sort.Strings(b.Directories)
		boundaries = append(boundaries, *b)
	}

	// Sort by total commits descending
	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i].TotalCommit > boundaries[j].TotalCommit
	})

	return boundaries
}

// FindUnclearOwnership returns directories where no single author dominates.
func FindUnclearOwnership(dirs []DirectoryOwnership, threshold float64) []DirectoryOwnership {
	var unclear []DirectoryOwnership
	for _, d := range dirs {
		if d.Confidence < threshold {
			unclear = append(unclear, d)
		}
	}
	return unclear
}

// GenerateCODEOWNERS renders a GitHub-style CODEOWNERS file from ownership boundaries.
func GenerateCODEOWNERS(boundaries []OwnershipBoundary) string {
	var buf bytes.Buffer
	buf.WriteString("# CODEOWNERS - Generated by nightshift ownership analysis\n")
	buf.WriteString("# Review and adjust before committing\n\n")

	for _, b := range boundaries {
		for _, dir := range b.Directories {
			fmt.Fprintf(&buf, "/%s/ %s\n", dir, b.Email)
		}
	}

	return buf.String()
}

// RenderOwnershipMarkdown generates a human-readable markdown report.
func RenderOwnershipMarkdown(report *OwnershipReport) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# Ownership Boundary Analysis\n\n")
	fmt.Fprintf(&buf, "*Generated: %s*\n\n", report.Timestamp.Format("2006-01-02 15:04:05"))

	// Summary
	fmt.Fprintf(&buf, "## Summary\n\n")
	fmt.Fprintf(&buf, "- **Directories analyzed**: %d\n", len(report.Directories))
	fmt.Fprintf(&buf, "- **Ownership boundaries**: %d owners\n", len(report.Boundaries))
	fmt.Fprintf(&buf, "- **Unclear ownership**: %d directories\n", len(report.Unclear))
	fmt.Fprintf(&buf, "- **Co-change pairs**: %d\n\n", len(report.Cohesion))

	// Ownership boundaries
	if len(report.Boundaries) > 0 {
		buf.WriteString("## Ownership Boundaries\n\n")
		for _, b := range report.Boundaries {
			fmt.Fprintf(&buf, "### %s (%s)\n\n", b.Owner, b.Email)
			fmt.Fprintf(&buf, "Total commits: %d\n\n", b.TotalCommit)
			for _, dir := range b.Directories {
				fmt.Fprintf(&buf, "- `%s/`\n", dir)
			}
			buf.WriteString("\n")
		}
	}

	// Directory details
	if len(report.Directories) > 0 {
		buf.WriteString("## Directory Ownership Details\n\n")
		buf.WriteString("| Directory | Dominant Owner | Confidence | Commits | Contributors |\n")
		buf.WriteString("|-----------|---------------|------------|---------|-------------|\n")
		for _, d := range report.Directories {
			fmt.Fprintf(&buf, "| `%s/` | %s | %.0f%% | %d | %d |\n",
				d.Dir, d.DominantOwner, d.Confidence*100, d.TotalCommits, len(d.Authors))
		}
		buf.WriteString("\n")
	}

	// Unclear ownership
	if len(report.Unclear) > 0 {
		buf.WriteString("## Unclear Ownership\n\n")
		buf.WriteString("These directories have no dominant contributor (confidence < 50%):\n\n")
		for _, d := range report.Unclear {
			fmt.Fprintf(&buf, "- **`%s/`** — %d commits across %d contributors (top: %s at %.0f%%)\n",
				d.Dir, d.TotalCommits, len(d.Authors), d.DominantOwner, d.Confidence*100)
		}
		buf.WriteString("\n")
	}

	// Cohesion
	if len(report.Cohesion) > 0 {
		buf.WriteString("## Co-Change Cohesion\n\n")
		buf.WriteString("Directories frequently modified in the same commits:\n\n")
		buf.WriteString("| Dir A | Dir B | Co-Changes | Strength |\n")
		buf.WriteString("|-------|-------|------------|----------|\n")
		limit := len(report.Cohesion)
		if limit > 20 {
			limit = 20
		}
		for _, c := range report.Cohesion[:limit] {
			fmt.Fprintf(&buf, "| `%s/` | `%s/` | %d | %.0f%% |\n",
				c.DirA, c.DirB, c.CoChanges, c.Strength*100)
		}
		if len(report.Cohesion) > 20 {
			fmt.Fprintf(&buf, "\n*... and %d more pairs*\n", len(report.Cohesion)-20)
		}
		buf.WriteString("\n")
	}

	return buf.String()
}
