---
sidebar_position: 4
title: Configuration
---

# Configuration

Nightshift uses YAML config files. Use `nightshift setup` for the guided global wizard, `nightshift init` to generate starter files, or edit YAML directly.

## Config Layering

Load order is:

1. Built-in defaults
2. Global config: `~/.config/nightshift/config.yaml`
3. Project config: `nightshift.yaml`
4. Environment overrides

Project config overrides the global config for that repo.

## Bootstrap Config

```bash
nightshift setup
nightshift init --global
nightshift init
```

Inspect and validate config from the CLI:

```bash
nightshift config
nightshift config get providers.preference
nightshift config set budget.max_percent 60 --global
nightshift config validate
```

`nightshift config set` is best for scalar values. Arrays and larger nested structures are usually easier to edit directly in YAML.

## Minimal Global Config

```yaml
schedule:
  cron: "0 2 * * *"
  max_projects: 1
  max_tasks: 1

budget:
  mode: daily
  max_percent: 75
  weekly_tokens: 700000
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
    dangerously_skip_permissions: false
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: false
  copilot:
    enabled: false
    data_path: "~/.copilot"
    dangerously_skip_permissions: false

projects:
  - path: ~/code/sidecar
  - path: ~/code/td

integrations:
  claude_md: true
  agents_md: true
  task_sources:
    - td:
        enabled: true
        teach_agent: true
```

## Schedule

Use cron syntax or interval-based scheduling:

```yaml
schedule:
  cron: "0 2 * * *"        # Every night at 2am
  # interval: "8h"         # Or run every 8 hours
  # max_projects: 2        # Default project cap for scheduled runs
  # max_tasks: 1           # Default task cap per project
```

## Budget

Control how much of your token budget Nightshift uses:

| Field | Default | Description |
|-------|---------|-------------|
| `mode` | `daily` | `daily` or `weekly` |
| `max_percent` | `75` | Max budget % to use per run |
| `reserve_percent` | `5` | Always keep this % available |
| `weekly_tokens` | `700000` | Fallback weekly budget when provider data is incomplete |
| `per_provider` | unset | Override budget per provider, for example `copilot: 500` |
| `billing_mode` | `subscription` | `subscription` or `api` |
| `calibrate_enabled` | `true` | Auto-calibrate from local CLI data |
| `snapshot_interval` | `30m` | How often Nightshift captures budget snapshots |
| `snapshot_retention_days` | `90` | How long snapshots are retained |
| `week_start_day` | `monday` | Week anchor for weekly budget calculations |
| `db_path` | platform default | Override the SQLite database path |

Example:

```yaml
budget:
  mode: weekly
  max_percent: 60
  reserve_percent: 10
  weekly_tokens: 700000
  per_provider:
    claude: 700000
    codex: 500000
    copilot: 300
```

## Providers

Nightshift supports `claude`, `codex`, and `copilot`.

| Key | Description |
|-----|-------------|
| `providers.preference` | Ordered provider preference used for automatic selection |
| `providers.PROVIDER.enabled` | Enable or disable a provider |
| `providers.PROVIDER.data_path` | Override the local data directory used for usage tracking |
| `providers.claude.dangerously_skip_permissions` | Allow Claude to bypass permission prompts |
| `providers.codex.dangerously_bypass_approvals_and_sandbox` | Allow headless Codex execution without approvals |
| `providers.copilot.dangerously_skip_permissions` | Allow Copilot to use the broad permission flags |

Example:

```yaml
providers:
  preference:
    - codex
    - claude
    - copilot
  claude:
    enabled: true
    data_path: "~/.claude"
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: true
  copilot:
    enabled: true
    data_path: "~/.copilot"
```

## Task Selection

Enable or disable tasks and set priorities:

```yaml
tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - bug-finder
    - skill-groom
  priorities:
    lint-fix: 1
    bug-finder: 2
  intervals:
    lint-fix: "24h"
    docs-backfill: "168h"
```

Each task has a default cooldown interval to prevent the same task from running too frequently on a project.

## Projects

The global config can manage multiple repos:

```yaml
projects:
  - path: ~/code/project1
    priority: 2
  - path: ~/code/project2
    priority: 1
  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archive
```

Project-local overrides belong in each repo's `nightshift.yaml`.

## Integrations

```yaml
integrations:
  claude_md: true
  agents_md: true
  task_sources:
    - td:
        enabled: true
        teach_agent: true
    - github_issues: true
```

Use `claude_md` and `agents_md` to ingest repo instructions automatically. Use `task_sources` to pull work from `td` or GitHub issues.

## Logging And Reporting

```yaml
logging:
  level: info
  path: ~/.local/share/nightshift/logs
  format: json

reporting:
  morning_summary: true
```

## Environment Overrides

Nightshift binds a few common overrides directly from the environment:

- `NIGHTSHIFT_BUDGET_MAX_PERCENT`
- `NIGHTSHIFT_BUDGET_MODE`
- `NIGHTSHIFT_LOG_LEVEL`
- `NIGHTSHIFT_LOG_PATH`

## File Locations

| Type | Location |
|------|----------|
| Run logs | `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log` |
| Audit logs | `~/.local/share/nightshift/audit/audit-YYYY-MM-DD.jsonl` |
| Summaries | `~/.local/share/nightshift/summaries/` |
| Database | `~/.local/share/nightshift/nightshift.db` |
| PID file | `~/.local/share/nightshift/nightshift.pid` |

If `state/state.json` exists from older versions, Nightshift migrates it to the SQLite database and renames the file to `state.json.migrated`.
