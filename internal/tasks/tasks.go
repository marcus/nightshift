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
	// CostLow is the lowest cost tier, roughly 10-50k tokens.
	CostLow CostTier = iota
	// CostMedium is a moderate cost tier, roughly 50-150k tokens.
	CostMedium
	// CostHigh is a high cost tier, roughly 150-500k tokens.
	CostHigh
	// CostVeryHigh is the highest cost tier, roughly 500k+ tokens.
	CostVeryHigh
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
	// RiskLow indicates a task with minimal blast radius.
	RiskLow RiskLevel = iota
	// RiskMedium indicates a task with moderate risk requiring review.
	RiskMedium
	// RiskHigh indicates a task with significant risk of side effects.
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
	// TaskLintFix is the task type for linter fixes.
	TaskLintFix TaskType = "lint-fix"
	// TaskBugFinder is the task type for the bug finder and fixer.
	TaskBugFinder TaskType = "bug-finder"
	// TaskAutoDRY is the task type for automatic DRY refactoring.
	TaskAutoDRY TaskType = "auto-dry"
	// TaskSkillGroom is the task type for grooming project-local agent skills.
	TaskSkillGroom TaskType = "skill-groom"
	// TaskAPIContractVerify is the task type for API contract verification.
	TaskAPIContractVerify TaskType = "api-contract-verify"
	// TaskBackwardCompat is the task type for backward-compatibility checks.
	TaskBackwardCompat TaskType = "backward-compat"
	// TaskBuildOptimize is the task type for build-time optimization.
	TaskBuildOptimize TaskType = "build-optimize"
	// TaskDocsBackfill is the task type for the documentation backfiller.
	TaskDocsBackfill TaskType = "docs-backfill"
	// TaskCommitNormalize is the task type for the commit message normalizer.
	TaskCommitNormalize TaskType = "commit-normalize"
	// TaskChangelogSynth is the task type for the changelog synthesizer.
	TaskChangelogSynth TaskType = "changelog-synth"
	// TaskReleaseNotes is the task type for the release note drafter.
	TaskReleaseNotes TaskType = "release-notes"
	// TaskADRDraft is the task type for the ADR drafter.
	TaskADRDraft TaskType = "adr-draft"
	// TaskTDReview is the task type for a td review session.
	TaskTDReview TaskType = "td-review"
)

// Category 2: "Here's what I found"
const (
	// TaskDocDrift is the task type for the doc drift detector.
	TaskDocDrift TaskType = "doc-drift"
	// TaskSemanticDiff is the task type for the semantic diff explainer.
	TaskSemanticDiff TaskType = "semantic-diff"
	// TaskDeadCode is the task type for the dead code detector.
	TaskDeadCode TaskType = "dead-code"
	// TaskDependencyRisk is the task type for the dependency risk scanner.
	TaskDependencyRisk TaskType = "dependency-risk"
	// TaskTestGap is the task type for the test gap finder.
	TaskTestGap TaskType = "test-gap"
	// TaskTestFlakiness is the task type for the test flakiness analyzer.
	TaskTestFlakiness TaskType = "test-flakiness"
	// TaskLoggingAudit is the task type for the logging quality auditor.
	TaskLoggingAudit TaskType = "logging-audit"
	// TaskMetricsCoverage is the task type for the metrics coverage analyzer.
	TaskMetricsCoverage TaskType = "metrics-coverage"
	// TaskPerfRegression is the task type for the performance regression spotter.
	TaskPerfRegression TaskType = "perf-regression"
	// TaskCostAttribution is the task type for the cost attribution estimator.
	TaskCostAttribution TaskType = "cost-attribution"
	// TaskSecurityFootgun is the task type for the security foot-gun finder.
	TaskSecurityFootgun TaskType = "security-footgun"
	// TaskPIIScanner is the task type for the PII exposure scanner.
	TaskPIIScanner TaskType = "pii-scanner"
	// TaskPrivacyPolicy is the task type for the privacy policy consistency checker.
	TaskPrivacyPolicy TaskType = "privacy-policy"
	// TaskSchemaEvolution is the task type for the schema evolution advisor.
	TaskSchemaEvolution TaskType = "schema-evolution"
	// TaskEventTaxonomy is the task type for the event taxonomy normalizer.
	TaskEventTaxonomy TaskType = "event-taxonomy"
	// TaskRoadmapEntropy is the task type for the roadmap entropy detector.
	TaskRoadmapEntropy TaskType = "roadmap-entropy"
	// TaskBusFactor is the task type for the bus-factor analyzer.
	TaskBusFactor TaskType = "bus-factor"
	// TaskKnowledgeSilo is the task type for the knowledge silo detector.
	TaskKnowledgeSilo TaskType = "knowledge-silo"
)

// Category 3: "Here are options"
const (
	// TaskGroomer is the task type for the task groomer.
	TaskGroomer TaskType = "task-groomer"
	// TaskGuideImprover is the task type for the guide and skill improver.
	TaskGuideImprover TaskType = "guide-improver"
	// TaskIdeaGenerator is the task type for the idea generator.
	TaskIdeaGenerator TaskType = "idea-generator"
	// TaskTechDebtClassify is the task type for the tech-debt classifier.
	TaskTechDebtClassify TaskType = "tech-debt-classify"
	// TaskWhyAnnotator is the task type for the "why does this exist" annotator.
	TaskWhyAnnotator TaskType = "why-annotator"
	// TaskEdgeCaseEnum is the task type for the edge-case enumerator.
	TaskEdgeCaseEnum TaskType = "edge-case-enum"
	// TaskErrorMsgImprove is the task type for the error-message improver.
	TaskErrorMsgImprove TaskType = "error-msg-improve"
	// TaskSLOSuggester is the task type for the SLO suggester.
	TaskSLOSuggester TaskType = "slo-suggester"
	// TaskUXCopySharpener is the task type for the UX copy sharpener.
	TaskUXCopySharpener TaskType = "ux-copy-sharpener"
	// TaskA11yLint is the task type for the accessibility linter.
	TaskA11yLint TaskType = "a11y-lint"
	// TaskServiceAdvisor is the task type for the service advisor.
	TaskServiceAdvisor TaskType = "service-advisor"
	// TaskOwnershipBoundary is the task type for the ownership boundary mapper.
	TaskOwnershipBoundary TaskType = "ownership-boundary"
	// TaskOncallEstimator is the task type for the oncall estimator.
	TaskOncallEstimator TaskType = "oncall-estimator"
)

// Category 4: "I tried it safely"
const (
	// TaskMigrationRehearsal is the task type for the migration rehearsal runner.
	TaskMigrationRehearsal TaskType = "migration-rehearsal"
	// TaskContractFuzzer is the task type for the integration contract fuzzer.
	TaskContractFuzzer TaskType = "contract-fuzzer"
	// TaskGoldenPath is the task type for the golden-path recorder.
	TaskGoldenPath TaskType = "golden-path"
	// TaskPerfProfile is the task type for performance profiling runs.
	TaskPerfProfile TaskType = "perf-profile"
	// TaskAllocationProfile is the task type for allocation and hot-path profiling.
	TaskAllocationProfile TaskType = "allocation-profile"
)

// Category 5: "Here's the map"
const (
	// TaskVisibilityInstrument is the task type for the visibility instrumentor.
	TaskVisibilityInstrument TaskType = "visibility-instrument"
	// TaskRepoTopology is the task type for the repo topology visualizer.
	TaskRepoTopology TaskType = "repo-topology"
	// TaskPermissionsMapper is the task type for the permissions and auth surface mapper.
	TaskPermissionsMapper TaskType = "permissions-mapper"
	// TaskDataLifecycle is the task type for the data lifecycle tracer.
	TaskDataLifecycle TaskType = "data-lifecycle"
	// TaskFeatureFlagMonitor is the task type for the feature flag lifecycle monitor.
	TaskFeatureFlagMonitor TaskType = "feature-flag-monitor"
	// TaskCISignalNoise is the task type for the CI signal-to-noise scorer.
	TaskCISignalNoise TaskType = "ci-signal-noise"
	// TaskHistoricalContext is the task type for the historical context summarizer.
	TaskHistoricalContext TaskType = "historical-context"
)

// Category 6: "For when things go sideways"
const (
	// TaskRunbookGen is the task type for the runbook generator.
	TaskRunbookGen TaskType = "runbook-gen"
	// TaskRollbackPlan is the task type for the rollback plan generator.
	TaskRollbackPlan TaskType = "rollback-plan"
	// TaskPostmortemGen is the task type for the incident postmortem draft generator.
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
		Type:     TaskPIIScanner,
		Category: CategoryAnalysis,
		Name:     "PII Exposure Scanner",
		Description: `Scan the repository for potential PII (Personally Identifiable Information) exposure. ` +
			`Search all source files, configs, scripts, and data files for these categories:` +
			"\n\n" +
			`1. HARDCODED PII PATTERNS — Look for literals matching: email addresses, phone numbers, ` +
			`SSNs (NNN-NN-NNNN), credit card numbers (13-19 digits), IP addresses used as identifiers, ` +
			`street addresses, dates of birth, passport/driver-license numbers, and full person names ` +
			`in structured data (JSON, YAML, CSV, SQL seeds, fixtures, test data).` +
			"\n\n" +
			`2. PII IN LOGS & ERROR MESSAGES — Find log/print/error statements that interpolate variables ` +
			`likely holding PII (user email, name, phone, address, SSN, token). Flag fmt.Sprintf, ` +
			`log.Printf, slog, zerolog, zap, logrus, or equivalent calls where PII fields appear ` +
			`unredacted. Check error-wrapping chains (fmt.Errorf, errors.Wrap) for embedded PII.` +
			"\n\n" +
			`3. ENV & SECRET FILES — Check .env, .env.*, config.yaml, config.json, docker-compose ` +
			`environment blocks, and CI workflow files for plaintext secrets, API keys, tokens, ` +
			`or credentials that could expose user data.` +
			"\n\n" +
			`4. UNENCRYPTED STORAGE — Identify database columns, struct fields, or file writes that ` +
			`store PII without encryption or hashing (e.g., plaintext password fields, raw SSN columns, ` +
			`unmasked credit card storage).` +
			"\n\n" +
			`5. GITIGNORE GAPS — Verify .gitignore covers .env*, *.pem, *.key, credentials.*, ` +
			`secrets.*, and common data-dump extensions (.sql, .csv, .xlsx containing user data). ` +
			`Flag tracked files that should be ignored.` +
			"\n\n" +
			`OUTPUT FORMAT — For each finding, report:` +
			"\n" +
			`- file: path relative to repo root` +
			"\n" +
			`- line: line number(s)` +
			"\n" +
			`- category: one of [hardcoded-pii, pii-in-logs, env-secret, unencrypted-storage, gitignore-gap]` +
			"\n" +
			`- severity: critical / high / medium / low` +
			"\n" +
			`- detail: what was found and why it's a risk` +
			"\n" +
			`- recommendation: specific fix (redact, hash, encrypt, add to .gitignore, use env var, etc.)` +
			"\n\n" +
			`Exclude vendored/third-party code. Treat test fixtures with realistic-looking PII as medium ` +
			`severity (prefer obviously fake data). Summarize total findings by category and severity at the end.`,
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
