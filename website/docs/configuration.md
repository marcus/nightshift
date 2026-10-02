---
sidebar_position: 4
title: Configuration
---

# Configuration

Nightshift uses YAML config files. Use `nightshift setup` for guided bootstrap, `nightshift init` or `nightshift init --global` to create a config file, and `nightshift config` to inspect or edit the merged view.

## Bootstrap Workflow

```bash
nightshift setup
nightshift init
nightshift init --global
nightshift config
nightshift config get budget.max_percent
nightshift config set budget.max_percent 15
nightshift config set --global logging.level debug
nightshift config validate
```

- `nightshift setup` walks through provider setup, projects, budget, schedule, PATH, and daemon installation.
- `nightshift init` creates `nightshift.yaml` in the current directory.
- `nightshift init --global` creates `~/.config/nightshift/config.yaml`.
- `nightshift config` shows the merged config plus the source paths.
- `nightshift config get` reads a nested value by key path.
- `nightshift config set` writes to the project config when one exists, otherwise to the global config. Use `--global` to force the global file.
- `nightshift config validate` checks the global file, project file, and merged config.

`nightshift config set` accepts booleans, integers, floats, and strings. For example, `true`, `15`, `12.5`, and `debug` are all parsed correctly.

## Config Sources

Nightshift reads config in this order:

1. Global config: `~/.config/nightshift/config.yaml`
2. Project config: `nightshift.yaml` in the current project directory
3. Environment overrides such as `NIGHTSHIFT_BUDGET_MAX_PERCENT`

Project config values override global config values, and environment variables override both. `nightshift config` reflects that same merge order when it prints the current configuration.

## Config Locations

| Type | Location |
|------|----------|
| Global | `~/.config/nightshift/config.yaml` |
| Project | `nightshift.yaml` |

## Minimal Config

```yaml
schedule:
  cron: "0 2 * * *"
  max_projects: 1
  max_tasks: 1

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
    dangerously_skip_permissions: true
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: true
  copilot:
    enabled: true
    data_path: "~/.copilot"
    dangerously_skip_permissions: false

projects:
  - path: ~/code/sidecar
  - path: ~/code/td
```

## Schedule

Use either cron or interval scheduling. Nightshift rejects configs that set both.

```yaml
schedule:
  cron: "0 2 * * *"        # Every night at 2am
  # interval: "8h"         # Or run every 8 hours
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/Denver"
  max_projects: 1
  max_tasks: 1
```

| Field | Default | Description |
|-------|---------|-------------|
| `cron` | - | Cron expression for scheduled runs |
| `interval` | - | Duration string for repeated runs |
| `window.start` | `22:00` | Start of the allowed execution window |
| `window.end` | `06:00` | End of the allowed execution window |
| `window.timezone` | local time | Time zone for the window |
| `max_projects` | `1` | Default max projects per run |
| `max_tasks` | `1` | Default max tasks per project |

## Budget

Control how much of your token budget Nightshift uses.

| Field | Default | Description |
|-------|---------|-------------|
| `mode` | `daily` | `daily` or `weekly` |
| `max_percent` | `75` | Max budget % to use per run |
| `reserve_percent` | `5` | Always keep this % available |
| `billing_mode` | `subscription` | `subscription` or `api` |
| `calibrate_enabled` | `true` | Enable subscription calibration via snapshots |
| `snapshot_interval` | `30m` | Automatic snapshot cadence |
| `snapshot_retention_days` | `90` | Snapshot retention window |
| `weekly_tokens` | `700000` | Fallback weekly budget |
| `per_provider` | - | Provider-specific weekly budgets |
| `week_start_day` | `monday` | Week boundary for calibration |
| `db_path` | `~/.local/share/nightshift/nightshift.db` | Override database path |
| `aggressive_end_of_week` | `false` | Spend more near the end of the week |

If `billing_mode: api`, Nightshift uses the explicit token budgets in `weekly_tokens` and `per_provider` instead of calibration.

## Providers

Nightshift supports Claude Code, Codex, and GitHub Copilot. It uses the providers listed in `providers.preference` order.

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
  copilot:
    enabled: true
    data_path: "~/.copilot"
```

| Field | Default | Description |
|-------|---------|-------------|
| `providers.preference` | `["claude", "codex", "copilot"]` | Provider priority order |
| `providers.claude.enabled` | `true` | Enable Claude provider |
| `providers.claude.data_path` | `~/.claude` | Claude Code data directory |
| `providers.claude.dangerously_skip_permissions` | `false` | Skip Claude permission prompts |
| `providers.codex.enabled` | `true` | Enable Codex provider |
| `providers.codex.data_path` | `~/.codex` | Codex data directory |
| `providers.codex.dangerously_bypass_approvals_and_sandbox` | `false` | Bypass Codex approvals and sandboxing |
| `providers.copilot.enabled` | `true` | Enable Copilot provider |
| `providers.copilot.data_path` | `~/.copilot` | Copilot request-tracking directory |
| `providers.copilot.dangerously_skip_permissions` | `false` | Allow Copilot to run with broader tool access |

Copilot tracks request counts, not token usage. Its budget view is an estimate based on monthly request limits.

## Task Selection

Enable and prioritize built-in tasks, disable specific tasks, or define custom tasks.

```yaml
tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - bug-finder
  priorities:
    lint-fix: 1
    bug-finder: 2
  intervals:
    lint-fix: "24h"
    docs-backfill: "168h"
  custom:
    - type: pr-review
      name: "PR Review Session"
      description: |
        Review open PRs and check for regressions.
        Create follow-up tasks for anything that needs attention.
      category: pr
      cost_tier: high
      risk_level: medium
      interval: "72h"
```

- `tasks.enabled` restricts the built-in tasks Nightshift may run.
- `tasks.disabled` explicitly blocks a task even if it is enabled elsewhere.
- `tasks.intervals` overrides cooldowns per task.
- `tasks.custom` defines user-authored tasks. `type`, `name`, and `description` are required.

## Integrations

```yaml
integrations:
  claude_md: true
  agents_md: true
  task_sources:
    - td:
        enabled: true
        teach_agent: true
```

| Field | Default | Description |
|-------|---------|-------------|
| `integrations.claude_md` | `true` | Read `CLAUDE.md` or `claude.md` for context |
| `integrations.agents_md` | `true` | Read `AGENTS.md` for context |
| `integrations.task_sources` | - | External task sources like `td` or GitHub issues |

## Multi-Project Setup

```yaml
projects:
  - path: ~/code/project1
    priority: 1                # Higher priority = processed first
    tasks:
      - lint-fix
      - docs-backfill
  - path: ~/code/project2
    priority: 2
  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archived
```

Each project can point at a path or a glob pattern. Use `exclude` to skip directories that match the pattern.

## Safe Defaults

| Feature | Default | Override |
|---------|---------|----------|
| Confirmation prompt in TTY | Yes | `--yes` |
| Confirmation prompt in non-TTY | Auto-skip | `--yes` or interactive terminal |
| Max projects per run | `1` | `--max-projects` or `schedule.max_projects` |
| Max tasks per project | `1` | `--max-tasks` or `schedule.max_tasks` |
| Max budget per run | `75%` | `budget.max_percent` |
| Reserve budget | `5%` | `budget.reserve_percent` |

## File Locations

| Type | Location |
|------|----------|
| Run logs | `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log` |
| Audit logs | `~/.local/share/nightshift/audit/audit-YYYY-MM-DD.jsonl` |
| Summaries | `~/.local/share/nightshift/summaries/` |
| Database | `~/.local/share/nightshift/nightshift.db` |
| PID file | `~/.local/share/nightshift/nightshift.pid` |

If `state/state.json` exists from older versions, Nightshift migrates it to the SQLite database and renames the file to `state.json.migrated`.
