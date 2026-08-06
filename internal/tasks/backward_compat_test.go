package tasks

import (
	"testing"
	"time"
)

// Backward-compatibility tests for the task registry.
// These tests pin the public API surface so that accidental changes
// to task type strings, categories, cost tiers, or risk levels are
// caught before release. Any change here is a breaking change for
// users who reference task types in config files or scripts.

// TestBackwardCompat_TaskTypeStringsStable verifies that all 59 built-in
// task type string values have not changed. Users reference these in
// config YAML (tasks.enabled, tasks.disabled, tasks.intervals, etc.),
// so renaming a task type is a breaking change.
func TestBackwardCompat_TaskTypeStringsStable(t *testing.T) {
	expected := map[TaskType]string{
		// Category 1: PR
		TaskLintFix:           "lint-fix",
		TaskBugFinder:         "bug-finder",
		TaskAutoDRY:           "auto-dry",
		TaskSkillGroom:        "skill-groom",
		TaskAPIContractVerify: "api-contract-verify",
		TaskBackwardCompat:    "backward-compat",
		TaskBuildOptimize:     "build-optimize",
		TaskDocsBackfill:      "docs-backfill",
		TaskCommitNormalize:   "commit-normalize",
		TaskChangelogSynth:    "changelog-synth",
		TaskReleaseNotes:      "release-notes",
		TaskADRDraft:          "adr-draft",
		TaskTDReview:          "td-review",
		// Category 2: Analysis
		TaskDocDrift:        "doc-drift",
		TaskSemanticDiff:    "semantic-diff",
		TaskDeadCode:        "dead-code",
		TaskDependencyRisk:  "dependency-risk",
		TaskTestGap:         "test-gap",
		TaskTestFlakiness:   "test-flakiness",
		TaskLoggingAudit:    "logging-audit",
		TaskMetricsCoverage: "metrics-coverage",
		TaskPerfRegression:  "perf-regression",
		TaskCostAttribution: "cost-attribution",
		TaskSecurityFootgun: "security-footgun",
		TaskPIIScanner:      "pii-scanner",
		TaskPrivacyPolicy:   "privacy-policy",
		TaskSchemaEvolution: "schema-evolution",
		TaskEventTaxonomy:   "event-taxonomy",
		TaskRoadmapEntropy:  "roadmap-entropy",
		TaskBusFactor:       "bus-factor",
		TaskKnowledgeSilo:   "knowledge-silo",
		// Category 3: Options
		TaskGroomer:           "task-groomer",
		TaskGuideImprover:     "guide-improver",
		TaskIdeaGenerator:     "idea-generator",
		TaskTechDebtClassify:  "tech-debt-classify",
		TaskWhyAnnotator:      "why-annotator",
		TaskEdgeCaseEnum:      "edge-case-enum",
		TaskErrorMsgImprove:   "error-msg-improve",
		TaskSLOSuggester:      "slo-suggester",
		TaskUXCopySharpener:   "ux-copy-sharpener",
		TaskA11yLint:          "a11y-lint",
		TaskServiceAdvisor:    "service-advisor",
		TaskOwnershipBoundary: "ownership-boundary",
		TaskOncallEstimator:   "oncall-estimator",
		// Category 4: Safe
		TaskMigrationRehearsal: "migration-rehearsal",
		TaskContractFuzzer:     "contract-fuzzer",
		TaskGoldenPath:         "golden-path",
		TaskPerfProfile:        "perf-profile",
		TaskAllocationProfile:  "allocation-profile",
		// Category 5: Map
		TaskVisibilityInstrument: "visibility-instrument",
		TaskRepoTopology:         "repo-topology",
		TaskPermissionsMapper:    "permissions-mapper",
		TaskDataLifecycle:        "data-lifecycle",
		TaskFeatureFlagMonitor:   "feature-flag-monitor",
		TaskCISignalNoise:        "ci-signal-noise",
		TaskHistoricalContext:    "historical-context",
		// Category 6: Emergency
		TaskRunbookGen:    "runbook-gen",
		TaskRollbackPlan:  "rollback-plan",
		TaskPostmortemGen: "postmortem-gen",
	}

	for taskConst, wantStr := range expected {
		if string(taskConst) != wantStr {
			t.Errorf("TaskType constant %q has value %q, want %q — renaming task types breaks user configs",
				wantStr, string(taskConst), wantStr)
		}
	}

	// Verify count: if a new task is added, the count increases.
	// A decrease means a task was removed (breaking change).
	allTypes := AllTaskTypes()
	if len(allTypes) < len(expected) {
		t.Errorf("AllTaskTypes() returned %d types, expected at least %d — a built-in task type was removed",
			len(allTypes), len(expected))
	}
}

// TestBackwardCompat_TaskCategoryAssignments verifies that each task
// remains in its original category. Changing a task's category affects
// budget allocation (different categories have different default intervals
// and cost expectations).
func TestBackwardCompat_TaskCategoryAssignments(t *testing.T) {
	categoryChecks := map[TaskType]TaskCategory{
		TaskLintFix:              CategoryPR,
		TaskBugFinder:            CategoryPR,
		TaskAutoDRY:              CategoryPR,
		TaskSkillGroom:           CategoryPR,
		TaskAPIContractVerify:    CategoryPR,
		TaskBackwardCompat:       CategoryPR,
		TaskBuildOptimize:        CategoryPR,
		TaskDocsBackfill:         CategoryPR,
		TaskDocDrift:             CategoryAnalysis,
		TaskSemanticDiff:         CategoryAnalysis,
		TaskDeadCode:             CategoryAnalysis,
		TaskTestGap:              CategoryAnalysis,
		TaskSecurityFootgun:      CategoryAnalysis,
		TaskPIIScanner:           CategoryAnalysis,
		TaskBusFactor:            CategoryAnalysis,
		TaskGroomer:              CategoryOptions,
		TaskGuideImprover:        CategoryOptions,
		TaskA11yLint:             CategoryOptions,
		TaskMigrationRehearsal:   CategorySafe,
		TaskContractFuzzer:       CategorySafe,
		TaskGoldenPath:           CategorySafe,
		TaskRepoTopology:         CategoryMap,
		TaskPermissionsMapper:    CategoryMap,
		TaskVisibilityInstrument: CategoryMap,
		TaskRunbookGen:           CategoryEmergency,
		TaskRollbackPlan:         CategoryEmergency,
		TaskPostmortemGen:        CategoryEmergency,
	}

	for taskType, wantCat := range categoryChecks {
		def, err := GetDefinition(taskType)
		if err != nil {
			t.Errorf("GetDefinition(%q) error: %v — task was removed from registry", taskType, err)
			continue
		}
		if def.Category != wantCat {
			t.Errorf("Task %q category = %d (%s), want %d (%s) — changing category breaks budget behavior",
				taskType, def.Category, def.Category.String(), wantCat, wantCat.String())
		}
	}
}

// TestBackwardCompat_CostTierAssignments verifies that key tasks maintain
// their cost tier. Changing a cost tier affects budget allocation calculations.
func TestBackwardCompat_CostTierAssignments(t *testing.T) {
	costChecks := map[TaskType]CostTier{
		TaskLintFix:            CostLow,
		TaskCommitNormalize:    CostLow,
		TaskBugFinder:          CostHigh,
		TaskAutoDRY:            CostHigh,
		TaskMigrationRehearsal: CostVeryHigh,
		TaskContractFuzzer:     CostVeryHigh,
	}

	for taskType, wantTier := range costChecks {
		def, err := GetDefinition(taskType)
		if err != nil {
			t.Errorf("GetDefinition(%q) error: %v", taskType, err)
			continue
		}
		if def.CostTier != wantTier {
			t.Errorf("Task %q CostTier = %s, want %s — changing cost tier affects budget calculations",
				taskType, def.CostTier.String(), wantTier.String())
		}
	}
}

// TestBackwardCompat_CostTierTokenRangesStable verifies that the token
// ranges for each cost tier haven't changed. These are used in budget
// allocation and user-facing displays.
func TestBackwardCompat_CostTierTokenRangesStable(t *testing.T) {
	ranges := []struct {
		tier    CostTier
		wantMin int
		wantMax int
	}{
		{CostLow, 10_000, 50_000},
		{CostMedium, 50_000, 150_000},
		{CostHigh, 150_000, 500_000},
		{CostVeryHigh, 500_000, 1_000_000},
	}

	for _, r := range ranges {
		min, max := r.tier.TokenRange()
		if min != r.wantMin || max != r.wantMax {
			t.Errorf("CostTier %s TokenRange() = (%d, %d), want (%d, %d) — token ranges must not change",
				r.tier.String(), min, max, r.wantMin, r.wantMax)
		}
	}
}

// TestBackwardCompat_CategoryDefaultIntervals verifies that default
// intervals per category haven't changed. Users depend on these for
// scheduling expectations.
func TestBackwardCompat_CategoryDefaultIntervals(t *testing.T) {
	intervals := map[TaskCategory]time.Duration{
		CategoryPR:        168 * time.Hour,  // 7 days
		CategoryAnalysis:  72 * time.Hour,   // 3 days
		CategoryOptions:   168 * time.Hour,  // 7 days
		CategorySafe:      336 * time.Hour,  // 14 days
		CategoryMap:       168 * time.Hour,  // 7 days
		CategoryEmergency: 720 * time.Hour,  // 30 days
	}

	for cat, want := range intervals {
		got := DefaultIntervalForCategory(cat)
		if got != want {
			t.Errorf("DefaultIntervalForCategory(%s) = %v, want %v — changing default intervals breaks scheduling",
				cat.String(), got, want)
		}
	}
}

// TestBackwardCompat_TDReviewDisabledByDefault verifies that td-review
// remains disabled by default, since it requires explicit opt-in.
func TestBackwardCompat_TDReviewDisabledByDefault(t *testing.T) {
	def, err := GetDefinition(TaskTDReview)
	if err != nil {
		t.Fatalf("GetDefinition(td-review): %v", err)
	}
	if !def.DisabledByDefault {
		t.Error("td-review must remain DisabledByDefault — enabling it by default would surprise users")
	}
}

// TestBackwardCompat_CustomTaskRegistration verifies that the custom task
// registration API works the same way as in v0.3.0+.
func TestBackwardCompat_CustomTaskRegistration(t *testing.T) {
	t.Cleanup(func() { UnregisterCustom("compat-custom") })

	def := TaskDefinition{
		Type:            "compat-custom",
		Category:        CategoryAnalysis,
		Name:            "Compat Test Task",
		Description:     "Tests backward compat of custom task registration",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 48 * time.Hour,
	}

	// Register must succeed
	if err := RegisterCustom(def); err != nil {
		t.Fatalf("RegisterCustom() error: %v", err)
	}

	// Must be retrievable
	got, err := GetDefinition("compat-custom")
	if err != nil {
		t.Fatalf("GetDefinition() after register: %v", err)
	}
	if got.Type != def.Type || got.Category != def.Category || got.CostTier != def.CostTier {
		t.Errorf("Retrieved definition doesn't match registered: got type=%q cat=%d tier=%d",
			got.Type, got.Category, got.CostTier)
	}

	// Must appear in AllDefinitions
	found := false
	for _, d := range AllDefinitions() {
		if d.Type == "compat-custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom task not found in AllDefinitions()")
	}

	// IsCustom must return true
	if !IsCustom("compat-custom") {
		t.Error("IsCustom() should return true for registered custom task")
	}

	// Re-registration must fail (duplicate)
	if err := RegisterCustom(def); err == nil {
		t.Error("RegisterCustom() should fail for duplicate type")
	}

	// Collision with built-in must fail
	builtIn := TaskDefinition{
		Type:            TaskLintFix,
		Category:        CategoryPR,
		Name:            "Collision",
		Description:     "should fail",
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 24 * time.Hour,
	}
	if err := RegisterCustom(builtIn); err == nil {
		t.Error("RegisterCustom() should fail when colliding with built-in type")
	}

	// Unregister must work
	UnregisterCustom("compat-custom")
	if IsCustom("compat-custom") {
		t.Error("IsCustom() should return false after UnregisterCustom()")
	}
}
