# Nightshift Technical Specification

This document describes the implemented configuration and runtime model for Nightshift. It is a reference for local development and advanced configuration; the website docs remain the user-facing guide.

## Architecture

Nightshift is a Go CLI built around a scheduled run pipeline:

1. Load global config from `~/.config/nightshift/config.yaml`.
2. Merge project config from `nightshift.yaml` in the target repo when a project path is active.
3. Apply `NIGHTSHIFT_` environment overrides.
4. Initialize logging, SQLite state, provider usage readers, budget calibration, and task selection.
5. Build a preflight plan: choose projects, choose an available provider, filter tasks by budget and cooldown, and print the summary.
6. Execute selected tasks through an agent CLI.
7. Record task/project state, run history, reports, and summaries.

Provider responsibilities are split:

- `internal/agents` runs CLI processes for Claude Code, Codex, and Copilot.
- `internal/providers` reads or estimates provider usage for budget decisions.
- `internal/budget` calculates per-run allowance.
- `internal/tasks` defines built-in and custom tasks, cost tiers, risk levels, intervals, and selection.
- `internal/orchestrator` coordinates prompt execution and result handling.

## Config Loading

Config precedence is:

1. Defaults in `internal/config/config.go`.
2. Global config at `~/.config/nightshift/config.yaml`.
3. Project config at `nightshift.yaml`.
4. Environment variables with the `NIGHTSHIFT_` prefix.

Project config overrides global config for the active project. Paths beginning with `~/` are expanded for provider paths, logs, and the database.

Currently explicit environment bindings include:

- `NIGHTSHIFT_BUDGET_MAX_PERCENT`
- `NIGHTSHIFT_BUDGET_MODE`
- `NIGHTSHIFT_LOG_LEVEL`
- `NIGHTSHIFT_LOG_PATH`

## Config Schema

```yaml
schedule:
  cron: "0 2 * * *"
  interval: ""
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/New_York"
  max_projects: 1
  max_tasks: 1

budget:
  mode: daily
  max_percent: 75
  aggressive_end_of_week: false
  reserve_percent: 5
  weekly_tokens: 700000
  per_provider:
    claude: 700000
    codex: 700000
    copilot: 75
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m
  snapshot_retention_days: 90
  week_start_day: monday
  db_path: "~/.local/share/nightshift/nightshift.db"

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
    enabled: true
    data_path: "~/.copilot"
    dangerously_skip_permissions: false

projects:
  - path: ~/code/project
    priority: 1
    tasks:
      - lint-fix
    config: nightshift.yaml
    pattern: ""
    exclude: []

tasks:
  enabled:
    - lint-fix
    - docs-backfill
  disabled: []
  priorities:
    lint-fix: 1
  intervals:
    lint-fix: "24h"
  custom:
    - type: dependency-notes
      name: Dependency Notes
      description: Summarize risky dependencies.
      category: analysis
      cost_tier: medium
      risk_level: low
      interval: "168h"

integrations:
  claude_md: true
  agents_md: true
  task_sources:
    - td:
        enabled: true
        teach_agent: true
    - github_issues: true
    - file: tasks.md

logging:
  level: info
  path: "~/.local/share/nightshift/logs"
  format: json

reporting:
  morning_summary: true
  email:
  slack_webhook:
```

Validation rules include:

- `schedule.cron` and `schedule.interval` are mutually exclusive.
- `budget.mode` must be `daily` or `weekly`.
- `budget.billing_mode` must be `subscription` or `api`.
- `budget.week_start_day` must be `monday` or `sunday`.
- `budget.max_percent` and `budget.reserve_percent` must be between 0 and 100.
- `logging.level` must be `debug`, `info`, `warn`, or `error`.
- `logging.format` must be `json` or `text`.
- Task interval strings must parse as Go durations, for example `24h` or `168h`.
- Provider preference entries must be unique and one of `claude`, `codex`, or `copilot`.
- Custom task types must match `[a-z0-9][a-z0-9-]*`.

When `budget.billing_mode: api` is set, calibration is disabled during config normalization.

## Provider Model

Nightshift supports three execution providers:

| Provider | Agent command | Budget source |
| --- | --- | --- |
| Claude | `claude --print` | Claude local usage data, calibration, or configured fallback |
| Codex | `codex exec` | Codex local usage data, calibration, or configured fallback |
| Copilot | `copilot -p` or `gh copilot -- -p` | Nightshift request counter |

Provider selection uses `providers.preference`, defaulting to `claude`, `codex`, `copilot`. A provider is eligible when:

- It is enabled in config.
- Its CLI binary is on `PATH`.
- The budget manager returns a positive allowance, unless `--ignore-budget` is set.

For Copilot, Nightshift prefers the standalone `copilot` binary and falls back to `gh` when the GitHub Copilot extension is installed.

## Permission Flags

Dangerous permission flags default to false in config defaults:

- `providers.claude.dangerously_skip_permissions`
- `providers.codex.dangerously_bypass_approvals_and_sandbox`
- `providers.copilot.dangerously_skip_permissions`

Codex's agent constructor still defaults to bypass mode for headless execution when no explicit option is passed. The command helper only passes the Codex bypass option when the config value is true, so existing headless fallback behavior is preserved.

For Copilot, `dangerously_skip_permissions` adds `--allow-all-tools --allow-all-urls`. Without it, Copilot still runs with `--no-ask-user --silent`.

## Budget Model

The budget manager resolves a weekly provider budget from calibration or config. `budget.per_provider.<name>` overrides `budget.weekly_tokens`; if neither produces a positive value, provider budget calculation fails.

Daily mode:

```text
daily_budget = weekly_budget / 7
available_today = daily_budget * (1 - used_percent / 100)
allowance = available_today * max_percent / 100
allowance = allowance - reserve_percent_of_daily_budget
allowance = allowance - predicted_daytime_usage
```

Weekly mode:

```text
remaining_weekly = weekly_budget * (1 - used_percent / 100)
allowance = (remaining_weekly / days_until_reset) * max_percent / 100
allowance = allowance * aggressive_end_of_week_multiplier
allowance = allowance - reserve_percent_of_remaining_weekly
allowance = allowance - predicted_daytime_usage
```

Copilot is request based. Nightshift's Copilot provider stores a monthly request count in `providers.copilot.data_path/nightshift-usage.json`, resets it on the first day of each UTC month, and estimates used percent from that count. In budget calculations, the configured weekly Copilot budget is approximated to a monthly request limit by multiplying by four.

## Task Selection

Task selection filters built-in and custom tasks in this order:

1. Enabled and not disabled.
2. Estimated max token cost fits the selected provider allowance.
3. Not currently assigned.
4. Cooldown has elapsed.
5. Highest score wins, unless `--random-task` is set.

Score is:

```text
configured priority + staleness bonus + context mention bonus + task source bonus
```

Context mention bonus comes from `CLAUDE.md` and `AGENTS.md`. Task source bonus comes from configured sources such as td or GitHub issues.

Cost tiers are:

- `low`: 10k-50k tokens
- `medium`: 50k-150k tokens
- `high`: 150k-500k tokens
- `very-high`: 500k-1M tokens

Default task intervals are 7 days for PR tasks, 3 days for analysis tasks, 14 days for safe execution tasks, and 30 days for emergency tasks, with individual task overrides in code and config.

## Storage Locations

| Data | Default location |
| --- | --- |
| Global config | `~/.config/nightshift/config.yaml` |
| Project config | `nightshift.yaml` |
| SQLite database | `~/.local/share/nightshift/nightshift.db` |
| Logs | `~/.local/share/nightshift/logs` |
| Run reports | `~/.local/share/nightshift/reports` |
| Summaries | `~/.local/share/nightshift/summaries` |
| Copilot request counter | `~/.copilot/nightshift-usage.json` |
| Claude data | `~/.claude` |
| Codex data | `~/.codex` |

Older `state/state.json` data is migrated to SQLite by the database layer and then renamed to `state.json.migrated`.

## Safety Model

Nightshift's primary safety boundary is process and workflow control:

- Runs begin with a preflight summary.
- Interactive runs ask for confirmation.
- Non-TTY runs auto-skip confirmation.
- `--dry-run` prints the plan and exits before execution.
- `--ignore-budget` is explicit and shown as a warning.
- Provider permission bypass flags are opt-in in config defaults.
- The orchestrator uses a bounded iteration count and a per-agent timeout.
- Task cooldowns reduce repeated churn on the same project.

Nightshift does not make an agent safe by itself. The effective write and network permissions are those granted to the selected provider CLI and any dangerous flags you enable.

## Operational Workflow

Common commands:

```bash
nightshift setup
nightshift doctor
nightshift preview --explain
nightshift run --dry-run
nightshift run --yes
nightshift task list
nightshift task show docs-backfill
nightshift task run docs-backfill --provider copilot --dry-run
nightshift budget
nightshift budget snapshot --local-only
nightshift budget calibrate
```

Related docs:

- [GitHub Copilot integration](COPILOT_INTEGRATION.md)
- [Run lifecycle](guides/run-lifecycle.md)
- [Codex budget tracking](guides/codex-budget-tracking.md)
- [Provider calibration](guides/provider-calibration.md)
