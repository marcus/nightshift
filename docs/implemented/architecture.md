# Architecture

This document describes how Nightshift is structured at the package level and
how a run flows from a CLI command through scheduling, orchestration, and a
provider to a generated branch or PR. It reflects the shipped codebase
(currently the `v0.3.x` line); the descriptions below are grounded in the
`// Package` doc comments and the public symbols of each package.

Nightshift is a single Go binary (`cmd/nightshift`) that runs AI coding agents
overnight to perform maintenance tasks — lint fixes, documentation backfill,
bug finding, changelog synthesis, and the rest of the built-in task catalog.
Everything it produces lands on a branch or PR; it never writes to the primary
branch.

## High-level flow

```
nightshift CLI  ──►  config            ──►  scheduler   ──►  orchestrator
(cmd/nightshift)     (internal/config)      (cron/now)       (internal/orchestrator)
                                                                     │
                                                                     ▼
                            ┌────────────────────────────────────────┴────────┐
                            │                                                  │
                        projects                                         task selector
                  (internal/projects)                                  (internal/tasks)
                            │                                                  │
                            └──────────────┬───────────────────────────────────┘
                                           ▼
                                     budget guard
                                  (internal/budget)
                                           │
                                           ▼
                                  provider / agent
                          (internal/providers, internal/agents)
                                           │
                                           ▼
                          run agent in a tmux session (internal/tmux)
                                           │
                                           ▼
                          collect result, commit, open/annotate PR
                                  (internal/orchestrator)
                                           │
                           ┌───────────────┴───────────────┐
                           ▼                               ▼
                       state / db                      reporting / stats
                  (internal/state, db)          (internal/reporting, stats)
```

1. **CLI** (`cmd/nightshift`, commands in `cmd/nightshift/commands`) parses the
   command and flags with Cobra. `run`, `task run`, and the scheduler all
   converge on the orchestrator.
2. **Config** is loaded from the global file
   (`~/.config/nightshift/config.yaml`) and any project-local `nightshift.yaml`,
   then validated.
3. **Security** checks run up front: `security.ValidateProjectPath` rejects
   sensitive directories, and credentials are resolved through
   `internal/security`.
4. **Scheduler** (`internal/scheduler`) triggers runs on the configured cron
   schedule; `nightshift run` / `task run` invoke the same path immediately.
5. **Orchestrator** (`internal/orchestrator`) discovers projects, selects
   eligible tasks (honoring per-task cooldowns), enforces the token budget,
   runs the chosen provider's agent, then commits the result and opens or
   annotates a PR.
6. **Providers/agents** (`internal/providers`, `internal/agents`) spawn the
   underlying AI CLI (Claude Code, Codex, or GitHub Copilot) and capture its
   usage.
7. **State, snapshots, reporting** persist run history and token usage and
   produce the morning summary.

## Package map

### Command entry points

| Package | Responsibility |
|---|---|
| `cmd/nightshift` | Binary entry point; delegates to the command package. |
| `cmd/nightshift/commands` | The Cobra CLI surface: `setup`, `run`, `preview`, `budget`, `task`, `status`, `doctor`, `daemon`, `report`, `stats`, `logs`, `snapshot`, `init`, `install`, `uninstall`, `busfactor`, `config`, etc. |
| `cmd/provider-calibration` | Standalone helper that calibrates provider usage/budget models from observed data (wired into the `calibrate-providers` Make target and `internal/calibrator`). |

### Core runtime

| Package | Responsibility |
|---|---|
| `internal/config` | Loads and validates configuration from the global path (`~/.config/nightshift/config.yaml`) and project-local `nightshift.yaml`; resolves provider preferences, budgets, projects, and task settings. |
| `internal/scheduler` | Time-based job scheduling on the configured cron expression. |
| `internal/orchestrator` | Coordinates a run: planning, agent execution, result capture, git commit, and PR creation/annotation (`RunTask`, `plan`, `annotatePR`). This is where "everything is a PR" is realized. |
| `internal/projects` | Multi-project discovery, resolution, and per-project budget allocation. |
| `internal/tasks` | The task catalog and registry, plus selection: priority scoring, cost/risk tiers, and cooldown filtering (`FilterByCooldown`). Defines all built-in task types such as `lint-fix`, `docs-backfill`, `bug-finder`, and `changelog-synth`. |

### Providers and agents

| Package | Responsibility |
|---|---|
| `internal/providers` | Interfaces and implementations for the AI coding agents — Claude Code, Codex, and GitHub Copilot — including usage queries (`GetTodayUsage`, `GetWeeklyUsage`). |
| `internal/agents` | Interfaces and implementations for spawning AI agents and capturing their results. |
| `internal/tmux` | Scrapes tmux sessions to detect running agent processes and their live usage, so Nightshift can observe long-running agent runs. |

### Budget and tuning

| Package | Responsibility |
|---|---|
| `internal/budget` | Token budget calculation and allocation; enforces the daily/weekly allotment and the configurable max (default 75% of remaining quota). |
| `internal/calibrator` | Tunes task budgets and scheduling from historical usage data. |
| `internal/snapshots` | Collects and stores periodic usage snapshots from providers (default every 30m). |
| `internal/trends` | Analyzes historical snapshot data to surface usage patterns and anomalies. |

### Safety and observability

| Package | Responsibility |
|---|---|
| `internal/security` | Credential management **and** the safety guards that keep agents from running in dangerous contexts: `ValidateProjectPath` (blocks `/`, `/tmp`, `/var`, `/etc`, `/usr`, and `$HOME`), `ValidateGitPush` (gates pushes behind `--allow-push`), and credential-format validation. |
| `internal/logging` | Structured logging with file rotation. |
| `internal/state` | Persistent state for runs (what ran, when, outcome). |
| `internal/db` | SQLite-backed storage for run state and snapshots. |
| `internal/stats` | Computes aggregate statistics from run data. |
| `internal/reporting` | Generates the morning summary report of what Nightshift did. |
| `internal/analysis` | Code-ownership and bus-factor analysis tooling (powers the `busfactor` command). |

### Onboarding and integration

| Package | Responsibility |
|---|---|
| `internal/setup` | Interactive onboarding wizard: provider configuration, project selection, budget calibration, and task presets. |
| `internal/integrations` | Readers for external configuration and task sources (e.g. GitHub issues). |

## Cross-cutting concerns

**The task registry.** Every built-in task is a `TaskDefinition` in
`internal/tasks/tasks.go` with a type, category (`pr`, `analysis`, `options`,
`safe`, `map`, `emergency`), cost tier, risk level, and a default re-run
interval. A completeness test asserts the registry stays in sync with its
constants. See [`docs/guides/adding-tasks.md`](../guides/adding-tasks.md) to
add one.

**The budget loop.** `internal/budget` decides whether a run may proceed based
on remaining quota; `internal/snapshots` feeds it current usage, and
`internal/calibrator` + `internal/trends` adjust budgets over time. `run`
shows a preflight summary of the selected provider, budget status, projects,
and planned tasks before executing.

**Security boundaries.** Because agents run with elevated permission flags
(`dangerously_skip_permissions` / `dangerously_bypass_approvals_and_sandbox`,
which default to `false`), `internal/security` is load-bearing: it blocks
sensitive root/home directories, gates git pushes, and manages credentials.
Any change that resolves a project path or pushes must go through these guards.

**Everything is a PR.** The orchestrator writes results to a branch, commits
with a traceable message, and opens or annotates a PR rather than touching the
primary branch. Closing the PR is the entire rollback path.

## Further reading

- [`docs/guides/adding-tasks.md`](../guides/adding-tasks.md) — adding a built-in task
- [`docs/guides/run-lifecycle.md`](../guides/run-lifecycle.md) — the run lifecycle
- [`docs/guides/provider-calibration.md`](../guides/provider-calibration.md) — calibrating providers
- [`docs/guides/codex-budget-tracking.md`](../guides/codex-budget-tracking.md) — budget tracking
- [`docs/implemented/initial-spec.md`](initial-spec.md) — original design spec
