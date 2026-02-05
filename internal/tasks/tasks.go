// Package tasks defines task structures and loading from various sources.
// Tasks can come from GitHub issues, local files, or inline definitions.
package tasks

import (
	"cmp"
	"fmt"
	"slices"
)

// CostTier represents the estimated token cost for a task.
type CostTier int

const (
	CostLow      CostTier = iota // 10-50k tokens
	CostMedium                   // 50-150k tokens
	CostHigh                     // 150-500k tokens
	CostVeryHigh                 // 500k+ tokens
)

func (c CostTier) String() string {
	switch c {
	case CostLow:
		return "Low (10-50k)"
	case CostMedium:
		return "Medium (50-150k)"
	case CostHigh:
		return "High (150-500k)"
	case CostVeryHigh:
		return "Very High (500k+)"
	default:
		return "Unknown"
	}
}

// TokenRange returns the min and max estimated tokens for this tier.
func (c CostTier) TokenRange() (min, max int) {
	switch c {
	case CostLow:
		return 10_000, 50_000
	case CostMedium:
		return 50_000, 150_000
	case CostHigh:
		return 150_000, 500_000
	case CostVeryHigh:
		return 500_000, 1_000_000 // Upper bound is approximate
	default:
		return 0, 0
	}
}

// RiskLevel represents the risk associated with a task.
type RiskLevel int

const (
	RiskLow RiskLevel = iota
	RiskMedium
	RiskHigh
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "Low"
	case RiskMedium:
		return "Medium"
	case RiskHigh:
		return "High"
	default:
		return "Unknown"
	}
}

// TaskCategory represents the type of output a task produces.
type TaskCategory int

const (
	// CategoryPR - "It's done - here's the PR"
	// Fully formed, review-ready artifacts.
	CategoryPR TaskCategory = iota

	// CategoryAnalysis - "Here's what I found"
	// Completed analysis with conclusions, no code touched.
	CategoryAnalysis

	// CategoryOptions - "Here are options - what do you want to do?"
	// Surfaces judgment calls, tradeoffs, design forks.
	CategoryOptions

	// CategorySafe - "I tried it safely"
	// Required execution/simulation but left no lasting side effects.
	CategorySafe

	// CategoryMap - "Here's the map"
	// Pure context laid out cleanly.
	CategoryMap

	// CategoryEmergency - "For when things go sideways"
	// Artifacts you hope to never need.
	CategoryEmergency
)

func (c TaskCategory) String() string {
	switch c {
	case CategoryPR:
		return "It's done - here's the PR"
	case CategoryAnalysis:
		return "Here's what I found"
	case CategoryOptions:
		return "Here are options"
	case CategorySafe:
		return "I tried it safely"
	case CategoryMap:
		return "Here's the map"
	case CategoryEmergency:
		return "For when things go sideways"
	default:
		return "Unknown"
	}
}

// TaskType represents a specific type of task.
type TaskType string

// Category 1: "It's done - here's the PR"
const (
	TaskLintFix              TaskType = "lint-fix"
	TaskBugFinder            TaskType = "bug-finder"
	TaskAutoDRY              TaskType = "auto-dry"
	TaskAPIContractVerify    TaskType = "api-contract-verify"
	TaskBackwardCompat       TaskType = "backward-compat"
	TaskBuildOptimize        TaskType = "build-optimize"
	TaskDocsBackfill         TaskType = "docs-backfill"
	TaskCommitNormalize      TaskType = "commit-normalize"
	TaskChangelogSynth       TaskType = "changelog-synth"
	TaskReleaseNotes         TaskType = "release-notes"
	TaskADRDraft             TaskType = "adr-draft"
)

// Category 2: "Here's what I found"
const (
	TaskDocDrift          TaskType = "doc-drift"
	TaskSemanticDiff      TaskType = "semantic-diff"
	TaskDeadCode          TaskType = "dead-code"
	TaskDependencyRisk    TaskType = "dependency-risk"
	TaskTestGap           TaskType = "test-gap"
	TaskTestFlakiness     TaskType = "test-flakiness"
	TaskLoggingAudit      TaskType = "logging-audit"
	TaskMetricsCoverage   TaskType = "metrics-coverage"
	TaskPerfRegression    TaskType = "perf-regression"
	TaskCostAttribution   TaskType = "cost-attribution"
	TaskSecurityFootgun   TaskType = "security-footgun"
	TaskPIIScanner        TaskType = "pii-scanner"
	TaskPrivacyPolicy     TaskType = "privacy-policy"
	TaskSchemaEvolution   TaskType = "schema-evolution"
	TaskEventTaxonomy     TaskType = "event-taxonomy"
	TaskRoadmapEntropy    TaskType = "roadmap-entropy"
	TaskBusFactor         TaskType = "bus-factor"
	TaskKnowledgeSilo     TaskType = "knowledge-silo"
)

// Category 3: "Here are options"
const (
	TaskGroomer           TaskType = "task-groomer"
	TaskGuideImprover     TaskType = "guide-improver"
	TaskIdeaGenerator     TaskType = "idea-generator"
	TaskTechDebtClassify  TaskType = "tech-debt-classify"
	TaskWhyAnnotator      TaskType = "why-annotator"
	TaskEdgeCaseEnum      TaskType = "edge-case-enum"
	TaskErrorMsgImprove   TaskType = "error-msg-improve"
	TaskSLOSuggester      TaskType = "slo-suggester"
	TaskUXCopySharpener   TaskType = "ux-copy-sharpener"
	TaskA11yLint          TaskType = "a11y-lint"
	TaskServiceAdvisor    TaskType = "service-advisor"
	TaskOwnershipBoundary TaskType = "ownership-boundary"
	TaskOncallEstimator   TaskType = "oncall-estimator"
)

// Category 4: "I tried it safely"
const (
	TaskMigrationRehearsal TaskType = "migration-rehearsal"
	TaskContractFuzzer     TaskType = "contract-fuzzer"
	TaskGoldenPath         TaskType = "golden-path"
	TaskPerfProfile        TaskType = "perf-profile"
	TaskAllocationProfile  TaskType = "allocation-profile"
)

// Category 5: "Here's the map"
const (
	TaskVisibilityInstrument TaskType = "visibility-instrument"
	TaskRepoTopology         TaskType = "repo-topology"
	TaskPermissionsMapper    TaskType = "permissions-mapper"
	TaskDataLifecycle        TaskType = "data-lifecycle"
	TaskFeatureFlagMonitor   TaskType = "feature-flag-monitor"
	TaskCISignalNoise        TaskType = "ci-signal-noise"
	TaskHistoricalContext    TaskType = "historical-context"
)

// Category 6: "For when things go sideways"
const (
	TaskRunbookGen    TaskType = "runbook-gen"
	TaskRollbackPlan  TaskType = "rollback-plan"
	TaskPostmortemGen TaskType = "postmortem-gen"
)

// TaskDefinition describes a built-in task type.
type TaskDefinition struct {
	Type        TaskType
	Category    TaskCategory
	Name        string
	Description string
	CostTier    CostTier
	RiskLevel   RiskLevel
}

// EstimatedTokens returns the token range for this task definition.
func (d TaskDefinition) EstimatedTokens() (min, max int) {
	return d.CostTier.TokenRange()
}

// registry holds all built-in task definitions.
var registry = map[TaskType]TaskDefinition{
	// Category 1: "It's done - here's the PR"
	TaskLintFix: {
		Type:        TaskLintFix,
		Category:    CategoryPR,
		Name:        "Linter Fixes",
		Description: "Automatically fix linting errors and style issues",
		CostTier:    CostLow,
		RiskLevel:   RiskLow,
	},
	TaskBugFinder: {
		Type:        TaskBugFinder,
		Category:    CategoryPR,
		Name:        "Bug Finder & Fixer",
		Description: "Identify and fix potential bugs in code",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskAutoDRY: {
		Type:        TaskAutoDRY,
		Category:    CategoryPR,
		Name:        "Auto DRY Refactoring",
		Description: "Identify and refactor duplicate code",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskAPIContractVerify: {
		Type:        TaskAPIContractVerify,
		Category:    CategoryPR,
		Name:        "API Contract Verification",
		Description: "Verify API contracts match implementation",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskBackwardCompat: {
		Type:        TaskBackwardCompat,
		Category:    CategoryPR,
		Name:        "Backward-Compatibility Checks",
		Description: "Check and ensure backward compatibility",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskBuildOptimize: {
		Type:        TaskBuildOptimize,
		Category:    CategoryPR,
		Name:        "Build Time Optimization",
		Description: "Optimize build configuration for faster builds",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskDocsBackfill: {
		Type:        TaskDocsBackfill,
		Category:    CategoryPR,
		Name:        "Documentation Backfiller",
		Description: "Generate missing documentation",
		CostTier:    CostLow,
		RiskLevel:   RiskLow,
	},
	TaskCommitNormalize: {
		Type:        TaskCommitNormalize,
		Category:    CategoryPR,
		Name:        "Commit Message Normalizer",
		Description: "Standardize commit message format",
		CostTier:    CostLow,
		RiskLevel:   RiskLow,
	},
	TaskChangelogSynth: {
		Type:        TaskChangelogSynth,
		Category:    CategoryPR,
		Name:        "Changelog Synthesizer",
		Description: "Generate changelog from commits",
		CostTier:    CostLow,
		RiskLevel:   RiskLow,
	},
	TaskReleaseNotes: {
		Type:        TaskReleaseNotes,
		Category:    CategoryPR,
		Name:        "Release Note Drafter",
		Description: "Draft release notes from changes",
		CostTier:    CostLow,
		RiskLevel:   RiskLow,
	},
	TaskADRDraft: {
		Type:        TaskADRDraft,
		Category:    CategoryPR,
		Name:        "ADR Drafter",
		Description: "Draft Architecture Decision Records",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},

	// Category 2: "Here's what I found"
	TaskDocDrift: {
		Type:        TaskDocDrift,
		Category:    CategoryAnalysis,
		Name:        "Doc Drift Detector",
		Description: "Detect documentation that's out of sync with code",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskSemanticDiff: {
		Type:        TaskSemanticDiff,
		Category:    CategoryAnalysis,
		Name:        "Semantic Diff Explainer",
		Description: "Explain the semantic meaning of code changes",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskDeadCode: {
		Type:        TaskDeadCode,
		Category:    CategoryAnalysis,
		Name:        "Dead Code Detector",
		Description: "Find unused code that can be removed",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskDependencyRisk: {
		Type:        TaskDependencyRisk,
		Category:    CategoryAnalysis,
		Name:        "Dependency Risk Scanner",
		Description: "Analyze dependencies for security and maintenance risks",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskTestGap: {
		Type:        TaskTestGap,
		Category:    CategoryAnalysis,
		Name:        "Test Gap Finder",
		Description: "Identify areas lacking test coverage",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskTestFlakiness: {
		Type:        TaskTestFlakiness,
		Category:    CategoryAnalysis,
		Name:        "Test Flakiness Analyzer",
		Description: "Identify and analyze flaky tests",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskLoggingAudit: {
		Type:        TaskLoggingAudit,
		Category:    CategoryAnalysis,
		Name:        "Logging Quality Auditor",
		Description: "Audit logging for completeness and quality",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskMetricsCoverage: {
		Type:        TaskMetricsCoverage,
		Category:    CategoryAnalysis,
		Name:        "Metrics Coverage Analyzer",
		Description: "Analyze metrics instrumentation coverage",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskPerfRegression: {
		Type:        TaskPerfRegression,
		Category:    CategoryAnalysis,
		Name:        "Performance Regression Spotter",
		Description: "Identify potential performance regressions",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskCostAttribution: {
		Type:        TaskCostAttribution,
		Category:    CategoryAnalysis,
		Name:        "Cost Attribution Estimator",
		Description: "Estimate resource costs by component",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskSecurityFootgun: {
		Type:        TaskSecurityFootgun,
		Category:    CategoryAnalysis,
		Name:        "Security Foot-Gun Finder",
		Description: "Find common security anti-patterns",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskPIIScanner: {
		Type:        TaskPIIScanner,
		Category:    CategoryAnalysis,
		Name:        "PII Exposure Scanner",
		Description: "Scan for potential PII exposure",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskPrivacyPolicy: {
		Type:        TaskPrivacyPolicy,
		Category:    CategoryAnalysis,
		Name:        "Privacy Policy Consistency Checker",
		Description: "Check code against privacy policy claims",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskSchemaEvolution: {
		Type:        TaskSchemaEvolution,
		Category:    CategoryAnalysis,
		Name:        "Schema Evolution Advisor",
		Description: "Analyze database schema changes",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskEventTaxonomy: {
		Type:        TaskEventTaxonomy,
		Category:    CategoryAnalysis,
		Name:        "Event Taxonomy Normalizer",
		Description: "Normalize event naming and structure",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskRoadmapEntropy: {
		Type:        TaskRoadmapEntropy,
		Category:    CategoryAnalysis,
		Name:        "Roadmap Entropy Detector",
		Description: "Detect roadmap scope creep and drift",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskBusFactor: {
		Type:        TaskBusFactor,
		Category:    CategoryAnalysis,
		Name:        "Bus-Factor Analyzer",
		Description: "Analyze code ownership concentration",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskKnowledgeSilo: {
		Type:        TaskKnowledgeSilo,
		Category:    CategoryAnalysis,
		Name:        "Knowledge Silo Detector",
		Description: "Identify knowledge silos in the team",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},

	// Category 3: "Here are options"
	TaskGroomer: {
		Type:        TaskGroomer,
		Category:    CategoryOptions,
		Name:        "Task Groomer",
		Description: "Refine and clarify task definitions",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskGuideImprover: {
		Type:        TaskGuideImprover,
		Category:    CategoryOptions,
		Name:        "Guide/Skill Improver",
		Description: "Suggest improvements to guides and skills",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskIdeaGenerator: {
		Type:        TaskIdeaGenerator,
		Category:    CategoryOptions,
		Name:        "Idea Generator",
		Description: "Generate improvement ideas for the codebase",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskTechDebtClassify: {
		Type:        TaskTechDebtClassify,
		Category:    CategoryOptions,
		Name:        "Tech-Debt Classifier",
		Description: "Classify and prioritize technical debt",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskWhyAnnotator: {
		Type:        TaskWhyAnnotator,
		Category:    CategoryOptions,
		Name:        "Why Does This Exist Annotator",
		Description: "Document the purpose of unclear code",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskEdgeCaseEnum: {
		Type:        TaskEdgeCaseEnum,
		Category:    CategoryOptions,
		Name:        "Edge-Case Enumerator",
		Description: "Enumerate potential edge cases",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskErrorMsgImprove: {
		Type:        TaskErrorMsgImprove,
		Category:    CategoryOptions,
		Name:        "Error-Message Improver",
		Description: "Suggest better error messages",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskSLOSuggester: {
		Type:        TaskSLOSuggester,
		Category:    CategoryOptions,
		Name:        "SLO/SLA Candidate Suggester",
		Description: "Suggest SLO/SLA candidates",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskUXCopySharpener: {
		Type:        TaskUXCopySharpener,
		Category:    CategoryOptions,
		Name:        "UX Copy Sharpener",
		Description: "Improve user-facing text",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskA11yLint: {
		Type:        TaskA11yLint,
		Category:    CategoryOptions,
		Name:        "Accessibility Linting",
		Description: "Non-checkbox accessibility analysis",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskServiceAdvisor: {
		Type:        TaskServiceAdvisor,
		Category:    CategoryOptions,
		Name:        "Should This Be a Service Advisor",
		Description: "Analyze service boundary decisions",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskOwnershipBoundary: {
		Type:        TaskOwnershipBoundary,
		Category:    CategoryOptions,
		Name:        "Ownership Boundary Suggester",
		Description: "Suggest code ownership boundaries",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskOncallEstimator: {
		Type:        TaskOncallEstimator,
		Category:    CategoryOptions,
		Name:        "Oncall Load Estimator",
		Description: "Estimate oncall load from code changes",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},

	// Category 4: "I tried it safely"
	TaskMigrationRehearsal: {
		Type:        TaskMigrationRehearsal,
		Category:    CategorySafe,
		Name:        "Migration Rehearsal Runner",
		Description: "Rehearse migrations without side effects",
		CostTier:    CostVeryHigh,
		RiskLevel:   RiskHigh,
	},
	TaskContractFuzzer: {
		Type:        TaskContractFuzzer,
		Category:    CategorySafe,
		Name:        "Integration Contract Fuzzer",
		Description: "Fuzz test integration contracts",
		CostTier:    CostVeryHigh,
		RiskLevel:   RiskHigh,
	},
	TaskGoldenPath: {
		Type:        TaskGoldenPath,
		Category:    CategorySafe,
		Name:        "Golden-Path Recorder",
		Description: "Record golden path test scenarios",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskPerfProfile: {
		Type:        TaskPerfProfile,
		Category:    CategorySafe,
		Name:        "Performance Profiling Runs",
		Description: "Run performance profiling",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskAllocationProfile: {
		Type:        TaskAllocationProfile,
		Category:    CategorySafe,
		Name:        "Allocation/Hot-Path Profiling",
		Description: "Profile memory allocation and hot paths",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},

	// Category 5: "Here's the map"
	TaskVisibilityInstrument: {
		Type:        TaskVisibilityInstrument,
		Category:    CategoryMap,
		Name:        "Visibility Instrumentor",
		Description: "Instrument code for observability",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskRepoTopology: {
		Type:        TaskRepoTopology,
		Category:    CategoryMap,
		Name:        "Repo Topology Visualizer",
		Description: "Visualize repository structure",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskPermissionsMapper: {
		Type:        TaskPermissionsMapper,
		Category:    CategoryMap,
		Name:        "Permissions/Auth Surface Mapper",
		Description: "Map permissions and auth surfaces",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskDataLifecycle: {
		Type:        TaskDataLifecycle,
		Category:    CategoryMap,
		Name:        "Data Lifecycle Tracer",
		Description: "Trace data lifecycle through the system",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskFeatureFlagMonitor: {
		Type:        TaskFeatureFlagMonitor,
		Category:    CategoryMap,
		Name:        "Feature Flag Lifecycle Monitor",
		Description: "Monitor feature flag usage and lifecycle",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskCISignalNoise: {
		Type:        TaskCISignalNoise,
		Category:    CategoryMap,
		Name:        "CI Signal-to-Noise Scorer",
		Description: "Score CI signal vs noise ratio",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},
	TaskHistoricalContext: {
		Type:        TaskHistoricalContext,
		Category:    CategoryMap,
		Name:        "Historical Context Summarizer",
		Description: "Summarize historical context of code",
		CostTier:    CostMedium,
		RiskLevel:   RiskLow,
	},

	// Category 6: "For when things go sideways"
	TaskRunbookGen: {
		Type:        TaskRunbookGen,
		Category:    CategoryEmergency,
		Name:        "Runbook Generator",
		Description: "Generate operational runbooks",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskRollbackPlan: {
		Type:        TaskRollbackPlan,
		Category:    CategoryEmergency,
		Name:        "Rollback Plan Generator",
		Description: "Generate rollback plans for changes",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
	TaskPostmortemGen: {
		Type:        TaskPostmortemGen,
		Category:    CategoryEmergency,
		Name:        "Incident Postmortem Draft Generator",
		Description: "Draft incident postmortem documents",
		CostTier:    CostHigh,
		RiskLevel:   RiskMedium,
	},
}

// GetDefinition returns the definition for a task type.
func GetDefinition(taskType TaskType) (TaskDefinition, error) {
	def, ok := registry[taskType]
	if !ok {
		return TaskDefinition{}, fmt.Errorf("unknown task type: %s", taskType)
	}
	return def, nil
}

// GetCostEstimate returns the estimated token cost range for a task type.
func GetCostEstimate(taskType TaskType) (min, max int, err error) {
	def, err := GetDefinition(taskType)
	if err != nil {
		return 0, 0, err
	}
	min, max = def.EstimatedTokens()
	return min, max, nil
}

// GetTasksByCategory returns all task definitions in a category.
func GetTasksByCategory(category TaskCategory) []TaskDefinition {
	var tasks []TaskDefinition
	for _, def := range registry {
		if def.Category == category {
			tasks = append(tasks, def)
		}
	}
	return tasks
}

// GetTasksByCostTier returns all task definitions with a given cost tier.
func GetTasksByCostTier(tier CostTier) []TaskDefinition {
	var tasks []TaskDefinition
	for _, def := range registry {
		if def.CostTier == tier {
			tasks = append(tasks, def)
		}
	}
	return tasks
}

// GetTasksByRiskLevel returns all task definitions with a given risk level.
func GetTasksByRiskLevel(risk RiskLevel) []TaskDefinition {
	var tasks []TaskDefinition
	for _, def := range registry {
		if def.RiskLevel == risk {
			tasks = append(tasks, def)
		}
	}
	return tasks
}

// AllTaskTypes returns all registered task types.
func AllTaskTypes() []TaskType {
	types := make([]TaskType, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}

// AllDefinitions returns all registered task definitions.
func AllDefinitions() []TaskDefinition {
	defs := make([]TaskDefinition, 0, len(registry))
	for _, def := range registry {
		defs = append(defs, def)
	}
	return defs
}

// Task represents a unit of work for an AI agent.
type Task struct {
	ID          string
	Title       string
	Description string
	Priority    int
	Type        TaskType // Optional: links to a TaskDefinition
	// TODO: Add more fields (labels, assignee, source, etc.)
}

// Queue holds tasks to be processed.
type Queue struct {
	// TODO: Add fields
}

// NewQueue creates an empty task queue.
func NewQueue() *Queue {
	// TODO: Implement
	return &Queue{}
}

// Add queues a task.
func (q *Queue) Add(t Task) {
	// TODO: Implement
}

// Next returns the highest priority task.
func (q *Queue) Next() *Task {
	// TODO: Implement
	return nil
}
