package semanticdiff

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Options configures the semantic diff analysis.
type Options struct {
	RepoPath string
	BaseRef  string
	HeadRef  string
	Since    time.Duration // alternative to BaseRef: go back N duration from HEAD
}

// Analyzer performs semantic diff analysis on a git repository.
type Analyzer struct {
	opts Options
}

// NewAnalyzer creates a new semantic diff analyzer.
func NewAnalyzer(opts Options) *Analyzer {
	return &Analyzer{opts: opts}
}

// Run executes the analysis and returns a report.
func (a *Analyzer) Run() (*Report, error) {
	baseRef, headRef, err := a.resolveRefs()
	if err != nil {
		return nil, fmt.Errorf("resolving refs: %w", err)
	}

	cs, err := ExtractDiff(a.opts.RepoPath, baseRef, headRef)
	if err != nil {
		return nil, fmt.Errorf("extracting diff: %w", err)
	}

	if len(cs.Commits) == 0 {
		return &Report{
			BaseRef: baseRef,
			HeadRef: headRef,
			Summary: "No changes found between the specified refs.",
		}, nil
	}

	classified := ClassifyChangeSet(cs)
	report := GenerateReport(cs, classified)
	return report, nil
}

// resolveRefs resolves base and head refs, applying defaults.
func (a *Analyzer) resolveRefs() (string, string, error) {
	headRef := a.opts.HeadRef
	if headRef == "" {
		headRef = "HEAD"
	}

	baseRef := a.opts.BaseRef
	if baseRef == "" && a.opts.Since > 0 {
		// Use git rev-list to find the commit closest to the since duration.
		sinceTime := time.Now().Add(-a.opts.Since)
		cmd := exec.Command("git", "log", "--format=%H", "--after="+sinceTime.Format(time.RFC3339), "--reverse")
		cmd.Dir = a.opts.RepoPath
		out, err := cmd.Output()
		if err != nil {
			return "", "", fmt.Errorf("finding base commit from --since: %w", err)
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			// Use the parent of the first commit in range, or the commit itself.
			baseRef = lines[0] + "~1"
		} else {
			return "", "", fmt.Errorf("no commits found in the last %s", a.opts.Since)
		}
	}

	if baseRef == "" {
		// Default: compare HEAD against its first parent (last commit).
		baseRef = headRef + "~1"
	}

	return baseRef, headRef, nil
}
