---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

## Root Command

```bash
nightshift --help
nightshift --version
nightshift --verbose
```

| Command | Description |
|---------|-------------|
| `nightshift setup` | Guided global configuration |
| `nightshift init` | Write a starter config file |
| `nightshift config` | Show, get, set, or validate config |
| `nightshift run` | Execute tasks now |
| `nightshift preview` | Show upcoming runs and prompts |
| `nightshift budget` | Check token budget status |
| `nightshift task` | Browse and run tasks |
| `nightshift doctor` | Check environment health |
| `nightshift status` | View run history |
| `nightshift logs` | Stream or export logs |
| `nightshift report` | Summarize recent runs |
| `nightshift stats` | Show aggregate statistics |
| `nightshift busfactor` | Analyze ownership concentration |
| `nightshift daemon` | Manage the background scheduler |
| `nightshift install` | Install a user service |
| `nightshift uninstall` | Remove the user service |

Global flags:

| Flag | Description |
|------|-------------|
| `-h`, `--help` | Show help |
| `-v`, `--version` | Show version |
| `--verbose` | Enable verbose output |

## Init and Config

```bash
nightshift init
nightshift init --global
nightshift init --force

nightshift config
nightshift config get budget.max_percent
nightshift config set budget.max_percent 15
nightshift config set providers.copilot.enabled false --global
nightshift config validate
```

## Run Options

`nightshift run` shows a preflight summary before executing, then prompts for
confirmation in interactive terminals.

```bash
nightshift run                          # Preflight + confirm + execute
nightshift run --yes                    # Skip confirmation
nightshift run --dry-run                # Show preflight, don't execute
nightshift run --max-projects 3         # Process up to 3 projects
nightshift run --max-tasks 2            # Run up to 2 tasks per project
nightshift run --random-task            # Pick a random eligible task
nightshift run --ignore-budget          # Bypass budget limits
nightshift run --project ~/code/myapp   # Target specific project
nightshift run --task lint-fix          # Run a specific task
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show the preflight summary and exit without executing |
| `--yes`, `-y` | `false` | Skip the confirmation prompt |
| `--max-projects` | `1` | Max projects to process |
| `--max-tasks` | `1` | Max tasks per project |
| `--random-task` | `false` | Choose a random eligible task |
| `--ignore-budget` | `false` | Bypass budget checks with a warning |
| `--project`, `-p` | | Target a specific project directory |
| `--task`, `-t` | | Run a specific task by name |

Non-interactive contexts such as daemon runs, cron, or piped output skip the
confirmation prompt automatically.

## Preview Commands

```bash
nightshift preview
nightshift preview -n 3
nightshift preview --long
nightshift preview --explain
nightshift preview --plain
nightshift preview --json
nightshift preview --write ./nightshift-prompts
```

## Task Commands

```bash
nightshift task list
nightshift task list --category pr
nightshift task list --cost low --json
nightshift task show lint-fix
nightshift task show lint-fix --prompt-only
nightshift task run lint-fix --provider claude
nightshift task run lint-fix --provider codex --dry-run
nightshift task run docs-backfill --provider copilot --branch main
```

Task-run flags:

| Flag | Description |
|------|-------------|
| `--provider` | Required provider: `claude`, `codex`, or `copilot` |
| `-p`, `--project` | Project directory to run in |
| `--dry-run` | Print the generated prompt without executing |
| `--timeout` | Execution timeout (default `30m`) |
| `-b`, `--branch` | Base branch for new feature branches |

## Budget Commands

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
```

## Logs, Reports, and Stats

```bash
nightshift logs --tail 100
nightshift logs --follow --component daemon
nightshift logs --since 2026-04-01 --level warn

nightshift report --period last-7d --report tasks
nightshift report --format markdown --paths

nightshift stats
nightshift stats --period last-30d --json
```

## Daemon and Service Commands

```bash
nightshift daemon start
nightshift daemon start --foreground
nightshift daemon start --timeout 45m
nightshift daemon status
nightshift daemon stop

nightshift install launchd
nightshift install systemd
nightshift install cron
nightshift uninstall
```

## Bus Factor

```bash
nightshift busfactor
nightshift busfactor --path ~/code/nightshift
nightshift busfactor --file internal/config/config.go
nightshift busfactor --since 2025-01-01 --json
nightshift busfactor --save
```
