---
sidebar_position: 4
title: Configuration
---

# Configuration

Nightshift loads configuration in this order:

1. Global config: `~/.config/nightshift/config.yaml`
2. Project config: `./nightshift.yaml`
3. Environment overrides

Use `nightshift setup` for the guided global setup, `nightshift init` to create starter files, and `nightshift config` to inspect or update values from the CLI.

## Configuration Workflow

```bash
# Guided setup for the global config
nightshift setup

# Create starter config files
nightshift init
nightshift init --global

# Inspect merged config and source paths
nightshift config

# Read values
nightshift config get budget.max_percent
nightshift config get providers.preference

# Update values
nightshift config set budget.max_percent 60
nightshift config set logging.level debug --global

# Validate both global and project config
nightshift config validate
```

Notes:

- `nightshift setup` writes the global config only.
- `nightshift init` creates `./nightshift.yaml` by default, or `~/.config/nightshift/config.yaml` with `--global`.
- `nightshift config set` writes to `./nightshift.yaml` if it exists in the current directory. Otherwise it writes to the global config.

## Minimal Global Config

```yaml
schedule:
  cron: "0 2 * * *"
  max_projects: 3
  max_tasks: 2

budget:
  mode: daily
  max_percent: 75
  reserve_percent: 5
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m

providers:
  preference:
    - claude
    - codex
    - copilot
  claude:
    enabled: true
    data_path: "~/.claude"
  codex:
    enabled: true
    data_path: "~/.codex"
  copilot:
    enabled: false
    data_path: "~/.copilot"

projects:
  - path: ~/code/nightshift
    priority: 3
  - path: ~/code/td
    priority: 2
```

## Project Config

Put `nightshift.yaml` in a repo root to override settings for that repo.

```yaml
tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - bug-finder
  priorities:
    lint-fix: 1
    docs-backfill: 2
  intervals:
    lint-fix: "24h"
    docs-backfill: "168h"

budget:
  max_percent: 50
```

Nightshift also supports project discovery from the global config:

```yaml
projects:
  - path: ~/code/nightshift
    priority: 3
  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archived
```

## Schedule

Use either `schedule.cron` or `schedule.interval`, not both.

| Key | Description |
|-----|-------------|
| `schedule.cron` | Cron expression for scheduled runs |
| `schedule.interval` | Duration-based schedule such as `8h` |
| `schedule.window.start` | Optional start time such as `22:00` |
| `schedule.window.end` | Optional end time such as `06:00` |
| `schedule.window.timezone` | IANA timezone such as `America/Los_Angeles` |
| `schedule.max_projects` | Default project cap for `nightshift run` when `--max-projects` is not set |
| `schedule.max_tasks` | Default per-project task cap when `--max-tasks` is not set |

Examples:

```yaml
schedule:
  cron: "0 2 * * *"
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/Los_Angeles"
```

```yaml
schedule:
  interval: "8h"
```

## Budget

| Key | Default | Description |
|-----|---------|-------------|
| `budget.mode` | `daily` | `daily` or `weekly` |
| `budget.max_percent` | `75` | Max percentage a single Nightshift run can use |
| `budget.reserve_percent` | `5` | Budget kept in reserve for daytime work |
| `budget.aggressive_end_of_week` | `false` | Spend more aggressively near the end of a weekly cycle |
| `budget.weekly_tokens` | `700000` | Fallback weekly budget estimate |
| `budget.per_provider` | unset | Per-provider token overrides |
| `budget.billing_mode` | `subscription` | `subscription` or `api` |
| `budget.calibrate_enabled` | `true` | Enable calibration snapshots |
| `budget.snapshot_interval` | `30m` | Snapshot cadence |
| `budget.snapshot_retention_days` | `90` | Snapshot retention window |
| `budget.week_start_day` | `monday` | Weekly reset boundary for calibration |
| `budget.db_path` | `~/.local/share/nightshift/nightshift.db` | Override the SQLite database path |

Subscription example:

```yaml
budget:
  mode: daily
  max_percent: 75
  reserve_percent: 5
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m
```

API billing example:

```yaml
budget:
  billing_mode: api
  weekly_tokens: 1000000
  per_provider:
    claude: 1000000
    codex: 500000
```

Useful commands:

```bash
nightshift budget
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
```

## Providers

Nightshift supports `claude`, `codex`, and `copilot`. Provider selection follows `providers.preference` and skips disabled providers.

| Key | Description |
|-----|-------------|
| `providers.preference` | Ordered provider preference list |
| `providers.<name>.enabled` | Enable or disable a provider |
| `providers.<name>.data_path` | Provider data directory used for usage tracking |
| `providers.claude.dangerously_skip_permissions` | Claude-specific unattended execution override |
| `providers.codex.dangerously_bypass_approvals_and_sandbox` | Codex-specific unattended execution override |
| `providers.copilot.dangerously_skip_permissions` | Copilot-specific unattended execution override |

Example:

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
  claude:
    enabled: true
    data_path: "~/.claude"
  codex:
    enabled: true
    data_path: "~/.codex"
  copilot:
    enabled: true
    data_path: "~/.copilot"
```

> There is no `--enable-writes` CLI flag in the current release. Nightshift always works through branches and PRs. Headless approval behavior is provider-specific, and the available config hooks live under `providers.*.dangerously_*`.

## Tasks

Use the `tasks` section to choose what Nightshift is allowed to run and how often.

```yaml
tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - bug-finder
  disabled:
    - td-review
  priorities:
    lint-fix: 1
    docs-backfill: 2
  intervals:
    lint-fix: "24h"
    docs-backfill: "168h"
  custom:
    - type: release-readiness
      name: "Release Readiness Review"
      description: |
        Review the repo for release blockers and summarize follow-up work.
      category: analysis
      cost_tier: medium
      risk_level: low
      interval: "72h"
```

Useful commands:

```bash
nightshift task list
nightshift task show docs-backfill
nightshift task run docs-backfill --provider claude --dry-run
```

## Integrations, Logging, and Reporting

| Key | Description |
|-----|-------------|
| `integrations.claude_md` | Read project-level `CLAUDE.md` instructions |
| `integrations.agents_md` | Read project-level `AGENTS.md` instructions |
| `integrations.task_sources` | External task sources such as `td` |
| `logging.level` | `debug`, `info`, `warn`, or `error` |
| `logging.path` | Directory for JSON or text logs |
| `logging.format` | `json` or `text` |
| `reporting.morning_summary` | Enable morning summary generation |
| `reporting.email` | Optional email destination |
| `reporting.slack_webhook` | Optional Slack webhook |

## Environment Overrides

Nightshift currently binds a small set of config keys directly from environment variables:

| Environment Variable | Config Key |
|----------------------|------------|
| `NIGHTSHIFT_BUDGET_MAX_PERCENT` | `budget.max_percent` |
| `NIGHTSHIFT_BUDGET_MODE` | `budget.mode` |
| `NIGHTSHIFT_LOG_LEVEL` | `logging.level` |
| `NIGHTSHIFT_LOG_PATH` | `logging.path` |

## File Locations

| Type | Location |
|------|----------|
| Global config | `~/.config/nightshift/config.yaml` |
| Project config | `./nightshift.yaml` |
| Logs | `~/.local/share/nightshift/logs/` |
| Run reports | `~/.local/share/nightshift/reports/` |
| Morning summaries | `~/.local/share/nightshift/summaries/` |
| Database | `~/.local/share/nightshift/nightshift.db` |
| PID file | `~/.local/share/nightshift/nightshift.pid` |
