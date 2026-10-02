// Package projects handles multi-project discovery, resolution, and budget allocation.
// Supports explicit project paths, glob patterns, and priority-based budget splitting.
package projects

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/state"
	"github.com/spf13/viper"
)

// Project represents a resolved project with merged configuration.
type Project struct {
	Path     string         // Absolute path to project
	Priority int            // Priority for ordering (higher = more important)
	Config   *config.Config // Merged configuration for this project
	Weight   float64        // Normalized weight for budget allocation
}

// Resolver handles project discovery and configuration merging.
type Resolver struct {
	globalCfg *config.Config
}

// NewResolver creates a resolver with the given global configuration.
func NewResolver(globalCfg *config.Config) *Resolver {
	return &Resolver{globalCfg: globalCfg}
}

// DiscoverProjects resolves all projects from configuration.
// Expands glob patterns, excludes specified paths, and merges per-project configs.
func (r *Resolver) DiscoverProjects() ([]Project, error) {
	var projects []Project

	for _, pc := range r.globalCfg.Projects {
		if pc.Pattern != "" {
			// Glob pattern discovery
			matches, err := ExpandGlobPatterns([]string{pc.Pattern}, pc.Exclude)
			if err != nil {
				return nil, err
			}
			for _, path := range matches {
				proj, err := r.resolveProject(path, pc.Priority)
				if err != nil {
					continue // Skip invalid projects
				}
				projects = append(projects, proj)
			}
		} else if pc.Path != "" {
			// Explicit path
			path := expandPath(pc.Path)
			proj, err := r.resolveProject(path, pc.Priority)
			if err != nil {
				continue // Skip invalid projects
			}
			projects = append(projects, proj)
		}
	}

	return projects, nil
}

// resolveProject creates a Project with merged configuration.
func (r *Resolver) resolveProject(path string, priority int) (Project, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return Project{}, err
	}

	// Verify directory exists
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return Project{}, os.ErrNotExist
	}

	// Merge project config with global
	mergedCfg, err := MergeProjectConfig(r.globalCfg, path)
	if err != nil {
		// Use global config if merge fails
		mergedCfg = r.globalCfg
	}

	return Project{
		Path:     path,
		Priority: priority,
		Config:   mergedCfg,
	}, nil
}

// ExpandGlobPatterns expands glob patterns and filters out excluded paths.
func ExpandGlobPatterns(patterns, excludes []string) ([]string, error) {
	var results []string
	excludeSet := make(map[string]bool)

	// Build exclude set (expanded)
	for _, exc := range excludes {
		excPath := expandPath(exc)
		excPath, err := filepath.Abs(excPath)
		if err != nil {
			continue
		}
		excludeSet[excPath] = true
	}

	for _, pattern := range patterns {
		pattern = expandPath(pattern)

		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			absMatch, err := filepath.Abs(match)
			if err != nil {
				continue
			}

			// Check if excluded
			if excludeSet[absMatch] {
				continue
			}

			// Check if any exclude is a parent
			excluded := false
			for exc := range excludeSet {
				if strings.HasPrefix(absMatch, exc+string(os.PathSeparator)) {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}

			// Verify it's a directory
			info, err := os.Stat(absMatch)
			if err != nil || !info.IsDir() {
				continue
			}

			results = append(results, absMatch)
		}
	}

	return results, nil
}

// SortByPriority orders projects by priority (highest first).
func SortByPriority(projects []Project) []Project {
	sorted := make([]Project, len(projects))
	copy(sorted, projects)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})
	return sorted
}

// BudgetAllocation contains budget info for a project.
type BudgetAllocation struct {
	Project    Project
	Tokens     int64   // Allocated token budget
	Percentage float64 // Percentage of total budget
}

// AllocateBudget distributes the total budget across projects by priority weight.
// Projects with higher priority get proportionally more budget.
func AllocateBudget(projects []Project, totalBudget int64) []BudgetAllocation {
	if len(projects) == 0 || totalBudget <= 0 {
		return nil
	}

	// Calculate total priority weight (use priority+1 to avoid zero weight)
	var totalWeight float64
	for i := range projects {
		projects[i].Weight = float64(projects[i].Priority + 1)
		totalWeight += projects[i].Weight
	}

	// Normalize weights
	for i := range projects {
		projects[i].Weight /= totalWeight
	}

	// Allocate budget
	allocations := make([]BudgetAllocation, len(projects))
	var allocated int64

	for i, proj := range projects {
		tokens := int64(float64(totalBudget) * proj.Weight)
		if i == len(projects)-1 {
			// Give remainder to last project to avoid rounding loss
			tokens = totalBudget - allocated
		}
		allocations[i] = BudgetAllocation{
			Project:    proj,
			Tokens:     tokens,
			Percentage: proj.Weight * 100,
		}
		allocated += tokens
	}

	return allocations
}

// FilterProcessedToday removes projects that were already processed today.
func FilterProcessedToday(projects []Project, s *state.State) []Project {
	var filtered []Project
	for _, p := range projects {
		if !s.WasProcessedToday(p.Path) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FilterNotProcessedSince returns projects not processed within the given duration.
func FilterNotProcessedSince(projects []Project, s *state.State, since time.Duration) []Project {
	cutoff := time.Now().Add(-since)
	var filtered []Project
	for _, p := range projects {
		lastRun := s.LastProjectRun(p.Path)
		if lastRun.IsZero() || lastRun.Before(cutoff) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// MergeProjectConfig loads and merges a per-project config with the global config.
// Per-project config overrides global settings.
func MergeProjectConfig(globalCfg *config.Config, projectPath string) (*config.Config, error) {
	projectConfigPath := filepath.Join(projectPath, config.ProjectConfigName)

	// Check if project config exists
	if _, err := os.Stat(projectConfigPath); os.IsNotExist(err) {
		return globalCfg, nil
	}

	// Load project config
	v := viper.New()
	v.SetConfigFile(projectConfigPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// Start with a copy of global config
	merged := *globalCfg

	// Merge specific fields from project config
	if v.IsSet("schedule.cron") {
		merged.Schedule.Cron = v.GetString("schedule.cron")
	}
	if v.IsSet("schedule.interval") {
		merged.Schedule.Interval = v.GetString("schedule.interval")
	}

	if v.IsSet("budget.max_percent") {
		merged.Budget.MaxPercent = v.GetInt("budget.max_percent")
	}
	if v.IsSet("budget.reserve_percent") {
		merged.Budget.ReservePercent = v.GetInt("budget.reserve_percent")
	}

	if v.IsSet("tasks.enabled") {
		merged.Tasks.Enabled = v.GetStringSlice("tasks.enabled")
	}
	if v.IsSet("tasks.disabled") {
		merged.Tasks.Disabled = v.GetStringSlice("tasks.disabled")
	}
	if v.IsSet("tasks.priorities") {
		intMap := make(map[string]int)
		for k, val := range v.GetStringMap("tasks.priorities") {
			switch v := val.(type) {
			case int:
				intMap[k] = v
			case int64:
				intMap[k] = int(v)
			case float64:
				intMap[k] = int(v)
			}
		}
		merged.Tasks.Priorities = intMap
	}

	if v.IsSet("providers.claude.enabled") {
		merged.Providers.Claude.Enabled = v.GetBool("providers.claude.enabled")
	}
	if v.IsSet("providers.codex.enabled") {
		merged.Providers.Codex.Enabled = v.GetBool("providers.codex.enabled")
	}

	if v.IsSet("logging.level") {
		merged.Logging.Level = v.GetString("logging.level")
	}

	return &merged, nil
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// ProjectSummary provides summary info for a project.
type ProjectSummary struct {
	Path           string
	Priority       int
	LastRun        time.Time
	RunCount       int
	ProcessedToday bool
}

// GetProjectSummaries returns summary info for all projects.
func GetProjectSummaries(projects []Project, s *state.State) []ProjectSummary {
	summaries := make([]ProjectSummary, len(projects))
	for i, p := range projects {
		ps := s.GetProjectState(p.Path)
		summaries[i] = ProjectSummary{
			Path:           p.Path,
			Priority:       p.Priority,
			ProcessedToday: s.WasProcessedToday(p.Path),
		}
		if ps != nil {
			summaries[i].LastRun = ps.LastRun
			summaries[i].RunCount = ps.RunCount
		}
	}
	return summaries
}

// SelectNext picks the next project to process based on priority and staleness.
// Returns nil if no projects are available.
func SelectNext(projects []Project, s *state.State) *Project {
	// Filter already processed today
	available := FilterProcessedToday(projects, s)
	if len(available) == 0 {
		return nil
	}

	// Sort by priority
	sorted := SortByPriority(available)

	// Among equal priority, prefer more stale
	slices.SortStableFunc(sorted, func(a, b Project) int {
		if a.Priority != b.Priority {
			return 0 // Already sorted by priority
		}
		// More stale (earlier last run) comes first
		aLast := s.LastProjectRun(a.Path)
		bLast := s.LastProjectRun(b.Path)
		if aLast.Before(bLast) {
			return -1
		}
		if aLast.After(bLast) {
			return 1
		}
		return 0
	})

	return &sorted[0]
}

// IsProjectPath checks if a path looks like a valid project directory.
// Checks for common project indicators like .git, go.mod, package.json, etc.
func IsProjectPath(path string) bool {
	indicators := []string{
		".git",
		"go.mod",
		"package.json",
		"Cargo.toml",
		"pyproject.toml",
		"requirements.txt",
		"Makefile",
		".nightshift.yaml",
	}

	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(path, ind)); err == nil {
			return true
		}
	}
	return false
}

// DiscoverProjectsInDir finds projects in a directory (non-recursive).
func DiscoverProjectsInDir(dir string) ([]string, error) {
	dir = expandPath(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if IsProjectPath(path) {
			projects = append(projects, path)
		}
	}
	return projects, nil
}
