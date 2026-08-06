package semanticdiff

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Report is the structured result of a semantic diff analysis.
type Report struct {
	BaseRef    string          `json:"base_ref"`
	HeadRef    string          `json:"head_ref"`
	Summary    string          `json:"summary"`
	Categories []CategoryGroup `json:"categories"`
	Highlights []string        `json:"highlights"`
	Stats      DiffStats       `json:"stats"`
}

// CategoryGroup groups changes under one semantic category.
type CategoryGroup struct {
	Category   Category           `json:"category"`
	Label      string             `json:"label"`
	Commits    []ClassifiedCommit `json:"commits"`
	FilesCount int                `json:"files_count"`
	Additions  int                `json:"additions"`
	Deletions  int                `json:"deletions"`
}

// DiffStats contains aggregate statistics.
type DiffStats struct {
	TotalCommits int `json:"total_commits"`
	TotalFiles   int `json:"total_files"`
	TotalAdded   int `json:"total_added"`
	TotalDeleted int `json:"total_deleted"`
}

// GenerateReport builds a Report from a classified change set.
func GenerateReport(cs *ChangeSet, classified []ClassifiedCommit) *Report {
	groups := groupByCategory(classified)
	stats := DiffStats{
		TotalCommits: len(cs.Commits),
		TotalFiles:   len(cs.AllFiles),
		TotalAdded:   cs.TotalAdditions(),
		TotalDeleted: cs.TotalDeletions(),
	}
	highlights := detectHighlights(cs, classified)
	summary := buildSummary(groups, stats)

	return &Report{
		BaseRef:    cs.BaseRef,
		HeadRef:    cs.HeadRef,
		Summary:    summary,
		Categories: groups,
		Highlights: highlights,
		Stats:      stats,
	}
}

// groupByCategory groups classified commits by their primary category.
func groupByCategory(classified []ClassifiedCommit) []CategoryGroup {
	catMap := make(map[Category]*CategoryGroup)
	order := []Category{}

	for _, cc := range classified {
		g, ok := catMap[cc.Category]
		if !ok {
			g = &CategoryGroup{
				Category: cc.Category,
				Label:    categoryLabel(cc.Category),
			}
			catMap[cc.Category] = g
			order = append(order, cc.Category)
		}
		g.Commits = append(g.Commits, cc)
		for _, f := range cc.Files {
			g.FilesCount++
			g.Additions += f.File.Additions
			g.Deletions += f.File.Deletions
		}
	}

	// Sort by priority.
	sort.Slice(order, func(i, j int) bool {
		return categoryPriority(order[i]) < categoryPriority(order[j])
	})

	groups := make([]CategoryGroup, 0, len(order))
	for _, cat := range order {
		groups = append(groups, *catMap[cat])
	}
	return groups
}

func categoryLabel(c Category) string {
	switch c {
	case CategoryFeature:
		return "New Features"
	case CategoryBugfix:
		return "Bug Fixes"
	case CategoryRefactor:
		return "Refactoring"
	case CategoryDeps:
		return "Dependency Updates"
	case CategoryConfig:
		return "Configuration Changes"
	case CategoryTest:
		return "Test Changes"
	case CategoryDocs:
		return "Documentation"
	case CategoryCleanup:
		return "Cleanup"
	case CategoryUnknown:
		return "Other Changes"
	default:
		return string(c)
	}
}

func categoryPriority(c Category) int {
	switch c {
	case CategoryFeature:
		return 0
	case CategoryBugfix:
		return 1
	case CategoryRefactor:
		return 2
	case CategoryDeps:
		return 3
	case CategoryConfig:
		return 4
	case CategoryTest:
		return 5
	case CategoryDocs:
		return 6
	case CategoryCleanup:
		return 7
	default:
		return 8
	}
}

// detectHighlights identifies high-impact changes.
func detectHighlights(cs *ChangeSet, classified []ClassifiedCommit) []string {
	var highlights []string

	for _, f := range cs.AllFiles {
		path := strings.ToLower(f.Path)

		// API surface changes
		if strings.Contains(path, "api") || strings.Contains(path, "handler") || strings.Contains(path, "route") {
			highlights = append(highlights, fmt.Sprintf("API surface changed: %s (+%d/-%d)", f.Path, f.Additions, f.Deletions))
		}

		// Schema/migration changes
		if strings.Contains(path, "migration") || strings.Contains(path, "schema") {
			highlights = append(highlights, fmt.Sprintf("Schema/migration changed: %s", f.Path))
		}

		// Security-sensitive files
		if strings.Contains(path, "auth") || strings.Contains(path, "security") || strings.Contains(path, "crypto") || strings.Contains(path, "permission") {
			highlights = append(highlights, fmt.Sprintf("Security-sensitive file changed: %s", f.Path))
		}

		// Large changes
		if f.Additions+f.Deletions > 200 {
			highlights = append(highlights, fmt.Sprintf("Large change: %s (+%d/-%d lines)", f.Path, f.Additions, f.Deletions))
		}
	}

	return dedup(highlights)
}

func dedup(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func buildSummary(groups []CategoryGroup, stats DiffStats) string {
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		parts = append(parts, fmt.Sprintf("%d %s", len(g.Commits), strings.ToLower(g.Label)))
	}
	return fmt.Sprintf("%d commits across %d files: %s. Net change: +%d/-%d lines.",
		stats.TotalCommits, stats.TotalFiles, strings.Join(parts, ", "),
		stats.TotalAdded, stats.TotalDeleted)
}

// RenderMarkdown renders the report as a markdown string.
func RenderMarkdown(r *Report) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# Semantic Diff: %s..%s\n\n", r.BaseRef, r.HeadRef)

	// TL;DR
	buf.WriteString("## TL;DR\n\n")
	fmt.Fprintf(&buf, "%s\n\n", r.Summary)

	// Stats
	buf.WriteString("## Stats\n\n")
	buf.WriteString("| Metric | Value |\n")
	buf.WriteString("|--------|-------|\n")
	fmt.Fprintf(&buf, "| Commits | %d |\n", r.Stats.TotalCommits)
	fmt.Fprintf(&buf, "| Files Changed | %d |\n", r.Stats.TotalFiles)
	fmt.Fprintf(&buf, "| Lines Added | +%d |\n", r.Stats.TotalAdded)
	fmt.Fprintf(&buf, "| Lines Deleted | -%d |\n\n", r.Stats.TotalDeleted)

	// Highlights
	if len(r.Highlights) > 0 {
		buf.WriteString("## Impact Highlights\n\n")
		for _, h := range r.Highlights {
			fmt.Fprintf(&buf, "- %s\n", h)
		}
		buf.WriteString("\n")
	}

	// Categories
	for _, g := range r.Categories {
		fmt.Fprintf(&buf, "## %s\n\n", g.Label)
		fmt.Fprintf(&buf, "*%d commits, %d files, +%d/-%d lines*\n\n", len(g.Commits), g.FilesCount, g.Additions, g.Deletions)
		for _, cc := range g.Commits {
			fmt.Fprintf(&buf, "- **%s** `%s`\n", cc.Commit.Subject, shortHash(cc.Commit.Hash))
			for _, f := range cc.Files {
				fmt.Fprintf(&buf, "  - `%s` (+%d/-%d)\n", f.File.Path, f.File.Additions, f.File.Deletions)
			}
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

func shortHash(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
