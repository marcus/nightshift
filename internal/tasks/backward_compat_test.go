package tasks

import (
	"testing"
	"time"
)

// TestBackwardCompat_TaskTypeConstants verifies that all built-in task type
// string constants are stable. Changing these would break config files,
// database records, and CLI usage referencing task types by name.
func TestBackwardCompat_TaskTypeConstants(t *testing.T) {
	// Exhaustive map of task type constant -> expected string value.
	// If a constant is renamed or its value changes, this test catches it.
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

	for taskType, wantStr := range expected {
		if string(taskType) != wantStr {
			t.Errorf("TaskType constant changed: got %q, want %q", string(taskType), wantStr)
		}
	}
}

// TestBackwardCompat_RegistryContainsAllBuiltinTasks verifies that the
// registry contains all expected built-in task types and none have been
// accidentally removed.
func TestBackwardCompat_RegistryContainsAllBuiltinTasks(t *testing.T) {
	requiredTypes := []TaskType{
		TaskLintFix, TaskBugFinder, TaskAutoDRY, TaskSkillGroom,
		TaskAPIContractVerify, TaskBackwardCompat, TaskBuildOptimize,
		TaskDocsBackfill, TaskCommitNormalize, TaskChangelogSynth,
		TaskReleaseNotes, TaskADRDraft, TaskTDReview,
		TaskDocDrift, TaskSemanticDiff, TaskDeadCode, TaskDependencyRisk,
		TaskTestGap, TaskTestFlakiness, TaskLoggingAudit, TaskMetricsCoverage,
		TaskPerfRegression, TaskCostAttribution, TaskSecurityFootgun,
		TaskPIIScanner, TaskPrivacyPolicy, TaskSchemaEvolution,
		TaskEventTaxonomy, TaskRoadmapEntropy, TaskBusFactor, TaskKnowledgeSilo,
		TaskGroomer, TaskGuideImprover, TaskIdeaGenerator, TaskTechDebtClassify,
		TaskWhyAnnotator, TaskEdgeCaseEnum, TaskErrorMsgImprove,
		TaskSLOSuggester, TaskUXCopySharpener, TaskA11yLint,
		TaskServiceAdvisor, TaskOwnershipBoundary, TaskOncallEstimator,
		TaskMigrationRehearsal, TaskContractFuzzer, TaskGoldenPath,
		TaskPerfProfile, TaskAllocationProfile,
		TaskVisibilityInstrument, TaskRepoTopology, TaskPermissionsMapper,
		TaskDataLifecycle, TaskFeatureFlagMonitor, TaskCISignalNoise,
		TaskHistoricalContext,
		TaskRunbookGen, TaskRollbackPlan, TaskPostmortemGen,
	}

	for _, tt := range requiredTypes {
		if _, err := GetDefinition(tt); err != nil {
			t.Errorf("built-in task %q missing from registry: %v", tt, err)
		}
	}
}

// TestBackwardCompat_CostTierValues verifies that CostTier iota values
// haven't shifted. These values may be stored in databases or serialized.
func TestBackwardCompat_CostTierValues(t *testing.T) {
	tests := []struct {
		tier CostTier
		want int
	}{
		{CostLow, 0},
		{CostMedium, 1},
		{CostHigh, 2},
		{CostVeryHigh, 3},
	}
	for _, tt := range tests {
		if int(tt.tier) != tt.want {
			t.Errorf("CostTier %v = %d, want %d", tt.tier.String(), int(tt.tier), tt.want)
		}
	}
}

// TestBackwardCompat_RiskLevelValues verifies that RiskLevel iota values
// haven't shifted.
func TestBackwardCompat_RiskLevelValues(t *testing.T) {
	tests := []struct {
		risk RiskLevel
		want int
	}{
		{RiskLow, 0},
		{RiskMedium, 1},
		{RiskHigh, 2},
	}
	for _, tt := range tests {
		if int(tt.risk) != tt.want {
			t.Errorf("RiskLevel %v = %d, want %d", tt.risk.String(), int(tt.risk), tt.want)
		}
	}
}

// TestBackwardCompat_CategoryValues verifies that TaskCategory iota values
// haven't shifted.
func TestBackwardCompat_CategoryValues(t *testing.T) {
	tests := []struct {
		cat  TaskCategory
		want int
	}{
		{CategoryPR, 0},
		{CategoryAnalysis, 1},
		{CategoryOptions, 2},
		{CategorySafe, 3},
		{CategoryMap, 4},
		{CategoryEmergency, 5},
	}
	for _, tt := range tests {
		if int(tt.cat) != tt.want {
			t.Errorf("TaskCategory %v = %d, want %d", tt.cat.String(), int(tt.cat), tt.want)
		}
	}
}

// TestBackwardCompat_DefaultIntervalForCategory verifies that default
// intervals for each category haven't changed, as users may rely on
// these defaults for scheduling.
func TestBackwardCompat_DefaultIntervalForCategory(t *testing.T) {
	tests := []struct {
		cat  TaskCategory
		want time.Duration
	}{
		{CategoryPR, 168 * time.Hour},
		{CategoryAnalysis, 72 * time.Hour},
		{CategoryOptions, 168 * time.Hour},
		{CategorySafe, 336 * time.Hour},
		{CategoryMap, 168 * time.Hour},
		{CategoryEmergency, 720 * time.Hour},
	}
	for _, tt := range tests {
		got := DefaultIntervalForCategory(tt.cat)
		if got != tt.want {
			t.Errorf("DefaultIntervalForCategory(%v) = %v, want %v", tt.cat.String(), got, tt.want)
		}
	}
}

// TestBackwardCompat_CustomTaskRegistration verifies that custom task
// registration and unregistration still work correctly.
func TestBackwardCompat_CustomTaskRegistration(t *testing.T) {
	customType := TaskType("test-custom-compat")

	// Ensure clean state
	UnregisterCustom(customType)

	def := TaskDefinition{
		Type:            customType,
		Category:        CategoryAnalysis,
		Name:            "Test Custom Compat",
		Description:     "Test task for backward compat",
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 24 * time.Hour,
	}

	// Register should succeed
	if err := RegisterCustom(def); err != nil {
		t.Fatalf("RegisterCustom: %v", err)
	}
	defer UnregisterCustom(customType)

	// Should be retrievable
	got, err := GetDefinition(customType)
	if err != nil {
		t.Fatalf("GetDefinition after register: %v", err)
	}
	if got.Name != def.Name {
		t.Errorf("Name = %q, want %q", got.Name, def.Name)
	}

	// Should be marked as custom
	if !IsCustom(customType) {
		t.Error("IsCustom = false, want true")
	}

	// Duplicate registration should fail
	if err := RegisterCustom(def); err == nil {
		t.Error("duplicate RegisterCustom should fail")
	}

	// Unregister
	UnregisterCustom(customType)

	// Should no longer exist
	if _, err := GetDefinition(customType); err == nil {
		t.Error("GetDefinition should fail after unregister")
	}
	if IsCustom(customType) {
		t.Error("IsCustom should be false after unregister")
	}
}

// TestBackwardCompat_CostTierTokenRanges verifies the token ranges for
// each cost tier haven't changed, as budget calculations depend on these.
func TestBackwardCompat_CostTierTokenRanges(t *testing.T) {
	tests := []struct {
		tier    CostTier
		wantMin int
		wantMax int
	}{
		{CostLow, 10_000, 50_000},
		{CostMedium, 50_000, 150_000},
		{CostHigh, 150_000, 500_000},
		{CostVeryHigh, 500_000, 1_000_000},
	}
	for _, tt := range tests {
		min, max := tt.tier.TokenRange()
		if min != tt.wantMin || max != tt.wantMax {
			t.Errorf("CostTier(%d).TokenRange() = (%d, %d), want (%d, %d)",
				tt.tier, min, max, tt.wantMin, tt.wantMax)
		}
	}
}

// TestBackwardCompat_TDReviewDisabledByDefault verifies that the td-review
// task remains opt-in only, preventing surprise task execution for users
// upgrading from older versions.
func TestBackwardCompat_TDReviewDisabledByDefault(t *testing.T) {
	def, err := GetDefinition(TaskTDReview)
	if err != nil {
		t.Fatalf("GetDefinition(TaskTDReview): %v", err)
	}
	if !def.DisabledByDefault {
		t.Error("TaskTDReview.DisabledByDefault should be true")
	}
}
