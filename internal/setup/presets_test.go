package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marcus/nightshift/internal/tasks"
)

func TestDetectRepoSignalsEmpty(t *testing.T) {
	signals := DetectRepoSignals([]string{})
	if signals.HasRelease || signals.HasADR {
		t.Fatal("expected no signals for empty projects")
	}
}

func TestDetectRepoSignalsChangelog(t *testing.T) {
	tmpdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpdir, "CHANGELOG.md"), []byte("# Changelog\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	signals := DetectRepoSignals([]string{tmpdir})
	if !signals.HasRelease {
		t.Fatal("expected HasRelease=true for CHANGELOG.md")
	}
}

func TestDetectRepoSignalsReleaseWorkflow(t *testing.T) {
	tmpdir := t.TempDir()
	workflowDir := filepath.Join(tmpdir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "release.yml"), []byte("name: release\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	signals := DetectRepoSignals([]string{tmpdir})
	if !signals.HasRelease {
		t.Fatal("expected HasRelease=true for release.yml")
	}
}

func TestDetectRepoSignalsADRDocs(t *testing.T) {
	tmpdir := t.TempDir()
	adrDir := filepath.Join(tmpdir, "docs", "adr")
	if err := os.MkdirAll(adrDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	signals := DetectRepoSignals([]string{tmpdir})
	if !signals.HasADR {
		t.Fatal("expected HasADR=true for docs/adr")
	}
}

func TestDetectRepoSignalsADRRoot(t *testing.T) {
	tmpdir := t.TempDir()
	adrDir := filepath.Join(tmpdir, "adr")
	if err := os.MkdirAll(adrDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	signals := DetectRepoSignals([]string{tmpdir})
	if !signals.HasADR {
		t.Fatal("expected HasADR=true for adr directory")
	}
}

func TestDetectRepoSignalsIgnoresEmpty(t *testing.T) {
	signals := DetectRepoSignals([]string{"", ""})
	if signals.HasRelease || signals.HasADR {
		t.Fatal("expected no signals for empty strings")
	}
}

func TestPresetTasksSafe(t *testing.T) {
	defs := []tasks.TaskDefinition{
		{Type: tasks.TaskLintFix, Category: tasks.CategoryPR, RiskLevel: tasks.RiskLow, CostTier: tasks.CostLow},
		{Type: tasks.TaskBugFinder, Category: tasks.CategoryPR, RiskLevel: tasks.RiskMedium, CostTier: tasks.CostHigh},
	}

	selected := PresetTasks(PresetSafe, defs, RepoSignals{})
	if !selected[tasks.TaskLintFix] {
		t.Fatal("expected TaskLintFix in safe preset")
	}
	if selected[tasks.TaskBugFinder] {
		t.Fatal("expected TaskBugFinder excluded from safe preset")
	}
}

func TestPresetTasksBalanced(t *testing.T) {
	defs := []tasks.TaskDefinition{
		{Type: tasks.TaskLintFix, Category: tasks.CategoryPR, RiskLevel: tasks.RiskLow, CostTier: tasks.CostLow},
		{Type: tasks.TaskBugFinder, Category: tasks.CategoryPR, RiskLevel: tasks.RiskMedium, CostTier: tasks.CostHigh},
		{Type: tasks.TaskAutoDRY, Category: tasks.CategoryPR, RiskLevel: tasks.RiskMedium, CostTier: tasks.CostHigh},
	}

	selected := PresetTasks(PresetBalanced, defs, RepoSignals{})
	if !selected[tasks.TaskLintFix] {
		t.Fatal("expected TaskLintFix in balanced preset")
	}
	// BugFinder and AutoDRY are heavy PR tasks, excluded from balanced
	if selected[tasks.TaskBugFinder] {
		t.Fatal("expected TaskBugFinder excluded from balanced preset")
	}
	if selected[tasks.TaskAutoDRY] {
		t.Fatal("expected TaskAutoDRY excluded from balanced preset")
	}
}

func TestPresetTasksAggressive(t *testing.T) {
	defs := []tasks.TaskDefinition{
		{Type: tasks.TaskLintFix, Category: tasks.CategoryPR, RiskLevel: tasks.RiskLow, CostTier: tasks.CostLow},
		{Type: tasks.TaskBugFinder, Category: tasks.CategoryPR, RiskLevel: tasks.RiskMedium, CostTier: tasks.CostHigh},
		{Type: tasks.TaskReleaseNotes, Category: tasks.CategoryPR, RiskLevel: tasks.RiskHigh, CostTier: tasks.CostHigh},
	}

	selected := PresetTasks(PresetAggressive, defs, RepoSignals{})
	if !selected[tasks.TaskLintFix] {
		t.Fatal("expected TaskLintFix in aggressive preset")
	}
	if !selected[tasks.TaskBugFinder] {
		t.Fatal("expected TaskBugFinder in aggressive preset")
	}
	// High risk not allowed in aggressive
	if selected[tasks.TaskReleaseNotes] {
		t.Fatal("expected TaskReleaseNotes excluded from aggressive preset")
	}
}

func TestPresetTasksFiltersByCategory(t *testing.T) {
	defs := []tasks.TaskDefinition{
		{Type: tasks.TaskLintFix, Category: tasks.CategoryPR, RiskLevel: tasks.RiskLow, CostTier: tasks.CostLow},
		{Type: tasks.TaskDocDrift, Category: tasks.CategoryAnalysis, RiskLevel: tasks.RiskLow, CostTier: tasks.CostLow},
		{Type: tasks.TaskGroomer, Category: tasks.CategoryOptions, RiskLevel: tasks.RiskLow, CostTier: tasks.CostLow},
	}

	selected := PresetTasks(PresetBalanced, defs, RepoSignals{})
	if !selected[tasks.TaskLintFix] {
		t.Fatal("expected TaskLintFix selected")
	}
	// CategoryAnalysis is allowed
	if !selected[tasks.TaskDocDrift] {
		t.Fatal("expected TaskDocDrift selected (CategoryAnalysis is allowed)")
	}
	// CategoryOptions is not allowed
	if selected[tasks.TaskGroomer] {
		t.Fatal("expected TaskGroomer excluded due to CategoryOptions")
	}
}

func TestPresetTasksReleaseSignal(t *testing.T) {
	defs := []tasks.TaskDefinition{
		{Type: tasks.TaskChangelogSynth, Category: tasks.CategoryPR, RiskLevel: tasks.RiskLow, CostTier: tasks.CostMedium},
	}

	// Without release signal
	selected := PresetTasks(PresetBalanced, defs, RepoSignals{HasRelease: false})
	if selected[tasks.TaskChangelogSynth] {
		t.Fatal("expected TaskChangelogSynth excluded without HasRelease")
	}

	// With release signal
	selected = PresetTasks(PresetBalanced, defs, RepoSignals{HasRelease: true})
	if !selected[tasks.TaskChangelogSynth] {
		t.Fatal("expected TaskChangelogSynth selected with HasRelease")
	}
}

func TestPresetTasksADRSignal(t *testing.T) {
	defs := []tasks.TaskDefinition{
		{Type: tasks.TaskADRDraft, Category: tasks.CategoryPR, RiskLevel: tasks.RiskLow, CostTier: tasks.CostMedium},
	}

	// Without ADR signal
	selected := PresetTasks(PresetBalanced, defs, RepoSignals{HasADR: false})
	if selected[tasks.TaskADRDraft] {
		t.Fatal("expected TaskADRDraft excluded without HasADR")
	}

	// With ADR signal
	selected = PresetTasks(PresetBalanced, defs, RepoSignals{HasADR: true})
	if !selected[tasks.TaskADRDraft] {
		t.Fatal("expected TaskADRDraft selected with HasADR")
	}
}

func TestHasAnyFileExists(t *testing.T) {
	tmpdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpdir, "test.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result := hasAny(tmpdir, []string{"test.md"})
	if !result {
		t.Fatal("expected hasAny to return true for existing file")
	}
}

func TestHasAnyDirExists(t *testing.T) {
	tmpdir := t.TempDir()
	testdir := filepath.Join(tmpdir, "testdir")
	if err := os.MkdirAll(testdir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	result := hasAny(tmpdir, []string{"testdir"})
	if !result {
		t.Fatal("expected hasAny to return true for existing directory")
	}
}

func TestHasAnyNotExists(t *testing.T) {
	tmpdir := t.TempDir()

	result := hasAny(tmpdir, []string{"nonexistent.md"})
	if result {
		t.Fatal("expected hasAny to return false for nonexistent path")
	}
}

func TestHasAnyMultiplePaths(t *testing.T) {
	tmpdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpdir, "found.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result := hasAny(tmpdir, []string{"notfound.md", "also-notfound.md", "found.md"})
	if !result {
		t.Fatal("expected hasAny to return true when one path exists")
	}
}

func TestIsReleaseTask(t *testing.T) {
	tests := []struct {
		taskType tasks.TaskType
		expected bool
	}{
		{tasks.TaskChangelogSynth, true},
		{tasks.TaskReleaseNotes, true},
		{tasks.TaskLintFix, false},
		{tasks.TaskBugFinder, false},
	}

	for _, test := range tests {
		result := isReleaseTask(test.taskType)
		if result != test.expected {
			t.Fatalf("isReleaseTask(%v) = %v, want %v", test.taskType, result, test.expected)
		}
	}
}

func TestIsHeavyPR(t *testing.T) {
	tests := []struct {
		taskType tasks.TaskType
		expected bool
	}{
		{tasks.TaskBugFinder, true},
		{tasks.TaskAutoDRY, true},
		{tasks.TaskLintFix, false},
		{tasks.TaskReleaseNotes, false},
	}

	for _, test := range tests {
		result := isHeavyPR(test.taskType)
		if result != test.expected {
			t.Fatalf("isHeavyPR(%v) = %v, want %v", test.taskType, result, test.expected)
		}
	}
}
