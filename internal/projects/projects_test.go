package projects

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/state"
)

func TestExpandGlobPatterns(t *testing.T) {
	// Create temp directories for testing
	tmpDir := t.TempDir()
	proj1 := filepath.Join(tmpDir, "proj1")
	proj2 := filepath.Join(tmpDir, "proj2")
	archived := filepath.Join(tmpDir, "archived")
	_ = os.Mkdir(proj1, 0755)
	_ = os.Mkdir(proj2, 0755)
	_ = os.Mkdir(archived, 0755)

	tests := []struct {
		name     string
		patterns []string
		excludes []string
		want     int
	}{
		{
			name:     "simple glob",
			patterns: []string{filepath.Join(tmpDir, "*")},
			want:     3,
		},
		{
			name:     "with exclude",
			patterns: []string{filepath.Join(tmpDir, "*")},
			excludes: []string{archived},
			want:     2,
		},
		{
			name:     "no matches",
			patterns: []string{filepath.Join(tmpDir, "nonexistent*")},
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandGlobPatterns(tt.patterns, tt.excludes)
			if err != nil {
				t.Fatalf("ExpandGlobPatterns() error = %v", err)
			}
			if len(got) != tt.want {
				t.Errorf("ExpandGlobPatterns() = %d results, want %d", len(got), tt.want)
			}
		})
	}
}

func TestSortByPriority(t *testing.T) {
	projects := []Project{
		{Path: "/low", Priority: 1},
		{Path: "/high", Priority: 10},
		{Path: "/medium", Priority: 5},
	}

	sorted := SortByPriority(projects)

	if sorted[0].Priority != 10 {
		t.Errorf("expected highest priority first, got %d", sorted[0].Priority)
	}
	if sorted[2].Priority != 1 {
		t.Errorf("expected lowest priority last, got %d", sorted[2].Priority)
	}
}

func TestAllocateBudget(t *testing.T) {
	projects := []Project{
		{Path: "/high", Priority: 9}, // weight 10
		{Path: "/low", Priority: 0},  // weight 1
	}
	// Total weight = 11, high gets 10/11 = ~90.9%, low gets 1/11 = ~9.1%

	allocs := AllocateBudget(projects, 1100)

	if len(allocs) != 2 {
		t.Fatalf("expected 2 allocations, got %d", len(allocs))
	}

	// High priority should get more
	if allocs[0].Tokens < allocs[1].Tokens {
		t.Errorf("high priority should get more tokens")
	}

	// Total should equal budget
	var total int64
	for _, a := range allocs {
		total += a.Tokens
	}
	if total != 1100 {
		t.Errorf("total allocation %d != budget 1100", total)
	}
}

func TestAllocateBudgetEmpty(t *testing.T) {
	allocs := AllocateBudget(nil, 1000)
	if allocs != nil {
		t.Errorf("expected nil for empty projects")
	}

	allocs = AllocateBudget([]Project{}, 1000)
	if allocs != nil {
		t.Errorf("expected nil for empty projects")
	}
}

func TestAllocateBudgetZeroBudget(t *testing.T) {
	projects := []Project{{Path: "/test", Priority: 1}}
	allocs := AllocateBudget(projects, 0)
	if allocs != nil {
		t.Errorf("expected nil for zero budget")
	}
}

func TestFilterProcessedToday(t *testing.T) {
	s := newTestState(t)

	projects := []Project{
		{Path: "/processed"},
		{Path: "/not-processed"},
	}

	// Mark one as processed
	s.RecordProjectRun("/processed")

	filtered := FilterProcessedToday(projects, s)
	if len(filtered) != 1 {
		t.Errorf("expected 1 project, got %d", len(filtered))
	}
	if filtered[0].Path != "/not-processed" {
		t.Errorf("expected /not-processed, got %s", filtered[0].Path)
	}
}

func TestFilterNotProcessedSince(t *testing.T) {
	s := newTestState(t)

	projects := []Project{
		{Path: "/recent"},
		{Path: "/stale"},
	}

	// Mark /recent as processed
	s.RecordProjectRun("/recent")

	// Filter for 1 hour threshold - /stale has never been processed
	filtered := FilterNotProcessedSince(projects, s, 1*time.Hour)
	if len(filtered) != 1 {
		t.Errorf("expected 1 stale project, got %d", len(filtered))
	}
}

func TestMergeProjectConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project config
	projectCfg := `
budget:
  max_percent: 20
tasks:
  enabled:
    - test
    - lint
`
	cfgPath := filepath.Join(tmpDir, config.ProjectConfigName)
	if err := os.WriteFile(cfgPath, []byte(projectCfg), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	globalCfg := &config.Config{
		Budget: config.BudgetConfig{
			MaxPercent: 10,
			Mode:       "daily",
		},
		Tasks: config.TasksConfig{
			Enabled: []string{"all"},
		},
	}

	merged, err := MergeProjectConfig(globalCfg, tmpDir)
	if err != nil {
		t.Fatalf("MergeProjectConfig() error = %v", err)
	}

	// Check override
	if merged.Budget.MaxPercent != 20 {
		t.Errorf("MaxPercent = %d, want 20", merged.Budget.MaxPercent)
	}

	// Check preserved global
	if merged.Budget.Mode != "daily" {
		t.Errorf("Mode = %s, want daily", merged.Budget.Mode)
	}
}

func TestMergeProjectConfigNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	globalCfg := &config.Config{
		Budget: config.BudgetConfig{MaxPercent: 10},
	}

	merged, err := MergeProjectConfig(globalCfg, tmpDir)
	if err != nil {
		t.Fatalf("MergeProjectConfig() error = %v", err)
	}

	if merged != globalCfg {
		t.Errorf("expected global config when no project config exists")
	}
}

func TestSelectNext(t *testing.T) {
	s := newTestState(t)

	projects := []Project{
		{Path: "/low", Priority: 1},
		{Path: "/high", Priority: 10},
	}

	next := SelectNext(projects, s)
	if next == nil {
		t.Fatal("expected a project, got nil")
		return
	}
	if next.Path != "/high" {
		t.Errorf("expected /high, got %s", next.Path)
	}
}

func TestSelectNextAllProcessed(t *testing.T) {
	s := newTestState(t)

	projects := []Project{
		{Path: "/proj1", Priority: 1},
	}

	s.RecordProjectRun("/proj1")

	next := SelectNext(projects, s)
	if next != nil {
		t.Errorf("expected nil when all processed, got %v", next)
	}
}

func TestIsProjectPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Not a project
	if IsProjectPath(tmpDir) {
		t.Error("empty dir should not be a project")
	}

	// Add .git
	_ = os.Mkdir(filepath.Join(tmpDir, ".git"), 0755)
	if !IsProjectPath(tmpDir) {
		t.Error("dir with .git should be a project")
	}
}

func TestDiscoverProjectsInDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some directories
	proj1 := filepath.Join(tmpDir, "proj1")
	proj2 := filepath.Join(tmpDir, "proj2")
	notProj := filepath.Join(tmpDir, "not-a-project")
	_ = os.Mkdir(proj1, 0755)
	_ = os.Mkdir(proj2, 0755)
	_ = os.Mkdir(notProj, 0755)

	// Make proj1 and proj2 look like projects
	_ = os.WriteFile(filepath.Join(proj1, "go.mod"), []byte("module proj1"), 0644)
	_ = os.WriteFile(filepath.Join(proj2, "package.json"), []byte("{}"), 0644)

	projects, err := DiscoverProjectsInDir(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverProjectsInDir() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestResolverDiscoverProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project directories
	proj1 := filepath.Join(tmpDir, "proj1")
	proj2 := filepath.Join(tmpDir, "proj2")
	_ = os.Mkdir(proj1, 0755)
	_ = os.Mkdir(proj2, 0755)

	cfg := &config.Config{
		Projects: []config.ProjectConfig{
			{Path: proj1, Priority: 5},
			{Path: proj2, Priority: 10},
		},
	}

	resolver := NewResolver(cfg)
	projects, err := resolver.DiscoverProjects()
	if err != nil {
		t.Fatalf("DiscoverProjects() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestResolverWithGlobPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project directories
	_ = os.Mkdir(filepath.Join(tmpDir, "proj1"), 0755)
	_ = os.Mkdir(filepath.Join(tmpDir, "proj2"), 0755)
	_ = os.Mkdir(filepath.Join(tmpDir, "archived"), 0755)

	cfg := &config.Config{
		Projects: []config.ProjectConfig{
			{
				Pattern: filepath.Join(tmpDir, "*"),
				Exclude: []string{filepath.Join(tmpDir, "archived")},
			},
		},
	}

	resolver := NewResolver(cfg)
	projects, err := resolver.DiscoverProjects()
	if err != nil {
		t.Fatalf("DiscoverProjects() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects (excluding archived), got %d", len(projects))
	}
}

func TestGetProjectSummaries(t *testing.T) {
	s := newTestState(t)

	projects := []Project{
		{Path: "/proj1", Priority: 5},
		{Path: "/proj2", Priority: 10},
	}

	// Record a run for proj1
	s.RecordProjectRun("/proj1")

	summaries := GetProjectSummaries(projects, s)
	if len(summaries) != 2 {
		t.Errorf("expected 2 summaries, got %d", len(summaries))
	}

	// proj1 should be processed today
	if !summaries[0].ProcessedToday {
		t.Error("proj1 should be marked as processed today")
	}
	if summaries[0].RunCount != 1 {
		t.Errorf("proj1 run count = %d, want 1", summaries[0].RunCount)
	}
}

func newTestState(t *testing.T) *state.State {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	st, err := state.New(database)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	return st
}
