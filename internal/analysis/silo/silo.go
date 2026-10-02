// Package silo identifies knowledge silos in a git repository: files or
// directories where commit activity is dominated by a single author.
package silo

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Commit represents a single commit touching one or more files.
type Commit struct {
	Author string
	When   time.Time
	Files  []string
}

// CommitSource yields commits for analysis. Injectable for testing.
type CommitSource interface {
	Commits() ([]Commit, error)
}

// Options configure silo detection.
type Options struct {
	Since              time.Time
	PathFilter         string  // optional path prefix filter
	DominanceThreshold float64 // e.g. 0.8 -> 80%
	GroupByDir         bool    // aggregate by top-level dir instead of files
}

// PathStats holds aggregated stats for one file or directory.
type PathStats struct {
	Path        string
	Commits     int
	Authors     int
	TopAuthor   string
	TopCommits  int
	Dominance   float64
	LastTouch   time.Time
	OtherLatest time.Time // most recent commit by a non-dominant author
	Risk        float64
	IsSilo      bool
}

// gitCommitSource is the default implementation backed by `git log`.
type gitCommitSource struct {
	repo  string
	since time.Time
}

// NewGitCommitSource creates a source that reads commits from a repo via git.
func NewGitCommitSource(repo string, since time.Time) CommitSource {
	return &gitCommitSource{repo: repo, since: since}
}

func (g *gitCommitSource) Commits() ([]Commit, error) {
	args := []string{"log", "--name-only", "--no-merges", "--format=__COMMIT__%n%ae%n%at"}
	if !g.since.IsZero() {
		args = append(args, "--since="+g.since.Format(time.RFC3339))
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = g.repo
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	return parseGitLog(string(out)), nil
}

func parseGitLog(out string) []Commit {
	var commits []Commit
	var cur *Commit
	state := 0 // 0=expect marker, 1=author, 2=time, 3=files
	for _, line := range strings.Split(out, "\n") {
		if line == "__COMMIT__" {
			if cur != nil {
				commits = append(commits, *cur)
			}
			cur = &Commit{}
			state = 1
			continue
		}
		if cur == nil {
			continue
		}
		switch state {
		case 1:
			cur.Author = strings.ToLower(strings.TrimSpace(line))
			state = 2
		case 2:
			var ts int64
			fmt.Sscanf(strings.TrimSpace(line), "%d", &ts)
			cur.When = time.Unix(ts, 0)
			state = 3
		case 3:
			line = strings.TrimSpace(line)
			if line != "" {
				cur.Files = append(cur.Files, line)
			}
		}
	}
	if cur != nil {
		commits = append(commits, *cur)
	}
	return commits
}

// Analyze aggregates commits and identifies silos according to opts.
func Analyze(src CommitSource, opts Options) ([]PathStats, error) {
	commits, err := src.Commits()
	if err != nil {
		return nil, err
	}
	if opts.DominanceThreshold <= 0 {
		opts.DominanceThreshold = 0.8
	}

	type bucket struct {
		commits map[string]int // author -> count
		last    time.Time
	}
	buckets := map[string]*bucket{}

	bump := func(path, author string, when time.Time) {
		b, ok := buckets[path]
		if !ok {
			b = &bucket{commits: map[string]int{}}
			buckets[path] = b
		}
		b.commits[author]++
		if when.After(b.last) {
			b.last = when
		}
	}

	for _, c := range commits {
		if !opts.Since.IsZero() && c.When.Before(opts.Since) {
			continue
		}
		for _, f := range c.Files {
			if opts.PathFilter != "" && !strings.HasPrefix(f, opts.PathFilter) {
				continue
			}
			key := f
			if opts.GroupByDir {
				key = topDir(f)
			}
			bump(key, c.Author, c.When)
		}
	}

	results := make([]PathStats, 0, len(buckets))
	for path, b := range buckets {
		total := 0
		topAuthor := ""
		topN := 0
		for a, n := range b.commits {
			total += n
			if n > topN {
				topN = n
				topAuthor = a
			}
		}
		if total == 0 {
			continue
		}
		dominance := float64(topN) / float64(total)
		authors := len(b.commits)
		isSilo := authors <= 1 || dominance >= opts.DominanceThreshold
		// Risk: weight dominance and bus-factor; small floor for recency.
		risk := dominance*0.6 + (1.0/float64(authors))*0.4
		results = append(results, PathStats{
			Path:       path,
			Commits:    total,
			Authors:    authors,
			TopAuthor:  topAuthor,
			TopCommits: topN,
			Dominance:  dominance,
			LastTouch:  b.last,
			Risk:       risk,
			IsSilo:     isSilo,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Risk != results[j].Risk {
			return results[i].Risk > results[j].Risk
		}
		return results[i].Commits > results[j].Commits
	})
	return results, nil
}

func topDir(p string) string {
	p = filepath.ToSlash(p)
	if i := strings.Index(p, "/"); i >= 0 {
		return p[:i]
	}
	return "."
}
