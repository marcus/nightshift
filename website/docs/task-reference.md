---
sidebar_position: 6
title: Task Reference
---

# Task Reference

Nightshift includes **59 built-in tasks** organized into 6 categories. Each task has a cost tier, risk level, and default cooldown interval.

Use `nightshift task list` to browse tasks, or `nightshift task show <name>` to see details for a specific task.

## PR Tasks — "It's done — here's the PR"

Fully formed, review-ready artifacts. These tasks create branches and open pull requests.

| Task | Name | Description | Cost | Risk | Cooldown |
|------|------|-------------|------|------|----------|
| `lint-fix` | Linter Fixes | Automatically fix linting errors and style issues | Low | Low | 24h |
| `bug-finder` | Bug Finder & Fixer | Identify and fix potential bugs in code | High | Medium | 72h |
| `auto-dry` | Auto DRY Refactoring | Identify and refactor duplicate code | High | Medium | 7d |
| `skill-groom` | Skill Grooming | Audit and update project-local agent skills to match the current codebase | High | Medium | 7d |
| `api-contract-verify` | API Contract Verification | Verify API contracts match implementation | Medium | Low | 7d |
| `backward-compat` | Backward-Compatibility Checks | Check and ensure backward compatibility | Medium | Low | 7d |
| `build-optimize` | Build Time Optimization | Optimize build configuration for faster builds | High | Medium | 7d |
| `docs-backfill` | Documentation Backfiller | Generate missing documentation | Low | Low | 7d |
| `commit-normalize` | Commit Message Normalizer | Standardize commit message format | Low | Low | 24h |
| `changelog-synth` | Changelog Synthesizer | Generate changelog from commits | Low | Low | 7d |
| `release-notes` | Release Note Drafter | Draft release notes from changes | Low | Low | 7d |
| `adr-draft` | ADR Drafter | Draft Architecture Decision Records | Medium | Low | 7d |
| `td-review` | TD Review Session | Review open td reviews, fix obvious bugs, create tasks for bigger issues | High | Medium | 72h |

:::note
`td-review` is **disabled by default** and must be explicitly opted in via `tasks.enabled`. It requires the td integration to be enabled (see [Integrations](/docs/integrations)).
:::

## Analysis Tasks — "Here's what I found"

Completed analysis with conclusions. These tasks produce reports without modifying code.

| Task | Name | Description | Cost | Risk | Cooldown |
|------|------|-------------|------|------|----------|
| `doc-drift` | Doc Drift Detector | Detect documentation that's out of sync with code | Medium | Low | 72h |
| `semantic-diff` | Semantic Diff Explainer | Explain the semantic meaning of code changes | Medium | Low | 72h |
| `dead-code` | Dead Code Detector | Find unused code that can be removed | Medium | Low | 72h |
| `dependency-risk` | Dependency Risk Scanner | Analyze dependencies for security and maintenance risks | Medium | Low | 72h |
| `test-gap` | Test Gap Finder | Identify areas lacking test coverage | Medium | Low | 72h |
| `test-flakiness` | Test Flakiness Analyzer | Identify and analyze flaky tests | Medium | Low | 72h |
| `logging-audit` | Logging Quality Auditor | Audit logging for completeness and quality | Medium | Low | 72h |
| `metrics-coverage` | Metrics Coverage Analyzer | Analyze metrics instrumentation coverage | Medium | Low | 72h |
| `perf-regression` | Performance Regression Spotter | Identify potential performance regressions | Medium | Low | 72h |
| `cost-attribution` | Cost Attribution Estimator | Estimate resource costs by component | Medium | Low | 72h |
| `security-footgun` | Security Foot-Gun Finder | Find common security anti-patterns | Medium | Low | 72h |
| `pii-scanner` | PII Exposure Scanner | Scan for potential PII exposure | Medium | Low | 72h |
| `privacy-policy` | Privacy Policy Consistency Checker | Check code against privacy policy claims | Medium | Low | 72h |
| `schema-evolution` | Schema Evolution Advisor | Analyze database schema changes | Medium | Low | 72h |
| `event-taxonomy` | Event Taxonomy Normalizer | Normalize event naming and structure | Medium | Low | 72h |
| `roadmap-entropy` | Roadmap Entropy Detector | Detect roadmap scope creep and drift | Medium | Low | 72h |
| `bus-factor` | Bus-Factor Analyzer | Analyze code ownership concentration | Medium | Low | 72h |
| `knowledge-silo` | Knowledge Silo Detector | Identify knowledge silos in the team | Medium | Low | 72h |

## Options Tasks — "Here are options"

These tasks surface judgment calls, tradeoffs, and design forks for human review.

| Task | Name | Description | Cost | Risk | Cooldown |
|------|------|-------------|------|------|----------|
| `task-groomer` | Task Groomer | Refine and clarify task definitions | Medium | Low | 7d |
| `guide-improver` | Guide/Skill Improver | Suggest improvements to guides and skills | Medium | Low | 7d |
| `idea-generator` | Idea Generator | Generate improvement ideas for the codebase | Medium | Low | 7d |
| `tech-debt-classify` | Tech-Debt Classifier | Classify and prioritize technical debt | Medium | Low | 7d |
| `why-annotator` | Why Does This Exist Annotator | Document the purpose of unclear code | Medium | Low | 7d |
| `edge-case-enum` | Edge-Case Enumerator | Enumerate potential edge cases | Medium | Low | 7d |
| `error-msg-improve` | Error-Message Improver | Suggest better error messages | Medium | Low | 7d |
| `slo-suggester` | SLO/SLA Candidate Suggester | Suggest SLO/SLA candidates | Medium | Low | 7d |
| `ux-copy-sharpener` | UX Copy Sharpener | Improve user-facing text | Medium | Low | 7d |
| `a11y-lint` | Accessibility Linting | Non-checkbox accessibility analysis | Medium | Low | 7d |
| `service-advisor` | Should This Be a Service Advisor | Analyze service boundary decisions | High | Medium | 7d |
| `ownership-boundary` | Ownership Boundary Suggester | Suggest code ownership boundaries | Medium | Low | 7d |
| `oncall-estimator` | Oncall Load Estimator | Estimate oncall load from code changes | Medium | Low | 7d |

## Safe Tasks — "I tried it safely"

Required execution or simulation but left no lasting side effects.

| Task | Name | Description | Cost | Risk | Cooldown |
|------|------|-------------|------|------|----------|
| `migration-rehearsal` | Migration Rehearsal Runner | Rehearse migrations without side effects | Very High | High | 14d |
| `contract-fuzzer` | Integration Contract Fuzzer | Fuzz test integration contracts | Very High | High | 14d |
| `golden-path` | Golden-Path Recorder | Record golden path test scenarios | High | Medium | 14d |
| `perf-profile` | Performance Profiling Runs | Run performance profiling | High | Medium | 14d |
| `allocation-profile` | Allocation/Hot-Path Profiling | Profile memory allocation and hot paths | High | Medium | 14d |

## Map Tasks — "Here's the map"

Pure context laid out cleanly. These tasks document and visualize system structure.

| Task | Name | Description | Cost | Risk | Cooldown |
|------|------|-------------|------|------|----------|
| `visibility-instrument` | Visibility Instrumentor | Instrument code for observability | High | Medium | 7d |
| `repo-topology` | Repo Topology Visualizer | Visualize repository structure | Medium | Low | 7d |
| `permissions-mapper` | Permissions/Auth Surface Mapper | Map permissions and auth surfaces | Medium | Low | 7d |
| `data-lifecycle` | Data Lifecycle Tracer | Trace data lifecycle through the system | Medium | Low | 7d |
| `feature-flag-monitor` | Feature Flag Lifecycle Monitor | Monitor feature flag usage and lifecycle | Medium | Low | 7d |
| `ci-signal-noise` | CI Signal-to-Noise Scorer | Score CI signal vs noise ratio | Medium | Low | 7d |
| `historical-context` | Historical Context Summarizer | Summarize historical context of code | Medium | Low | 7d |

## Emergency Tasks — "For when things go sideways"

Artifacts you hope to never need. These tasks prepare for incident response.

| Task | Name | Description | Cost | Risk | Cooldown |
|------|------|-------------|------|------|----------|
| `runbook-gen` | Runbook Generator | Generate operational runbooks | High | Medium | 30d |
| `rollback-plan` | Rollback Plan Generator | Generate rollback plans for changes | High | Medium | 30d |
| `postmortem-gen` | Incident Postmortem Draft Generator | Draft incident postmortem documents | High | Medium | 30d |

## Cost Tiers

| Tier | Token Range | Task Count |
|------|-------------|------------|
| Low | 10–50k tokens | 6 |
| Medium | 50–150k tokens | 37 |
| High | 150–500k tokens | 14 |
| Very High | 500k+ tokens | 2 |

## Risk Levels

| Level | Description | Task Count |
|-------|-------------|------------|
| Low | Safe to run autonomously | 43 |
| Medium | May modify code; review recommended | 14 |
| High | Significant execution; careful review required | 2 |
