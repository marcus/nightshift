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

Nightshift loads and merges config in this order, from lowest to highest precedence:

1. Global config: `~/.config/nightshift/config.yaml`
2. Project config: `nightshift.yaml` in the current project directory
3. Bound environment overrides

Project config values override global config values, and environment variables override both. The currently bound variables are `NIGHTSHIFT_BUDGET_MAX_PERCENT`, `NIGHTSHIFT_BUDGET_MODE`, `NIGHTSHIFT_LOG_LEVEL`, and `NIGHTSHIFT_LOG_PATH`. Other `NIGHTSHIFT_*` names are not guaranteed to unmarshal into the config struct.

When a command accepts `--project`, Nightshift loads `nightshift.yaml` from that directory. Otherwise it uses the current working directory. `nightshift config set` updates the current directory's project file only when that file already exists; otherwise it writes the global file. It writes first and then warns if the merged result fails validation, so review the file after a warning.

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
  claude:
    enabled: true
    data_path: "~/.claude"
    dangerously_skip_permissions: false
  codex:
    enabled: false
  copilot:
    enabled: false

projects:
  - path: ~/code/sidecar
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
| `window` | none | Optional execution-window object; both start and end must be valid `HH:MM` values when present |
| `window.start` | - | Inclusive start of the allowed execution window |
| `window.end` | - | Exclusive end; an earlier end spans midnight |
| `window.timezone` | local time | IANA time zone for the window |
| `max_projects` | `0` | Positive values replace the manual `run` default when its flag is omitted; `0` leaves the CLI default of 1 |
| `max_tasks` | `0` | Positive values replace the manual `run` default when its flag is omitted; `0` leaves the CLI default of 1 |

The config validator rejects cron plus interval, but it permits both to be empty. `preview` and `daemon start` then report that no schedule is configured. Cron is five fields; interval uses a positive Go duration.

## Budget

Control how much of your token budget Nightshift uses.

| Field | Default | Description |
|-------|---------|-------------|
| `mode` | `daily` | `daily` or `weekly` |
| `max_percent` | `75` | Max budget % to use per run; validation accepts `0`-`100`, and calculation treats `0` as the default `75` |
| `reserve_percent` | `5` | Deduct this % of the current budget base after applying `max_percent` |
| `billing_mode` | `subscription` | `subscription` or `api` |
| `calibrate_enabled` | `true` | Enable subscription calibration via snapshots |
| `snapshot_interval` | `30m` | Automatic snapshot cadence |
| `snapshot_retention_days` | `90` | Snapshot retention window |
| `weekly_tokens` | `700000` | Fallback weekly budget |
| `per_provider` | - | Provider-specific weekly budgets |
| `week_start_day` | `monday` | Week boundary for calibration |
| `db_path` | `~/.local/share/nightshift/nightshift.db` | Override database path |
| `aggressive_end_of_week` | `false` | Weekly mode multiplier: currently 1x with two days left and 2x with one day left |

If `billing_mode: api`, Nightshift turns calibration off after loading and uses the explicit token budgets in `weekly_tokens` and `per_provider`. See [Budget Management](/docs/budget) for the exact daily/weekly formulas and provider-specific usage sources.

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

Every provider entry is decoded through the same struct, so YAML may contain both dangerous keys under any provider. Execution currently reads only Claude's `dangerously_skip_permissions`, Codex's `dangerously_bypass_approvals_and_sandbox`, and Copilot's `dangerously_skip_permissions`; the other combinations have no effect.

For automatic runs, provider preference is filtered by enabled status, CLI discovery, and positive allowance. The first remaining entry wins. Direct `task run` requires an explicit provider.

The safety fields default to false in configuration. Claude and Copilot pass their dangerous flags only when enabled. Codex is an exception in this release: its headless agent defaults the approval/sandbox bypass on, and a false config value preserves that agent default because unset and explicit false are indistinguishable. Disable Codex entirely if that behavior is not acceptable.

Copilot's usage model is request-based rather than token-based, but the execution path currently does not increment its local counter. The standalone `copilot` binary is preferred; see [Integrations](/docs/integrations) for fallback details and limitations.

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

An empty `tasks.enabled` means all tasks except definitions marked disabled-by-default are eligible. `tasks.disabled` always wins. Custom `type` values must match `[a-z0-9][a-z0-9-]*` and be unique. Optional categories are `pr`, `analysis`, `options`, `safe`, `map`, and `emergency`; cost tiers are `low`, `medium`, `high`, and `very-high`; risks are `low`, `medium`, and `high`. Empty optional fields default to analysis, medium cost, low risk, and the category's standard cooldown.

`run`, `preview`, and the setup wizard register configured custom tasks. The current persistent daemon and direct `task list`, `task show`, and `task run` command paths do not load them, so use `nightshift run --task CUSTOM_TYPE` for direct custom-task execution.

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
| `integrations.task_sources` | - | Reader configuration for td, GitHub issues, or a file field |

The readers are implemented, but current run/daemon/preview paths do not invoke their manager, and the file source has no reader. See [Integrations](/docs/integrations) before relying on these fields for scheduling.

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

The schema includes `path`, `priority`, `tasks`, `config`, `pattern`, and `exclude`. The reusable project resolver implements explicit paths, non-recursive filesystem globs, exact/subtree exclusions, priority ordering, and a limited per-project merge.

The current CLI execution paths do not use that resolver: `run`, `daemon`, and `preview` read only `projects[].path`, silently skip missing paths, and preserve declaration order. `pattern`, `exclude`, `priority`, per-project `tasks`, and `config` do not currently affect those commands. Expand patterns into explicit `path` entries for production configuration.

## Logging and Reporting

```yaml
logging:
  level: info
  path: ~/.local/share/nightshift/logs
  format: json

reporting:
  morning_summary: true
  # email: user@example.com
  # slack_webhook: https://hooks.slack.com/...
```

| Field | Default | Description |
|-------|---------|-------------|
| `logging.level` | `info` | `debug`, `info`, `warn`, or `error` |
| `logging.path` | `~/.local/share/nightshift/logs` | Daily log directory; `~` is expanded |
| `logging.format` | `json` | `json` or plain console-style `text` |
| `reporting.morning_summary` | `true` | Write the dated Markdown summary after runs |
| `reporting.email` | none | Notification address; requires `NIGHTSHIFT_SMTP_HOST` and optionally SMTP port/user/pass/from environment values |
| `reporting.slack_webhook` | none | POST the generated summary to this Slack webhook |

Nightshift retains its dated structured log files for seven days; that retention is not currently configurable through the main schema.

## Safe Defaults

| Feature | Default | Override |
|---------|---------|----------|
| Confirmation prompt in TTY | Yes | `--yes` |
| Confirmation prompt in non-TTY | Auto-skip | `--yes` or interactive terminal |
| Max projects per run | `1` | `--max-projects` or `schedule.max_projects` |
| Max tasks per project | `1` | `--max-tasks` or `schedule.max_tasks` |
| Max budget per run | `75%` | `budget.max_percent` |
| Reserve budget | `5%` | `budget.reserve_percent` |
| Claude permission bypass | Off | `providers.claude.dangerously_skip_permissions` |
| Copilot broad tool/URL access | Off | `providers.copilot.dangerously_skip_permissions` |

These per-run limits describe `nightshift run` and service installations, which invoke that command. The persistent daemon currently processes every explicit project path and selects up to five tasks per project instead of using the schedule limit fields.

## File Locations

| Type | Location |
|------|----------|
| Run logs | `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log` |
| Audit logs | `~/.local/share/nightshift/audit/audit-YYYY-MM-DD.jsonl` |
| Summaries | `~/.local/share/nightshift/summaries/` |
| Database | `~/.local/share/nightshift/nightshift.db` |
| PID file | `~/.local/share/nightshift/nightshift.pid` |

If `state/state.json` exists from older versions, Nightshift migrates it to the SQLite database and renames the file to `state.json.migrated`.
