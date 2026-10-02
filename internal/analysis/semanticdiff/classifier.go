package semanticdiff

import (
	"path/filepath"
	"strings"
)

// Category represents a semantic category for a change.
type Category string

const (
	CategoryFeature  Category = "feature"
	CategoryBugfix   Category = "bugfix"
	CategoryRefactor Category = "refactor"
	CategoryDeps     Category = "dependency-update"
	CategoryConfig   Category = "config-change"
	CategoryTest     Category = "test-change"
	CategoryDocs     Category = "docs-change"
	CategoryCleanup  Category = "cleanup"
	CategoryUnknown  Category = "uncategorized"
)

// ClassifiedChange pairs a file change with its semantic category.
type ClassifiedChange struct {
	File     FileChange
	Category Category
	Reason   string // short explanation of why this category was chosen
}

// ClassifiedCommit pairs a commit with its primary category.
type ClassifiedCommit struct {
	Commit   Commit
	Category Category
	Reason   string
	Files    []ClassifiedChange
}

// ClassifyChangeSet classifies every commit and file in the change set.
func ClassifyChangeSet(cs *ChangeSet) []ClassifiedCommit {
	var result []ClassifiedCommit
	for _, c := range cs.Commits {
		cc := classifyCommit(c)
		result = append(result, cc)
	}
	return result
}

// classifyCommit determines the primary category of a commit.
func classifyCommit(c Commit) ClassifiedCommit {
	var classified []ClassifiedChange
	for _, f := range c.Files {
		cat, reason := classifyFile(f, c.Subject, c.Body)
		classified = append(classified, ClassifiedChange{
			File:     f,
			Category: cat,
			Reason:   reason,
		})
	}

	primary, reason := classifyFromMessage(c.Subject, c.Body)
	if primary == CategoryUnknown && len(classified) > 0 {
		primary, reason = majorityCategory(classified)
	}

	return ClassifiedCommit{
		Commit:   c,
		Category: primary,
		Reason:   reason,
		Files:    classified,
	}
}

// classifyFile determines the category of a single file change.
func classifyFile(f FileChange, subject, body string) (Category, string) {
	path := strings.ToLower(f.Path)
	base := strings.ToLower(filepath.Base(f.Path))
	ext := strings.ToLower(filepath.Ext(f.Path))

	// Test files
	if isTestFile(path, base) {
		return CategoryTest, "test file path"
	}

	// Documentation
	if isDocsFile(path, base, ext) {
		return CategoryDocs, "documentation file"
	}

	// Dependencies
	if isDepsFile(base) {
		return CategoryDeps, "dependency manifest"
	}

	// Configuration
	if isConfigFile(path, base, ext) {
		return CategoryConfig, "configuration file"
	}

	// Fall back to commit message analysis.
	cat, reason := classifyFromMessage(subject, body)
	if cat != CategoryUnknown {
		return cat, reason
	}

	// Heuristic: deletion-heavy changes are often cleanup.
	if f.Deletions > 0 && f.Additions == 0 {
		return CategoryCleanup, "pure deletion"
	}

	return CategoryUnknown, ""
}

// classifyFromMessage determines category from commit message patterns.
func classifyFromMessage(subject, body string) (Category, string) {
	msg := strings.ToLower(subject + " " + body)

	// Conventional commit prefixes
	subjectLower := strings.ToLower(subject)
	if strings.HasPrefix(subjectLower, "fix:") || strings.HasPrefix(subjectLower, "fix(") {
		return CategoryBugfix, "conventional commit: fix"
	}
	if strings.HasPrefix(subjectLower, "feat:") || strings.HasPrefix(subjectLower, "feat(") {
		return CategoryFeature, "conventional commit: feat"
	}
	if strings.HasPrefix(subjectLower, "docs:") || strings.HasPrefix(subjectLower, "docs(") {
		return CategoryDocs, "conventional commit: docs"
	}
	if strings.HasPrefix(subjectLower, "test:") || strings.HasPrefix(subjectLower, "test(") {
		return CategoryTest, "conventional commit: test"
	}
	if strings.HasPrefix(subjectLower, "refactor:") || strings.HasPrefix(subjectLower, "refactor(") {
		return CategoryRefactor, "conventional commit: refactor"
	}
	if strings.HasPrefix(subjectLower, "chore:") || strings.HasPrefix(subjectLower, "chore(") {
		return CategoryCleanup, "conventional commit: chore"
	}
	if strings.HasPrefix(subjectLower, "build:") || strings.HasPrefix(subjectLower, "build(") || strings.HasPrefix(subjectLower, "ci:") || strings.HasPrefix(subjectLower, "ci(") {
		return CategoryConfig, "conventional commit: build/ci"
	}
	if strings.HasPrefix(subjectLower, "deps:") || strings.HasPrefix(subjectLower, "deps(") {
		return CategoryDeps, "conventional commit: deps"
	}

	// Keyword patterns in message
	bugKeywords := []string{"fix", "bug", "patch", "hotfix", "resolve", "issue"}
	for _, kw := range bugKeywords {
		if strings.Contains(msg, kw) {
			return CategoryBugfix, "keyword: " + kw
		}
	}

	featureKeywords := []string{"add ", "new ", "feature", "implement", "introduce"}
	for _, kw := range featureKeywords {
		if strings.Contains(msg, kw) {
			return CategoryFeature, "keyword: " + strings.TrimSpace(kw)
		}
	}

	refactorKeywords := []string{"refactor", "restructure", "reorganize", "simplify", "extract", "rename"}
	for _, kw := range refactorKeywords {
		if strings.Contains(msg, kw) {
			return CategoryRefactor, "keyword: " + kw
		}
	}

	depKeywords := []string{"bump", "upgrade", "dependency", "dependabot", "go.sum", "go.mod"}
	for _, kw := range depKeywords {
		if strings.Contains(msg, kw) {
			return CategoryDeps, "keyword: " + kw
		}
	}

	cleanupKeywords := []string{"cleanup", "clean up", "remove unused", "delete", "deprecate", "drop"}
	for _, kw := range cleanupKeywords {
		if strings.Contains(msg, kw) {
			return CategoryCleanup, "keyword: " + kw
		}
	}

	return CategoryUnknown, ""
}

func isTestFile(path, base string) bool {
	if strings.Contains(path, "_test.go") || strings.Contains(path, "_test.") {
		return true
	}
	if strings.Contains(path, "/test/") || strings.Contains(path, "/tests/") || strings.Contains(path, "/testdata/") {
		return true
	}
	if strings.HasPrefix(path, "test/") || strings.HasPrefix(path, "tests/") || strings.HasPrefix(path, "testdata/") {
		return true
	}
	if strings.HasSuffix(base, ".test.js") || strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".spec.ts") {
		return true
	}
	return false
}

func isDocsFile(path, base, ext string) bool {
	if ext == ".md" || ext == ".rst" || ext == ".txt" || ext == ".adoc" {
		return true
	}
	if strings.Contains(path, "/docs/") || strings.Contains(path, "/doc/") {
		return true
	}
	if base == "readme" || base == "readme.md" || base == "changelog" || base == "changelog.md" || base == "license" || base == "license.md" {
		return true
	}
	return false
}

func isDepsFile(base string) bool {
	depFiles := []string{
		"go.mod", "go.sum",
		"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"requirements.txt", "poetry.lock", "pipfile.lock",
		"gemfile.lock", "cargo.lock", "cargo.toml",
		"composer.lock",
	}
	for _, df := range depFiles {
		if base == df {
			return true
		}
	}
	return false
}

func isConfigFile(path, base, ext string) bool {
	if ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".ini" {
		if !strings.Contains(path, "/test") && !strings.Contains(path, "/doc") {
			return true
		}
	}
	configFiles := []string{
		"dockerfile", ".dockerignore", "docker-compose.yml", "docker-compose.yaml",
		"makefile", ".goreleaser.yml", ".goreleaser.yaml", ".golangci.yml",
		".gitignore", ".editorconfig", ".eslintrc", ".prettierrc",
		"tsconfig.json", "webpack.config.js",
	}
	for _, cf := range configFiles {
		if base == cf {
			return true
		}
	}
	if strings.Contains(path, ".github/") || strings.Contains(path, ".circleci/") {
		return true
	}
	return false
}

// majorityCategory returns the most common category among classified changes.
func majorityCategory(changes []ClassifiedChange) (Category, string) {
	counts := make(map[Category]int)
	for _, c := range changes {
		counts[c.Category]++
	}
	best := CategoryUnknown
	bestCount := 0
	for cat, count := range counts {
		if count > bestCount {
			best = cat
			bestCount = count
		}
	}
	return best, "majority of files"
}
