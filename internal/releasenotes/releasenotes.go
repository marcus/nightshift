// Package releasenotes collects git commits between two refs, parses
// conventional commit messages, groups them by type, and renders
// formatted markdown release notes matching the CHANGELOG.md style.
package releasenotes

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// CommitType represents the conventional commit type.
type CommitType int

const (
	TypeFeature CommitType = iota
	TypeFix
	TypeOther
)

// Commit holds a parsed conventional commit.
type Commit struct {
	Hash        string
	Type        CommitType
	RawType     string
	Scope       string
	Description string
	PRNumber    string
	IssueRefs   []string
	Author      string
	Breaking    bool
}

// Group is a labeled set of commits for rendering.
type Group struct {
	Title   string
	Commits []Commit
}

// Options configures the release notes generation.
type Options struct {
	From    string // start ref (tag or sha), default: latest tag
	To      string // end ref, default: HEAD
	Version string // version label for the header
	RepoDir string // git working directory (empty = cwd)
}

// commitDelimiter separates fields in the git log format string.
const commitDelimiter = "---FIELD---"

// logFormat is the git log --format string we use.
var logFormat = strings.Join([]string{"%H", "%s", "%an"}, commitDelimiter)

// conventionalRe matches a conventional commit subject line.
// Groups: 1=type, 2=scope (optional, with parens), 3=breaking marker, 4=description
var conventionalRe = regexp.MustCompile(`^(\w+)(?:\(([^)]*)\))?(!)?\s*:\s*(.+)$`)

// prRefRe matches a trailing (#123) PR reference.
var prRefRe = regexp.MustCompile(`\(#(\d+)\)\s*$`)

// issueRefRe matches "fixes #N" or "closes #N" references in the description.
var issueRefRe = regexp.MustCompile(`(?i)(?:fix(?:es)?|close(?:s)?|resolve(?:s)?)\s+#(\d+)`)

// Generate produces the full release notes markdown string.
func Generate(opts Options) (string, error) {
	raw, err := collectCommits(opts)
	if err != nil {
		return "", err
	}
	commits := ParseCommits(raw)
	if len(commits) == 0 {
		return "", fmt.Errorf("no commits found between %s and %s", opts.From, opts.To)
	}
	groups := GroupCommits(commits)
	return Render(opts.Version, groups), nil
}

// collectCommits shells out to git log and returns the raw output lines.
func collectCommits(opts Options) ([]string, error) {
	from := opts.From
	if from == "" {
		tag, err := latestTag(opts.RepoDir)
		if err != nil {
			return nil, fmt.Errorf("determine latest tag: %w", err)
		}
		from = tag
	}
	to := opts.To
	if to == "" {
		to = "HEAD"
	}

	rangeArg := from + ".." + to
	args := []string{"log", "--format=" + logFormat, rangeArg}
	cmd := exec.Command("git", args...)
	if opts.RepoDir != "" {
		cmd.Dir = opts.RepoDir
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log %s: %w", rangeArg, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}

// latestTag returns the most recent reachable tag.
func latestTag(dir string) (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no tags found: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ParseCommits parses raw git log lines into structured Commits.
func ParseCommits(lines []string) []Commit {
	var commits []Commit
	for _, line := range lines {
		if line == "" {
			continue
		}
		c := parseLine(line)
		commits = append(commits, c)
	}
	return commits
}

func parseLine(line string) Commit {
	parts := strings.SplitN(line, commitDelimiter, 3)
	var hash, subject, author string
	if len(parts) >= 1 {
		hash = parts[0]
	}
	if len(parts) >= 2 {
		subject = parts[1]
	}
	if len(parts) >= 3 {
		author = parts[2]
	}

	c := Commit{
		Hash:   hash,
		Author: author,
	}

	// Try to extract PR number from subject
	if m := prRefRe.FindStringSubmatch(subject); m != nil {
		c.PRNumber = m[1]
		subject = strings.TrimSpace(prRefRe.ReplaceAllString(subject, ""))
	}

	// Try conventional commit format
	if m := conventionalRe.FindStringSubmatch(subject); m != nil {
		c.RawType = strings.ToLower(m[1])
		c.Scope = m[2]
		c.Breaking = m[3] == "!"
		c.Description = m[4]
		c.Type = classifyType(c.RawType)
	} else {
		// Not a conventional commit — goes into Other
		c.RawType = ""
		c.Description = subject
		c.Type = TypeOther
	}

	// Extract issue references from description
	if matches := issueRefRe.FindAllStringSubmatch(c.Description, -1); matches != nil {
		for _, m := range matches {
			c.IssueRefs = append(c.IssueRefs, m[1])
		}
	}

	return c
}

func classifyType(t string) CommitType {
	switch t {
	case "feat":
		return TypeFeature
	case "fix":
		return TypeFix
	default:
		return TypeOther
	}
}

// GroupCommits buckets commits by type into ordered groups.
// Empty groups are omitted.
func GroupCommits(commits []Commit) []Group {
	buckets := map[CommitType][]Commit{}
	for _, c := range commits {
		buckets[c.Type] = append(buckets[c.Type], c)
	}

	order := []struct {
		t     CommitType
		title string
	}{
		{TypeFeature, "Features"},
		{TypeFix, "Bug Fixes"},
		{TypeOther, "Other"},
	}

	var groups []Group
	for _, o := range order {
		if cs, ok := buckets[o.t]; ok && len(cs) > 0 {
			groups = append(groups, Group{Title: o.title, Commits: cs})
		}
	}
	return groups
}

// Render formats grouped commits as markdown release notes.
func Render(version string, groups []Group) string {
	var b strings.Builder

	if version == "" {
		version = "Unreleased"
	}

	// Version header
	date := time.Now().Format("2006-01-02")
	if version == "Unreleased" {
		fmt.Fprintf(&b, "## [%s]\n", version)
	} else {
		fmt.Fprintf(&b, "## [%s] - %s\n", version, date)
	}

	for _, g := range groups {
		fmt.Fprintf(&b, "\n### %s\n", g.Title)
		for _, c := range g.Commits {
			b.WriteString(renderCommit(c))
		}
	}

	return b.String()
}

func renderCommit(c Commit) string {
	var b strings.Builder
	b.WriteString("- ")

	// Bold title from scope or first phrase
	title := c.Description
	if c.Scope != "" {
		title = c.Scope
	}
	fmt.Fprintf(&b, "**%s**", title)

	// Description after scope
	if c.Scope != "" {
		fmt.Fprintf(&b, " — %s", c.Description)
	}

	// PR link
	if c.PRNumber != "" {
		fmt.Fprintf(&b, " (#%s)", c.PRNumber)
	}

	// Breaking change marker
	if c.Breaking {
		b.WriteString(" **BREAKING**")
	}

	b.WriteString("\n")
	return b.String()
}
