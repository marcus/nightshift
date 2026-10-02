// Package releasenotes generates release notes from git history between tags.
package releasenotes

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// CommitCategory classifies a commit by its type.
type CommitCategory string

const (
	CategoryFeatures     CommitCategory = "Features"
	CategoryFixes        CommitCategory = "Bug Fixes"
	CategorySecurity     CommitCategory = "Security"
	CategoryPerformance  CommitCategory = "Performance"
	CategoryDocs         CommitCategory = "Documentation"
	CategoryRefactor     CommitCategory = "Refactoring"
	CategoryTests        CommitCategory = "Tests"
	CategoryBuild        CommitCategory = "Build & CI"
	CategoryBreaking     CommitCategory = "Breaking Changes"
	CategoryOther        CommitCategory = "Other Changes"
)

// categoryOrder defines the display order of categories.
var categoryOrder = []CommitCategory{
	CategoryBreaking,
	CategoryFeatures,
	CategorySecurity,
	CategoryFixes,
	CategoryPerformance,
	CategoryRefactor,
	CategoryDocs,
	CategoryTests,
	CategoryBuild,
	CategoryOther,
}

// Commit represents a parsed git commit.
type Commit struct {
	Hash      string
	ShortHash string
	Subject   string
	Body      string
	Author    string
	Date      time.Time
	Category  CommitCategory
	Scope     string // Optional scope from conventional commits, e.g. "config" in "feat(config): ..."
	Breaking  bool
}

// TagInfo represents a git tag with its associated metadata.
type TagInfo struct {
	Name string
	Hash string
	Date time.Time
}

// ReleaseNotes holds the generated release notes content.
type ReleaseNotes struct {
	Version    string
	PrevTag    string
	Date       time.Time
	Categories map[CommitCategory][]Commit
	AllCommits []Commit
	RepoPath   string
}

// Generator creates release notes from git history.
type Generator struct {
	repoPath string
}

// NewGenerator creates a release notes generator for the given repository.
func NewGenerator(repoPath string) *Generator {
	return &Generator{repoPath: repoPath}
}

// Options controls release note generation behavior.
type Options struct {
	// Tag to generate notes for. If empty, uses HEAD.
	Tag string
	// PrevTag to compare against. If empty, auto-detects previous tag.
	PrevTag string
	// IncludeCommitHashes includes short commit hashes in output.
	IncludeCommitHashes bool
	// IncludeAuthors includes author names in output.
	IncludeAuthors bool
	// GroupByCategory groups commits by conventional commit type.
	GroupByCategory bool
}

// DefaultOptions returns sensible defaults for release note generation.
func DefaultOptions() Options {
	return Options{
		IncludeCommitHashes: true,
		IncludeAuthors:      false,
		GroupByCategory:     true,
	}
}

// Generate creates release notes from git history.
func (g *Generator) Generate(opts Options) (*ReleaseNotes, error) {
	// Determine the tag range
	tag := opts.Tag
	if tag == "" {
		latest, err := g.latestTag()
		if err != nil || latest == "" {
			tag = "HEAD"
		} else {
			tag = latest
		}
	}

	prevTag := opts.PrevTag
	if prevTag == "" {
		var err error
		prevTag, err = g.previousTag(tag)
		if err != nil {
			prevTag = "" // Will use full history
		}
	}

	// Get commits in range
	commits, err := g.commitsInRange(prevTag, tag)
	if err != nil {
		return nil, fmt.Errorf("getting commits: %w", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits found between %q and %q", prevTag, tag)
	}

	// Categorize commits
	categories := make(map[CommitCategory][]Commit)
	for i := range commits {
		commits[i].Category, commits[i].Scope, commits[i].Breaking = classifyCommit(commits[i].Subject)
		cat := commits[i].Category
		if commits[i].Breaking {
			categories[CategoryBreaking] = append(categories[CategoryBreaking], commits[i])
		}
		categories[cat] = append(categories[cat], commits[i])
	}

	version := tag
	if version == "HEAD" {
		version = "Unreleased"
	}

	return &ReleaseNotes{
		Version:    version,
		PrevTag:    prevTag,
		Date:       time.Now(),
		Categories: categories,
		AllCommits: commits,
		RepoPath:   g.repoPath,
	}, nil
}

// latestTag returns the most recent semver tag reachable from HEAD.
func (g *Generator) latestTag() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0", "HEAD")
	cmd.Dir = g.repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// previousTag returns the tag before the given tag.
func (g *Generator) previousTag(tag string) (string, error) {
	ref := tag
	if ref == "HEAD" {
		// For HEAD, find the latest tag first, then the one before it
		latest, err := g.latestTag()
		if err != nil || latest == "" {
			return "", fmt.Errorf("no tags found")
		}
		return latest, nil
	}

	// Find the tag before this one
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0", ref+"^")
	cmd.Dir = g.repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// commitsInRange returns commits between two refs.
func (g *Generator) commitsInRange(from, to string) ([]Commit, error) {
	rangeSpec := to
	if from != "" {
		rangeSpec = from + ".." + to
	}

	// Use a delimiter that won't appear in commit messages
	const delim = "---NIGHTSHIFT-COMMIT-DELIM---"
	format := strings.Join([]string{"%H", "%h", "%s", "%b", "%an", "%aI"}, "%x00") + delim

	cmd := exec.Command("git", "log", "--format="+format, rangeSpec)
	cmd.Dir = g.repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	entries := strings.Split(raw, delim)
	var commits []Commit
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.Split(entry, "\x00")
		if len(parts) < 6 {
			continue
		}

		date, _ := time.Parse(time.RFC3339, strings.TrimSpace(parts[5]))

		commits = append(commits, Commit{
			Hash:      strings.TrimSpace(parts[0]),
			ShortHash: strings.TrimSpace(parts[1]),
			Subject:   strings.TrimSpace(parts[2]),
			Body:      strings.TrimSpace(parts[3]),
			Author:    strings.TrimSpace(parts[4]),
			Date:      date,
		})
	}

	return commits, nil
}

// Tags returns all semver-like tags in chronological order.
func (g *Generator) Tags() ([]TagInfo, error) {
	cmd := exec.Command("git", "tag", "--sort=-creatordate", "--format=%(refname:short)%00%(objectname:short)%00%(creatordate:iso-strict)")
	cmd.Dir = g.repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git tag: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	var tags []TagInfo
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 3 {
			continue
		}
		date, _ := time.Parse(time.RFC3339, strings.TrimSpace(parts[2]))
		tags = append(tags, TagInfo{
			Name: parts[0],
			Hash: parts[1],
			Date: date,
		})
	}

	return tags, nil
}

// conventionalCommitRe matches conventional commit subjects like:
// feat(scope): description
// fix: description
// feat!: breaking change
var conventionalCommitRe = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!)?\s*:\s*(.+)$`)

// classifyCommit categorizes a commit based on its subject line.
func classifyCommit(subject string) (CommitCategory, string, bool) {
	m := conventionalCommitRe.FindStringSubmatch(subject)
	if m == nil {
		return inferCategory(subject), "", false
	}

	prefix := strings.ToLower(m[1])
	scope := m[2]
	breaking := m[3] == "!"

	cat := prefixToCategory(prefix)
	return cat, scope, breaking
}

// prefixToCategory maps conventional commit prefixes to categories.
func prefixToCategory(prefix string) CommitCategory {
	switch prefix {
	case "feat", "feature":
		return CategoryFeatures
	case "fix", "bugfix":
		return CategoryFixes
	case "security", "sec":
		return CategorySecurity
	case "perf", "performance":
		return CategoryPerformance
	case "docs", "doc":
		return CategoryDocs
	case "refactor":
		return CategoryRefactor
	case "test", "tests":
		return CategoryTests
	case "build", "ci", "chore":
		return CategoryBuild
	default:
		return CategoryOther
	}
}

// inferCategory attempts to classify non-conventional commit messages by keywords.
func inferCategory(subject string) CommitCategory {
	lower := strings.ToLower(subject)

	switch {
	case strings.HasPrefix(lower, "add ") || strings.HasPrefix(lower, "implement ") ||
		strings.HasPrefix(lower, "introduce "):
		return CategoryFeatures
	case strings.HasPrefix(lower, "fix ") || strings.Contains(lower, "bugfix"):
		return CategoryFixes
	case strings.Contains(lower, "security") || strings.Contains(lower, "vulnerability") ||
		strings.Contains(lower, "cve"):
		return CategorySecurity
	case strings.Contains(lower, "perf") || strings.Contains(lower, "optimize") ||
		strings.Contains(lower, "speed"):
		return CategoryPerformance
	case strings.Contains(lower, "refactor") || strings.Contains(lower, "restructur") ||
		strings.Contains(lower, "clean up"):
		return CategoryRefactor
	case strings.HasPrefix(lower, "doc") || strings.Contains(lower, "documentation") ||
		strings.Contains(lower, "readme") || strings.Contains(lower, "changelog"):
		return CategoryDocs
	case strings.Contains(lower, "test"):
		return CategoryTests
	case strings.HasPrefix(lower, "build") || strings.HasPrefix(lower, "ci") ||
		strings.Contains(lower, "ci/cd") || strings.Contains(lower, "makefile") ||
		strings.Contains(lower, "goreleaser"):
		return CategoryBuild
	default:
		return CategoryOther
	}
}

// Render generates the release notes as a markdown string.
func (rn *ReleaseNotes) Render(opts Options) string {
	var buf strings.Builder

	// Header
	dateStr := rn.Date.Format("2006-01-02")
	buf.WriteString(fmt.Sprintf("# Release Notes: %s\n\n", rn.Version))
	buf.WriteString(fmt.Sprintf("**Date:** %s\n", dateStr))
	if rn.PrevTag != "" {
		buf.WriteString(fmt.Sprintf("**Compared to:** %s\n", rn.PrevTag))
	}
	buf.WriteString(fmt.Sprintf("**Commits:** %d\n\n", len(rn.AllCommits)))

	if !opts.GroupByCategory {
		// Flat list
		for _, c := range rn.AllCommits {
			buf.WriteString(formatCommitLine(c, opts))
		}
		return buf.String()
	}

	// Grouped by category
	for _, cat := range categoryOrder {
		commits, ok := rn.Categories[cat]
		if !ok || len(commits) == 0 {
			continue
		}

		// Deduplicate: breaking commits appear under both Breaking and their own category
		if cat == CategoryBreaking {
			buf.WriteString(fmt.Sprintf("## %s\n\n", cat))
		} else {
			// Filter out commits already listed under Breaking
			filtered := filterNonBreaking(commits)
			if len(filtered) == 0 {
				continue
			}
			commits = filtered
			buf.WriteString(fmt.Sprintf("## %s\n\n", cat))
		}

		// Sort commits by date descending
		sort.Slice(commits, func(i, j int) bool {
			return commits[i].Date.After(commits[j].Date)
		})

		for _, c := range commits {
			buf.WriteString(formatCommitLine(c, opts))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// filterNonBreaking removes commits that are marked as breaking changes.
func filterNonBreaking(commits []Commit) []Commit {
	var out []Commit
	for _, c := range commits {
		if !c.Breaking {
			out = append(out, c)
		}
	}
	return out
}

// formatCommitLine formats a single commit as a markdown list item.
func formatCommitLine(c Commit, opts Options) string {
	subject := c.Subject

	// Strip conventional commit prefix for cleaner output
	if m := conventionalCommitRe.FindStringSubmatch(subject); m != nil {
		subject = m[4] // The description part after "type(scope): "
	}

	// Capitalize first letter
	if len(subject) > 0 {
		subject = strings.ToUpper(subject[:1]) + subject[1:]
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("- %s", subject))

	if c.Scope != "" {
		parts = append(parts, fmt.Sprintf("(%s)", c.Scope))
	}

	if opts.IncludeCommitHashes {
		parts = append(parts, fmt.Sprintf("[`%s`]", c.ShortHash))
	}

	if opts.IncludeAuthors {
		parts = append(parts, fmt.Sprintf("— %s", c.Author))
	}

	return strings.Join(parts, " ") + "\n"
}
