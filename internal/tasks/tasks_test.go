package tasks

import (
	"testing"
	"time"
)

func TestCostTierString(t *testing.T) {
	tests := []struct {
		tier CostTier
		want string
	}{
		{CostLow, "Low (10-50k)"},
		{CostMedium, "Medium (50-150k)"},
		{CostHigh, "High (150-500k)"},
		{CostVeryHigh, "Very High (500k+)"},
		{CostTier(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.tier.String(); got != tt.want {
			t.Errorf("CostTier(%d).String() = %q, want %q", tt.tier, got, tt.want)
		}
	}
}

func TestCostTierTokenRange(t *testing.T) {
	tests := []struct {
		tier    CostTier
		wantMin int
		wantMax int
	}{
		{CostLow, 10_000, 50_000},
		{CostMedium, 50_000, 150_000},
		{CostHigh, 150_000, 500_000},
		{CostVeryHigh, 500_000, 1_000_000},
		{CostTier(99), 0, 0},
	}
	for _, tt := range tests {
		min, max := tt.tier.TokenRange()
		if min != tt.wantMin || max != tt.wantMax {
			t.Errorf("CostTier(%d).TokenRange() = (%d, %d), want (%d, %d)",
				tt.tier, min, max, tt.wantMin, tt.wantMax)
		}
	}
}

func TestRiskLevelString(t *testing.T) {
	tests := []struct {
		risk RiskLevel
		want string
	}{
		{RiskLow, "Low"},
		{RiskMedium, "Medium"},
		{RiskHigh, "High"},
		{RiskLevel(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.risk.String(); got != tt.want {
			t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.risk, got, tt.want)
		}
	}
}

func TestTaskCategoryString(t *testing.T) {
	tests := []struct {
		cat  TaskCategory
		want string
	}{
		{CategoryPR, "It's done - here's the PR"},
		{CategoryAnalysis, "Here's what I found"},
		{CategoryOptions, "Here are options"},
		{CategorySafe, "I tried it safely"},
		{CategoryMap, "Here's the map"},
		{CategoryEmergency, "For when things go sideways"},
		{TaskCategory(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.cat.String(); got != tt.want {
			t.Errorf("TaskCategory(%d).String() = %q, want %q", tt.cat, got, tt.want)
		}
	}
}

func TestGetDefinition(t *testing.T) {
	// Valid task type
	def, err := GetDefinition(TaskLintFix)
	if err != nil {
		t.Fatalf("GetDefinition(TaskLintFix) returned error: %v", err)
	}
	if def.Type != TaskLintFix {
		t.Errorf("GetDefinition(TaskLintFix).Type = %q, want %q", def.Type, TaskLintFix)
	}
	if def.Category != CategoryPR {
		t.Errorf("GetDefinition(TaskLintFix).Category = %d, want %d", def.Category, CategoryPR)
	}
	if def.CostTier != CostLow {
		t.Errorf("GetDefinition(TaskLintFix).CostTier = %d, want %d", def.CostTier, CostLow)
	}

	// Unknown task type
	_, err = GetDefinition("unknown-task")
	if err == nil {
		t.Error("GetDefinition(unknown) should return error")
	}
}

func TestGetCostEstimate(t *testing.T) {
	// Low cost task
	min, max, err := GetCostEstimate(TaskLintFix)
	if err != nil {
		t.Fatalf("GetCostEstimate(TaskLintFix) error: %v", err)
	}
	if min != 10_000 || max != 50_000 {
		t.Errorf("GetCostEstimate(TaskLintFix) = (%d, %d), want (10000, 50000)", min, max)
	}

	// Very high cost task
	min, max, err = GetCostEstimate(TaskMigrationRehearsal)
	if err != nil {
		t.Fatalf("GetCostEstimate(TaskMigrationRehearsal) error: %v", err)
	}
	if min != 500_000 || max != 1_000_000 {
		t.Errorf("GetCostEstimate(TaskMigrationRehearsal) = (%d, %d), want (500000, 1000000)", min, max)
	}

	// Unknown task
	_, _, err = GetCostEstimate("unknown-task")
	if err == nil {
		t.Error("GetCostEstimate(unknown) should return error")
	}
}

func TestGetTasksByCategory(t *testing.T) {
	// PR category should have multiple tasks
	prTasks := GetTasksByCategory(CategoryPR)
	if len(prTasks) == 0 {
		t.Error("GetTasksByCategory(CategoryPR) returned empty slice")
	}
	for _, task := range prTasks {
		if task.Category != CategoryPR {
			t.Errorf("GetTasksByCategory(CategoryPR) returned task with category %d", task.Category)
		}
	}

	// Verify all 6 categories have tasks
	categories := []TaskCategory{
		CategoryPR, CategoryAnalysis, CategoryOptions,
		CategorySafe, CategoryMap, CategoryEmergency,
	}
	for _, cat := range categories {
		tasks := GetTasksByCategory(cat)
		if len(tasks) == 0 {
			t.Errorf("GetTasksByCategory(%d) returned empty slice", cat)
		}
	}
}

func TestGetTasksByCostTier(t *testing.T) {
	// Low cost should include lint-fix
	lowTasks := GetTasksByCostTier(CostLow)
	if len(lowTasks) == 0 {
		t.Error("GetTasksByCostTier(CostLow) returned empty slice")
	}
	found := false
	for _, task := range lowTasks {
		if task.Type == TaskLintFix {
			found = true
		}
		if task.CostTier != CostLow {
			t.Errorf("GetTasksByCostTier(CostLow) returned task with tier %d", task.CostTier)
		}
	}
	if !found {
		t.Error("GetTasksByCostTier(CostLow) should include TaskLintFix")
	}

	// VeryHigh should include migration-rehearsal
	vhTasks := GetTasksByCostTier(CostVeryHigh)
	found = false
	for _, task := range vhTasks {
		if task.Type == TaskMigrationRehearsal {
			found = true
		}
	}
	if !found {
		t.Error("GetTasksByCostTier(CostVeryHigh) should include TaskMigrationRehearsal")
	}
}

func TestGetTasksByRiskLevel(t *testing.T) {
	// Low risk tasks
	lowRisk := GetTasksByRiskLevel(RiskLow)
	if len(lowRisk) == 0 {
		t.Error("GetTasksByRiskLevel(RiskLow) returned empty slice")
	}
	for _, task := range lowRisk {
		if task.RiskLevel != RiskLow {
			t.Errorf("GetTasksByRiskLevel(RiskLow) returned task with risk %d", task.RiskLevel)
		}
	}

	// High risk should include migration-rehearsal
	highRisk := GetTasksByRiskLevel(RiskHigh)
	found := false
	for _, task := range highRisk {
		if task.Type == TaskMigrationRehearsal {
			found = true
		}
	}
	if !found {
		t.Error("GetTasksByRiskLevel(RiskHigh) should include TaskMigrationRehearsal")
	}
}

func TestAllTaskTypes(t *testing.T) {
	types := AllTaskTypes()
	if len(types) == 0 {
		t.Error("AllTaskTypes() returned empty slice")
	}
	// Should have at least 50 task types based on SPEC
	if len(types) < 50 {
		t.Errorf("AllTaskTypes() returned %d types, expected at least 50", len(types))
	}
}

func TestAllDefinitions(t *testing.T) {
	defs := AllDefinitions()
	if len(defs) == 0 {
		t.Error("AllDefinitions() returned empty slice")
	}
	// Should match number of types
	types := AllTaskTypes()
	if len(defs) != len(types) {
		t.Errorf("AllDefinitions() returned %d, AllTaskTypes() returned %d", len(defs), len(types))
	}
}

func TestTaskDefinitionEstimatedTokens(t *testing.T) {
	def, _ := GetDefinition(TaskLintFix)
	min, max := def.EstimatedTokens()
	if min != 10_000 || max != 50_000 {
		t.Errorf("TaskDefinition.EstimatedTokens() = (%d, %d), want (10000, 50000)", min, max)
	}
}

func TestRegistryCompleteness(t *testing.T) {
	// All task type constants should be in registry
	taskTypes := []TaskType{
		// Category 1
		TaskLintFix, TaskBugFinder, TaskAutoDRY, TaskAPIContractVerify,
		TaskBackwardCompat, TaskBuildOptimize, TaskDocsBackfill,
		TaskCommitNormalize, TaskChangelogSynth, TaskReleaseNotes, TaskADRDraft,
		TaskTDReview,
		// Category 2
		TaskDocDrift, TaskSemanticDiff, TaskDeadCode, TaskDependencyRisk,
		TaskTestGap, TaskTestFlakiness, TaskLoggingAudit, TaskMetricsCoverage,
		TaskPerfRegression, TaskCostAttribution, TaskSecurityFootgun,
		TaskPIIScanner, TaskPrivacyPolicy, TaskSchemaEvolution,
		TaskEventTaxonomy, TaskRoadmapEntropy, TaskBusFactor, TaskKnowledgeSilo,
		// Category 3
		TaskGroomer, TaskGuideImprover, TaskIdeaGenerator, TaskTechDebtClassify,
		TaskWhyAnnotator, TaskEdgeCaseEnum, TaskErrorMsgImprove, TaskSLOSuggester,
		TaskUXCopySharpener, TaskA11yLint, TaskServiceAdvisor,
		TaskOwnershipBoundary, TaskOncallEstimator,
		// Category 4
		TaskMigrationRehearsal, TaskContractFuzzer, TaskGoldenPath,
		TaskPerfProfile, TaskAllocationProfile,
		// Category 5
		TaskVisibilityInstrument, TaskRepoTopology, TaskPermissionsMapper,
		TaskDataLifecycle, TaskFeatureFlagMonitor, TaskCISignalNoise,
		TaskHistoricalContext,
		// Category 6
		TaskRunbookGen, TaskRollbackPlan, TaskPostmortemGen,
	}

	for _, tt := range taskTypes {
		if _, err := GetDefinition(tt); err != nil {
			t.Errorf("Task type %q not in registry: %v", tt, err)
		}
	}
}

func TestAllDefinitionsHaveDefaultInterval(t *testing.T) {
	for _, def := range AllDefinitions() {
		if def.DefaultInterval == 0 {
			t.Errorf("Task %q (%s) has zero DefaultInterval", def.Type, def.Name)
		}
	}
}

func TestDefaultIntervalForCategory(t *testing.T) {
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
		{TaskCategory(99), 168 * time.Hour},
	}
	for _, tt := range tests {
		if got := DefaultIntervalForCategory(tt.cat); got != tt.want {
			t.Errorf("DefaultIntervalForCategory(%d) = %v, want %v", tt.cat, got, tt.want)
		}
	}
}

func TestDisabledByDefault(t *testing.T) {
	def, err := GetDefinition(TaskTDReview)
	if err != nil {
		t.Fatalf("GetDefinition(TaskTDReview) error: %v", err)
	}
	if !def.DisabledByDefault {
		t.Error("TaskTDReview should be DisabledByDefault")
	}

	// Non-disabled-by-default tasks should have false
	def, _ = GetDefinition(TaskLintFix)
	if def.DisabledByDefault {
		t.Error("TaskLintFix should not be DisabledByDefault")
	}
}

func TestDefaultDisabledTaskTypes(t *testing.T) {
	types := DefaultDisabledTaskTypes()
	if len(types) == 0 {
		t.Error("DefaultDisabledTaskTypes() returned empty slice")
	}
	found := false
	for _, tt := range types {
		if tt == TaskTDReview {
			found = true
		}
	}
	if !found {
		t.Error("DefaultDisabledTaskTypes() should include TaskTDReview")
	}
}

func TestRegisterCustom_Success(t *testing.T) {
	def := TaskDefinition{
		Type: "custom-test", Category: CategoryAnalysis,
		Name: "Test", Description: "test task",
		CostTier: CostMedium, RiskLevel: RiskLow,
		DefaultInterval: 72 * time.Hour,
	}
	t.Cleanup(func() { UnregisterCustom("custom-test") })

	err := RegisterCustom(def)
	if err != nil {
		t.Fatalf("RegisterCustom() unexpected error: %v", err)
	}

	got, err := GetDefinition("custom-test")
	if err != nil {
		t.Fatalf("GetDefinition() after register: %v", err)
	}
	if got.Type != def.Type {
		t.Errorf("Type = %q, want %q", got.Type, def.Type)
	}
	if got.Category != def.Category {
		t.Errorf("Category = %d, want %d", got.Category, def.Category)
	}
	if got.Name != def.Name {
		t.Errorf("Name = %q, want %q", got.Name, def.Name)
	}
	if got.CostTier != def.CostTier {
		t.Errorf("CostTier = %d, want %d", got.CostTier, def.CostTier)
	}
	if got.DefaultInterval != def.DefaultInterval {
		t.Errorf("DefaultInterval = %v, want %v", got.DefaultInterval, def.DefaultInterval)
	}
}

func TestRegisterCustom_BuiltInCollision(t *testing.T) {
	def := TaskDefinition{
		Type: TaskLintFix, Category: CategoryPR,
		Name: "Collision", Description: "should fail",
		CostTier: CostLow, RiskLevel: RiskLow,
		DefaultInterval: 24 * time.Hour,
	}
	err := RegisterCustom(def)
	if err == nil {
		t.Error("expected error for built-in collision")
	}
}

func TestRegisterCustom_DuplicateCustom(t *testing.T) {
	t.Cleanup(func() { UnregisterCustom("dup-test") })
	_ = RegisterCustom(TaskDefinition{
		Type: "dup-test", Category: CategoryAnalysis,
		Name: "Dup1", Description: "first",
		CostTier: CostLow, RiskLevel: RiskLow,
		DefaultInterval: 72 * time.Hour,
	})
	err := RegisterCustom(TaskDefinition{
		Type: "dup-test", Category: CategoryAnalysis,
		Name: "Dup2", Description: "second",
		CostTier: CostLow, RiskLevel: RiskLow,
		DefaultInterval: 72 * time.Hour,
	})
	if err == nil {
		t.Error("expected error for duplicate custom registration")
	}
}

func TestIsCustom(t *testing.T) {
	t.Cleanup(func() { UnregisterCustom("is-custom-test") })
	_ = RegisterCustom(TaskDefinition{
		Type: "is-custom-test", Category: CategoryAnalysis,
		Name: "IsCustom", Description: "test",
		CostTier: CostLow, RiskLevel: RiskLow,
		DefaultInterval: 72 * time.Hour,
	})
	if !IsCustom("is-custom-test") {
		t.Error("expected true for custom type")
	}
	if IsCustom(TaskLintFix) {
		t.Error("expected false for built-in type")
	}
}

func TestUnregisterCustom(t *testing.T) {
	_ = RegisterCustom(TaskDefinition{
		Type: "unreg-test", Category: CategoryAnalysis,
		Name: "Unreg", Description: "test",
		CostTier: CostLow, RiskLevel: RiskLow,
		DefaultInterval: 72 * time.Hour,
	})
	UnregisterCustom("unreg-test")
	_, err := GetDefinition("unreg-test")
	if err == nil {
		t.Error("expected error after unregister")
	}
	if IsCustom("unreg-test") {
		t.Error("should not be custom after unregister")
	}
}

func TestClearCustom(t *testing.T) {
	_ = RegisterCustom(TaskDefinition{
		Type: "clear-test-1", Category: CategoryAnalysis,
		Name: "Clear1", Description: "test",
		CostTier: CostLow, RiskLevel: RiskLow,
		DefaultInterval: 72 * time.Hour,
	})
	_ = RegisterCustom(TaskDefinition{
		Type: "clear-test-2", Category: CategoryAnalysis,
		Name: "Clear2", Description: "test",
		CostTier: CostLow, RiskLevel: RiskLow,
		DefaultInterval: 72 * time.Hour,
	})
	ClearCustom()
	if IsCustom("clear-test-1") || IsCustom("clear-test-2") {
		t.Error("should be cleared")
	}
	if _, err := GetDefinition("clear-test-1"); err == nil {
		t.Error("clear-test-1 should not be in registry after ClearCustom")
	}
	if _, err := GetDefinition("clear-test-2"); err == nil {
		t.Error("clear-test-2 should not be in registry after ClearCustom")
	}
}

func TestSpecificDefaultIntervalOverrides(t *testing.T) {
	overrides := map[TaskType]time.Duration{
		TaskLintFix:         24 * time.Hour,
		TaskCommitNormalize: 24 * time.Hour,
		TaskBugFinder:       72 * time.Hour,
		TaskSecurityFootgun: 72 * time.Hour,
		TaskPIIScanner:      72 * time.Hour,
		TaskTestGap:         72 * time.Hour,
		TaskDeadCode:        72 * time.Hour,
	}
	for tt, want := range overrides {
		def, err := GetDefinition(tt)
		if err != nil {
			t.Fatalf("GetDefinition(%q) error: %v", tt, err)
		}
		if def.DefaultInterval != want {
			t.Errorf("Task %q DefaultInterval = %v, want %v", tt, def.DefaultInterval, want)
		}
	}
}
