// Package tasks defines task structures and loading from various sources.
// Tasks can come from GitHub issues, local files, or inline definitions.
package tasks

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// CostTier represents the estimated token cost for a task.
type CostTier int

const (
	CostLow      CostTier = iota // 10-50k tokens
	CostMedium                   // 50-150k tokens
	CostHigh                     // 150-500k tokens
	CostVeryHigh                 // 500k+ tokens
)

// String returns a human-readable label for the cost tier.
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

// String returns a human-readable label for the risk level.
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

// String returns a human-readable description of the task category.
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
	TaskLintFix           TaskType = "lint-fix"
	TaskBugFinder         TaskType = "bug-finder"
	TaskAutoDRY           TaskType = "auto-dry"
	TaskSkillGroom        TaskType = "skill-groom"
	TaskAPIContractVerify TaskType = "api-contract-verify"
	TaskBackwardCompat    TaskType = "backward-compat"
	TaskBuildOptimize     TaskType = "build-optimize"
	TaskDocsBackfill      TaskType = "docs-backfill"
	TaskCommitNormalize   TaskType = "commit-normalize"
	TaskChangelogSynth    TaskType = "changelog-synth"
	TaskReleaseNotes      TaskType = "release-notes"
	TaskADRDraft          TaskType = "adr-draft"
	TaskTDReview          TaskType = "td-review"
)

// Category 2: "Here's what I found"
const (
	TaskDocDrift        TaskType = "doc-drift"
	TaskSemanticDiff    TaskType = "semantic-diff"
	TaskDeadCode        TaskType = "dead-code"
	TaskDependencyRisk  TaskType = "dependency-risk"
	TaskTestGap         TaskType = "test-gap"
	TaskTestFlakiness   TaskType = "test-flakiness"
	TaskLoggingAudit    TaskType = "logging-audit"
	TaskMetricsCoverage TaskType = "metrics-coverage"
	TaskPerfRegression  TaskType = "perf-regression"
	TaskCostAttribution TaskType = "cost-attribution"
	TaskSecurityFootgun TaskType = "security-footgun"
	TaskPIIScanner      TaskType = "pii-scanner"
	TaskPrivacyPolicy   TaskType = "privacy-policy"
	TaskSchemaEvolution TaskType = "schema-evolution"
	TaskEventTaxonomy   TaskType = "event-taxonomy"
	TaskRoadmapEntropy  TaskType = "roadmap-entropy"
	TaskBusFactor       TaskType = "bus-factor"
	TaskKnowledgeSilo   TaskType = "knowledge-silo"
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
	Type              TaskType
	Category          TaskCategory
	Name              string
	Description       string
	CostTier          CostTier
	RiskLevel         RiskLevel
	DefaultInterval   time.Duration
	DisabledByDefault bool // Requires explicit opt-in via tasks.enabled
}

// DefaultIntervalForCategory returns the default re-run interval for a task category.
func DefaultIntervalForCategory(cat TaskCategory) time.Duration {
	switch cat {
	case CategoryPR:
		return 168 * time.Hour // 7 days
	case CategoryAnalysis:
		return 72 * time.Hour // 3 days
	case CategoryOptions:
		return 168 * time.Hour // 7 days
	case CategorySafe:
		return 336 * time.Hour // 14 days
	case CategoryMap:
		return 168 * time.Hour // 7 days
	case CategoryEmergency:
		return 720 * time.Hour // 30 days
	default:
		return 168 * time.Hour // 7 days
	}
}

// EstimatedTokens returns the token range for this task definition.
func (d TaskDefinition) EstimatedTokens() (min, max int) {
	return d.CostTier.TokenRange()
}

// customTypes tracks which task types were registered via RegisterCustom.
var customTypes = map[TaskType]bool{}

// registry holds all built-in task definitions.
var registry = map[TaskType]TaskDefinition{
	// Category 1: "It's done - here's the PR"
	TaskLintFix: {
		Type:            TaskLintFix,
		Category:        CategoryPR,
		Name:            "Linter Fixes",
		Description:     "Automatically fix linting errors and style issues",
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 24 * time.Hour,
	},
	TaskBugFinder: {
		Type:            TaskBugFinder,
		Category:        CategoryPR,
		Name:            "Bug Finder & Fixer",
		Description:     "Identify and fix potential bugs in code",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 72 * time.Hour,
	},
	TaskAutoDRY: {
		Type:            TaskAutoDRY,
		Category:        CategoryPR,
		Name:            "Auto DRY Refactoring",
		Description:     "Identify and refactor duplicate code",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 168 * time.Hour,
	},
	TaskSkillGroom: {
		Type:     TaskSkillGroom,
		Category: CategoryPR,
		Name:     "Skill Grooming",
		Description: `Audit and update project-local agent skills to match the current codebase.
Use README.md as the primary project context for commands, architecture, and workflows.
For Agent Skills documentation lookup, fetch https://agentskills.io/llms.txt first and use it as the index before reading specific spec pages.
Inspect .claude/skills and .codex/skills for SKILL.md files, validate frontmatter and naming rules against the spec, and fix stale references to files/scripts/paths.
Apply safe updates directly, and leave concise follow-ups for anything uncertain.`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 168 * time.Hour,
	},
	TaskAPIContractVerify: {
		Type:            TaskAPIContractVerify,
		Category:        CategoryPR,
		Name:            "API Contract Verification",
		Description:     "Verify API contracts match implementation",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskBackwardCompat: {
		Type:            TaskBackwardCompat,
		Category:        CategoryPR,
		Name:            "Backward-Compatibility Checks",
		Description:     "Check and ensure backward compatibility",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskBuildOptimize: {
		Type:            TaskBuildOptimize,
		Category:        CategoryPR,
		Name:            "Build Time Optimization",
		Description:     "Optimize build configuration for faster builds",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 168 * time.Hour,
	},
	TaskDocsBackfill: {
		Type:            TaskDocsBackfill,
		Category:        CategoryPR,
		Name:            "Documentation Backfiller",
		Description:     "Generate missing documentation",
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskCommitNormalize: {
		Type:            TaskCommitNormalize,
		Category:        CategoryPR,
		Name:            "Commit Message Normalizer",
		Description:     "Standardize commit message format",
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 24 * time.Hour,
	},
	TaskChangelogSynth: {
		Type:            TaskChangelogSynth,
		Category:        CategoryPR,
		Name:            "Changelog Synthesizer",
		Description:     "Generate changelog from commits",
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskReleaseNotes: {
		Type:            TaskReleaseNotes,
		Category:        CategoryPR,
		Name:            "Release Note Drafter",
		Description:     "Draft release notes from changes",
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskADRDraft: {
		Type:            TaskADRDraft,
		Category:        CategoryPR,
		Name:            "ADR Drafter",
		Description:     "Draft Architecture Decision Records",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskTDReview: {
		Type:     TaskTDReview,
		Category: CategoryPR,
		Name:     "TD Review Session",
		Description: `Start a td review session and do a detailed review of open reviews. ` +
			`For obvious fixes, create a td bug task with a detailed description of the problem ` +
			`and fix them immediately. Create new td tasks with detailed descriptions for bigger ` +
			`bugs or issues that should be fixed in a later session. Verify that changes have ` +
			`tests—if not, create td tasks to add test coverage. For reviews that can be processed ` +
			`in parallel, use subagents. Once tasks related to previously opened bugs are complete, ` +
			`close the in-progress tasks.`,
		CostTier:          CostHigh,
		RiskLevel:         RiskMedium,
		DefaultInterval:   72 * time.Hour,
		DisabledByDefault: true,
	},

	// Category 2: "Here's what I found"
	TaskDocDrift: {
		Type:            TaskDocDrift,
		Category:        CategoryAnalysis,
		Name:            "Doc Drift Detector",
		Description:     "Detect documentation that's out of sync with code",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskSemanticDiff: {
		Type:            TaskSemanticDiff,
		Category:        CategoryAnalysis,
		Name:            "Semantic Diff Explainer",
		Description:     "Explain the semantic meaning of code changes",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskDeadCode: {
		Type:            TaskDeadCode,
		Category:        CategoryAnalysis,
		Name:            "Dead Code Detector",
		Description:     "Find unused code that can be removed",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskDependencyRisk: {
		Type:            TaskDependencyRisk,
		Category:        CategoryAnalysis,
		Name:            "Dependency Risk Scanner",
		Description:     "Analyze dependencies for security and maintenance risks",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskTestGap: {
		Type:            TaskTestGap,
		Category:        CategoryAnalysis,
		Name:            "Test Gap Finder",
		Description:     "Identify areas lacking test coverage",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskTestFlakiness: {
		Type:            TaskTestFlakiness,
		Category:        CategoryAnalysis,
		Name:            "Test Flakiness Analyzer",
		Description:     "Identify and analyze flaky tests",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskLoggingAudit: {
		Type:            TaskLoggingAudit,
		Category:        CategoryAnalysis,
		Name:            "Logging Quality Auditor",
		Description:     "Audit logging for completeness and quality",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskMetricsCoverage: {
		Type:            TaskMetricsCoverage,
		Category:        CategoryAnalysis,
		Name:            "Metrics Coverage Analyzer",
		Description:     "Analyze metrics instrumentation coverage",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskPerfRegression: {
		Type:            TaskPerfRegression,
		Category:        CategoryAnalysis,
		Name:            "Performance Regression Spotter",
		Description:     "Identify potential performance regressions",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskCostAttribution: {
		Type:            TaskCostAttribution,
		Category:        CategoryAnalysis,
		Name:            "Cost Attribution Estimator",
		Description:     "Estimate resource costs by component",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskSecurityFootgun: {
		Type:            TaskSecurityFootgun,
		Category:        CategoryAnalysis,
		Name:            "Security Foot-Gun Finder",
		Description:     "Find common security anti-patterns",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskPIIScanner: {
		Type:            TaskPIIScanner,
		Category:        CategoryAnalysis,
		Name:            "PII Exposure Scanner",
		Description:     "Scan for potential PII exposure",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskPrivacyPolicy: {
		Type:            TaskPrivacyPolicy,
		Category:        CategoryAnalysis,
		Name:            "Privacy Policy Consistency Checker",
		Description:     "Check code against privacy policy claims",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskSchemaEvolution: {
		Type:            TaskSchemaEvolution,
		Category:        CategoryAnalysis,
		Name:            "Schema Evolution Advisor",
		Description:     "Analyze database schema changes",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskEventTaxonomy: {
		Type:            TaskEventTaxonomy,
		Category:        CategoryAnalysis,
		Name:            "Event Taxonomy Normalizer",
		Description:     "Normalize event naming and structure",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskRoadmapEntropy: {
		Type:            TaskRoadmapEntropy,
		Category:        CategoryAnalysis,
		Name:            "Roadmap Entropy Detector",
		Description:     "Detect roadmap scope creep and drift",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskBusFactor: {
		Type:            TaskBusFactor,
		Category:        CategoryAnalysis,
		Name:            "Bus-Factor Analyzer",
		Description:     "Analyze code ownership concentration",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskKnowledgeSilo: {
		Type:            TaskKnowledgeSilo,
		Category:        CategoryAnalysis,
		Name:            "Knowledge Silo Detector",
		Description:     "Identify knowledge silos in the team",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},

	// Category 3: "Here are options"
	TaskGroomer: {
		Type:            TaskGroomer,
		Category:        CategoryOptions,
		Name:            "Task Groomer",
		Description:     "Refine and clarify task definitions",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskGuideImprover: {
		Type:            TaskGuideImprover,
		Category:        CategoryOptions,
		Name:            "Guide/Skill Improver",
		Description:     "Suggest improvements to guides and skills",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskIdeaGenerator: {
		Type:            TaskIdeaGenerator,
		Category:        CategoryOptions,
		Name:            "Idea Generator",
		Description:     "Generate improvement ideas for the codebase",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskTechDebtClassify: {
		Type:            TaskTechDebtClassify,
		Category:        CategoryOptions,
		Name:            "Tech-Debt Classifier",
		Description:     "Classify and prioritize technical debt",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskWhyAnnotator: {
		Type:            TaskWhyAnnotator,
		Category:        CategoryOptions,
		Name:            "Why Does This Exist Annotator",
		Description:     "Document the purpose of unclear code",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskEdgeCaseEnum: {
		Type:            TaskEdgeCaseEnum,
		Category:        CategoryOptions,
		Name:            "Edge-Case Enumerator",
		Description:     "Enumerate potential edge cases",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskErrorMsgImprove: {
		Type:            TaskErrorMsgImprove,
		Category:        CategoryOptions,
		Name:            "Error-Message Improver",
		Description:     "Suggest better error messages",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskSLOSuggester: {
		Type:            TaskSLOSuggester,
		Category:        CategoryOptions,
		Name:            "SLO/SLA Candidate Suggester",
		Description:     "Suggest SLO/SLA candidates",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskUXCopySharpener: {
		Type:            TaskUXCopySharpener,
		Category:        CategoryOptions,
		Name:            "UX Copy Sharpener",
		Description:     "Improve user-facing text",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskA11yLint: {
		Type:            TaskA11yLint,
		Category:        CategoryOptions,
		Name:            "Accessibility Linting",
		Description:     "Non-checkbox accessibility analysis",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskServiceAdvisor: {
		Type:            TaskServiceAdvisor,
		Category:        CategoryOptions,
		Name:            "Should This Be a Service Advisor",
		Description:     "Analyze service boundary decisions",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 168 * time.Hour,
	},
	TaskOwnershipBoundary: {
		Type:            TaskOwnershipBoundary,
		Category:        CategoryOptions,
		Name:            "Ownership Boundary Suggester",
		Description:     "Suggest code ownership boundaries",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskOncallEstimator: {
		Type:            TaskOncallEstimator,
		Category:        CategoryOptions,
		Name:            "Oncall Load Estimator",
		Description:     "Estimate oncall load from code changes",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},

	// Category 4: "I tried it safely"
	TaskMigrationRehearsal: {
		Type:            TaskMigrationRehearsal,
		Category:        CategorySafe,
		Name:            "Migration Rehearsal Runner",
		Description:     "Rehearse migrations without side effects",
		CostTier:        CostVeryHigh,
		RiskLevel:       RiskHigh,
		DefaultInterval: 336 * time.Hour,
	},
	TaskContractFuzzer: {
		Type:            TaskContractFuzzer,
		Category:        CategorySafe,
		Name:            "Integration Contract Fuzzer",
		Description:     "Fuzz test integration contracts",
		CostTier:        CostVeryHigh,
		RiskLevel:       RiskHigh,
		DefaultInterval: 336 * time.Hour,
	},
	TaskGoldenPath: {
		Type:            TaskGoldenPath,
		Category:        CategorySafe,
		Name:            "Golden-Path Recorder",
		Description:     "Record golden path test scenarios",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 336 * time.Hour,
	},
	TaskPerfProfile: {
		Type:            TaskPerfProfile,
		Category:        CategorySafe,
		Name:            "Performance Profiling Runs",
		Description:     "Run performance profiling",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 336 * time.Hour,
	},
	TaskAllocationProfile: {
		Type:            TaskAllocationProfile,
		Category:        CategorySafe,
		Name:            "Allocation/Hot-Path Profiling",
		Description:     "Profile memory allocation and hot paths",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 336 * time.Hour,
	},

	// Category 5: "Here's the map"
	TaskVisibilityInstrument: {
		Type:            TaskVisibilityInstrument,
		Category:        CategoryMap,
		Name:            "Visibility Instrumentor",
		Description:     "Instrument code for observability",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 168 * time.Hour,
	},
	TaskRepoTopology: {
		Type:            TaskRepoTopology,
		Category:        CategoryMap,
		Name:            "Repo Topology Visualizer",
		Description:     "Visualize repository structure",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskPermissionsMapper: {
		Type:            TaskPermissionsMapper,
		Category:        CategoryMap,
		Name:            "Permissions/Auth Surface Mapper",
		Description:     "Map permissions and auth surfaces",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskDataLifecycle: {
		Type:            TaskDataLifecycle,
		Category:        CategoryMap,
		Name:            "Data Lifecycle Tracer",
		Description:     "Trace data lifecycle through the system",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskFeatureFlagMonitor: {
		Type:            TaskFeatureFlagMonitor,
		Category:        CategoryMap,
		Name:            "Feature Flag Lifecycle Monitor",
		Description:     "Monitor feature flag usage and lifecycle",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskCISignalNoise: {
		Type:            TaskCISignalNoise,
		Category:        CategoryMap,
		Name:            "CI Signal-to-Noise Scorer",
		Description:     "Score CI signal vs noise ratio",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskHistoricalContext: {
		Type:            TaskHistoricalContext,
		Category:        CategoryMap,
		Name:            "Historical Context Summarizer",
		Description:     "Summarize historical context of code",
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},

	// Category 6: "For when things go sideways"
	TaskRunbookGen: {
		Type:            TaskRunbookGen,
		Category:        CategoryEmergency,
		Name:            "Runbook Generator",
		Description:     "Generate operational runbooks",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 720 * time.Hour,
	},
	TaskRollbackPlan: {
		Type:            TaskRollbackPlan,
		Category:        CategoryEmergency,
		Name:            "Rollback Plan Generator",
		Description:     "Generate rollback plans for changes",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 720 * time.Hour,
	},
	TaskPostmortemGen: {
		Type:            TaskPostmortemGen,
		Category:        CategoryEmergency,
		Name:            "Incident Postmortem Draft Generator",
		Description:     "Draft incident postmortem documents",
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 720 * time.Hour,
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

// DefaultDisabledTaskTypes returns task types that are disabled by default
// and require explicit opt-in via the tasks.enabled config list.
func DefaultDisabledTaskTypes() []TaskType {
	var types []TaskType
	for _, def := range registry {
		if def.DisabledByDefault {
			types = append(types, def.Type)
		}
	}
	return types
}

// AllDefinitionsSorted returns all registered task definitions sorted by
// Category first, then by Name within each category. This provides stable,
// deterministic ordering for CLI output.
func AllDefinitionsSorted() []TaskDefinition {
	defs := AllDefinitions()
	slices.SortFunc(defs, func(a, b TaskDefinition) int {
		if c := cmp.Compare(a.Category, b.Category); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return defs
}

// RegisterCustom registers a custom task definition. Returns an error if the
// type is already registered (built-in or custom).
func RegisterCustom(def TaskDefinition) error {
	if _, exists := registry[def.Type]; exists {
		return fmt.Errorf("task type %q already registered", def.Type)
	}
	registry[def.Type] = def
	customTypes[def.Type] = true
	return nil
}

// UnregisterCustom removes a custom task type. Built-in types are not affected.
func UnregisterCustom(taskType TaskType) {
	if customTypes[taskType] {
		delete(registry, taskType)
		delete(customTypes, taskType)
	}
}

// IsCustom reports whether a task type was registered via RegisterCustom.
func IsCustom(taskType TaskType) bool {
	return customTypes[taskType]
}

// ClearCustom removes all custom task types from the registry.
func ClearCustom() {
	for t := range customTypes {
		delete(registry, t)
	}
	customTypes = map[TaskType]bool{}
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
