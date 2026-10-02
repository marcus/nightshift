# Architecture

High-level overview of the Nightshift codebase. Update this document
whenever top-level directories are added, removed, or significantly
refactored.

## Top-Level Layout

```
cmd/                  CLI entry points (main packages)
  nightshift/         Primary user-facing binary
  provider-calibration/  Offline tool that compares Claude/Codex usage
internal/             Application code (not importable externally)
docs/                 Reference docs and migration notes
scripts/              Local developer scripts (pre-commit hook, etc.)
website/              Docusaurus site published as nightshift.haplab.com
.claude/              Project-local Claude skills and agents
```

## Runtime Flow

```
nightshift <cmd>
    │
    ▼
cmd/nightshift/commands  (Cobra command tree)
    │
    ▼
internal/config          ── loads ~/.config/nightshift/config.yaml
internal/projects        ── resolves target repos
internal/budget          ── checks remaining daily allotment
internal/scheduler       ── decides whether a run should fire now
internal/tasks           ── scores and selects tasks per project
internal/orchestrator    ── drives providers, streams output
internal/providers       ── concrete backends (Claude, Codex, Copilot)
internal/snapshots       ── persists usage telemetry
internal/state / db      ── SQLite-backed durable state
internal/reporting       ── morning summary reports
internal/logging         ── structured logs with rotation
```

## Package Responsibilities

| Package | Responsibility |
|---------|----------------|
| `internal/agents` | Interfaces and implementations for spawning AI agents. |
| `internal/analysis` | Code ownership and bus-factor analysis. |
| `internal/budget` | Token budget calculation and allocation. |
| `internal/calibrator` | Tune task budgets and scheduling from history. |
| `internal/config` | Load and validate the nightshift config. |
| `internal/db` | SQLite-backed storage for state and snapshots. |
| `internal/integrations` | Readers for external config and task sources. |
| `internal/logging` | Structured logging with file rotation. |
| `internal/orchestrator` | Coordinate AI agents across tasks. |
| `internal/projects` | Multi-project discovery, resolution, budget split. |
| `internal/providers` | Interfaces and implementations for coding agents. |
| `internal/reporting` | Generate morning summary reports. |
| `internal/scheduler` | Time-based job scheduling. |
| `internal/security` | Audit logging for operations. |
| `internal/setup` | Interactive configuration and preset selection. |
| `internal/snapshots` | Periodic usage data capture from providers. |
| `internal/state` | Persistent state for nightshift runs. |
| `internal/stats` | Aggregate statistics from run data. |
| `internal/tasks` | Task selection and priority scoring. |
| `internal/tmux` | Scrape tmux sessions to detect agent processes. |
| `internal/trends` | Analyze snapshots to surface patterns and anomalies. |

## Data Flow

1. **Config load** (`internal/config`) — parses
   `~/.config/nightshift/config.yaml` into strongly-typed structures.
2. **Project resolution** (`internal/projects`) — expands configured
   paths, verifies repos, and computes per-project budget shares.
3. **Budget gating** (`internal/budget` + `internal/snapshots`) — reads
   recent usage snapshots, compares against `budget.max_percent`, and
   decides whether to proceed.
4. **Task scoring** (`internal/tasks`) — applies priorities, cooldowns,
   and eligibility filters to rank candidate tasks per project.
5. **Orchestration** (`internal/orchestrator` + `internal/providers`) —
   spawns the chosen provider CLI for the top-ranked task, streams
   output, and enforces timeouts.
6. **Persistence** (`internal/state` + `internal/db`) — records runs,
   outcomes, and provider usage.
7. **Reporting** (`internal/reporting`) — assembles the morning
   summary for the operator.

## External Dependencies

- **SQLite** via `modernc.org/sqlite` — durable state, snapshots, and
  run history. Database file lives under
  `~/.local/share/nightshift/`.
- **Claude Code CLI** (`claude`) — invoked for the `claude` provider.
  Auth is whatever `claude /login` configured.
- **Codex CLI** (`codex`) — invoked for the `codex` provider. Auth is
  whatever `codex --login` configured.
- **GitHub Copilot CLI** (`@github/copilot`) — optional Copilot
  provider; see [COPILOT_INTEGRATION.md](COPILOT_INTEGRATION.md) if
  present.
- **tmux** — optional, used by `internal/tmux` to correlate running
  agent processes with their sessions.
- **gum** — optional, used to page preview output when available.

## Safety Model

Nightshift never writes directly to the primary branch. Every task
lands as its own branch (and, where `gh` auth allows, an open PR).
The preflight summary in `nightshift run` shows provider, budget,
projects, and planned tasks before work begins; in TTYs the user
confirms, in non-TTYs confirmation is auto-skipped.
