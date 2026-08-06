---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

Nightshift ships a single `nightshift` binary with commands for setup, execution, reporting, and service management.

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Enable verbose output |
| `-h`, `--help` | Show help for the current command |
| `-v`, `--version` | Show the current Nightshift version |

## Root Commands

| Command | Description |
|---------|-------------|
| `nightshift setup` | Interactive onboarding wizard |
| `nightshift init` | Create a starter config file |
| `nightshift config` | Show, get, set, or validate config |
| `nightshift doctor` | Check config, providers, database, and budget health |
| `nightshift preview` | Preview upcoming runs without executing |
| `nightshift run` | Execute tasks immediately |
| `nightshift task` | List, inspect, or run a single task |
| `nightshift budget` | Show current provider budget status |
| `nightshift status` | Show recent run history |
| `nightshift logs` | Tail, filter, or export logs |
| `nightshift report` | Render run reports in human or machine-readable formats |
| `nightshift stats` | Show aggregate statistics across runs |
| `nightshift busfactor` | Analyze code ownership concentration |
| `nightshift daemon` | Start, stop, or inspect the background daemon |
| `nightshift install` | Install a launchd, systemd, or cron service |
| `nightshift uninstall` | Remove the installed system service |
| `nightshift completion` | Generate shell completion scripts |

## Setup and Configuration

### `nightshift setup`

Runs the interactive onboarding wizard. It creates or updates the global config, checks provider availability, captures a snapshot, previews the next run, and can install the daemon.

```bash
nightshift setup
```

### `nightshift init`

Creates a starter config file.

```bash
nightshift init
nightshift init --global
nightshift init --force
```

| Flag | Description |
|------|-------------|
| `--global` | Create `~/.config/nightshift/config.yaml` instead of `./nightshift.yaml` |
| `-f`, `--force` | Overwrite an existing config without prompting |

### `nightshift config`

Running `nightshift config` with no subcommand prints the merged configuration plus the global and project config source paths.

```bash
nightshift config
nightshift config get budget.max_percent
nightshift config get providers.preference
nightshift config set budget.max_percent 60
nightshift config set logging.level debug --global
nightshift config validate
```

Subcommands:

| Subcommand | Description |
|------------|-------------|
| `config get KEY` | Print a config value by key path |
| `config set KEY VALUE` | Write a config value to the current project config if `./nightshift.yaml` exists, otherwise to the global config |
| `config validate` | Validate the current global and project configs |

`config set` supports:

| Flag | Description |
|------|-------------|
| `-g`, `--global` | Always write to the global config |

### `nightshift doctor`

Checks configuration, schedule resolution, service installation, daemon state, provider CLIs, provider data paths, snapshots, and budget readiness.

```bash
nightshift doctor
```

## Previewing and Running Work

### `nightshift preview`

Shows what Nightshift would do next without executing tasks or mutating state.

```bash
nightshift preview
nightshift preview -n 5
nightshift preview --project ~/code/nightshift
nightshift preview --task docs-backfill
nightshift preview --explain
nightshift preview --json
nightshift preview --write ./nightshift-prompts
```

| Flag | Description |
|------|-------------|
| `-n`, `--runs` | Number of upcoming runs to preview |
| `-p`, `--project` | Preview only a specific project path |
| `-t`, `--task` | Preview only a specific task type |
| `--long` | Show full prompts |
| `--write` | Write prompt files to a directory |
| `--explain` | Include budget and task-filter explanations |
| `--plain` | Disable gum pager output |
| `--json` | Emit JSON output, including full prompts |

### `nightshift run`

Executes tasks immediately. In interactive terminals, Nightshift shows a preflight summary and asks for confirmation. In non-TTY environments such as cron, CI, and the daemon, confirmation is auto-skipped.

```bash
nightshift run
nightshift run --dry-run
nightshift run --yes
nightshift run --project ~/code/nightshift --task docs-backfill
nightshift run --max-projects 3 --max-tasks 2
nightshift run --random-task
nightshift run --branch main --timeout 45m
nightshift run --ignore-budget
```

| Flag | Description |
|------|-------------|
| `--dry-run` | Show the preflight summary and exit without executing |
| `-p`, `--project` | Limit execution to one project directory |
| `-t`, `--task` | Run one specific task type |
| `--max-projects` | Cap how many projects are processed in the run |
| `--max-tasks` | Cap how many tasks run per project |
| `--random-task` | Pick one random eligible task instead of the highest-scored task |
| `--ignore-budget` | Bypass budget checks |
| `-b`, `--branch` | Use a specific base branch for new feature branches |
| `--timeout` | Set the per-agent execution timeout |
| `--no-color` | Disable colored output |
| `-y`, `--yes` | Skip the confirmation prompt |

### `nightshift daemon`

Runs Nightshift on the configured schedule in the background.

```bash
nightshift daemon start
nightshift daemon start --foreground
nightshift daemon status
nightshift daemon stop
```

Subcommands:

| Subcommand | Description |
|------------|-------------|
| `daemon start` | Start the background daemon |
| `daemon status` | Show whether the daemon is running |
| `daemon stop` | Stop the daemon by sending `SIGTERM` |

`daemon start` flags:

| Flag | Description |
|------|-------------|
| `-f`, `--foreground` | Run in the foreground instead of daemonizing |
| `--timeout` | Set the per-agent execution timeout used by scheduled runs |

## Tasks and Budgets

### `nightshift task`

Use `task` to inspect the built-in task library or run one task directly.

```bash
nightshift task list
nightshift task list --category pr --cost low
nightshift task show docs-backfill
nightshift task show lint-fix --prompt-only
nightshift task run docs-backfill --provider claude
nightshift task run docs-backfill --provider codex --dry-run --timeout 45m
nightshift task run docs-backfill --provider copilot --branch develop
```

Subcommands:

| Subcommand | Description |
|------------|-------------|
| `task list` | List available tasks, cost tiers, and categories |
| `task show &lt;task-type&gt;` | Show metadata plus the planning prompt |
| `task run &lt;task-type&gt; --provider PROVIDER` | Execute one task immediately |

`task list` flags:

| Flag | Description |
|------|-------------|
| `--category` | Filter by category: `pr`, `analysis`, `options`, `safe`, `map`, `emergency` |
| `--cost` | Filter by cost tier: `low`, `medium`, `high`, `veryhigh` |
| `--json` | Emit JSON output |

`task show` flags:

| Flag | Description |
|------|-------------|
| `-p`, `--project` | Use a specific project path when building prompt context |
| `--prompt-only` | Print only the raw prompt text |
| `--json` | Emit JSON output |

`task run` flags:

| Flag | Description |
|------|-------------|
| `--provider` | Required. Choose `claude`, `codex`, or `copilot` |
| `-p`, `--project` | Run in a specific project directory |
| `--dry-run` | Print the prompt without executing |
| `--timeout` | Set the execution timeout |
| `-b`, `--branch` | Use a specific base branch for new feature branches |

### `nightshift budget`

Shows budget status for all enabled providers or one specific provider.

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget --provider codex
nightshift budget --provider copilot
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate --provider claude
```

Subcommands:

| Subcommand | Description |
|------------|-------------|
| `budget snapshot` | Capture a calibration snapshot |
| `budget history` | Show recent snapshots |
| `budget calibrate` | Show inferred calibration status |

Shared budget flags:

| Flag | Description |
|------|-------------|
| `-p`, `--provider` | Filter to `claude`, `codex`, or `copilot` |

`budget snapshot` flags:

| Flag | Description |
|------|-------------|
| `--local-only` | Skip tmux scraping and store a local-only snapshot |

`budget history` flags:

| Flag | Description |
|------|-------------|
| `-n` | Number of snapshots to show |

## Reporting and Observability

### `nightshift status`

Shows recent runs or a single-day summary.

```bash
nightshift status
nightshift status --today
nightshift status -n 10
```

| Flag | Description |
|------|-------------|
| `-n`, `--last` | Show the last `N` runs |
| `--today` | Show today's activity summary |

### `nightshift logs`

Tails or filters JSON log files from `~/.local/share/nightshift/logs` by default.

```bash
nightshift logs
nightshift logs --follow
nightshift logs --level warn --since 2026-04-17
nightshift logs --component run --match docs-backfill
nightshift logs --summary
nightshift logs --export ./nightshift.log
```

| Flag | Description |
|------|-------------|
| `-n`, `--tail` | Number of log lines to show |
| `-f`, `--follow` | Stream new log output |
| `-e`, `--export` | Export matching logs to a file |
| `--since` / `--until` | Filter by time range |
| `--level` | Filter by minimum log level |
| `--component` | Filter by component substring |
| `--match` | Filter by message substring |
| `--summary` | Show only summary stats |
| `--raw` | Show raw log lines |
| `--path` | Override the log directory |
| `--no-color` | Disable colored output |

### `nightshift report`

Renders structured reports from previous runs.

```bash
nightshift report
nightshift report --report tasks
nightshift report --period last-run
nightshift report --since 2026-04-17 --until 2026-04-18
nightshift report --format markdown
nightshift report --report raw --format plain --paths
```

| Flag | Description |
|------|-------------|
| `-r`, `--report` | Choose `overview`, `tasks`, `projects`, `budget`, or `raw` |
| `-p`, `--period` | Choose `last-night`, `last-run`, `last-24h`, `last-7d`, `today`, `yesterday`, or `all` |
| `-n`, `--runs` | Limit how many runs to include |
| `--since` / `--until` | Use an explicit time window instead of a named period |
| `--format` | Choose `fancy`, `plain`, `markdown`, or `json` |
| `--paths` | Include report and log file paths |
| `--max-items` | Limit highlights per run |
| `--no-color` | Disable colored output |

### `nightshift stats`

Aggregates results across all recorded runs.

```bash
nightshift stats
nightshift stats --period last-7d
nightshift stats --json
```

| Flag | Description |
|------|-------------|
| `-p`, `--period` | Choose `all`, `last-7d`, `last-30d`, or `last-night` |
| `--json` | Emit JSON output |

## Analysis, Services, and Utilities

### `nightshift busfactor`

Analyzes commit ownership concentration for the current repository or another path.

```bash
nightshift busfactor
nightshift busfactor ~/code/nightshift
nightshift busfactor --since 2026-01-01 --file cmd/nightshift/commands/run.go
nightshift busfactor --json
nightshift busfactor --save
```

| Flag | Description |
|------|-------------|
| `-p`, `--path` | Repository or directory path |
| `-f`, `--file` | Restrict analysis to one file or path pattern |
| `--since` / `--until` | Filter commit history by date |
| `--json` | Emit JSON output |
| `--save` | Store results in the Nightshift database |
| `--db` | Override the database path |

### `nightshift install`

Installs a background service. If no service type is provided, Nightshift auto-detects the best option for the current OS.

```bash
nightshift install
nightshift install launchd
nightshift install systemd
nightshift install cron
```

Supported targets:

| Target | Platform |
|--------|----------|
| `launchd` | macOS |
| `systemd` | Linux user services |
| `cron` | Any Unix-like system |

### `nightshift uninstall`

Removes the installed system service.

```bash
nightshift uninstall
```

### `nightshift completion`

Generates shell completion scripts.

```bash
nightshift completion bash
nightshift completion zsh
nightshift completion fish
nightshift completion powershell
```
