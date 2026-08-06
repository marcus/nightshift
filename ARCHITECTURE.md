# Architecture

This document describes the high-level structure of Nightshift: the module
layout, the command layer, and how the `internal/` packages group by
responsibility. It is intended as a map for contributors; it deliberately does
not duplicate the runtime walkthrough, which lives in
[docs/guides/run-lifecycle.md](docs/guides/run-lifecycle.md).

> **Accuracy note.** Package descriptions below are taken from each package's
> own `// Package ...` doc comment, and the dependency edges are taken from each
> package's imports. Statements about *how* components compose at runtime that
> are not directly supported by those two sources are marked **(assumption)**.

## Overview

Nightshift is a single Go module, `github.com/marcus/nightshift` (see `go.mod`,
Go 1.24). It is a CLI that orchestrates AI coding agents (Claude Code, Codex,
GitHub Copilot) to work on development tasks during off-hours, within a
configurable token budget. Runs are triggered on a schedule (cron / launchd /
systemd / the built-in daemon), execute a plan → implement → review loop through
an agent, and record their results as structured logs, SQLite state, and Markdown
reports.

Everything lands as a branch or pull request; the primary branch is never written
to directly.

## The command layer (`cmd/`)

There are two binaries:

| Path | Package | Role |
|------|---------|------|
| `cmd/nightshift/` | `main` | CLI entry point; calls `commands.Execute()`. |
| `cmd/nightshift/commands/` | `commands` | All cobra subcommands (`setup`, `run`, `preview`, `budget`, `task`, `doctor`, `status`, `logs`, `stats`, `daemon`, `config`, `report`, `snapshot`, `install`, `init`, `busfactor`, …). |
| `cmd/provider-calibration/` | `main` | Standalone diagnostic tool that aggregates token/session metrics from provider logs to suggest a budget-calibration multiplier. Not built into the main binary; run via `make calibrate-providers`. |

The `commands` package is the **composition root** for a run: `run.go` wires
together the `config`, `budget`, `calibrator`, `db`, `logging`, `orchestrator`,
`providers`, `reporting`, `state`, `tasks`, `trends`, and `agents` packages to
carry out a scheduled or manual run.

## Internal packages (`internal/`)

The packages under `internal/` are grouped below by responsibility. Each bullet's
description is the package's own doc comment; the import relationships are
verified from source.

### Configuration, state, and storage

- **`config`** — handles loading and validating nightshift configuration.
- **`state`** — manages persistent state for nightshift runs.
- **`db`** — provides SQLite-backed storage for nightshift state and snapshots.
  Used as the shared persistence layer: `tasks`, `projects`, `snapshots`,
  `trends`, `calibrator`, `stats`, and the run command all depend on it.

`config` is a foundational dependency — `budget`, `scheduler`, `tasks`,
`reporting`, `integrations`, `projects`, and `calibrator` all import it.

### Agents, providers, and integrations

- **`agents`** — provides interfaces and implementations for spawning AI agents.
- **`providers`** — defines interfaces and implementations for AI coding agents
  (Claude, Codex, Copilot). A near-leaf package (no `internal/` dependencies).
- **`integrations`** — provides readers for external configuration and task
  sources (e.g. `AGENTS.md`, `CLAUDE.md`, GitHub, td). Imports only `config`.

`agents` and `providers` are distinct: `providers` defines the agent backends and
interfaces, while `agents` provides the spawning machinery. **(assumption)** The
`orchestrator` drives `agents` to execute work through the chosen provider.

### Task selection and execution

- **`tasks`** — provides task selection and priority scoring. Imports `config`,
  `db`, `state`.
- **`scheduler`** — handles time-based job scheduling. Imports `config`.
- **`orchestrator`** — coordinates AI agents working on tasks. Imports `agents`,
  `budget`, `logging`, `tasks`.

### Budget and calibration

- **`budget`** — implements token budget calculation and allocation. Imports
  `config` and `providers`.
- **`calibrator`** — tunes task budgets and scheduling based on historical usage
  data. Imports `budget`, `config`, `db`.
- **`snapshots`** — collects and stores periodic usage data from AI providers.
  Imports `db` and `tmux`.
- **`trends`** — analyzes historical snapshot data to surface usage patterns and
  anomalies. Imports `db`.
- **`stats`** — computes aggregate statistics from nightshift run data. Imports
  `budget`, `db`, `reporting`.
- **`tmux`** — scrapes tmux sessions to detect running AI agent processes and
  their usage. A leaf package (no `internal/` dependencies).

These form a **usage feedback loop**: `tmux` scrapes live usage → `snapshots`
persists it to `db` → `trends` analyzes the history in `db` → `calibrator` reads
that history to retune `budget`. **(assumption)** The exact trigger and cadence
of calibration is governed by configuration (`budget.calibrate_enabled`,
`budget.snapshot_interval`).

### Security

- **`security`** — provides credential management for nightshift. A leaf package
  (no `internal/` dependencies).

### Observability and reporting

- **`logging`** — provides structured logging with file rotation.
- **`reporting`** — generates morning summary reports for nightshift runs.
  Imports `config` and `logging`.

### Analysis

- **`analysis`** — provides code ownership and bus-factor analysis tools (backing
  the `busfactor` command).

### Projects and setup

- **`projects`** — handles multi-project discovery, resolution, and budget
  allocation. Imports `config`, `db`, `state`.
- **`setup`** — provides interactive configuration and task preset selection for
  new projects (backing the guided `setup` command).

## Runtime lifecycle

The sequence from a scheduled trigger to a finished run — config/logging init,
SQLite state, budget allowance, provider selection, the orchestrator's
plan → implement → review loop, and report generation — is documented with a
sequence diagram in
[docs/guides/run-lifecycle.md](docs/guides/run-lifecycle.md). Refer there rather
than reproducing the flow here.

## Where output goes

(From [run-lifecycle.md](docs/guides/run-lifecycle.md).)

- **Structured logs**: `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log`
- **Run report**: `~/.local/share/nightshift/reports/run-YYYY-MM-DD-HHMMSS.md`
- **Daily summary** (when `reporting.morning_summary: true`):
  `~/.local/share/nightshift/summaries/summary-YYYY-MM-DD.md`
- **State**: SQLite under `~/.local/share/nightshift/` (managed by `db`)

## Open assumptions

To avoid fabrication, the following are explicitly assumptions rather than
verified facts, and should be confirmed against the code before being relied on:

- The `orchestrator` drives `agents` through the selected `providers` backend.
- The calibration feedback loop (`tmux` → `snapshots` → `trends`/`calibrator` →
  `budget`) is triggered and cadenced by `budget.calibrate_enabled` and
  `budget.snapshot_interval`.
- The `daemon` cobra command wraps `scheduler` to run tasks in the background
  (per the run-lifecycle guide, which mentions "the daemon scheduler calls
  `runScheduledTasks`").

If you verify any of these while working in the code, please update this file to
convert the assumption into a stated fact.
