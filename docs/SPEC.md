# Nightshift Technical Specification

This document describes Nightshift's implemented runtime configuration,
provider model, budget model, task selection, storage locations, and operating
workflow.

## Architecture

Nightshift is a Go CLI that selects repository maintenance tasks, chooses an
available AI provider, and runs a plan, implementation, and review workflow in a
target repository.

Main components:

- `cmd/nightshift/commands`: Cobra commands such as `run`, `task`, `budget`,
  `doctor`, `setup`, and `daemon`.
- `internal/config`: YAML loading, defaults, environment overrides, validation,
  and path expansion.
- `internal/providers`: provider-side usage and budget data for Claude, Codex,
  and Copilot.
- `internal/agents`: command execution wrappers for Claude Code, Codex, and
  GitHub Copilot.
- `internal/budget`: allowance calculation and reset-time reporting.
- `internal/tasks`: built-in task definitions, custom tasks, scoring, and
  cooldowns.
- `internal/orchestrator`: task execution loop, run metadata, and PR-oriented
  prompts.
- `internal/state` and `internal/db`: SQLite-backed run and task state.

## Configuration Loading

Nightshift loads configuration in this order:

1. global config at `~/.config/nightshift/config.yaml`;
2. project config named `nightshift.yaml` in the selected project;
3. environment variables with the `NIGHTSHIFT_` prefix.

Run `nightshift setup` for interactive configuration. Project config is loaded
after the global config, so project-level values can override global values for
that repository.

## Configuration Schema

Representative schema:

```yaml
schedule:
  cron: "0 2 * * *"
  interval: "8h"
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
    copilot: 100
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
  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archived

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
    - type: dependency-review
      name: "Dependency Review"
      description: "Review dependency health and open a PR for safe updates."
      category: safe
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

- `schedule.cron` and `schedule.interval` are mutually exclusive;
- `budget.mode` must be `daily` or `weekly`;
- `budget.billing_mode` must be `subscription` or `api`;
- `budget.week_start_day` must be `monday` or `sunday`;
- `budget.max_percent` must be between 0 and 100;
- `budget.reserve_percent` must be between 0 and 100;
- `logging.level` must be `debug`, `info`, `warn`, or `error`;
- `logging.format` must be `json` or `text`;
- provider preferences must be unique and limited to `claude`, `codex`, and
  `copilot`;
- custom task types, categories, cost tiers, risk levels, and intervals are
  validated before use.

When `budget.billing_mode` is `api`, calibration is disabled during config
normalization.

## Provider Model

Nightshift separates providers from agents:

- providers expose usage and budget data;
- agents execute prompts through local CLIs.

Supported execution providers:

- `claude`: Claude Code CLI, data path default `~/.claude`;
- `codex`: Codex CLI, data path default `~/.codex`;
- `copilot`: GitHub Copilot CLI, data path default `~/.copilot`.

Automatic provider selection follows `providers.preference`. It skips disabled
providers, providers whose CLI is unavailable, and providers with no remaining
allowance unless `--ignore-budget` is set.

For Copilot, selection prefers the standalone `copilot` command when present and
falls back to `gh` with the Copilot extension.

## Budget Model

Nightshift calculates an allowance before selecting tasks:

- daily mode uses `weekly_budget / 7` as the starting daily budget;
- weekly mode uses remaining weekly budget divided by days until reset;
- `max_percent` limits how much of the available budget a run may use;
- `reserve_percent` is subtracted from the calculated allowance;
- predicted daytime usage may be reserved when trend data is available.

Claude and Codex are token-based providers. Codex can use rate-limit data from
session JSONL files, with token totals as a fallback. Subscription calibration
uses snapshots to infer a weekly token budget and falls back to
`budget.weekly_tokens` or `budget.per_provider`.

Copilot is modeled as local monthly request usage because GitHub does not expose
authoritative Copilot usage data to Nightshift. The provider can read and write
`nightshift-usage.json` under the Copilot data path, but current task execution
paths do not call the increment helper after Copilot runs. Copilot budget
enforcement is therefore approximate and only reflects existing local counter
data, not total GitHub-side Copilot usage.

## Task Selection

Tasks are selected per project after provider allowance is known.

Selection inputs:

- task definitions and estimated token ranges;
- `tasks.enabled` and `tasks.disabled`;
- per-task priorities from `tasks.priorities`;
- per-project task overrides;
- cooldowns from task defaults or `tasks.intervals`;
- project state recorded in SQLite;
- optional `--task`, `--random-task`, `--max-projects`, and `--max-tasks`.

Custom tasks require `type`, `name`, and `description`. Optional metadata
includes `category`, `cost_tier`, `risk_level`, and `interval`.

## Runtime Workflow

`nightshift run` performs these steps:

1. load config and initialize logging;
2. open the SQLite database;
3. clear stale task assignments;
4. initialize providers, calibration, trends, and budget manager;
5. resolve projects;
6. select a base branch;
7. register custom tasks;
8. build and display a preflight plan;
9. confirm execution unless skipped by `--yes`, `--dry-run`, or a non-TTY
   environment;
10. run each selected task through the orchestrator;
11. record state and write run reports.

`nightshift task run <task> --provider <provider>` bypasses automatic provider
selection and runs one task in the selected project with the requested provider.

## Storage Locations

Default locations:

| Data | Location |
| --- | --- |
| Global config | `~/.config/nightshift/config.yaml` |
| Project config | `nightshift.yaml` |
| SQLite database | `~/.local/share/nightshift/nightshift.db` |
| Logs | `~/.local/share/nightshift/logs` |
| Reports | `~/.local/share/nightshift/reports` |
| Summaries | `~/.local/share/nightshift/summaries` |
| Claude data | `~/.claude` |
| Codex data | `~/.codex` |
| Copilot data | `~/.copilot` |

`budget.db_path`, `logging.path`, and provider `data_path` fields can override
the defaults.

## Safety Model

Nightshift is designed for unattended execution, but dangerous provider flags
remain explicit:

- Claude only receives `--dangerously-skip-permissions` when
  `providers.claude.dangerously_skip_permissions` is enabled.
- Codex only receives the bypass option when
  `providers.codex.dangerously_bypass_approvals_and_sandbox` is enabled.
- Copilot only receives `--allow-all-tools --allow-all-urls` when
  `providers.copilot.dangerously_skip_permissions` is enabled.

Interactive `nightshift run` displays a preflight summary and asks for
confirmation. Non-TTY runs auto-confirm. Use `--dry-run` to inspect planned work
without executing it.

## Related Docs

- [Copilot integration](COPILOT_INTEGRATION.md)
- [Run lifecycle](guides/run-lifecycle.md)
- [Codex budget tracking](guides/codex-budget-tracking.md)
- [Configuration docs](../website/docs/configuration.md)
- [Budget docs](../website/docs/budget.md)
- [Tasks docs](../website/docs/tasks.md)
