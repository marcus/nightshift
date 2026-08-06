---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

Run `nightshift --help` or `nightshift COMMAND --help` for the full generated help text. This page focuses on the commands and flags most people reach for.

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Enable verbose output |
| `--version`, `-v` | Print the installed Nightshift version |

## Top-Level Commands

| Command | Description |
|---------|-------------|
| `nightshift setup` | Guided global onboarding wizard |
| `nightshift init` | Create a starter config file |
| `nightshift config` | Show, read, write, or validate config |
| `nightshift preview` | Preview upcoming runs without executing |
| `nightshift run` | Execute tasks immediately |
| `nightshift budget` | Inspect budget status and snapshots |
| `nightshift task` | Browse tasks, inspect prompts, or run one directly |
| `nightshift logs` | Tail, filter, or export logs |
| `nightshift report` | Summarize recent Nightshift activity |
| `nightshift stats` | Show aggregate run and token statistics |
| `nightshift busfactor` | Analyze ownership concentration in a repo |
| `nightshift daemon` | Start, stop, or inspect the background daemon |
| `nightshift install` | Install a launchd, systemd, or cron service |
| `nightshift uninstall` | Remove the installed service |
| `nightshift doctor` | Check config, providers, snapshots, and DB health |
| `nightshift status` | Show recent runs or today's activity |

## Setup And Configuration

### `nightshift setup`

Interactive onboarding that writes the global config, validates providers, runs a snapshot, previews the next run, and can install the daemon.

```bash
nightshift setup
```

### `nightshift init`

Create starter config files without the wizard.

```bash
nightshift init
nightshift init --global
nightshift init --force
```

| Flag | Description |
|------|-------------|
| `--global` | Write `~/.config/nightshift/config.yaml` instead of `./nightshift.yaml` |
| `--force`, `-f` | Overwrite an existing config without prompting |

### `nightshift config`

Show the merged config and work with specific keys.

```bash
nightshift config
nightshift config get budget.max_percent
nightshift config set budget.max_percent 60 --global
nightshift config validate
```

Subcommands:

| Subcommand | Description |
|------------|-------------|
| `config get KEY` | Print a specific config value |
| `config set KEY VALUE` | Write a specific config value |
| `config validate` | Validate global, project, and merged config |

`config set` writes to `nightshift.yaml` when it exists in the current repo. Otherwise it writes to the global config unless `--global` is provided explicitly.

## Running Work

### `nightshift run`

`nightshift run` shows a preflight summary before executing. Interactive terminals prompt for confirmation; non-TTY environments skip confirmation automatically.

```bash
nightshift run
nightshift run --dry-run
nightshift run --yes
nightshift run --max-projects 3 --max-tasks 2
nightshift run --random-task
nightshift run --ignore-budget
nightshift run --project ~/code/myapp --task lint-fix
nightshift run --branch develop
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show the preflight summary and exit |
| `--yes`, `-y` | `false` | Skip the confirmation prompt |
| `--project`, `-p` | unset | Limit the run to one project path |
| `--task`, `-t` | unset | Run a specific task type |
| `--max-projects` | `1` | Max projects to process when `--project` is not set |
| `--max-tasks` | `1` | Max tasks per project when `--task` is not set |
| `--random-task` | `false` | Choose a random eligible task instead of the top-scored task |
| `--ignore-budget` | `false` | Bypass budget checks |
| `--branch`, `-b` | current branch | Base branch for new feature branches |
| `--timeout` | `30m` | Per-agent execution timeout |
| `--no-color` | `false` | Disable colored output |

### `nightshift preview`

Preview the next scheduled runs without changing state.

```bash
nightshift preview
nightshift preview -n 5
nightshift preview --long
nightshift preview --explain
nightshift preview --json
nightshift preview --write ./nightshift-prompts
```

Key flags:

| Flag | Description |
|------|-------------|
| `--runs`, `-n` | Number of upcoming runs to preview |
| `--project`, `-p` | Preview only one project |
| `--task`, `-t` | Preview only one task type |
| `--long` | Show full prompts instead of truncated previews |
| `--explain` | Show budget and cooldown explanations |
| `--json` | Machine-readable output |
| `--plain` | Disable the `gum` pager |
| `--write DIR` | Write prompt files to a directory |

### `nightshift task`

Browse tasks, inspect their prompts, or run one directly against a specific provider.

```bash
nightshift task list
nightshift task list --category pr
nightshift task list --cost low --json
nightshift task show lint-fix
nightshift task show lint-fix --prompt-only
nightshift task run lint-fix --provider claude
nightshift task run docs-backfill --provider copilot --dry-run
```

Subcommands:

| Subcommand | Description |
|------------|-------------|
| `task list` | List available tasks with filters |
| `task show TASK_TYPE` | Show task metadata and the planning prompt |
| `task run TASK_TYPE --provider PROVIDER` | Execute one task immediately |

## Budget And Reporting

### `nightshift budget`

Inspect provider budget state or work with usage snapshots.

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget --provider copilot
nightshift budget snapshot --provider codex
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
```

Subcommands:

| Subcommand | Description |
|------------|-------------|
| `budget snapshot` | Capture a usage snapshot |
| `budget history` | Show recent snapshots |
| `budget calibrate` | Show inferred calibration status |

### `nightshift logs`

Tail and filter Nightshift logs.

```bash
nightshift logs
nightshift logs --tail 200 --level warn
nightshift logs --follow --component daemon
nightshift logs --match copilot --since 2026-04-01
nightshift logs --export ./nightshift.log
```

Common flags:

| Flag | Description |
|------|-------------|
| `--follow`, `-f` | Stream new log lines |
| `--tail`, `-n` | Number of lines to show |
| `--component` | Filter by component substring |
| `--level` | Minimum level: `debug`, `info`, `warn`, `error` |
| `--match` | Filter by message substring |
| `--since`, `--until` | Time window filters |
| `--summary` | Show summary only |
| `--raw` | Print raw log lines |
| `--export FILE` | Write matching logs to a file |

### `nightshift report`

Summarize recent work in a more structured format than `status`.

```bash
nightshift report
nightshift report --report overview --period last-night
nightshift report --report tasks --period last-7d --format markdown
nightshift report --report raw --format json --runs 1
```

| Flag | Default | Description |
|------|---------|-------------|
| `--report`, `-r` | `overview` | `overview`, `tasks`, `projects`, `budget`, or `raw` |
| `--period`, `-p` | `last-night` | `last-night`, `last-run`, `last-24h`, `last-7d`, `today`, `yesterday`, or `all` |
| `--format` | `fancy` | `fancy`, `plain`, `markdown`, or `json` |
| `--runs`, `-n` | `3` | Max runs to include (`0` means all) |
| `--max-items` | `5` | Max highlights per run |
| `--paths` | `false` | Include report and log file paths |
| `--since`, `--until` | unset | Explicit time bounds |

### `nightshift stats`

Show aggregate run counts, task outcomes, token usage, and projections.

```bash
nightshift stats
nightshift stats --period last-7d
nightshift stats --json
```

### `nightshift status`

Show recent runs or a same-day summary.

```bash
nightshift status
nightshift status --last 10
nightshift status --today
```

## Ownership Analysis

### `nightshift busfactor`

Analyze commit concentration for a repo, directory, or file subset.

```bash
nightshift busfactor .
nightshift busfactor --path ~/code/nightshift --since 2026-01-01
nightshift busfactor --file 'internal/**/*.go' --save
nightshift busfactor --json
```

Key flags:

| Flag | Description |
|------|-------------|
| `--path`, `-p` | Repo or directory to analyze |
| `--file`, `-f` | Restrict analysis to a file or pattern |
| `--since`, `--until` | Date range filters |
| `--save` | Persist results to the database |
| `--json` | Machine-readable output |
| `--db` | Override the database path |

## Background Execution

### `nightshift install`

Install a scheduled service. If you omit the service type, Nightshift auto-detects one from the current OS.

```bash
nightshift install
nightshift install launchd
nightshift install systemd
nightshift install cron
nightshift uninstall
```

Supported service types:

| Type | Environment |
|------|-------------|
| `launchd` | macOS user LaunchAgent |
| `systemd` | Linux user service and timer |
| `cron` | Generic cron entry |

### `nightshift daemon`

Run the scheduler as a foreground or background daemon.

```bash
nightshift daemon start
nightshift daemon start --foreground
nightshift daemon start --foreground --timeout 45m
nightshift daemon status
nightshift daemon stop
```

Subcommands:

| Subcommand | Description |
|------------|-------------|
| `daemon start` | Start the daemon |
| `daemon status` | Check whether the daemon is running |
| `daemon stop` | Stop the running daemon |

`daemon start` supports:

| Flag | Default | Description |
|------|---------|-------------|
| `--foreground`, `-f` | `false` | Stay in the foreground instead of daemonizing |
| `--timeout` | `30m` | Per-agent execution timeout |

## Diagnostics

```bash
nightshift doctor
nightshift status --today
```

Use `nightshift doctor` after changing config or troubleshooting provider data and snapshot health. To smoke-test a specific provider CLI, run `nightshift task run lint-fix --provider PROVIDER --dry-run`.
