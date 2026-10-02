---
sidebar_position: 4
title: Configuration
---

# Configuration

Nightshift merges configuration from three layers:

1. Global config at `~/.config/nightshift/config.yaml`
2. Project config at `nightshift.yaml` in the repo root
3. Environment variable overrides

Use `nightshift setup` for the guided flow, or `nightshift init` when you want a
starter file you can edit directly.

## Bootstrap and Inspect

```bash
nightshift init
nightshift init --global
nightshift config
nightshift config get budget.max_percent
nightshift config set budget.max_percent 15
nightshift config set providers.copilot.enabled false --global
nightshift config validate
```

`nightshift config set` writes to the project config if one exists in the
current repo; otherwise it writes to the global config. Use `--global` to force
the write target.

## Minimal Config

```yaml
schedule:
  cron: "0 2 * * *"

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
    dangerously_skip_permissions: true

projects:
  - path: ~/code/sidecar
  - path: ~/code/td
```

## Schedule

Use cron syntax or interval scheduling:

```yaml
schedule:
  cron: "0 2 * * *"
  # interval: "8h"
  max_projects: 1
  max_tasks: 1
  # window:
  #   start: "22:00"
  #   end: "06:00"
  #   timezone: "America/Los_Angeles"
```

## Budget

| Field | Default | Description |
|-------|---------|-------------|
| `mode` | `daily` | `daily` or `weekly` |
| `max_percent` | `75` | Max budget percent to use per run |
| `reserve_percent` | `5` | Budget to keep in reserve |
| `weekly_tokens` | `700000` | Fallback weekly budget |
| `billing_mode` | `subscription` | `subscription` or `api` |
| `calibrate_enabled` | `true` | Auto-calibrate from local CLI usage data |
| `snapshot_interval` | `30m` | Snapshot frequency |
| `snapshot_retention_days` | `90` | Snapshot retention window |
| `week_start_day` | `monday` | `monday` or `sunday` |
| `db_path` | platform default | Override Nightshift database path |

## Providers

Nightshift supports Claude Code, Codex, and GitHub Copilot. The
`providers.preference` list controls fallback order during previews and runs.

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
```

Provider-specific notes:

- `providers.claude.dangerously_skip_permissions` skips Claude permission prompts.
- `providers.codex.dangerously_bypass_approvals_and_sandbox` controls Codex's
  headless execution flag.
- `providers.copilot.dangerously_skip_permissions` enables Copilot's
  `--allow-all-tools` and `--allow-all-urls` flags.

## Task Selection

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
```

Each task has a default cooldown interval per project. Override that with
`tasks.intervals`.

`skill-groom` is enabled by default even if you do not list it in
`tasks.enabled`. Add it to `tasks.disabled` if you want to opt out.

## Multi-Project Setup

```yaml
projects:
  - path: ~/code/project1
    priority: 1
    tasks:
      - lint-fix
      - docs-backfill
  - path: ~/code/project2
    priority: 2

  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archived
```

## Integrations

Use integrations to pull in repo instructions and external task sources:

```yaml
integrations:
  claude_md: true
  agents_md: true
  task_sources:
    - td:
        enabled: true
        teach_agent: true
    - github_issues: true
    - file: TODO.md
```

## File Locations

| Type | Location |
|------|----------|
| Run logs | `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log` |
| Audit logs | `~/.local/share/nightshift/audit/audit-YYYY-MM-DD.jsonl` |
| Summaries | `~/.local/share/nightshift/summaries/` |
| Database | `~/.local/share/nightshift/nightshift.db` |
| PID file | `~/.local/share/nightshift/nightshift.pid` |

If `state/state.json` exists from older versions, Nightshift migrates it to the
SQLite database and renames the file to `state.json.migrated`.
