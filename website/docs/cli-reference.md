---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

## Command Overview

| Command | Purpose |
|---------|---------|
| `nightshift setup` | Guided onboarding: config, providers, snapshot, preview, optional daemon install |
| `nightshift init` | Create a project or global config file |
| `nightshift config` | Show, edit, and validate config |
| `nightshift run` | Execute configured tasks immediately |
| `nightshift preview` | Preview upcoming runs and prompts |
| `nightshift budget` | Inspect budget status and calibration |
| `nightshift task` | List, inspect, and run individual tasks |
| `nightshift daemon` | Manage the background daemon |
| `nightshift install` | Install a launchd, systemd, or cron service |
| `nightshift uninstall` | Remove the installed service |
| `nightshift doctor` | Check config and environment health |
| `nightshift status` | Show run history |
| `nightshift logs` | Stream or export logs |
| `nightshift stats` | Show aggregate usage statistics |
| `nightshift report` | Generate a run report |
| `nightshift busfactor` | Analyze ownership concentration |

## Bootstrap Commands

### `nightshift setup`

Interactive onboarding. It creates or updates the global config, checks providers, runs a snapshot, previews the next run, and can install or enable the daemon service.

```bash
nightshift setup
```

### `nightshift init`

Create a config file directly instead of using the wizard.

```bash
nightshift init
nightshift init --global
nightshift init --force
```

| Flag | Default | Description |
|------|---------|-------------|
| `--global` | `false` | Create `~/.config/nightshift/config.yaml` instead of `nightshift.yaml` |
| `--force`, `-f` | `false` | Overwrite an existing config without prompting |

### `nightshift config`

Show, read, and modify merged configuration.

```bash
nightshift config
nightshift config get budget.max_percent
nightshift config set providers.copilot.enabled true
nightshift config validate
nightshift config set budget.max_percent 80 --global
```

| Command | Purpose |
|---------|---------|
| `nightshift config` | Print the merged config and source paths |
| `nightshift config get KEY` | Print a single value by dotted key path |
| `nightshift config set KEY VALUE` | Write a value back to config |
| `nightshift config validate` | Validate global, project, and merged config |

`nightshift config set` writes to the project config when one exists, otherwise to the global config. Use `--global` to force global writes.

## Run

`nightshift run` shows a preflight summary before execution. In interactive terminals it prompts for confirmation; non-TTY contexts skip the prompt automatically.

```bash
nightshift run
nightshift run --yes
nightshift run --dry-run
nightshift run --max-projects 3
nightshift run --max-tasks 2
nightshift run --random-task
nightshift run --ignore-budget
nightshift run --project ~/code/myapp --task lint-fix
nightshift run --branch develop --timeout 45m
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show the preflight summary and exit without executing |
| `--project`, `-p` | _unset_ | Target a specific project directory |
| `--task`, `-t` | _unset_ | Run a specific task by name |
| `--max-projects` | `1` | Max projects to process per run, unless `schedule.max_projects` provides a config default |
| `--max-tasks` | `1` | Max tasks to run per project, unless `schedule.max_tasks` provides a config default |
| `--ignore-budget` | `false` | Bypass budget checks with a warning |
| `--yes`, `-y` | `false` | Skip the confirmation prompt |
| `--random-task` | `false` | Pick one eligible task at random |
| `--branch`, `-b` | _current branch_ | Base branch for feature branches and metadata |
| `--timeout` | `30m` | Per-agent execution timeout |
| `--no-color` | `false` | Disable colored output |

`--random-task` is mutually exclusive with `--task`. `--project` and `--task` override the wider `--max-projects` and `--max-tasks` loops.

## Daemon

```bash
nightshift daemon start
nightshift daemon start --foreground
nightshift daemon stop
nightshift daemon status
```

| Command | Purpose |
|---------|---------|
| `nightshift daemon start` | Start the background scheduler |
| `nightshift daemon start --foreground` | Run the daemon in the foreground for debugging |
| `nightshift daemon stop` | Stop the running daemon |
| `nightshift daemon status` | Show PID and runtime status |

`nightshift daemon start` also accepts `--timeout`, which sets the per-agent execution timeout for scheduled runs.

## Service Install

```bash
nightshift install
nightshift install launchd
nightshift install systemd
nightshift install cron
nightshift uninstall
```

| Command | Purpose |
|---------|---------|
| `nightshift install` | Auto-detect the host service manager |
| `nightshift install launchd` | Install a launchd service on macOS |
| `nightshift install systemd` | Install a user systemd service on Linux |
| `nightshift install cron` | Install a cron entry on any platform |
| `nightshift uninstall` | Remove the installed service |

## Task Commands

```bash
nightshift task list
nightshift task list --category pr
nightshift task list --cost low --json
nightshift task show lint-fix
nightshift task show lint-fix --prompt-only
nightshift task run lint-fix --provider claude
nightshift task run lint-fix --provider copilot --dry-run
```

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | _required_ for `task run` | Provider to run against: `claude`, `codex`, or `copilot` |
| `--project`, `-p` | _unset_ | Project directory to use in prompt context |
| `--dry-run` | `false` | Show the generated prompt without executing |
| `--timeout` | `30m` | Execution timeout |
| `--branch`, `-b` | _current branch_ | Base branch for feature branch metadata |
| `--prompt-only` | `false` | For `task show`, print only the raw prompt text |
| `--json` | `false` | Emit JSON for `task list` and `task show` |
| `--category` | _unset_ | Filter task list by category |
| `--cost` | _unset_ | Filter task list by cost tier |

## Budget and Status

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget --provider codex
nightshift budget --provider copilot
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
nightshift status --today
nightshift logs --tail 50
```

| Command | Purpose |
|---------|---------|
| `nightshift budget --provider` | Show a specific provider's budget state |
| `nightshift budget snapshot --local-only` | Record a local-only usage snapshot |
| `nightshift budget history -n N` | Show recent snapshots |
| `nightshift budget calibrate` | Recompute calibration data |
| `nightshift status --today` | Show today's activity summary |
| `nightshift logs` | View or export logs |

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Enable verbose output |
