// Package semanticdiff extracts git diffs and classifies changes by semantic category.
package semanticdiff

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Commit represents a single git commit with metadata.
type Commit struct {
	Hash    string
	Subject string
	Body    string
	Author  string
	Files   []FileChange
}

// FileChange represents a single file's changes within a commit.
type FileChange struct {
	Path       string
	OldPath    string // non-empty if renamed
	Additions  int
	Deletions  int
	IsBinary   bool
	IsRename   bool
	IsNew      bool
	IsDeleted  bool
	DiffOutput string // raw diff hunk text for this file
}

// ChangeSet is the full diff extraction result between two refs.
type ChangeSet struct {
	BaseRef  string
	HeadRef  string
	Commits  []Commit
	AllFiles []FileChange // aggregated across all commits
}

// TotalAdditions returns the sum of additions across all files.
func (cs *ChangeSet) TotalAdditions() int {
	total := 0
	for _, f := range cs.AllFiles {
		total += f.Additions
	}
	return total
}

// TotalDeletions returns the sum of deletions across all files.
func (cs *ChangeSet) TotalDeletions() int {
	total := 0
	for _, f := range cs.AllFiles {
		total += f.Deletions
	}
	return total
}

// ExtractDiff extracts a structured ChangeSet between two git refs.
func ExtractDiff(repoPath, baseRef, headRef string) (*ChangeSet, error) {
	commits, err := parseCommits(repoPath, baseRef, headRef)
	if err != nil {
		return nil, fmt.Errorf("parsing commits: %w", err)
	}

	allFiles, err := parseNumstat(repoPath, baseRef, headRef)
	if err != nil {
		return nil, fmt.Errorf("parsing numstat: %w", err)
	}

	return &ChangeSet{
		BaseRef:  baseRef,
		HeadRef:  headRef,
		Commits:  commits,
		AllFiles: allFiles,
	}, nil
}

// parseCommits extracts commit metadata between two refs.
func parseCommits(repoPath, baseRef, headRef string) ([]Commit, error) {
	// Use a delimiter unlikely to appear in commit messages.
	const sep = "---NIGHTSHIFT-SEP---"
	format := fmt.Sprintf("%%H%s%%s%s%%b%s%%an%s", sep, sep, sep, sep)

	cmd := exec.Command("git", "log", "--format="+format, baseRef+".."+headRef)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	// Each commit ends with the trailing separator.
	entries := strings.Split(raw, sep+"\n")
	var commits []Commit
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, sep, 4)
		if len(parts) < 4 {
			continue
		}
		c := Commit{
			Hash:    strings.TrimSpace(parts[0]),
			Subject: strings.TrimSpace(parts[1]),
			Body:    strings.TrimSpace(parts[2]),
			Author:  strings.TrimSpace(parts[3]),
		}
		commits = append(commits, c)
	}

	// Attach file lists to each commit.
	for i := range commits {
		files, err := parseCommitFiles(repoPath, commits[i].Hash)
		if err != nil {
			continue // non-fatal
		}
		commits[i].Files = files
	}

	return commits, nil
}

// parseCommitFiles returns the file changes for a single commit.
func parseCommitFiles(repoPath, hash string) ([]FileChange, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "-r", "--numstat", "-M", hash)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseNumstatLines(string(out))
}

// parseNumstat runs git diff --numstat between two refs and returns aggregated file changes.
func parseNumstat(repoPath, baseRef, headRef string) ([]FileChange, error) {
	cmd := exec.Command("git", "diff", "--numstat", "-M", baseRef+".."+headRef)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --numstat: %w", err)
	}
	return parseNumstatLines(string(out))
}

// parseNumstatLines parses lines of git numstat output into FileChange structs.
func parseNumstatLines(output string) ([]FileChange, error) {
	var files []FileChange
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		fc := FileChange{}

		// Binary files show "-" for additions/deletions.
		if fields[0] == "-" && fields[1] == "-" {
			fc.IsBinary = true
		} else {
			fc.Additions, _ = strconv.Atoi(fields[0])
			fc.Deletions, _ = strconv.Atoi(fields[1])
		}

		path := fields[2]
		// Detect renames: "old => new" or "{prefix/old => prefix/new}"
		if len(fields) >= 4 && strings.Contains(line, "=>") {
			fc.IsRename = true
			fc.OldPath = extractRenamePart(fields[2:], true)
			fc.Path = extractRenamePart(fields[2:], false)
		} else {
			fc.Path = path
		}

		files = append(files, fc)
	}
	return files, nil
}

// extractRenamePart extracts old or new path from rename notation.
func extractRenamePart(fields []string, wantOld bool) string {
	joined := strings.Join(fields, " ")
	// Handle {a => b} style
	if idx := strings.Index(joined, " => "); idx >= 0 {
		if wantOld {
			old := joined[:idx]
			old = strings.TrimPrefix(old, "{")
			return strings.TrimSpace(old)
		}
		newPart := joined[idx+4:]
		newPart = strings.TrimSuffix(newPart, "}")
		return strings.TrimSpace(newPart)
	}
	return joined
}
