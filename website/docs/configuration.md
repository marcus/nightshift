---
sidebar_position: 4
title: Configuration
---

# Configuration

Nightshift reads YAML configuration and applies built-in defaults when a field is omitted.

## Create and inspect configuration

Use the guided wizard for a complete global setup:

```bash
nightshift setup
```

For a generated file you can edit directly:

```bash
nightshift init --global       # ~/.config/nightshift/config.yaml
nightshift init                # ./nightshift.yaml
nightshift config validate
```

`--force` overwrites an existing file without the confirmation prompt. You can inspect the
effective configuration or change scalar values without opening an editor:

```bash
nightshift config
nightshift config get providers.preference
nightshift config set budget.max_percent 60
nightshift config set logging.level debug --global
```

`config set` writes to `nightshift.yaml` when that file exists in the current directory;
otherwise it writes to the global file. It parses `true`/`false`, integers, and decimal numbers,
and treats other values as strings.

The `init` templates are opinionated examples rather than serialized built-in defaults. In
particular, the current global template prefers only Claude and Codex, explicitly enables their
unattended permission-bypass flags, and contains legacy task aliases such as `lint` and `docs`.
Review the generated file and use `nightshift task list` for current task slugs.

## Source precedence

From lowest to highest precedence:

1. Built-in defaults.
2. `~/.config/nightshift/config.yaml`.
3. One project `nightshift.yaml`: normally from the current directory, or from the directory
   supplied to `run --project`, `preview --project`, or `task run --project`.
4. `NIGHTSHIFT_` environment overrides.
5. Command-line flags that override behavior for the command being run.

The project filename is exactly `nightshift.yaml`; `.nightshift.yaml` is not loaded automatically.
The explicitly bound environment variables are:

| Variable | Configuration key |
|----------|-------------------|
| `NIGHTSHIFT_BUDGET_MAX_PERCENT` | `budget.max_percent` |
| `NIGHTSHIFT_BUDGET_MODE` | `budget.mode` |
| `NIGHTSHIFT_LOG_LEVEL` | `logging.level` |
| `NIGHTSHIFT_LOG_PATH` | `logging.path` |

## Complete example

```yaml
schedule:
  cron: "0 2 * * *"
  max_projects: 2
  max_tasks: 2
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/Los_Angeles"

budget:
  mode: daily
  max_percent: 75
  aggressive_end_of_week: false
  reserve_percent: 5
  weekly_tokens: 700000
  per_provider:
    claude: 700000
    codex: 500000
    copilot: 500000
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m
  snapshot_retention_days: 90
  week_start_day: monday
  db_path: ~/.local/share/nightshift/nightshift.db

providers:
  preference: [claude, codex, copilot]
  claude:
    enabled: true
    data_path: ~/.claude
    dangerously_skip_permissions: false
  codex:
    enabled: true
    data_path: ~/.codex
    dangerously_bypass_approvals_and_sandbox: false
  copilot:
    enabled: true
    data_path: ~/.copilot
    dangerously_skip_permissions: false

projects:
  - path: ~/code/sidecar
    priority: 2
  - path: ~/code/td
    priority: 1

tasks:
  enabled: [lint-fix, docs-backfill, bug-finder]
  disabled: [td-review]
  priorities:
    bug-finder: 3
    docs-backfill: 2
  intervals:
    lint-fix: 24h
    docs-backfill: 168h

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

Choose exactly one scheduler:

| Key | Default | Rules |
|-----|---------|-------|
| `schedule.cron` | unset | Five-field minute/hour/day/month/weekday expression |
| `schedule.interval` | unset | Positive Go duration such as `30m`, `8h`, or `24h` |
| `schedule.window.start` | unset | `HH:MM`, hour 00-23 and minute 00-59 |
| `schedule.window.end` | unset | `HH:MM`; the end is exclusive |
| `schedule.window.timezone` | local timezone | IANA name such as `America/Los_Angeles` |
| `schedule.max_projects` | `0` | Positive value supplies the default for `run --max-projects`; otherwise the CLI default is 1 |
| `schedule.max_tasks` | `0` | Positive value supplies the default for `run --max-tasks`; otherwise the CLI default is 1 |

`cron` and `interval` are mutually exclusive. A missing schedule is valid for manual commands,
but `preview` and `daemon start` require one. Windows may stay within one day or cross midnight.

`config validate` checks mutual exclusion, but it does not parse the cron expression, interval,
window times, or timezone. Scheduler construction in `doctor`, `preview`, and `daemon start`
performs those checks. A five-field cron expression and a positive Go duration are required at
that point.

## Budget

| Key | Default | Rules |
|-----|---------|-------|
| `budget.mode` | `daily` | `daily` or `weekly` |
| `budget.max_percent` | `75` | 0-100 |
| `budget.aggressive_end_of_week` | `false` | Increase weekly-mode spending in the final two days |
| `budget.reserve_percent` | `5` | 0-100 |
| `budget.weekly_tokens` | `700000` | Fallback weekly allowance |
| `budget.per_provider` | unset | Provider-specific weekly token overrides |
| `budget.billing_mode` | `subscription` | `subscription` or `api` |
| `budget.calibrate_enabled` | `true` | Enable subscription usage calibration |
| `budget.snapshot_interval` | `30m` | Daemon snapshot cadence |
| `budget.snapshot_retention_days` | `90` | Non-negative snapshot retention |
| `budget.week_start_day` | `monday` | `monday` or `sunday` |
| `budget.db_path` | `~/.local/share/nightshift/nightshift.db` | SQLite state and snapshot database |

API billing automatically disables calibration after configuration is loaded. See
[Budget Management](/docs/budget) for the operational workflow.

## Providers

Claude, Codex, and Copilot are enabled by default. The default preference order is:

```yaml
providers:
  preference: [claude, codex, copilot]
```

At run time, Nightshift walks that order and chooses the first enabled provider whose executable
is available and whose calculated allowance is positive. This selection checks executable
presence, not authentication. Preference names are case-insensitive when used and must be unique
values from `claude`, `codex`, and `copilot` during validation.

`data_path` points to a provider's local usage/session data. It does not select an executable;
executables must be discoverable through `PATH`. The effective data-path defaults are
`~/.claude`, `~/.codex`, and `~/.copilot`.

The permission-bypass fields opt autonomous runs into the corresponding provider flags. Their
configuration defaults are `false`. Codex execution itself preserves its headless bypass default
when the setting is not enabled, so disable the Codex provider if you do not want it selected for
autonomous execution.

`preview` is an exception to provider preference: it uses the first enabled provider in the fixed
Claude, Codex, Copilot order and does not check executable availability. `task run` is another
exception: its required `--provider` is used even when that provider is disabled in configuration.

## Projects

Top-level commands currently resolve explicit `path` entries:

```yaml
projects:
  - path: ~/code/project-one
    priority: 2
  - path: ~/code/project-two
    priority: 1
```

The schema also accepts `priority`, `tasks`, `config`, `pattern`, and `exclude` fields:

```yaml
projects:
  - pattern: ~/code/open-source/*
    priority: 1
    exclude:
      - ~/code/open-source/archived
```

The project resolver package implements pattern expansion, exclusions, priority ordering, and a
limited per-project merge, but `run`, `preview`, and the daemon do not call that resolver. Their
current path enumeration uses only existing `projects[].path` values and preserves configuration
order. List concrete paths for unattended execution; the other project fields are accepted by
the schema but do not affect those commands.

Nightshift also does not automatically merge a `nightshift.yaml` found inside each configured
project. To use a project's file, invoke `run --project PATH`, `preview --project PATH`, or
`task run --project PATH`, or start Nightshift with that project as the current directory.

## Task customization

An empty `tasks.enabled` list means all built-in tasks are eligible. `tasks.disabled` always wins.
Higher priority numbers increase selection score. Interval values must be valid Go durations.

```yaml
tasks:
  custom:
    - type: dependency-review
      name: Dependency Review
      description: |
        Review direct dependencies for stale or risky versions.
        Open a pull request for safe updates.
      category: analysis
      cost_tier: medium
      risk_level: low
      interval: 72h
```

Custom `type` values use lowercase letters, numbers, and hyphens. `type`, `name`, and
`description` are required. Categories are `pr`, `analysis`, `options`, `safe`, `map`, or
`emergency`; cost tiers are `low`, `medium`, `high`, or `very-high`; risk levels are `low`,
`medium`, or `high`.

`run` and `preview` register configured custom tasks. The `task list`, `task show`, and `task run`
commands currently expose only built-in task definitions.

## Integrations, logging, and reporting

`integrations.claude_md` and `integrations.agents_md` default to `true`. The schema also accepts
td, GitHub issue, and file task-source entries. Reader implementations exist for Claude/agent
instruction files, td, and GitHub issues, but no current CLI command invokes the integration
manager, so these settings do not yet add prompt context or tasks. See
[Integrations](/docs/integrations).

Logging defaults to level `info`, JSON format, and
`~/.local/share/nightshift/logs`. Valid levels are `debug`, `info`, `warn`, and `error`; valid
formats are `json` and `text`. `reporting.morning_summary` defaults to `true`. Email and Slack
webhook fields exist in the configuration schema. Notification implementations also exist, but
the run finalizer currently saves the local summary without calling notification dispatch.
