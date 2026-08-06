---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

Run `nightshift --help` or `nightshift <command> --help` for the current command surface and flag details. The sections below focus on the commands most people use regularly, with extra detail for reporting and diagnostics.

## Top-Level Commands

| Command | Description |
|---------|-------------|
| `nightshift budget` | Show budget status |
| `nightshift busfactor` | Analyze code ownership concentration (bus factor) |
| `nightshift completion` | Generate shell completions |
| `nightshift config` | Manage configuration |
| `nightshift daemon` | Manage background daemon |
| `nightshift doctor` | Check Nightshift configuration and environment |
| `nightshift init` | Create configuration file |
| `nightshift install` | Install system service |
| `nightshift logs` | View logs |
| `nightshift preview` | Preview the next scheduled runs |
| `nightshift report` | Show what nightshift did |
| `nightshift run` | Execute tasks |
| `nightshift setup` | Interactive onboarding wizard |
| `nightshift stats` | Show aggregate statistics |
| `nightshift status` | Show run history |
| `nightshift task` | Manage and run tasks |
| `nightshift uninstall` | Remove system service |

## Run Options

`nightshift run` shows a preflight summary before executing, then prompts for confirmation in interactive terminals.

```bash
nightshift run                          # Preflight + confirm + execute (1 project, 1 task)
nightshift run --yes                    # Skip confirmation
nightshift run --dry-run                # Show preflight, don't execute
nightshift run --max-projects 3         # Process up to 3 projects
nightshift run --max-tasks 2            # Run up to 2 tasks per project
nightshift run --random-task            # Pick a random eligible task
nightshift run --ignore-budget          # Bypass budget limits (use with caution)
nightshift run --branch develop         # Use a specific base branch for feature work
nightshift run --timeout 45m            # Override per-agent timeout
nightshift run --project ~/code/myapp   # Target specific project (ignores --max-projects)
nightshift run --task lint-fix          # Run specific task (ignores --max-tasks)
```

| Flag | Default | Description |
|------|---------|-------------|
| `--branch`, `-b` | current branch | Base branch for new feature branches |
| `--dry-run` | `false` | Show preflight summary and exit without executing |
| `--yes`, `-y` | `false` | Skip confirmation prompt |
| `--max-projects` | `1` | Max projects to process (ignored when `--project` is set) |
| `--max-tasks` | `1` | Max tasks per project (ignored when `--task` is set) |
| `--random-task` | `false` | Pick a random task from eligible tasks instead of the highest-scored one |
| `--ignore-budget` | `false` | Bypass budget checks with a warning |
| `--no-color` | `false` | Disable colored output |
| `--project`, `-p` | | Target a specific project directory |
| `--task`, `-t` | | Run a specific task by name |
| `--timeout` | `30m` | Per-agent execution timeout |

Non-interactive contexts (daemon, cron, piped output) skip the confirmation prompt automatically.

## Preview Options

```bash
nightshift preview                # Default view
nightshift preview -n 3           # Next 3 runs
nightshift preview --long         # Detailed view
nightshift preview --explain      # With prompt previews
nightshift preview --plain        # No pager
nightshift preview --json         # JSON output
nightshift preview --write ./dir  # Write prompts to files
```

## Task Commands

```bash
nightshift task list              # All tasks
nightshift task list --category pr
nightshift task list --cost low --json
nightshift task show lint-fix
nightshift task show lint-fix --prompt-only
nightshift task run lint-fix --provider claude
nightshift task run lint-fix --provider codex --dry-run
```

## Budget Commands

`nightshift budget snapshot` captures a calibration snapshot from local usage data and, when enabled, tmux-scraped provider percentages.

```bash
nightshift budget                 # Current status
nightshift budget --provider claude
nightshift budget snapshot
nightshift budget history -n 10
nightshift budget calibrate
```

## Shell Completion

Use `nightshift completion <shell>` to print a completion script for your shell. The [Installation](./installation.md) guide covers persistent setup paths for each shell.

```bash
source <(nightshift completion bash)
source <(nightshift completion zsh)
nightshift completion fish | source
```

```powershell
nightshift completion powershell | Out-String | Invoke-Expression
```

## Status

`nightshift status` is the fastest way to check recent run history without opening a full report or scanning raw logs.

```bash
nightshift status
nightshift status --today
nightshift status --last 10
```

| Flag | Description |
|------|-------------|
| `--today` | Show today's activity summary |
| `--last`, `-n` | Show the last N runs |

## Logs

Use `nightshift logs` for real-time debugging, targeted filtering, and exporting raw log output.

```bash
nightshift logs
nightshift logs --summary
nightshift logs --follow
nightshift logs --component scheduler --tail 100
nightshift logs --match budget --since "2026-04-09 22:00"
nightshift logs --export ./nightshift.log
```

| Flag | Description |
|------|-------------|
| `--component` | Filter by component substring |
| `--match` | Filter by message substring |
| `--level` | Minimum log level: `debug`, `info`, `warn`, or `error` |
| `--since`, `--until` | Bound the time window |
| `--tail`, `-n` | Limit how many lines to show |
| `--summary` | Show summary counts instead of full lines |
| `--follow`, `-f` | Stream new log lines in real time |
| `--export`, `-e` | Write matching logs to a file |
| `--raw` | Skip formatting and show raw log lines |
| `--path` | Override the log directory |

## Stats

`nightshift stats` aggregates runs, task outcomes, token usage, budget projections, and per-project activity.

```bash
nightshift stats
nightshift stats --period last-night
nightshift stats --period last-7d
nightshift stats --json
```

| Flag | Description |
|------|-------------|
| `--period`, `-p` | Time period: `all`, `last-7d`, `last-30d`, or `last-night` |
| `--json` | Emit machine-readable JSON |

## Report

`nightshift report` gives you a polished morning-after summary, with alternate report types and output formats when you need to share or automate follow-up.

```bash
nightshift report
nightshift report --report tasks
nightshift report --period today --format markdown
nightshift report --report budget --format json
nightshift report --runs 1 --paths
```

| Flag | Description |
|------|-------------|
| `--report`, `-r` | Report type: `overview`, `tasks`, `projects`, `budget`, or `raw` |
| `--period`, `-p` | Time period: `last-night`, `last-run`, `last-24h`, `last-7d`, `today`, `yesterday`, or `all` |
| `--format` | Output format: `fancy`, `plain`, `markdown`, or `json` |
| `--runs`, `-n` | Limit how many runs to include |
| `--max-items` | Limit highlights per run |
| `--paths` | Include report and log file paths |
| `--since`, `--until` | Bound the time window explicitly |

## Top-Level Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Verbose output |
| `--version`, `-v` | Show the current version |

Most commands add their own flags on top of these, so use `nightshift <command> --help` when you need the exact options for a specific workflow.
