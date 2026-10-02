---
sidebar_position: 4
title: Configuration
---

# Configuration

Nightshift reads YAML config in this order: global config, then project config, then environment variables. Use `nightshift setup` for guided onboarding, or use `nightshift init` and `nightshift config` if you prefer to manage the files directly.

## Bootstrap Workflow

```bash
# Guided global setup
nightshift setup

# Create a project config in the current directory
nightshift init

# Create or overwrite the global config
nightshift init --global

# Inspect the merged config
nightshift config

# Validate the current files and merged result
nightshift config validate
```

## Config Locations

| Scope | Location |
|-------|----------|
| Global | `~/.config/nightshift/config.yaml` |
| Project | `nightshift.yaml` in the current repo |

`nightshift config set` writes to the project config when one exists, otherwise it falls back to the global config. Use `--global` to force global writes.

## Example Config

```yaml
schedule:
  cron: "0 2 * * *"
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/Los_Angeles"
  max_projects: 3
  max_tasks: 2

budget:
  mode: daily
  max_percent: 75
  reserve_percent: 5
  weekly_tokens: 700000
  billing_mode: subscription
  calibrate_enabled: true

providers:
  preference:
    - claude
    - codex
    - copilot
  claude:
    enabled: true
    data_path: "~/.claude"
    dangerously_skip_permissions: true
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: true
  copilot:
    enabled: true
    data_path: "~/.copilot"

projects:
  - path: ~/code/sidecar
    priority: 1
  - path: ~/code/nightshift
    priority: 2

tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - skill-groom
  priorities:
    lint-fix: 1
    docs-backfill: 2
  intervals:
    lint-fix: "24h"
    docs-backfill: "168h"

integrations:
  claude_md: true
  agents_md: true
  task_sources:
    - td:
        enabled: true
        teach_agent: true
    - github_issues: true

logging:
  level: info
  path: ~/.local/share/nightshift/logs
  format: json

reporting:
  morning_summary: true
```

## Schedule

| Field | Default | Description |
|-------|---------|-------------|
| `schedule.cron` | _unset_ | Cron expression for scheduled runs |
| `schedule.interval` | _unset_ | Duration string for interval-based runs |
| `schedule.window.start` | _unset_ | Earliest allowed start time |
| `schedule.window.end` | _unset_ | Latest allowed end time |
| `schedule.window.timezone` | _unset_ | Time zone used for the window |
| `schedule.max_projects` | `0` | Default max projects per `nightshift run` when no CLI flag is set |
| `schedule.max_tasks` | `0` | Default max tasks per project when no CLI flag is set |

`schedule.cron` and `schedule.interval` are mutually exclusive. If you omit both, `nightshift daemon start` will refuse to run until one is set.

## Budget

| Field | Default | Description |
|-------|---------|-------------|
| `budget.mode` | `daily` | `daily` or `weekly` budget math |
| `budget.max_percent` | `75` | Maximum percentage of the budget to use per run |
| `budget.aggressive_end_of_week` | `false` | Increase weekly usage near the end of the week |
| `budget.reserve_percent` | `5` | Amount of budget to keep in reserve |
| `budget.weekly_tokens` | `700000` | Weekly fallback budget |
| `budget.per_provider` | _unset_ | Per-provider token limits |
| `budget.billing_mode` | `subscription` | `subscription` or `api` |
| `budget.calibrate_enabled` | `true` | Enable usage calibration |
| `budget.snapshot_interval` | `30m` | Snapshot cadence |
| `budget.snapshot_retention_days` | `90` | How long to keep snapshots |
| `budget.week_start_day` | `monday` | Week boundary for weekly math |
| `budget.db_path` | `~/.local/share/nightshift/nightshift.db` | Override the SQLite database path |

## Providers

| Field | Default | Description |
|-------|---------|-------------|
| `providers.preference` | `["claude", "codex", "copilot"]` | Provider order when Nightshift picks an agent |
| `providers.claude.enabled` | `true` | Enable Claude Code execution |
| `providers.claude.data_path` | `~/.claude` | Claude data directory |
| `providers.claude.dangerously_skip_permissions` | `false` | Pass Claude's permissive flag |
| `providers.codex.enabled` | `true` | Enable Codex execution |
| `providers.codex.data_path` | `~/.codex` | Codex data directory |
| `providers.codex.dangerously_bypass_approvals_and_sandbox` | `false` | Pass Codex's permissive flag |
| `providers.copilot.enabled` | `true` | Enable Copilot execution |
| `providers.copilot.data_path` | `~/.copilot` | Copilot usage data directory |
| `providers.copilot.dangerously_skip_permissions` | `false` | Pass Copilot's permissive flag |

Nightshift prefers the first enabled provider in `providers.preference` that is available in `PATH` and still has budget remaining.

## Projects

| Field | Description |
|-------|-------------|
| `projects[].path` | Absolute or home-relative path to a repository |
| `projects[].priority` | Higher values are processed first |
| `projects[].tasks` | Task override list for that project |
| `projects[].config` | Optional per-project config filename |
| `projects[].pattern` | Glob pattern for discovery |
| `projects[].exclude` | Paths to skip when using a glob |

## Tasks

| Field | Description |
|-------|-------------|
| `tasks.enabled` | Explicitly enabled task types |
| `tasks.priorities` | Per-task priority overrides |
| `tasks.disabled` | Explicitly disabled task types |
| `tasks.intervals` | Per-task cooldown overrides, expressed as durations |
| `tasks.custom` | User-defined custom tasks |

Custom tasks require `type`, `name`, and `description`. Optional fields include `category`, `cost_tier`, `risk_level`, and `interval`.

## Integrations

| Field | Description |
|-------|-------------|
| `integrations.claude_md` | Read `CLAUDE.md`/`claude.md` for context |
| `integrations.agents_md` | Read `AGENTS.md`/`agents.md` for context |
| `integrations.task_sources` | External task sources such as td or GitHub issues |

The `task_sources` list accepts entries like:

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true
    - github_issues: true
```

See [Integrations](/docs/integrations) for the current task-source behavior.

## Logging

| Field | Default | Description |
|-------|---------|-------------|
| `logging.level` | `info` | `debug`, `info`, `warn`, or `error` |
| `logging.path` | `~/.local/share/nightshift/logs` | Log directory |
| `logging.format` | `json` | `json` or `text` |

## Reporting

| Field | Default | Description |
|-------|---------|-------------|
| `reporting.morning_summary` | `true` | Generate the morning summary |
| `reporting.email` | _unset_ | Optional email recipient |
| `reporting.slack_webhook` | _unset_ | Optional Slack webhook |

## Config Commands

```bash
nightshift config
nightshift config get providers.copilot.enabled
nightshift config set budget.max_percent 80
nightshift config set providers.copilot.enabled false --global
nightshift config validate
```

`nightshift config get` reads the merged config, while `set` writes back to the config file that matches your scope. `set` parses booleans, numbers, and strings; edit YAML directly for lists or nested maps.
