// Package changelog synthesizes changelogs from git commit history.
package changelog

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Category represents a changelog section.
type Category string

const (
	CategoryFeatures Category = "Features"
	CategoryFixes    Category = "Bug Fixes"
	CategoryDocs     Category = "Documentation"
	CategoryOther    Category = "Other"
)

// CommitInfo holds parsed data from a single git commit.
type CommitInfo struct {
	Hash    string
	Subject string
	Prefix  string
	Scope   string
	Title   string
	PR      int
}

// CategoryGroup holds commits grouped under a changelog category.
type CategoryGroup struct {
	Category Category
	Commits  []CommitInfo
}

// prefixToCategory maps conventional commit prefixes to changelog categories.
var prefixToCategory = map[string]Category{
	"feat":     CategoryFeatures,
	"fix":      CategoryFixes,
	"docs":     CategoryDocs,
	"refactor": CategoryOther,
	"chore":    CategoryOther,
	"ci":       CategoryOther,
	"test":     CategoryOther,
}

var (
	// Matches: feat(scope): title  or  fix: title
	conventionalRe = regexp.MustCompile(`^(\w+)(?:\(([^)]*)\))?:\s*(.+)$`)
	// Matches: (#123) anywhere in the string
	prNumberRe = regexp.MustCompile(`\(#(\d+)\)`)
)

// ParseCommit extracts structured info from a conventional commit subject line.
func ParseCommit(hash, subject string) CommitInfo {
	ci := CommitInfo{Hash: hash, Subject: subject}

	// Extract PR number
	if m := prNumberRe.FindStringSubmatch(subject); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &ci.PR)
	}

	// Parse conventional commit prefix
	if m := conventionalRe.FindStringSubmatch(subject); len(m) > 1 {
		ci.Prefix = m[1]
		ci.Scope = m[2]
		ci.Title = m[3]
	} else {
		ci.Title = subject
	}

	return ci
}

// ClassifyCommit returns the changelog category for a commit prefix.
func ClassifyCommit(prefix string) Category {
	if cat, ok := prefixToCategory[strings.ToLower(prefix)]; ok {
		return cat
	}
	return CategoryOther
}

// Generator produces changelogs from git history.
type Generator struct {
	RepoDir string
}

// NewGenerator creates a Generator for the given repository directory.
func NewGenerator(repoDir string) *Generator {
	return &Generator{RepoDir: repoDir}
}

// Generate reads commits between fromRef and toRef and returns grouped categories.
func (g *Generator) Generate(fromRef, toRef string) ([]CategoryGroup, error) {
	subjects, err := g.gitLog(fromRef, toRef)
	if err != nil {
		return nil, err
	}
	return GroupCommits(subjects), nil
}

// gitLog shells out to git log and returns hash+subject lines.
func (g *Generator) gitLog(fromRef, toRef string) ([]string, error) {
	rangeArg := fromRef + ".." + toRef
	cmd := exec.Command("git", "log", "--pretty=format:%H %s", rangeArg)
	cmd.Dir = g.RepoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log %s: %w", rangeArg, err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// LatestTag returns the most recent reachable tag, or "" if none.
func (g *Generator) LatestTag() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = g.RepoDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GroupCommits parses raw "hash subject" lines and groups them by category.
func GroupCommits(lines []string) []CategoryGroup {
	groups := map[Category][]CommitInfo{}
	for _, line := range lines {
		hash, subject, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		ci := ParseCommit(hash, subject)
		cat := ClassifyCommit(ci.Prefix)
		groups[cat] = append(groups[cat], ci)
	}
	return sortedGroups(groups)
}

// sortedGroups returns categories in a stable display order.
func sortedGroups(m map[Category][]CommitInfo) []CategoryGroup {
	order := []Category{CategoryFeatures, CategoryFixes, CategoryDocs, CategoryOther}
	var result []CategoryGroup
	for _, cat := range order {
		if commits, ok := m[cat]; ok {
			result = append(result, CategoryGroup{Category: cat, Commits: commits})
		}
	}
	// Any unknown categories (shouldn't happen, but be safe)
	seen := map[Category]bool{}
	for _, c := range order {
		seen[c] = true
	}
	var extra []Category
	for c := range m {
		if !seen[c] {
			extra = append(extra, c)
		}
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i] < extra[j] })
	for _, c := range extra {
		result = append(result, CategoryGroup{Category: c, Commits: m[c]})
	}
	return result
}

// RenderMarkdown formats grouped commits as markdown matching CHANGELOG.md style.
func RenderMarkdown(version string, groups []CategoryGroup) string {
	var sb strings.Builder
	if version != "" {
		date := time.Now().Format("2006-01-02")
		fmt.Fprintf(&sb, "## [%s] - %s\n", version, date)
	}
	for _, g := range groups {
		sb.WriteString("\n### ")
		sb.WriteString(string(g.Category))
		sb.WriteString("\n")
		for _, c := range g.Commits {
			sb.WriteString("- ")
			title := strings.TrimSpace(c.Title)
			// Strip trailing PR reference from title since we append it
			title = prNumberRe.ReplaceAllString(title, "")
			title = strings.TrimSpace(title)
			sb.WriteString("**")
			sb.WriteString(title)
			sb.WriteString("**")
			if c.PR > 0 {
				fmt.Fprintf(&sb, " (#%d)", c.PR)
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// RenderPlain formats grouped commits as plain text.
func RenderPlain(version string, groups []CategoryGroup) string {
	var sb strings.Builder
	if version != "" {
		fmt.Fprintf(&sb, "%s\n", version)
		sb.WriteString(strings.Repeat("=", len(version)))
		sb.WriteString("\n")
	}
	for _, g := range groups {
		sb.WriteString("\n")
		sb.WriteString(string(g.Category))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("-", len(string(g.Category))))
		sb.WriteString("\n")
		for _, c := range g.Commits {
			title := strings.TrimSpace(c.Title)
			title = prNumberRe.ReplaceAllString(title, "")
			title = strings.TrimSpace(title)
			sb.WriteString("- ")
			sb.WriteString(title)
			if c.PR > 0 {
				fmt.Fprintf(&sb, " (#%d)", c.PR)
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
