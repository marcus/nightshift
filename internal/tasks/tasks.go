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
		Type:     TaskLintFix,
		Category: CategoryPR,
		Name:     "Linter Fixes",
		Description: `Run all configured linters for the project and automatically fix reported issues. ` +
			`Detect the project language and build system (go vet, golangci-lint, eslint, ruff, etc.) ` +
			`from config files and Makefile/taskfile targets.` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Identify linter configs (.golangci.yml, .eslintrc, pyproject.toml, etc.) and run linters ` +
			`with auto-fix flags where available (--fix, -w).` +
			"\n" +
			`2. For issues that cannot be auto-fixed, apply the fix manually: formatting, import ordering, ` +
			`unused variables, unreachable code, naming conventions.` +
			"\n" +
			`3. Run the linter again after fixes to confirm zero remaining issues.` +
			"\n" +
			`4. Run tests to verify fixes did not change behavior.` +
			"\n\n" +
			`SCOPE: Only fix genuine lint violations. Do not refactor, restructure, or ` +
			`add features. Do not modify linter configuration. If a lint rule is contentious, ` +
			`skip it and note it in the PR description.` +
			"\n\n" +
			`OUTPUT: A PR with one commit per logical group of fixes. PR description lists ` +
			`linters run, issue counts before/after, and any skipped rules.`,
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 24 * time.Hour,
	},
	TaskBugFinder: {
		Type:     TaskBugFinder,
		Category: CategoryPR,
		Name:     "Bug Finder & Fixer",
		Description: `Systematically scan the codebase for latent bugs and produce fixes with tests. ` +
			`Focus on bugs that are provably wrong, not style preferences.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- Nil/null pointer dereferences and missing nil checks` +
			"\n" +
			`- Off-by-one errors in loops, slices, and range bounds` +
			"\n" +
			`- Resource leaks (unclosed files, HTTP bodies, database connections)` +
			"\n" +
			`- Error returns that are silently discarded` +
			"\n" +
			`- Race conditions (shared state without synchronization)` +
			"\n" +
			`- Integer overflow/underflow in arithmetic` +
			"\n" +
			`- Incorrect string/byte conversions, encoding issues` +
			"\n" +
			`- Logic errors in conditionals (inverted checks, missing cases in switches)` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Read each package/module, focusing on error paths, boundary conditions, and concurrency.` +
			"\n" +
			`2. For each bug found, write a failing test that demonstrates the bug.` +
			"\n" +
			`3. Apply the minimal fix, then verify the test passes.` +
			"\n" +
			`4. Run the full test suite to confirm no regressions.` +
			"\n\n" +
			`SCOPE: Fix only clear bugs—not code smells, style issues, or potential improvements. ` +
			`Each fix should be small and obvious. If a bug requires a design change, report it ` +
			`but do not fix it.` +
			"\n\n" +
			`OUTPUT: PR with one commit per bug. Each commit message explains the bug, how it ` +
			`manifests, and the fix. PR description has a summary table of bugs found.`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 72 * time.Hour,
	},
	TaskAutoDRY: {
		Type:     TaskAutoDRY,
		Category: CategoryPR,
		Name:     "Auto DRY Refactoring",
		Description: `Identify duplicated or near-duplicate code blocks and refactor them into shared ` +
			`abstractions. Only extract when duplication is real and the abstraction is clear.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- Functions or methods with identical/near-identical bodies across packages` +
			"\n" +
			`- Repeated error-handling boilerplate that can be consolidated` +
			"\n" +
			`- Copy-pasted struct transformations or mapping logic` +
			"\n" +
			`- Duplicated validation logic, string formatting, or config parsing patterns` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Search for structurally similar code blocks (3+ lines, appearing 3+ times).` +
			"\n" +
			`2. Evaluate whether an extraction improves clarity—not all duplication is bad. ` +
			`Two similar blocks may diverge intentionally.` +
			"\n" +
			`3. Extract to a well-named helper function in the most appropriate package.` +
			"\n" +
			`4. Update all call sites and verify tests pass.` +
			"\n\n" +
			`SCOPE: Do not create abstractions for code that is only duplicated twice. ` +
			`Do not introduce interfaces or generics unless clearly warranted. ` +
			`Prefer flat helpers over deep abstraction hierarchies.` +
			"\n\n" +
			`OUTPUT: PR with each extraction as a separate commit. PR description lists ` +
			`each duplication cluster: where it was found, the new shared location, ` +
			`and how many call sites were consolidated.`,
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
		Type:     TaskAPIContractVerify,
		Category: CategoryPR,
		Name:     "API Contract Verification",
		Description: `Verify that API contracts (OpenAPI specs, protobuf definitions, GraphQL schemas, ` +
			`JSON Schema, or CLI flag definitions) match the actual implementation.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- OpenAPI/Swagger YAML/JSON vs. handler code: routes, parameters, request/response bodies, status codes` +
			"\n" +
			`- Protobuf .proto files vs. generated code and server implementations` +
			"\n" +
			`- CLI cobra/flag definitions vs. documentation and help text` +
			"\n" +
			`- Exported Go types used in JSON marshaling vs. documented schemas` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Locate contract definition files and their corresponding implementations.` +
			"\n" +
			`2. Compare field names, types, required/optional status, enum values, and defaults.` +
			"\n" +
			`3. Check that error responses match documented error schemas.` +
			"\n" +
			`4. Fix the implementation or spec (prefer fixing spec when implementation is tested and correct).` +
			"\n" +
			`5. Run tests to verify consistency.` +
			"\n\n" +
			`OUTPUT: PR with contract fixes. PR description includes a table of mismatches found: ` +
			`field, expected (from contract), actual (from code), resolution.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskBackwardCompat: {
		Type:     TaskBackwardCompat,
		Category: CategoryPR,
		Name:     "Backward-Compatibility Checks",
		Description: `Analyze recent changes for backward-compatibility breaks and add protective measures. ` +
			`Focus on the public surface area: exported APIs, CLI flags, config file formats, ` +
			`serialized data formats, and database schemas.` +
			"\n\n" +
			`WHAT TO CHECK:` +
			"\n" +
			`- Removed or renamed exported functions, types, constants, or struct fields` +
			"\n" +
			`- Changed function signatures (added required params, changed return types)` +
			"\n" +
			`- Removed or renamed CLI flags/subcommands` +
			"\n" +
			`- Config file key renames or structural changes without migration` +
			"\n" +
			`- Changed serialization formats (JSON/YAML field names, wire protocol changes)` +
			"\n" +
			`- Database migrations that drop columns or change types without backfill` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Compare current HEAD against the last release tag or main branch.` +
			"\n" +
			`2. For each breaking change, determine if it's intentional or accidental.` +
			"\n" +
			`3. For accidental breaks, restore compatibility. For intentional breaks, ` +
			`add deprecation aliases, migration helpers, or version guards.` +
			"\n" +
			`4. Run tests to verify both old and new usage paths work.` +
			"\n\n" +
			`OUTPUT: PR with compatibility fixes. PR description lists each break, ` +
			`its impact, and the remediation applied.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskBuildOptimize: {
		Type:     TaskBuildOptimize,
		Category: CategoryPR,
		Name:     "Build Time Optimization",
		Description: `Analyze and optimize build configuration to reduce build times and artifact sizes.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Go: build flags, CGO usage, ldflags, module graph depth, unnecessary dependencies` +
			"\n" +
			`- Docker: layer ordering, multi-stage builds, cache invalidation, base image size` +
			"\n" +
			`- CI: parallelism opportunities, cached steps, unnecessary rebuilds` +
			"\n" +
			`- Makefile/Taskfile: dependency graph, phony targets, parallel execution` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Profile the build: run with timing flags and identify the slowest stages.` +
			"\n" +
			`2. Check for unnecessary dependencies inflating compile time (go mod graph analysis).` +
			"\n" +
			`3. Verify Docker layers are ordered from least to most frequently changing.` +
			"\n" +
			`4. Look for CI steps that can be parallelized or cached.` +
			"\n" +
			`5. Apply optimizations and measure improvement.` +
			"\n\n" +
			`SCOPE: Do not change application behavior. Only modify build tooling, ` +
			`CI config, and dependency declarations. Measure before and after.` +
			"\n\n" +
			`OUTPUT: PR with build optimizations. PR description includes before/after ` +
			`timing comparisons and binary size changes where applicable.`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 168 * time.Hour,
	},
	TaskDocsBackfill: {
		Type:     TaskDocsBackfill,
		Category: CategoryPR,
		Name:     "Documentation Backfiller",
		Description: `Find exported symbols, packages, and CLI commands that lack documentation ` +
			`and add clear, concise doc comments.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- Exported Go functions, types, methods, and constants without doc comments` +
			"\n" +
			`- Packages without package-level doc comments` +
			"\n" +
			`- CLI commands/flags without help text or descriptions` +
			"\n" +
			`- README sections that reference features but lack usage examples` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Use go vet or static analysis to find undocumented exports.` +
			"\n" +
			`2. Read the implementation to understand what each symbol does.` +
			"\n" +
			`3. Write doc comments following Go conventions: start with the symbol name, ` +
			`describe what it does (not how), note any important caveats.` +
			"\n" +
			`4. Keep comments brief—one to three sentences for most symbols.` +
			"\n\n" +
			`SCOPE: Only add missing docs. Do not rewrite or expand existing doc comments. ` +
			`Do not add comments to unexported symbols. Do not generate README files.` +
			"\n\n" +
			`OUTPUT: PR with documentation additions. PR description tallies symbols ` +
			`documented by package.`,
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskCommitNormalize: {
		Type:     TaskCommitNormalize,
		Category: CategoryPR,
		Name:     "Commit Message Normalizer",
		Description: `Analyze recent commit messages and set up or enforce a consistent commit ` +
			`message format based on the project's established conventions.` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Read the last 50 commit messages to identify the dominant convention ` +
			`(Conventional Commits, gitmoji, imperative tense, ticket prefixes, etc.).` +
			"\n" +
			`2. Check for existing commit-msg hooks, commitlint config, or .gitmessage templates.` +
			"\n" +
			`3. If conventions exist but are inconsistently applied, add or update tooling ` +
			`to enforce them (commit-msg hook, commitlint config).` +
			"\n" +
			`4. If no convention exists, propose one based on the most common pattern ` +
			`and add lightweight enforcement.` +
			"\n\n" +
			`SCOPE: Do not rewrite git history. Only add or update enforcement tooling ` +
			`and documentation of the convention. Do not change existing commits.` +
			"\n\n" +
			`OUTPUT: PR with hook/config changes. PR description explains the detected ` +
			`convention, any deviations found, and the enforcement mechanism added.`,
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 24 * time.Hour,
	},
	TaskChangelogSynth: {
		Type:     TaskChangelogSynth,
		Category: CategoryPR,
		Name:     "Changelog Synthesizer",
		Description: `Generate or update a CHANGELOG.md from git history since the last release tag.` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Find the most recent release tag (vX.Y.Z or similar).` +
			"\n" +
			`2. Collect all commits since that tag, grouped by type ` +
			`(features, fixes, breaking changes, other).` +
			"\n" +
			`3. Filter out merge commits, CI-only changes, and trivial chores.` +
			"\n" +
			`4. Write human-readable entries: describe the user-facing impact, ` +
			`not the implementation detail. Link to PRs/issues where available.` +
			"\n" +
			`5. If CHANGELOG.md exists, prepend the new section. If not, create it ` +
			`following Keep a Changelog format.` +
			"\n\n" +
			`SCOPE: Only synthesize from merged commits. Do not fabricate entries. ` +
			`Use commit messages and PR titles as source material.` +
			"\n\n" +
			`OUTPUT: PR with updated CHANGELOG.md. PR description shows the ` +
			`version range covered and entry count by category.`,
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskReleaseNotes: {
		Type:     TaskReleaseNotes,
		Category: CategoryPR,
		Name:     "Release Note Drafter",
		Description: `Draft user-facing release notes for the next version, suitable for ` +
			`GitHub Releases, blog posts, or announcement channels.` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Determine the version being released (from version files, tags, or config).` +
			"\n" +
			`2. Gather changes since the last release: commits, merged PRs, closed issues.` +
			"\n" +
			`3. Group into sections: Highlights, New Features, Improvements, Bug Fixes, ` +
			`Breaking Changes, Deprecations.` +
			"\n" +
			`4. Write each entry from the user's perspective: what changed for them, ` +
			`not internal implementation details.` +
			"\n" +
			`5. Include upgrade instructions for any breaking changes.` +
			"\n" +
			`6. Add contributor acknowledgments if applicable.` +
			"\n\n" +
			`SCOPE: Focus on user-facing changes. Omit internal refactors, CI changes, ` +
			`and dependency bumps unless they affect behavior. Keep tone concise and professional.` +
			"\n\n" +
			`OUTPUT: PR adding a release notes draft file. PR description includes ` +
			`the target version and a summary of what's covered.`,
		CostTier:        CostLow,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskADRDraft: {
		Type:     TaskADRDraft,
		Category: CategoryPR,
		Name:     "ADR Drafter",
		Description: `Identify recent architectural decisions in the codebase that lack ` +
			`Architecture Decision Records and draft ADRs for them.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- New package/module boundaries introduced in recent commits` +
			"\n" +
			`- Technology choices (new dependencies, framework switches, database changes)` +
			"\n" +
			`- Significant pattern changes (new error handling strategy, config approach)` +
			"\n" +
			`- Existing docs/adr/ or similar directory indicating ADR convention` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Review commits from the last 30 days for architectural changes.` +
			"\n" +
			`2. Check if ADRs already exist for these decisions.` +
			"\n" +
			`3. For undocumented decisions, draft ADRs using the project's existing template ` +
			`or Michael Nygard's format: Title, Status, Context, Decision, Consequences.` +
			"\n" +
			`4. Infer context and rationale from commit messages, PR descriptions, ` +
			`and code comments. Mark uncertain rationale with [VERIFY].` +
			"\n\n" +
			`SCOPE: Draft ADRs as proposals. Use status "Proposed" so a human ` +
			`can review and accept. Do not change code.` +
			"\n\n" +
			`OUTPUT: PR with new ADR files. PR description lists each decision ` +
			`documented and its inferred rationale.`,
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
		Type:     TaskDocDrift,
		Category: CategoryAnalysis,
		Name:     "Doc Drift Detector",
		Description: `Detect documentation that has drifted out of sync with the actual code. ` +
			`Compare documented behavior, APIs, flags, and examples against the current implementation.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- README.md: CLI usage examples, installation instructions, config examples` +
			"\n" +
			`- Code comments referencing functions, files, or behavior that no longer exists` +
			"\n" +
			`- Godoc/JSDoc/docstring parameter lists vs actual function signatures` +
			"\n" +
			`- Example code blocks that would fail if executed` +
			"\n" +
			`- TODO/FIXME comments referencing completed or abandoned work` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Inventory all documentation files and significant code comments.` +
			"\n" +
			`2. For each documented claim, verify it against the code: does the flag exist? ` +
			`Does the function accept those parameters? Does the example compile/run?` +
			"\n" +
			`3. Classify drift by severity: critical (misleading, causes errors), ` +
			`medium (outdated but not harmful), low (cosmetic or stale wording).` +
			"\n\n" +
			`OUTPUT FORMAT — For each finding:` +
			"\n" +
			`- file: path and line range` +
			"\n" +
			`- drift_type: stale-reference | wrong-signature | dead-example | outdated-instruction` +
			"\n" +
			`- severity: critical / medium / low` +
			"\n" +
			`- detail: what the doc says vs what the code does` +
			"\n" +
			`- suggestion: corrected text or "remove"`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskSemanticDiff: {
		Type:     TaskSemanticDiff,
		Category: CategoryAnalysis,
		Name:     "Semantic Diff Explainer",
		Description: `Analyze recent code changes and explain their semantic meaning—what behavioral ` +
			`differences they introduce, beyond the syntactic diff.` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Get the diff between HEAD and the last release tag (or last N commits if no tag).` +
			"\n" +
			`2. For each changed function/method, explain: what it did before, what it does now, ` +
			`and what user-visible behavior changed.` +
			"\n" +
			`3. Identify side effects: does a change affect error messages users see? ` +
			`Does it change timing, ordering, or default values?` +
			"\n" +
			`4. Flag changes that look like refactors but subtly alter behavior ` +
			`(e.g., changing iteration order, nil vs empty slice returns).` +
			"\n\n" +
			`OUTPUT FORMAT — For each semantic change:` +
			"\n" +
			`- location: file:function` +
			"\n" +
			`- change_type: behavior-change | new-behavior | removed-behavior | refactor-only` +
			"\n" +
			`- before: description of old behavior` +
			"\n" +
			`- after: description of new behavior` +
			"\n" +
			`- impact: who/what is affected (users, callers, downstream systems)` +
			"\n" +
			`- risk: low / medium / high` +
			"\n\n" +
			`Summarize with a count of behavioral changes by risk level.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskDeadCode: {
		Type:     TaskDeadCode,
		Category: CategoryAnalysis,
		Name:     "Dead Code Detector",
		Description: `Find code that is unreachable, unused, or effectively dead and report ` +
			`it for removal.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- Exported functions/types/constants with zero callers outside their own package` +
			"\n" +
			`- Unexported functions with zero callers` +
			"\n" +
			`- Entire files that are never imported or referenced` +
			"\n" +
			`- Struct fields that are set but never read (or vice versa)` +
			"\n" +
			`- Switch/if branches that can never be reached given type constraints` +
			"\n" +
			`- Build-tagged files for platforms the project doesn't target` +
			"\n" +
			`- Commented-out code blocks` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Use static analysis and grep to find unused symbols.` +
			"\n" +
			`2. Check for dynamic usage (reflection, string-based lookup, plugin systems) ` +
			`before flagging—false positives are worse than misses.` +
			"\n" +
			`3. Verify via tests that removing the code doesn't break anything.` +
			"\n\n" +
			`OUTPUT FORMAT — For each finding:` +
			"\n" +
			`- file: path and line range` +
			"\n" +
			`- symbol: name of the dead code` +
			"\n" +
			`- type: unused-export | unused-unexported | dead-file | unreachable-branch | commented-out` +
			"\n" +
			`- confidence: high / medium (based on whether dynamic usage was ruled out)` +
			"\n" +
			`- lines: count of removable lines` +
			"\n\n" +
			`Summarize total removable lines and confidence distribution.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskDependencyRisk: {
		Type:     TaskDependencyRisk,
		Category: CategoryAnalysis,
		Name:     "Dependency Risk Scanner",
		Description: `Analyze project dependencies for security vulnerabilities, maintenance risks, ` +
			`and license concerns.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- go.mod/go.sum, package.json/package-lock.json, requirements.txt, Cargo.toml` +
			"\n" +
			`- Known CVEs via govulncheck, npm audit, or equivalent` +
			"\n" +
			`- Maintenance signals: last commit date, open issue count, bus factor of maintainers` +
			"\n" +
			`- License compatibility: identify copyleft, restrictive, or unclear licenses` +
			"\n" +
			`- Dependency depth: transitive dependencies pulling in large or risky subtrees` +
			"\n" +
			`- Pinning: unpinned or floating version references` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Run available vulnerability scanners (govulncheck, npm audit).` +
			"\n" +
			`2. For each direct dependency, check last release date and maintenance activity.` +
			"\n" +
			`3. Flag dependencies that are unmaintained (>12 months inactive), ` +
			`have known vulnerabilities, or have license concerns.` +
			"\n" +
			`4. Suggest alternatives for high-risk dependencies.` +
			"\n\n" +
			`OUTPUT FORMAT — For each risky dependency:` +
			"\n" +
			`- package: name@version` +
			"\n" +
			`- risk_type: vulnerability | unmaintained | license | deep-transitive` +
			"\n" +
			`- severity: critical / high / medium / low` +
			"\n" +
			`- detail: specific CVE, maintenance gap, or license issue` +
			"\n" +
			`- recommendation: upgrade, replace, vendor, or accept-risk with rationale`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskTestGap: {
		Type:     TaskTestGap,
		Category: CategoryAnalysis,
		Name:     "Test Gap Finder",
		Description: `Identify code paths, functions, and scenarios that lack test coverage.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Packages/files with no corresponding _test.go or test files` +
			"\n" +
			`- Exported functions with zero test callers` +
			"\n" +
			`- Error paths and edge cases in existing tested functions that are not exercised` +
			"\n" +
			`- Recently changed code (last 30 days) without corresponding test changes` +
			"\n" +
			`- Critical paths (config parsing, auth, data mutation) with shallow coverage` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Run coverage tools (go test -coverprofile) and identify uncovered lines.` +
			"\n" +
			`2. Cross-reference with code complexity: high-complexity uncovered code is highest priority.` +
			"\n" +
			`3. Check for test files that exist but only test the happy path.` +
			"\n" +
			`4. Prioritize gaps by risk: data corruption paths > error handling > edge cases > cosmetic.` +
			"\n\n" +
			`OUTPUT FORMAT — For each gap:` +
			"\n" +
			`- file: path and function/method name` +
			"\n" +
			`- gap_type: untested-function | untested-error-path | untested-edge-case | no-test-file` +
			"\n" +
			`- priority: critical / high / medium / low` +
			"\n" +
			`- suggestion: brief description of what test(s) to write` +
			"\n\n" +
			`Summarize with coverage percentage by package and a top-10 priority list.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskTestFlakiness: {
		Type:     TaskTestFlakiness,
		Category: CategoryAnalysis,
		Name:     "Test Flakiness Analyzer",
		Description: `Identify tests that are likely flaky and diagnose the root cause of their instability.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- Tests using time.Sleep, fixed ports, or real network calls` +
			"\n" +
			`- Tests depending on map iteration order or goroutine scheduling` +
			"\n" +
			`- Tests reading/writing shared filesystem state without cleanup` +
			"\n" +
			`- Tests with race conditions (shared global state, t.Parallel without isolation)` +
			"\n" +
			`- Tests comparing floating point values without epsilon` +
			"\n" +
			`- Tests that pass individually but fail under -count=10 or -race` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Scan test files for known flakiness patterns (sleep, os.Getenv in tests, ` +
			`shared temp dirs, hardcoded ports).` +
			"\n" +
			`2. Run tests with -count=5 -race to surface intermittent failures.` +
			"\n" +
			`3. For each flaky test found, identify the root cause and categorize it.` +
			"\n\n" +
			`OUTPUT FORMAT — For each flaky test:` +
			"\n" +
			`- test: TestName in package/path` +
			"\n" +
			`- pattern: timing-dependent | order-dependent | shared-state | race-condition | external-dep` +
			"\n" +
			`- severity: high (fails often) / medium (fails occasionally) / low (theoretical risk)` +
			"\n" +
			`- root_cause: specific explanation` +
			"\n" +
			`- fix: concrete remediation steps`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskLoggingAudit: {
		Type:     TaskLoggingAudit,
		Category: CategoryAnalysis,
		Name:     "Logging Quality Auditor",
		Description: `Audit logging statements for completeness, consistency, and operational usefulness.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Log levels: are errors logged at error level, not info? Are debug logs gated?` +
			"\n" +
			`- Structured fields: are logs using structured logging (slog, zerolog, zap) ` +
			`consistently, or mixing fmt.Printf with structured loggers?` +
			"\n" +
			`- Context propagation: do logs include request IDs, trace IDs, or correlation keys?` +
			"\n" +
			`- Error paths: are all error returns logged at least once before being discarded?` +
			"\n" +
			`- Sensitive data: are passwords, tokens, or PII being logged? (cross-ref with PII scanner)` +
			"\n" +
			`- Noise: are there high-frequency debug/info logs that would drown out signals in production?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Inventory all logging calls and their log levels.` +
			"\n" +
			`2. Map error-return paths and verify each has a log statement or is explicitly propagated.` +
			"\n" +
			`3. Check for log level misuse (e.g., log.Error for non-errors, log.Info in hot loops).` +
			"\n" +
			`4. Verify structured fields use consistent key names across packages.` +
			"\n\n" +
			`OUTPUT FORMAT — For each finding:` +
			"\n" +
			`- file: path and line` +
			"\n" +
			`- issue: wrong-level | missing-log | sensitive-data | unstructured | noisy | inconsistent-keys` +
			"\n" +
			`- severity: high / medium / low` +
			"\n" +
			`- detail: what's wrong and why it matters operationally` +
			"\n" +
			`- recommendation: specific fix`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskMetricsCoverage: {
		Type:     TaskMetricsCoverage,
		Category: CategoryAnalysis,
		Name:     "Metrics Coverage Analyzer",
		Description: `Analyze whether key operations are instrumented with metrics and identify gaps ` +
			`in observability coverage.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- HTTP/gRPC handlers: request count, latency, error rate per endpoint` +
			"\n" +
			`- Database operations: query duration, connection pool utilization, error rates` +
			"\n" +
			`- External API calls: latency, retry counts, failure rates` +
			"\n" +
			`- Queue/worker operations: processing time, queue depth, dead letters` +
			"\n" +
			`- Business-critical operations: what domain events are measured?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Find the metrics library in use (prometheus, statsd, otel, etc.).` +
			"\n" +
			`2. Map all metric registrations to their corresponding operations.` +
			"\n" +
			`3. Walk key code paths (handlers, workers, cron jobs) and check for instrumentation.` +
			"\n" +
			`4. Flag operations with no metrics and high-traffic paths with insufficient dimensionality.` +
			"\n\n" +
			`OUTPUT FORMAT — For each gap:` +
			"\n" +
			`- operation: description of the uninstrumented operation` +
			"\n" +
			`- location: file:function` +
			"\n" +
			`- priority: critical / high / medium / low` +
			"\n" +
			`- suggested_metrics: metric name, type (counter/histogram/gauge), and labels` +
			"\n\n" +
			`Summarize with a coverage ratio: instrumented operations / total operations.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskPerfRegression: {
		Type:     TaskPerfRegression,
		Category: CategoryAnalysis,
		Name:     "Performance Regression Spotter",
		Description: `Review recent code changes for patterns that commonly cause performance regressions.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- N+1 query patterns introduced in loops` +
			"\n" +
			`- Unbounded allocations (growing slices/maps without capacity hints)` +
			"\n" +
			`- String concatenation in loops instead of strings.Builder` +
			"\n" +
			`- Synchronous I/O in hot paths that could be batched or async` +
			"\n" +
			`- Regex compilation inside loops instead of pre-compiled` +
			"\n" +
			`- Lock contention from overly broad mutex scopes` +
			"\n" +
			`- Missing context cancellation propagation (long-running ops without ctx)` +
			"\n" +
			`- JSON marshal/unmarshal in hot paths instead of streaming` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Get the diff since the last release or last 30 days.` +
			"\n" +
			`2. For each changed function, analyze algorithmic complexity and allocation patterns.` +
			"\n" +
			`3. Check for O(n²) patterns, missing capacity pre-allocation, and unnecessary copies.` +
			"\n" +
			`4. Flag only likely regressions, not theoretical concerns.` +
			"\n\n" +
			`OUTPUT FORMAT — For each potential regression:` +
			"\n" +
			`- file: path:line` +
			"\n" +
			`- pattern: n-plus-1 | unbounded-alloc | hot-path-io | lock-contention | complexity-increase` +
			"\n" +
			`- severity: high / medium / low (based on expected traffic and data volume)` +
			"\n" +
			`- detail: what the regression is and under what conditions it manifests` +
			"\n" +
			`- recommendation: specific optimization`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskCostAttribution: {
		Type:     TaskCostAttribution,
		Category: CategoryAnalysis,
		Name:     "Cost Attribution Estimator",
		Description: `Estimate compute, storage, and API costs by component to identify the most ` +
			`expensive parts of the system.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- External API calls: identify rate-limited or pay-per-call APIs (LLM providers, ` +
			`cloud services, SaaS APIs) and estimate call volume from code paths` +
			"\n" +
			`- Database queries: identify expensive query patterns (full table scans, ` +
			`large joins, frequent writes)` +
			"\n" +
			`- Storage: data written to disk, blob storage, or caches—what grows unbounded?` +
			"\n" +
			`- Compute: CPU-intensive operations, goroutine/thread spawning patterns` +
			"\n" +
			`- Network: large payload transfers, frequent polling, chatty protocols` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Map all external service integrations and their pricing models.` +
			"\n" +
			`2. Trace high-frequency code paths and estimate per-request cost.` +
			"\n" +
			`3. Identify cost multipliers (retries, fan-out, polling intervals).` +
			"\n" +
			`4. Rank components by estimated cost contribution.` +
			"\n\n" +
			`OUTPUT FORMAT — For each cost center:` +
			"\n" +
			`- component: module/package name` +
			"\n" +
			`- cost_driver: api-calls | storage | compute | network` +
			"\n" +
			`- estimated_impact: high / medium / low (relative ranking)` +
			"\n" +
			`- detail: what drives the cost and at what scale` +
			"\n" +
			`- optimization: specific ways to reduce cost`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskSecurityFootgun: {
		Type:     TaskSecurityFootgun,
		Category: CategoryAnalysis,
		Name:     "Security Foot-Gun Finder",
		Description: `Scan the codebase for common security anti-patterns that could lead to ` +
			`vulnerabilities. Focus on patterns that are easy to introduce accidentally.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- Command injection: os/exec with unsanitized user input, shell=true patterns` +
			"\n" +
			`- Path traversal: filepath.Join with user-controlled segments without validation` +
			"\n" +
			`- SQL injection: string concatenation in SQL queries instead of parameterized queries` +
			"\n" +
			`- XSS: unescaped user input in HTML templates` +
			"\n" +
			`- Insecure crypto: hardcoded keys, weak algorithms (MD5/SHA1 for security), ` +
			`predictable random (math/rand for security-sensitive operations)` +
			"\n" +
			`- SSRF: HTTP requests to user-controlled URLs without allowlisting` +
			"\n" +
			`- Timing attacks: non-constant-time comparison of secrets/tokens` +
			"\n" +
			`- Insecure defaults: TLS verification disabled, permissive CORS, debug mode in prod config` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Grep for known dangerous function calls (exec.Command, sql.Query with + or Sprintf).` +
			"\n" +
			`2. Trace data flow from external inputs to dangerous sinks.` +
			"\n" +
			`3. Check crypto usage against current best practices.` +
			"\n" +
			`4. Review HTTP client configurations for security settings.` +
			"\n\n" +
			`OUTPUT FORMAT — For each finding:` +
			"\n" +
			`- file: path:line` +
			"\n" +
			`- pattern: command-injection | path-traversal | sql-injection | xss | weak-crypto | ssrf | timing-attack | insecure-default` +
			"\n" +
			`- severity: critical / high / medium / low` +
			"\n" +
			`- detail: the specific anti-pattern and how it could be exploited` +
			"\n" +
			`- fix: concrete remediation (parameterize, sanitize, use constant-time compare, etc.)`,
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
		Type:     TaskPrivacyPolicy,
		Category: CategoryAnalysis,
		Name:     "Privacy Policy Consistency Checker",
		Description: `Compare the project's privacy policy or data-handling documentation against ` +
			`actual code behavior to find inconsistencies.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Privacy policy claims about what data is collected vs. what the code actually collects` +
			"\n" +
			`- Stated data retention periods vs. actual deletion/TTL logic in code` +
			"\n" +
			`- Claimed data sharing practices vs. actual third-party API integrations` +
			"\n" +
			`- Opt-out mechanisms described in policy vs. implemented in code` +
			"\n" +
			`- Cookie/tracking disclosures vs. actual tracking code` +
			"\n" +
			`- GDPR/CCPA compliance claims vs. data export/deletion endpoints` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Locate privacy policy, terms of service, or data-handling docs.` +
			"\n" +
			`2. Extract each concrete claim (we collect X, we share with Y, we retain for Z days).` +
			"\n" +
			`3. Search the codebase for the corresponding implementation.` +
			"\n" +
			`4. Flag mismatches: undisclosed collection, missing deletion logic, ` +
			`undocumented third-party sharing.` +
			"\n\n" +
			`OUTPUT FORMAT — For each inconsistency:` +
			"\n" +
			`- claim: what the policy says` +
			"\n" +
			`- reality: what the code does` +
			"\n" +
			`- severity: critical (legal risk) / high / medium / low` +
			"\n" +
			`- recommendation: update policy, update code, or add missing implementation`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskSchemaEvolution: {
		Type:     TaskSchemaEvolution,
		Category: CategoryAnalysis,
		Name:     "Schema Evolution Advisor",
		Description: `Analyze database schema changes (migrations, model definitions) for safety, ` +
			`rollback-ability, and compatibility.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Migration files: SQL migrations, ORM migration definitions, schema change scripts` +
			"\n" +
			`- Column additions: are they nullable or have defaults? (non-nullable without default breaks deploys)` +
			"\n" +
			`- Column removals: is the column still referenced in code? Is there a backfill/migration?` +
			"\n" +
			`- Type changes: can existing data be safely cast? Is there data loss risk?` +
			"\n" +
			`- Index changes: will new indexes lock tables? Are removed indexes still needed for queries?` +
			"\n" +
			`- Foreign key changes: cascading deletes, orphaned references` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Find migration files and schema definitions.` +
			"\n" +
			`2. For each migration, evaluate: can it be rolled back? Does it require downtime? ` +
			`Is it compatible with the previous version of the application running concurrently?` +
			"\n" +
			`3. Check that code changes and schema changes are consistent (new columns are used, ` +
			`removed columns are no longer referenced).` +
			"\n\n" +
			`OUTPUT FORMAT — For each schema change:` +
			"\n" +
			`- migration: file name or identifier` +
			"\n" +
			`- operation: add-column | drop-column | change-type | add-index | add-constraint` +
			"\n" +
			`- risk: critical / high / medium / low` +
			"\n" +
			`- rollback_safe: yes / no / partial` +
			"\n" +
			`- detail: specific risks (data loss, locking, backward incompatibility)` +
			"\n" +
			`- recommendation: safe migration strategy`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskEventTaxonomy: {
		Type:     TaskEventTaxonomy,
		Category: CategoryAnalysis,
		Name:     "Event Taxonomy Normalizer",
		Description: `Audit event names, analytics tracking calls, and structured event emissions ` +
			`for naming consistency and completeness.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Analytics events: naming convention (camelCase, snake_case, dot.separated)` +
			"\n" +
			`- Event property schemas: consistent property names across similar events` +
			"\n" +
			`- Coverage: are all user-facing actions tracked? Are there dead events (emitted but never consumed)?` +
			"\n" +
			`- Duplication: multiple events tracking the same action with different names` +
			"\n" +
			`- Structured event types: webhook payloads, domain events, message queue topics` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Grep for event emission calls (analytics.Track, emit, publish, send, etc.).` +
			"\n" +
			`2. Extract all event names and their property schemas.` +
			"\n" +
			`3. Check for naming convention violations and inconsistencies.` +
			"\n" +
			`4. Identify events with different property schemas for the same logical action.` +
			"\n\n" +
			`OUTPUT FORMAT — For each issue:` +
			"\n" +
			`- event: event name` +
			"\n" +
			`- issue: naming-violation | inconsistent-properties | duplicate | dead-event | missing-event` +
			"\n" +
			`- location: file:line where emitted` +
			"\n" +
			`- detail: what's wrong` +
			"\n" +
			`- recommendation: normalized name, property schema, or removal` +
			"\n\n" +
			`Include a complete event catalog table at the end: event name, emitter location, ` +
			`property count, consumer count.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskRoadmapEntropy: {
		Type:     TaskRoadmapEntropy,
		Category: CategoryAnalysis,
		Name:     "Roadmap Entropy Detector",
		Description: `Detect signs of scope creep, abandoned initiatives, and roadmap drift by ` +
			`analyzing code artifacts, TODOs, and incomplete features.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- TODO/FIXME/HACK comments older than 90 days` +
			"\n" +
			`- Feature flags that have been "temporary" for months` +
			"\n" +
			`- Half-implemented features: code with stub functions, empty handlers, ` +
			`or "not yet implemented" errors` +
			"\n" +
			`- Packages/files added but never integrated into the main flow` +
			"\n" +
			`- Multiple competing implementations of the same concept` +
			"\n" +
			`- Dead configuration options that no code path reads` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Collect all TODO/FIXME/HACK comments with their git blame dates.` +
			"\n" +
			`2. Find stub implementations (functions that only return errors or nil).` +
			"\n" +
			`3. Identify feature flags and check their age.` +
			"\n" +
			`4. Look for code that was added in a feature branch but never fully connected.` +
			"\n\n" +
			`OUTPUT FORMAT — For each finding:` +
			"\n" +
			`- type: stale-todo | abandoned-feature | competing-impl | dead-config | stuck-flag` +
			"\n" +
			`- location: file:line or package` +
			"\n" +
			`- age: how long it's been in this state` +
			"\n" +
			`- detail: what appears to be unfinished and why` +
			"\n" +
			`- recommendation: complete, remove, or document as intentionally deferred`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskBusFactor: {
		Type:     TaskBusFactor,
		Category: CategoryAnalysis,
		Name:     "Bus-Factor Analyzer",
		Description: `Analyze code ownership concentration to identify single points of failure ` +
			`in team knowledge.` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Run git log analysis to determine per-file and per-directory author distribution.` +
			"\n" +
			`2. Calculate bus factor per package: how many contributors would need to leave ` +
			`before no one understands the code? (Bus factor = number of contributors who ` +
			`authored 80%+ of the changes.)` +
			"\n" +
			`3. Weight by recency: recent-only contributors matter more than historical ones.` +
			"\n" +
			`4. Cross-reference with code complexity: high-complexity, single-author code is highest risk.` +
			"\n\n" +
			`OUTPUT FORMAT:` +
			"\n" +
			`- Top-level summary: overall repo bus factor score` +
			"\n" +
			`- Per-package breakdown:` +
			"\n" +
			`  - package: path` +
			"\n" +
			`  - bus_factor: number (1 = critical risk, 2 = caution, 3+ = healthy)` +
			"\n" +
			`  - primary_author: who wrote most of it` +
			"\n" +
			`  - last_other_contributor: who else touched it and when` +
			"\n" +
			`  - complexity: low / medium / high` +
			"\n" +
			`  - risk: critical / high / medium / low` +
			"\n\n" +
			`Sort by risk descending. Recommend knowledge-sharing actions for critical areas.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},
	TaskKnowledgeSilo: {
		Type:     TaskKnowledgeSilo,
		Category: CategoryAnalysis,
		Name:     "Knowledge Silo Detector",
		Description: `Identify areas of the codebase where knowledge is concentrated in a single ` +
			`person or small group, creating organizational risk.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Code areas where only one author has made changes in the last 6 months` +
			"\n" +
			`- Complex subsystems with no documentation or tests (hard to onboard into)` +
			"\n" +
			`- Custom protocols, encoding formats, or algorithms with single-author implementations` +
			"\n" +
			`- Deployment scripts, infrastructure code, or CI pipelines owned by one person` +
			"\n" +
			`- Third-party integrations where only one person understands the vendor's API` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Analyze git log to find single-contributor files and directories.` +
			"\n" +
			`2. Check for documentation: do siloed areas have README, comments, or ADRs?` +
			"\n" +
			`3. Assess onboarding difficulty: could a new contributor modify this code ` +
			`safely without the original author?` +
			"\n" +
			`4. Factor in code review history: has anyone else reviewed changes here?` +
			"\n\n" +
			`OUTPUT FORMAT — For each silo:` +
			"\n" +
			`- area: package, directory, or subsystem` +
			"\n" +
			`- owner: primary contributor` +
			"\n" +
			`- risk_level: critical / high / medium` +
			"\n" +
			`- documentation: none / minimal / adequate` +
			"\n" +
			`- test_coverage: none / partial / good` +
			"\n" +
			`- recommendation: pair program, write docs, add tests, or cross-train`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 72 * time.Hour,
	},

	// Category 3: "Here are options"
	TaskGroomer: {
		Type:     TaskGroomer,
		Category: CategoryOptions,
		Name:     "Task Groomer",
		Description: `Audit all task descriptions in the task registry and identify stubs that need ` +
			`to be expanded into detailed agent prompts.` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Read internal/tasks/tasks.go and enumerate all TaskDefinition entries in the registry.` +
			"\n" +
			`2. Classify each description as "detailed" (multi-line with methodology/output format) ` +
			`or "stub" (one-liner without actionable instructions).` +
			"\n" +
			`3. For each stub, draft an expanded description following the pattern of existing ` +
			`detailed descriptions (TaskPIIScanner, TaskSkillGroom, TaskTDReview): what to examine, ` +
			`methodology steps, output format, and scope boundaries.` +
			"\n" +
			`4. Present expanded descriptions as options for review before applying.` +
			"\n\n" +
			`EVALUATION CRITERIA for a good description:` +
			"\n" +
			`- Specific enough that an agent can execute without clarification` +
			"\n" +
			`- Includes WHAT TO LOOK FOR or WHAT TO EXAMINE section` +
			"\n" +
			`- Includes METHODOLOGY with numbered steps` +
			"\n" +
			`- Includes OUTPUT FORMAT with expected fields` +
			"\n" +
			`- Includes SCOPE boundaries (what NOT to do)` +
			"\n\n" +
			`OUTPUT: A list of stub descriptions found with proposed expansions, ` +
			`grouped by category. Include a count of stubs vs detailed descriptions.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskGuideImprover: {
		Type:     TaskGuideImprover,
		Category: CategoryOptions,
		Name:     "Guide/Skill Improver",
		Description: `Review project guides, READMEs, CONTRIBUTING docs, and agent skill files ` +
			`for clarity, accuracy, and completeness, then suggest improvements.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- README.md: does it cover setup, usage, architecture, and contribution?` +
			"\n" +
			`- CONTRIBUTING.md: are PR guidelines, code style, and test requirements clear?` +
			"\n" +
			`- Agent skill files (.claude/skills/, .codex/skills/): frontmatter accuracy, ` +
			`trigger conditions, and instruction clarity` +
			"\n" +
			`- CLAUDE.md: are project conventions and constraints documented?` +
			"\n" +
			`- Developer guides: onboarding docs, architecture docs, runbooks` +
			"\n\n" +
			`EVALUATION CRITERIA:` +
			"\n" +
			`- Can a new contributor get the project running from the README alone?` +
			"\n" +
			`- Are examples up to date and runnable?` +
			"\n" +
			`- Is the writing concise and scannable?` +
			"\n" +
			`- Are there gaps where a reader would get stuck?` +
			"\n\n" +
			`OUTPUT: Prioritized list of suggestions, each with:` +
			"\n" +
			`- file: which guide` +
			"\n" +
			`- section: where the issue is` +
			"\n" +
			`- issue: gap | outdated | unclear | verbose | missing-example` +
			"\n" +
			`- priority: high / medium / low` +
			"\n" +
			`- suggestion: specific text or structural change`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskIdeaGenerator: {
		Type:     TaskIdeaGenerator,
		Category: CategoryOptions,
		Name:     "Idea Generator",
		Description: `Analyze the codebase and generate concrete improvement ideas across categories: ` +
			`developer experience, performance, reliability, and user-facing features.` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Read the project structure, README, and recent git history to understand ` +
			`the project's purpose and trajectory.` +
			"\n" +
			`2. Identify friction points: repetitive manual steps, missing automation, ` +
			`error-prone workflows, and slow feedback loops.` +
			"\n" +
			`3. Look for patterns in issues and TODOs that suggest unmet needs.` +
			"\n" +
			`4. Generate ideas that are specific and actionable, not generic advice.` +
			"\n\n" +
			`IDEA CATEGORIES:` +
			"\n" +
			`- DX: developer workflow improvements, better error messages, faster iteration` +
			"\n" +
			`- Reliability: error handling gaps, retry logic, graceful degradation` +
			"\n" +
			`- Performance: caching opportunities, batching, lazy loading` +
			"\n" +
			`- Features: capabilities suggested by the existing architecture but not yet built` +
			"\n\n" +
			`OUTPUT: For each idea:` +
			"\n" +
			`- title: short, descriptive name` +
			"\n" +
			`- category: dx | reliability | performance | feature` +
			"\n" +
			`- effort: small (hours) / medium (days) / large (weeks)` +
			"\n" +
			`- impact: high / medium / low` +
			"\n" +
			`- description: what to build and why it matters` +
			"\n" +
			`- starting_point: which files/functions to modify first` +
			"\n\n" +
			`Sort by impact/effort ratio. Generate 5-15 ideas.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskTechDebtClassify: {
		Type:     TaskTechDebtClassify,
		Category: CategoryOptions,
		Name:     "Tech-Debt Classifier",
		Description: `Survey the codebase for technical debt, classify each instance, and produce ` +
			`a prioritized remediation backlog.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- TODO/FIXME/HACK/XXX comments with context` +
			"\n" +
			`- Deprecated function usage, compatibility shims, version-guarded code` +
			"\n" +
			`- Copy-pasted logic that should be abstracted` +
			"\n" +
			`- Outdated dependencies pinned to old versions for compatibility` +
			"\n" +
			`- Missing error handling, swallowed errors, panic-based flow control` +
			"\n" +
			`- God functions/files (>300 lines or >10 responsibilities)` +
			"\n" +
			`- Inconsistent patterns across packages (different logging, config, testing approaches)` +
			"\n\n" +
			`CLASSIFICATION:` +
			"\n" +
			`- Type: design-debt | code-debt | test-debt | dependency-debt | doc-debt` +
			"\n" +
			`- Interest rate: how fast is this getting worse? (compounding / stable / isolated)` +
			"\n" +
			`- Payoff effort: how hard is it to fix? (trivial / moderate / significant / rewrite)` +
			"\n\n" +
			`OUTPUT: For each debt item:` +
			"\n" +
			`- location: file:line or package` +
			"\n" +
			`- type: classification from above` +
			"\n" +
			`- interest_rate: compounding / stable / isolated` +
			"\n" +
			`- effort: trivial / moderate / significant` +
			"\n" +
			`- description: what the debt is and why it matters` +
			"\n" +
			`- recommendation: specific remediation approach` +
			"\n\n" +
			`Sort by interest_rate descending (fix compounding debt first), then by effort ascending.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskWhyAnnotator: {
		Type:     TaskWhyAnnotator,
		Category: CategoryOptions,
		Name:     "Why Does This Exist Annotator",
		Description: `Find code that is non-obvious, surprising, or appears unnecessary, ` +
			`research why it exists, and suggest clarifying comments or refactors.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- Workarounds with no explanation (magic numbers, special-case logic, platform checks)` +
			"\n" +
			`- Code that contradicts the surrounding style or architecture` +
			"\n" +
			`- Functions that seem like they duplicate stdlib functionality` +
			"\n" +
			`- Error handling that catches and ignores specific errors` +
			"\n" +
			`- Build constraints, conditional compilation, or platform-specific code paths` +
			"\n" +
			`- Retry logic, backoff patterns, or timeouts without documented rationale` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Read each package and flag code that would confuse a new contributor.` +
			"\n" +
			`2. Use git blame and commit messages to research the original intent.` +
			"\n" +
			`3. Check related issues or PRs for context.` +
			"\n" +
			`4. For each finding, propose either: a clarifying comment (if the code is correct ` +
			`but non-obvious) or a refactor (if the original reason no longer applies).` +
			"\n\n" +
			`OUTPUT: For each unclear code section:` +
			"\n" +
			`- location: file:line range` +
			"\n" +
			`- what: description of the non-obvious code` +
			"\n" +
			`- why: researched reason it exists (or "[UNKNOWN]" if not determinable)` +
			"\n" +
			`- recommendation: add-comment (with proposed text) | refactor | remove`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskEdgeCaseEnum: {
		Type:     TaskEdgeCaseEnum,
		Category: CategoryOptions,
		Name:     "Edge-Case Enumerator",
		Description: `Systematically enumerate edge cases for critical code paths that may not ` +
			`be handled correctly.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Input parsing: empty strings, unicode, very long inputs, null bytes, special chars` +
			"\n" +
			`- Numeric operations: zero, negative, overflow, NaN, infinity, max-int boundaries` +
			"\n" +
			`- Collections: empty, single-element, very large, nil vs empty` +
			"\n" +
			`- Concurrency: simultaneous access, shutdown during operation, partial completion` +
			"\n" +
			`- Network: timeout, partial response, connection reset, DNS failure` +
			"\n" +
			`- File system: permission denied, disk full, symlinks, paths with spaces` +
			"\n" +
			`- Time: midnight, DST transitions, leap seconds, clock skew, zero time` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Identify the critical code paths (config parsing, CLI arg handling, ` +
			`data processing, API handlers).` +
			"\n" +
			`2. For each input parameter, enumerate edge-case values.` +
			"\n" +
			`3. Trace the code path to determine if each edge case is handled.` +
			"\n" +
			`4. Classify: handled, partially handled, unhandled.` +
			"\n\n" +
			`OUTPUT: For each edge case:` +
			"\n" +
			`- function: where the edge case applies` +
			"\n" +
			`- input: the edge-case value or condition` +
			"\n" +
			`- current_behavior: what happens now (crash, wrong result, silent failure, correct handling)` +
			"\n" +
			`- status: handled | partial | unhandled` +
			"\n" +
			`- priority: critical / high / medium / low` +
			"\n" +
			`- suggestion: how to handle it properly`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskErrorMsgImprove: {
		Type:     TaskErrorMsgImprove,
		Category: CategoryOptions,
		Name:     "Error-Message Improver",
		Description: `Review error messages in the codebase and suggest improvements to make them ` +
			`more actionable for users and operators.` +
			"\n\n" +
			`WHAT TO LOOK FOR:` +
			"\n" +
			`- Generic messages: "an error occurred", "invalid input", "failed"` +
			"\n" +
			`- Missing context: errors without the value that caused them, the operation attempted, ` +
			`or the expected format` +
			"\n" +
			`- No remediation: errors that tell what went wrong but not how to fix it` +
			"\n" +
			`- Inconsistent formatting: mixed casing, punctuation, error wrapping styles` +
			"\n" +
			`- Technical jargon in user-facing messages (stack traces, internal type names)` +
			"\n" +
			`- Missing error codes or classification for programmatic handling` +
			"\n\n" +
			`EVALUATION CRITERIA for a good error message:` +
			"\n" +
			`- States what happened specifically` +
			"\n" +
			`- Includes the relevant value or context` +
			"\n" +
			`- Suggests how to fix it (when possible)` +
			"\n" +
			`- Uses consistent formatting across the project` +
			"\n\n" +
			`OUTPUT: For each poor error message:` +
			"\n" +
			`- location: file:line` +
			"\n" +
			`- current: the existing error message` +
			"\n" +
			`- issue: generic | missing-context | no-remediation | inconsistent | jargon` +
			"\n" +
			`- suggested: improved error message text` +
			"\n" +
			`- priority: high (user-facing) / medium (operator-facing) / low (internal)`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskSLOSuggester: {
		Type:     TaskSLOSuggester,
		Category: CategoryOptions,
		Name:     "SLO/SLA Candidate Suggester",
		Description: `Analyze the system's operations and suggest Service Level Objectives (SLOs) ` +
			`and Service Level Agreement (SLA) candidates based on actual capabilities.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- API endpoints: what latency percentiles and error rates are achievable?` +
			"\n" +
			`- Background jobs: what throughput and completion time guarantees are realistic?` +
			"\n" +
			`- Data pipelines: what freshness and completeness can be promised?` +
			"\n" +
			`- Availability: what uptime is achievable given the architecture?` +
			"\n" +
			`- Existing metrics: what's already measured that could back an SLO?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Inventory all user-facing operations and internal services.` +
			"\n" +
			`2. For each, identify the key quality metrics (latency, error rate, throughput, freshness).` +
			"\n" +
			`3. Check existing instrumentation: can these metrics be measured today?` +
			"\n" +
			`4. Propose SLO targets based on architecture analysis (not guessing).` +
			"\n\n" +
			`OUTPUT: For each SLO candidate:` +
			"\n" +
			`- service: component or endpoint` +
			"\n" +
			`- indicator: what to measure (p99 latency, error rate, availability)` +
			"\n" +
			`- suggested_target: proposed SLO value (e.g., p99 < 500ms, error rate < 0.1%)` +
			"\n" +
			`- measurable_today: yes / no (and what instrumentation is needed if no)` +
			"\n" +
			`- rationale: why this target is appropriate for this operation` +
			"\n" +
			`- priority: high (customer-facing) / medium (internal) / low (nice-to-have)`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskUXCopySharpener: {
		Type:     TaskUXCopySharpener,
		Category: CategoryOptions,
		Name:     "UX Copy Sharpener",
		Description: `Review all user-facing text in the application (CLI output, help text, ` +
			`error messages, prompts, UI labels) and suggest improvements for clarity and consistency.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- CLI --help output: is it scannable? Are descriptions consistent in tone and length?` +
			"\n" +
			`- Progress messages: are they informative without being noisy?` +
			"\n" +
			`- Confirmation prompts: do they clearly state the consequences?` +
			"\n" +
			`- Error messages: see TaskErrorMsgImprove for overlap; here focus on tone and wording` +
			"\n" +
			`- Success/completion messages: are they useful or just noise?` +
			"\n" +
			`- Terminology: is the same concept called the same thing everywhere?` +
			"\n\n" +
			`EVALUATION CRITERIA:` +
			"\n" +
			`- Consistent voice and tone throughout the application` +
			"\n" +
			`- Uses the user's terminology, not internal engineering jargon` +
			"\n" +
			`- Scannable: important info first, details after` +
			"\n" +
			`- Actionable: tells users what to do, not just what happened` +
			"\n\n" +
			`OUTPUT: For each improvement:` +
			"\n" +
			`- location: file:line` +
			"\n" +
			`- current: existing text` +
			"\n" +
			`- issue: jargon | inconsistent-tone | verbose | unclear | missing-context` +
			"\n" +
			`- suggested: improved text` +
			"\n" +
			`- priority: high / medium / low (based on user visibility and frequency)`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskA11yLint: {
		Type:     TaskA11yLint,
		Category: CategoryOptions,
		Name:     "Accessibility Linting",
		Description: `Perform deep accessibility analysis beyond automated checklist tools. ` +
			`Evaluate the application from the perspective of users with diverse abilities.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Keyboard navigation: can all interactive elements be reached and operated ` +
			`without a mouse? Is focus order logical?` +
			"\n" +
			`- Screen reader experience: are ARIA labels meaningful? Is content order sensible ` +
			`when linearized? Are dynamic updates announced?` +
			"\n" +
			`- Color contrast: do text/background combinations meet WCAG AA ratios?` +
			"\n" +
			`- Motion: are animations respectful of prefers-reduced-motion?` +
			"\n" +
			`- CLI accessibility: is output parseable by screen readers? ` +
			`Are colors optional (NO_COLOR support)? Is output width-aware?` +
			"\n" +
			`- Error states: are errors announced to assistive tech and not just visual?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Inventory all interaction points (forms, buttons, links, CLI commands).` +
			"\n" +
			`2. Evaluate each against WCAG 2.1 AA criteria—not just the automated ones.` +
			"\n" +
			`3. Identify semantic HTML issues that automated tools miss.` +
			"\n" +
			`4. Test keyboard-only interaction flows mentally or via code analysis.` +
			"\n\n" +
			`OUTPUT: For each issue:` +
			"\n" +
			`- location: file:line or component/command` +
			"\n" +
			`- wcag_criterion: specific WCAG success criterion (e.g., 2.1.1 Keyboard)` +
			"\n" +
			`- issue: description of the barrier` +
			"\n" +
			`- impact: who is affected and how` +
			"\n" +
			`- priority: critical / high / medium / low` +
			"\n" +
			`- suggestion: specific fix`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskServiceAdvisor: {
		Type:     TaskServiceAdvisor,
		Category: CategoryOptions,
		Name:     "Should This Be a Service Advisor",
		Description: `Analyze the codebase architecture to identify components that should (or should not) ` +
			`be extracted into separate services, and present the tradeoffs.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Package boundaries: which packages have high coupling vs. high cohesion?` +
			"\n" +
			`- Scaling requirements: are there components with different scaling needs ` +
			`(CPU-bound vs. I/O-bound, burst vs. steady)?` +
			"\n" +
			`- Deployment frequency: are some components changing much faster than others?` +
			"\n" +
			`- Data ownership: which components own which data, and do they share databases?` +
			"\n" +
			`- Failure isolation: would a crash in one component take down unrelated functionality?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Map the dependency graph between packages.` +
			"\n" +
			`2. Identify clusters of high cohesion separated by thin interfaces.` +
			"\n" +
			`3. For each potential extraction, evaluate: network boundary cost, data consistency ` +
			`challenges, operational overhead, and team structure alignment.` +
			"\n" +
			`4. Present each as a decision with tradeoffs, not a recommendation.` +
			"\n\n" +
			`OUTPUT: For each boundary decision:` +
			"\n" +
			`- component: what could be extracted` +
			"\n" +
			`- current_coupling: how tightly bound it is (high / medium / low)` +
			"\n" +
			`- extract_pros: benefits of separation` +
			"\n" +
			`- extract_cons: costs and risks of separation` +
			"\n" +
			`- keep_pros: benefits of keeping it monolithic` +
			"\n" +
			`- recommendation: extract | keep | defer (with reasoning)`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 168 * time.Hour,
	},
	TaskOwnershipBoundary: {
		Type:     TaskOwnershipBoundary,
		Category: CategoryOptions,
		Name:     "Ownership Boundary Suggester",
		Description: `Analyze the codebase structure and suggest clear ownership boundaries ` +
			`for teams or individuals based on code topology and change patterns.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Package/directory structure vs. actual change patterns (who changes what together?)` +
			"\n" +
			`- Cross-cutting changes: which files frequently change together across packages?` +
			"\n" +
			`- CODEOWNERS file: does it exist? Is it accurate and granular enough?` +
			"\n" +
			`- Interface boundaries: where are the natural seams between ownership areas?` +
			"\n" +
			`- Shared code: which utilities/helpers are used across multiple ownership areas?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Analyze git log for co-change patterns (files that change in the same commit).` +
			"\n" +
			`2. Map the dependency graph to identify natural boundaries.` +
			"\n" +
			`3. Compare current CODEOWNERS (if any) against actual change patterns.` +
			"\n" +
			`4. Identify areas with unclear or overlapping ownership.` +
			"\n\n" +
			`OUTPUT: Proposed ownership map:` +
			"\n" +
			`- area: directory or package path` +
			"\n" +
			`- suggested_owner: team or role (not specific people)` +
			"\n" +
			`- rationale: why this boundary makes sense` +
			"\n" +
			`- shared_dependencies: code shared with other areas that needs co-ownership rules` +
			"\n" +
			`- boundary_quality: clean (clear interface) / messy (high cross-boundary coupling)`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskOncallEstimator: {
		Type:     TaskOncallEstimator,
		Category: CategoryOptions,
		Name:     "Oncall Load Estimator",
		Description: `Estimate the operational oncall burden based on code complexity, error handling ` +
			`patterns, and failure modes to help plan staffing and prioritize reliability work.` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- Error handling quality: are errors recoverable or do they require human intervention?` +
			"\n" +
			`- Retry/backoff patterns: are transient failures handled automatically?` +
			"\n" +
			`- Monitoring coverage: what alerts exist and what gaps remain?` +
			"\n" +
			`- Failure modes: what can go wrong and how visible is it?` +
			"\n" +
			`- Recovery procedures: are there runbooks, or would oncall need to investigate from scratch?` +
			"\n" +
			`- External dependencies: how many third-party services could page you?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Map all failure modes: network errors, disk full, OOM, config errors, ` +
			`dependency outages, data corruption.` +
			"\n" +
			`2. For each, assess: is it auto-recoverable, alertable, or silent?` +
			"\n" +
			`3. Estimate oncall toil: how many of these failures require manual intervention?` +
			"\n" +
			`4. Identify the highest-toil areas and suggest automation.` +
			"\n\n" +
			`OUTPUT: Oncall burden assessment:` +
			"\n" +
			`- area: component or subsystem` +
			"\n" +
			`- failure_modes: count and types` +
			"\n" +
			`- auto_recoverable: percentage of failures handled automatically` +
			"\n" +
			`- estimated_pages_per_week: low (0-1) / medium (2-5) / high (5+)` +
			"\n" +
			`- toil_reduction: specific improvements to reduce oncall burden` +
			"\n" +
			`- priority: critical / high / medium / low`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},

	// Category 4: "I tried it safely"
	TaskMigrationRehearsal: {
		Type:     TaskMigrationRehearsal,
		Category: CategorySafe,
		Name:     "Migration Rehearsal Runner",
		Description: `Rehearse database and data migrations in a safe, isolated environment ` +
			`to validate they work correctly before production deployment.` +
			"\n\n" +
			`SAFETY GUARDRAILS:` +
			"\n" +
			`- NEVER run migrations against production databases or live systems` +
			"\n" +
			`- Use only local/test databases, Docker containers, or in-memory stores` +
			"\n" +
			`- Verify the target is not a production endpoint before executing` +
			"\n" +
			`- Clean up all test resources after the rehearsal` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Find all pending migration files (SQL, ORM migrations, data scripts).` +
			"\n" +
			`2. Set up a clean test database (via Docker, SQLite, or existing test infrastructure).` +
			"\n" +
			`3. Apply migrations in order and verify each succeeds.` +
			"\n" +
			`4. Test rollback: apply migrations forward, then roll back, then re-apply.` +
			"\n" +
			`5. Load representative test data and verify application queries still work.` +
			"\n" +
			`6. Measure migration duration and resource usage.` +
			"\n\n" +
			`OUTPUT:` +
			"\n" +
			`- migration_count: number of migrations rehearsed` +
			"\n" +
			`- result: all-pass | partial-failure | blocked` +
			"\n" +
			`- duration: total execution time` +
			"\n" +
			`- rollback_safe: yes / no / partial (per migration)` +
			"\n" +
			`- issues: any errors, warnings, or data integrity concerns` +
			"\n" +
			`- recommendation: safe to deploy / needs fixes (with specifics)`,
		CostTier:        CostVeryHigh,
		RiskLevel:       RiskHigh,
		DefaultInterval: 336 * time.Hour,
	},
	TaskContractFuzzer: {
		Type:     TaskContractFuzzer,
		Category: CategorySafe,
		Name:     "Integration Contract Fuzzer",
		Description: `Fuzz-test integration points (APIs, message formats, config parsers) ` +
			`with unexpected inputs to find edge cases and crashes.` +
			"\n\n" +
			`SAFETY GUARDRAILS:` +
			"\n" +
			`- Only fuzz against local test servers or in-process handlers` +
			"\n" +
			`- NEVER send fuzzed data to production or shared staging environments` +
			"\n" +
			`- Set resource limits (timeout, memory cap) to prevent runaway fuzzing` +
			"\n" +
			`- Clean up any generated test artifacts` +
			"\n\n" +
			`WHAT TO FUZZ:` +
			"\n" +
			`- JSON/YAML/TOML parsers with malformed, deeply nested, or oversized inputs` +
			"\n" +
			`- CLI argument parsing with unusual flag combinations and special characters` +
			"\n" +
			`- Config file loading with missing fields, wrong types, and extra keys` +
			"\n" +
			`- API request handlers with boundary values, empty bodies, and corrupt headers` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Identify all input parsing functions and integration boundaries.` +
			"\n" +
			`2. Write Go fuzz tests (testing.F) or generate input corpuses for existing fuzz targets.` +
			"\n" +
			`3. Run fuzzing for a bounded duration (e.g., 30 seconds per target).` +
			"\n" +
			`4. Collect crashes, panics, and unexpected error behaviors.` +
			"\n\n" +
			`OUTPUT:` +
			"\n" +
			`- targets_fuzzed: count and list of functions fuzzed` +
			"\n" +
			`- duration: total fuzzing time` +
			"\n" +
			`- crashes: count and details of each crash (input, stack trace, root cause)` +
			"\n" +
			`- edge_cases: unexpected but non-crashing behaviors` +
			"\n" +
			`- recommendation: fixes for each crash, fuzz tests to add permanently`,
		CostTier:        CostVeryHigh,
		RiskLevel:       RiskHigh,
		DefaultInterval: 336 * time.Hour,
	},
	TaskGoldenPath: {
		Type:     TaskGoldenPath,
		Category: CategorySafe,
		Name:     "Golden-Path Recorder",
		Description: `Record golden-path test scenarios that capture the expected behavior of ` +
			`critical user workflows end-to-end.` +
			"\n\n" +
			`SAFETY GUARDRAILS:` +
			"\n" +
			`- Use only test/dev configurations, never production credentials or endpoints` +
			"\n" +
			`- Record against local or containerized instances` +
			"\n" +
			`- Do not persist sensitive data in golden files` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Identify the critical user workflows (setup, primary use cases, ` +
			`configuration, error recovery).` +
			"\n" +
			`2. For each workflow, define the sequence of inputs and expected outputs.` +
			"\n" +
			`3. Execute each workflow and capture: command inputs, stdout/stderr output, ` +
			`exit codes, and any generated files.` +
			"\n" +
			`4. Save as golden test files (testdata/*.golden) with clear naming.` +
			"\n" +
			`5. Create or update test functions that replay and compare against golden files.` +
			"\n\n" +
			`OUTPUT:` +
			"\n" +
			`- workflows_recorded: count and names` +
			"\n" +
			`- golden_files: list of generated golden test files` +
			"\n" +
			`- test_functions: list of test functions that use the golden files` +
			"\n" +
			`- coverage_improvement: which code paths are now covered that weren't before`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 336 * time.Hour,
	},
	TaskPerfProfile: {
		Type:     TaskPerfProfile,
		Category: CategorySafe,
		Name:     "Performance Profiling Runs",
		Description: `Run CPU and latency profiling on the application to identify performance ` +
			`bottlenecks and hot paths.` +
			"\n\n" +
			`SAFETY GUARDRAILS:` +
			"\n" +
			`- Profile only against local test workloads, not production traffic` +
			"\n" +
			`- Set bounded execution time for profiling runs` +
			"\n" +
			`- Do not leave profiling endpoints enabled in committed code` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Identify profiling targets: benchmark tests, CLI commands, or test scenarios ` +
			`that exercise critical paths.` +
			"\n" +
			`2. Run Go CPU profiles (go test -cpuprofile) on benchmark tests.` +
			"\n" +
			`3. Analyze profiles with go tool pprof to identify top CPU consumers.` +
			"\n" +
			`4. If benchmark tests don't exist for hot paths, create minimal ones.` +
			"\n" +
			`5. Compare against previous profiles if available (check testdata/ or profiles/).` +
			"\n\n" +
			`OUTPUT:` +
			"\n" +
			`- profiled_targets: list of functions/scenarios profiled` +
			"\n" +
			`- top_cpu_consumers: top 10 functions by CPU time (function, package, percentage)` +
			"\n" +
			`- bottlenecks: identified performance bottlenecks with context` +
			"\n" +
			`- recommendations: specific optimizations ranked by expected impact` +
			"\n" +
			`- profile_files: paths to saved profile data for future comparison`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 336 * time.Hour,
	},
	TaskAllocationProfile: {
		Type:     TaskAllocationProfile,
		Category: CategorySafe,
		Name:     "Allocation/Hot-Path Profiling",
		Description: `Profile memory allocation patterns to identify excessive allocations, ` +
			`memory leaks, and GC pressure in hot paths.` +
			"\n\n" +
			`SAFETY GUARDRAILS:` +
			"\n" +
			`- Profile only against local test workloads` +
			"\n" +
			`- Set bounded execution time and memory limits` +
			"\n" +
			`- Clean up generated profile files after analysis` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Run Go memory profiles (go test -memprofile, -benchmem) on benchmark tests.` +
			"\n" +
			`2. Identify functions with high allocation counts (allocs/op) and sizes (bytes/op).` +
			"\n" +
			`3. Look for allocation patterns in hot loops: string concatenation, slice growth ` +
			`without pre-allocation, interface boxing, fmt.Sprintf in tight loops.` +
			"\n" +
			`4. Check for potential memory leaks: goroutines that never exit, maps that grow ` +
			`without eviction, retained references preventing GC.` +
			"\n" +
			`5. Run with -race flag to detect concurrent access issues.` +
			"\n\n" +
			`OUTPUT:` +
			"\n" +
			`- profiled_targets: list of functions/scenarios profiled` +
			"\n" +
			`- top_allocators: top 10 functions by allocation count/size` +
			"\n" +
			`- hot_path_allocations: allocations in performance-critical paths` +
			"\n" +
			`- leak_candidates: potential memory leaks with evidence` +
			"\n" +
			`- recommendations: specific fixes (pre-allocate, pool, sync.Pool, avoid interface boxing)` +
			"\n" +
			`- gc_impact: estimated GC pressure reduction from proposed fixes`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 336 * time.Hour,
	},

	// Category 5: "Here's the map"
	TaskVisibilityInstrument: {
		Type:     TaskVisibilityInstrument,
		Category: CategoryMap,
		Name:     "Visibility Instrumentor",
		Description: `Map the current observability instrumentation across the codebase and ` +
			`identify gaps where visibility is missing.` +
			"\n\n" +
			`WHAT TO MAP:` +
			"\n" +
			`- Logging: which operations are logged, at what level, with what context?` +
			"\n" +
			`- Metrics: which operations have counters, histograms, or gauges?` +
			"\n" +
			`- Tracing: which operations create spans? Is trace context propagated across boundaries?` +
			"\n" +
			`- Health checks: what health/readiness endpoints exist?` +
			"\n" +
			`- Alerting: what conditions trigger alerts (from code or config)?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Grep for logging, metrics, and tracing library calls.` +
			"\n" +
			`2. Map each instrumentation point to the operation it covers.` +
			"\n" +
			`3. Walk critical code paths and mark gaps: operations with no logging, ` +
			`no metrics, or no tracing.` +
			"\n" +
			`4. Evaluate context propagation: do traces connect across goroutines, ` +
			`HTTP calls, and message queues?` +
			"\n\n" +
			`OUTPUT: Visibility map as a table:` +
			"\n" +
			`- operation: name of the operation or code path` +
			"\n" +
			`- logging: yes / partial / no` +
			"\n" +
			`- metrics: yes / partial / no` +
			"\n" +
			`- tracing: yes / partial / no` +
			"\n" +
			`- gap_priority: critical / high / medium / low` +
			"\n" +
			`- recommendation: specific instrumentation to add`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 168 * time.Hour,
	},
	TaskRepoTopology: {
		Type:     TaskRepoTopology,
		Category: CategoryMap,
		Name:     "Repo Topology Visualizer",
		Description: `Generate a structural map of the repository showing package dependencies, ` +
			`module boundaries, and communication patterns.` +
			"\n\n" +
			`WHAT TO MAP:` +
			"\n" +
			`- Package dependency graph (which packages import which)` +
			"\n" +
			`- Layer violations (e.g., a "model" package importing a "handler" package)` +
			"\n" +
			`- Circular dependencies or near-circular dependency chains` +
			"\n" +
			`- Entry points: main packages, CLI commands, HTTP handlers` +
			"\n" +
			`- Shared utilities: packages imported by many other packages` +
			"\n" +
			`- External boundaries: where the code talks to the outside world` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Parse import statements across all Go files.` +
			"\n" +
			`2. Build the dependency graph and identify layers.` +
			"\n" +
			`3. Detect architectural violations (imports going the wrong direction).` +
			"\n" +
			`4. Generate a text-based or Mermaid diagram of the topology.` +
			"\n\n" +
			`OUTPUT:` +
			"\n" +
			`- Package dependency graph (text or Mermaid format)` +
			"\n" +
			`- Layers identified: entry points → business logic → data access → utilities` +
			"\n" +
			`- Violations: list of imports that cross layer boundaries incorrectly` +
			"\n" +
			`- Hotspots: packages with unusually high fan-in or fan-out` +
			"\n" +
			`- Suggestions: structural improvements if violations or hotspots are found`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskPermissionsMapper: {
		Type:     TaskPermissionsMapper,
		Category: CategoryMap,
		Name:     "Permissions/Auth Surface Mapper",
		Description: `Map all permission checks, authentication gates, and authorization ` +
			`decisions in the codebase to create a security surface map.` +
			"\n\n" +
			`WHAT TO MAP:` +
			"\n" +
			`- Authentication: where are credentials validated? (middleware, login handlers, token checks)` +
			"\n" +
			`- Authorization: where are permission/role checks performed?` +
			"\n" +
			`- Unprotected endpoints: routes or operations missing auth checks` +
			"\n" +
			`- API keys and tokens: where are they validated and what scopes do they grant?` +
			"\n" +
			`- File permissions: where does the code set or check file/directory permissions?` +
			"\n" +
			`- Privilege escalation paths: can a low-privilege operation reach a high-privilege one?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Find all auth middleware, decorators, and guard functions.` +
			"\n" +
			`2. Map which endpoints/operations are protected and which are open.` +
			"\n" +
			`3. Trace the auth decision path: from credential presentation to resource access.` +
			"\n" +
			`4. Identify gaps: operations that should be gated but aren't.` +
			"\n\n" +
			`OUTPUT: Auth surface map:` +
			"\n" +
			`- endpoint/operation: name and path` +
			"\n" +
			`- auth_type: none | api-key | token | session | mutual-tls` +
			"\n" +
			`- authorization: none | role-based | scope-based | owner-only` +
			"\n" +
			`- protected: yes / no / partial` +
			"\n" +
			`- risk: critical / high / medium / low (for unprotected operations)` +
			"\n" +
			`- recommendation: what auth to add for unprotected operations`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskDataLifecycle: {
		Type:     TaskDataLifecycle,
		Category: CategoryMap,
		Name:     "Data Lifecycle Tracer",
		Description: `Trace how data flows through the system from ingestion to storage ` +
			`to deletion, mapping the complete lifecycle.` +
			"\n\n" +
			`WHAT TO MAP:` +
			"\n" +
			`- Data ingestion points: where does data enter? (API endpoints, CLI input, ` +
			`file reads, message queues, webhooks)` +
			"\n" +
			`- Transformations: how is data transformed, validated, or enriched in transit?` +
			"\n" +
			`- Storage: where is data persisted? (database, files, cache, external services)` +
			"\n" +
			`- Access patterns: what reads the stored data and how?` +
			"\n" +
			`- Deletion/retention: how and when is data removed? Are there TTLs, cleanup jobs, ` +
			`or manual deletion endpoints?` +
			"\n" +
			`- Data export: where does data leave the system? (API responses, exports, ` +
			`third-party integrations)` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Identify all data types/models in the system.` +
			"\n" +
			`2. For each data type, trace: creation → storage → access → update → deletion.` +
			"\n" +
			`3. Map external boundaries where data crosses system borders.` +
			"\n" +
			`4. Identify data that is stored but never deleted (potential compliance risk).` +
			"\n\n" +
			`OUTPUT: Data lifecycle map per data type:` +
			"\n" +
			`- data_type: name and description` +
			"\n" +
			`- ingestion: how it enters (endpoint, function)` +
			"\n" +
			`- storage: where it's persisted (db table, file, cache key)` +
			"\n" +
			`- retention: TTL or deletion policy (if any)` +
			"\n" +
			`- export: where it leaves the system` +
			"\n" +
			`- gaps: missing deletion logic, unencrypted storage, or compliance concerns`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskFeatureFlagMonitor: {
		Type:     TaskFeatureFlagMonitor,
		Category: CategoryMap,
		Name:     "Feature Flag Lifecycle Monitor",
		Description: `Map all feature flags in the codebase and assess their lifecycle status ` +
			`to identify stale flags that should be cleaned up.` +
			"\n\n" +
			`WHAT TO MAP:` +
			"\n" +
			`- Flag definitions: where are feature flags declared? (config files, code constants, ` +
			`environment variables, external flag services)` +
			"\n" +
			`- Flag usage: where are flags checked in the code? How many code paths depend on each?` +
			"\n" +
			`- Flag age: when was each flag introduced? (git blame)` +
			"\n" +
			`- Flag state: is the flag still actively toggled, or has it been permanently ` +
			`on/off for a long time?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Grep for feature flag patterns (env lookups, config checks, flag library calls).` +
			"\n" +
			`2. Map each flag to all code locations that check it.` +
			"\n" +
			`3. Use git blame to determine flag age.` +
			"\n" +
			`4. Classify each flag's lifecycle status.` +
			"\n\n" +
			`OUTPUT: Feature flag inventory:` +
			"\n" +
			`- flag: name or identifier` +
			"\n" +
			`- age: when introduced (date)` +
			"\n" +
			`- usage_count: number of code locations checking this flag` +
			"\n" +
			`- status: active | stale (>90 days unchanged) | permanent (always on/off)` +
			"\n" +
			`- recommendation: keep | clean-up (remove flag, keep winning code path) | review` +
			"\n\n" +
			`Highlight flags older than 90 days that appear permanently enabled or disabled.`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskCISignalNoise: {
		Type:     TaskCISignalNoise,
		Category: CategoryMap,
		Name:     "CI Signal-to-Noise Scorer",
		Description: `Evaluate CI pipeline configuration for signal-to-noise ratio: ` +
			`are CI checks catching real problems, or generating noise that trains developers to ignore failures?` +
			"\n\n" +
			`WHAT TO EXAMINE:` +
			"\n" +
			`- CI workflow files (.github/workflows/, .gitlab-ci.yml, Jenkinsfile, etc.)` +
			"\n" +
			`- Test execution: are tests run with appropriate flags (-race, -timeout)?` +
			"\n" +
			`- Linter configuration: are lint rules producing actionable results?` +
			"\n" +
			`- Build steps: are there redundant or unnecessary steps?` +
			"\n" +
			`- Flaky tests: are known-flaky tests quarantined or do they block merges?` +
			"\n" +
			`- Pipeline duration: how long does CI take? Are there optimization opportunities?` +
			"\n" +
			`- Failure patterns: what types of failures are most common?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Read all CI configuration files.` +
			"\n" +
			`2. Map each CI step to what it validates (code quality, correctness, security, etc.).` +
			"\n" +
			`3. Score each step: high-signal (catches real bugs), medium-signal, ` +
			`low-signal (rarely catches anything useful), noise (frequently false-positive).` +
			"\n" +
			`4. Estimate pipeline duration and identify the critical path.` +
			"\n\n" +
			`OUTPUT: CI scorecard:` +
			"\n" +
			`- step: CI step name` +
			"\n" +
			`- signal_score: high / medium / low / noise` +
			"\n" +
			`- duration: estimated time for this step` +
			"\n" +
			`- value: what it catches` +
			"\n" +
			`- recommendation: keep | optimize | remove | add-missing-check`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},
	TaskHistoricalContext: {
		Type:     TaskHistoricalContext,
		Category: CategoryMap,
		Name:     "Historical Context Summarizer",
		Description: `Summarize the historical context of key code areas: why they exist, ` +
			`how they evolved, and what decisions shaped their current form.` +
			"\n\n" +
			`WHAT TO MAP:` +
			"\n" +
			`- Major architectural changes visible in git history` +
			"\n" +
			`- Large refactors: when and why they happened` +
			"\n" +
			`- Technology migrations: old approach → new approach, and is the migration complete?` +
			"\n" +
			`- Bug-fix patterns: are the same areas repeatedly fixed?` +
			"\n" +
			`- Growth trajectory: which areas are expanding vs. stabilizing vs. shrinking?` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Run git log analysis grouped by directory/package.` +
			"\n" +
			`2. Identify major change clusters (large diffs, many files touched simultaneously).` +
			"\n" +
			`3. Read commit messages and PR references for these clusters to understand intent.` +
			"\n" +
			`4. Map the evolution timeline for each major subsystem.` +
			"\n" +
			`5. Identify patterns: increasing velocity (active development), ` +
			`decreasing velocity (stable/abandoned), or churn (repeated rewrites).` +
			"\n\n" +
			`OUTPUT: Historical context map:` +
			"\n" +
			`- area: package or subsystem` +
			"\n" +
			`- age: when first introduced` +
			"\n" +
			`- major_changes: timeline of significant modifications with reasons` +
			"\n" +
			`- current_trajectory: active | stable | declining | churning` +
			"\n" +
			`- key_decisions: important choices visible in history with their rationale` +
			"\n" +
			`- context_for_newcomers: what a new contributor should know about this area`,
		CostTier:        CostMedium,
		RiskLevel:       RiskLow,
		DefaultInterval: 168 * time.Hour,
	},

	// Category 6: "For when things go sideways"
	TaskRunbookGen: {
		Type:     TaskRunbookGen,
		Category: CategoryEmergency,
		Name:     "Runbook Generator",
		Description: `Generate operational runbooks for common failure scenarios based on the ` +
			`system's architecture, dependencies, and failure modes.` +
			"\n\n" +
			`WHAT TO COVER:` +
			"\n" +
			`- Service startup/shutdown procedures` +
			"\n" +
			`- Database connection failures and recovery` +
			"\n" +
			`- External dependency outages (APIs, third-party services)` +
			"\n" +
			`- Disk space / resource exhaustion` +
			"\n" +
			`- Configuration errors and how to diagnose them` +
			"\n" +
			`- Data corruption detection and recovery` +
			"\n" +
			`- Performance degradation diagnosis` +
			"\n\n" +
			`REQUIRED SECTIONS per runbook:` +
			"\n" +
			`1. SYMPTOMS — What does this failure look like? (error messages, metrics, user reports)` +
			"\n" +
			`2. DIAGNOSIS — Step-by-step investigation (what to check, in what order)` +
			"\n" +
			`3. MITIGATION — Immediate actions to reduce impact` +
			"\n" +
			`4. RESOLUTION — How to fix the root cause` +
			"\n" +
			`5. VERIFICATION — How to confirm the fix worked` +
			"\n" +
			`6. ESCALATION — When and to whom to escalate` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Analyze code for external dependencies and failure handling.` +
			"\n" +
			`2. Identify error conditions that require human intervention.` +
			"\n" +
			`3. Generate runbooks using concrete commands and paths from the actual codebase.` +
			"\n" +
			`4. Include relevant log grep patterns, health check URLs, and config file locations.` +
			"\n\n" +
			`OUTPUT: One runbook file per failure scenario. Each includes all required sections ` +
			`with copy-pasteable commands. Generate an index of all runbooks with severity ` +
			`and estimated time-to-resolve.`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 720 * time.Hour,
	},
	TaskRollbackPlan: {
		Type:     TaskRollbackPlan,
		Category: CategoryEmergency,
		Name:     "Rollback Plan Generator",
		Description: `Generate rollback plans for recent changes, ensuring any deployment ` +
			`can be safely reversed.` +
			"\n\n" +
			`WHAT TO ANALYZE:` +
			"\n" +
			`- Code changes since the last release: are they independently rollback-safe?` +
			"\n" +
			`- Database migrations: can they be reversed without data loss?` +
			"\n" +
			`- Config changes: will old config work with old code?` +
			"\n" +
			`- API changes: will old clients work with old server?` +
			"\n" +
			`- Feature flags: can new features be disabled without rollback?` +
			"\n\n" +
			`REQUIRED SECTIONS per rollback plan:` +
			"\n" +
			`1. TRIGGER — When should this rollback be initiated? (error rate threshold, ` +
			`user reports, monitoring alert)` +
			"\n" +
			`2. PRE-ROLLBACK CHECKS — What to verify before rolling back` +
			"\n" +
			`3. ROLLBACK STEPS — Exact commands to execute, in order` +
			"\n" +
			`4. DATA CONSIDERATIONS — Will rollback cause data loss or inconsistency?` +
			"\n" +
			`5. VERIFICATION — How to confirm rollback succeeded` +
			"\n" +
			`6. POST-ROLLBACK — What to communicate and what follow-up is needed` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. List all changes between current HEAD and the last release tag.` +
			"\n" +
			`2. For each change, assess rollback safety.` +
			"\n" +
			`3. Identify changes that cannot be rolled back (destructive migrations, ` +
			`external side effects) and document mitigation strategies.` +
			"\n" +
			`4. Generate step-by-step rollback commands.` +
			"\n\n" +
			`OUTPUT: Rollback plan with all required sections. Flag any changes that are ` +
			`not rollback-safe with severity and workarounds. Include estimated rollback ` +
			`duration and blast radius.`,
		CostTier:        CostHigh,
		RiskLevel:       RiskMedium,
		DefaultInterval: 720 * time.Hour,
	},
	TaskPostmortemGen: {
		Type:     TaskPostmortemGen,
		Category: CategoryEmergency,
		Name:     "Incident Postmortem Draft Generator",
		Description: `Generate a draft incident postmortem document by analyzing recent code changes, ` +
			`error patterns, and system behavior during an incident.` +
			"\n\n" +
			`REQUIRED SECTIONS:` +
			"\n" +
			`1. SUMMARY — One-paragraph description of what happened` +
			"\n" +
			`2. IMPACT — Who was affected, for how long, and to what degree` +
			"\n" +
			`3. TIMELINE — Chronological sequence of events (detection → response → mitigation → resolution)` +
			"\n" +
			`4. ROOT CAUSE — Technical root cause analysis (5 Whys or similar)` +
			"\n" +
			`5. CONTRIBUTING FACTORS — What made the incident worse or harder to detect` +
			"\n" +
			`6. WHAT WENT WELL — What helped during the response` +
			"\n" +
			`7. ACTION ITEMS — Specific, assigned follow-up tasks with priority and deadline` +
			"\n\n" +
			`METHODOLOGY:` +
			"\n" +
			`1. Analyze recent commits around the incident timeframe for the triggering change.` +
			"\n" +
			`2. Review error handling in the affected code paths.` +
			"\n" +
			`3. Identify monitoring gaps that delayed detection.` +
			"\n" +
			`4. Generate the timeline from git history, deploy logs, and code analysis.` +
			"\n" +
			`5. Draft blameless root cause analysis.` +
			"\n" +
			`6. Propose concrete action items with owners.` +
			"\n\n" +
			`ESCALATION GUIDANCE:` +
			"\n" +
			`- All postmortems should be reviewed by the team within 48 hours` +
			"\n" +
			`- Action items should have an owner and a due date` +
			"\n" +
			`- Mark items as P0 (do before next deploy), P1 (this week), P2 (this month)` +
			"\n\n" +
			`OUTPUT: Draft postmortem document with all required sections. Mark areas ` +
			`needing human input with [FILL IN]. Include a summary of action items at the top.`,
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
