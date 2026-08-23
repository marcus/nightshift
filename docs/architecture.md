# Architecture

This document maps Nightshift's Go packages and shows how a run flows through
them. For the step-by-step run lifecycle (including where logs and reports are
written), see [guides/run-lifecycle.md](guides/run-lifecycle.md).

## Overview

Nightshift is a CLI + daemon that runs AI coding agents (Claude Code, Codex,
Copilot) on maintenance tasks across your projects, on a schedule, within
token budgets. The codebase is a single Go module:

```
cmd/                     executable entry points
  nightshift/            the main CLI binary
    commands/            cobra command definitions (one file per command)
  provider-calibration/  standalone calibration analysis tool
internal/                all library packages (not importable externally)
website/                 documentation site (Docusaurus)
docs/                    long-form guides and reference docs
```

## Command Layer

| Package | Purpose |
|---------|---------|
| `cmd/nightshift` | CLI entry point: wires config, logging, and database into the commands. |
| `cmd/nightshift/commands` | Cobra command definitions — `run`, `preview`, `task`, `budget`, `commit`, `busfactor`, `config`, `init`, `doctor`, `daemon`, `install`/`uninstall`, `logs`, `report`, `status`, `stats`, `setup`. Thin layer: parses flags, then delegates to `internal/*`. |
| `cmd/provider-calibration` | Offline tool that summarizes per-provider token usage distributions from local session data; informs budget calibration heuristics. See [guides/provider-calibration.md](guides/provider-calibration.md). |

## Core Runtime

| Package | Purpose |
|---------|---------|
| `internal/config` | Loads, merges (global + project), and validates configuration. |
| `internal/scheduler` | Time-based job scheduling (cron or interval) used by the daemon. |
| `internal/orchestrator` | Coordinates agents working on tasks — the plan → implement → review loop. |
| `internal/agents` | Interfaces and implementations for spawning AI agent processes. |
| `internal/providers` | Provider abstraction and selection (Claude, Codex, Copilot): preference order, availability, and per-provider settings. |
| `internal/tasks` | Task registry, selection, priority scoring, and cooldown intervals. |
| `internal/projects` | Multi-project discovery, resolution, and per-project budget allocation. |
| `internal/tmux` | Scrapes tmux sessions to detect running agent processes and their usage output. |
| `internal/logging` | Structured logging with file rotation. |

## Budget & Usage Data

| Package | Purpose |
|---------|---------|
| `internal/budget` | Token budget calculation, allowance, and enforcement. |
| `internal/snapshots` | Collects and stores periodic usage snapshots from provider data files. |
| `internal/calibrator` | Tunes task budgets and scheduling from historical usage. |
| `internal/trends` | Analyzes snapshot history to surface usage patterns and anomalies. |
| `internal/stats` | Aggregate statistics over past runs (backs `nightshift stats`). |

## Persistence & Output

| Package | Purpose |
|---------|---------|
| `internal/db` | SQLite-backed storage for state and snapshots. |
| `internal/state` | Persistent run state (what ran, when, cooldowns). |
| `internal/reporting` | Run reports and morning summaries (backs `nightshift report`). |
| `internal/security` | Audit logging for nightshift operations. |
| `internal/setup` | Interactive onboarding (backs `nightshift setup`). |
| `internal/integrations` | Readers for external configuration and task sources. |

## Analysis & Tooling

| Package | Purpose |
|---------|---------|
| `internal/analysis` | Code ownership and bus-factor analysis (backs `nightshift busfactor`). |
| `internal/commits` | Conventional Commits normalization (backs `nightshift commit normalize`). |

## How a Run Flows

1. **Trigger** — cron/launchd/systemd executes `nightshift run`, or the daemon
   (`internal/scheduler`) fires a scheduled tick.
2. **Setup** — `cmd/nightshift` loads config (`internal/config`) and initializes
   logging (`internal/logging`), then opens the database (`internal/db`) and
   loads run state (`internal/state`).
3. **Budget & provider** — `internal/budget` calculates the remaining
   allowance from `internal/snapshots` data; `internal/providers` picks a
   provider by preference and budget.
4. **Selection** — `internal/projects` resolves target projects;
   `internal/tasks` scores and selects eligible tasks (respecting cooldowns).
5. **Execution** — `internal/orchestrator` drives the agent
   (`internal/agents`) through plan → implement → review, inside tmux
   (`internal/tmux`) when scraping is needed.
6. **Recording** — task and project results are written to the database;
   `internal/reporting` saves the run report and (optionally) the morning
   summary. `nightshift status`, `report`, and `stats` read this data back.

The sequence diagram in [guides/run-lifecycle.md](guides/run-lifecycle.md)
traces the same flow end to end.
