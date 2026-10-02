---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

## Top-Level Commands

| Command | Description |
|---------|-------------|
| `nightshift setup` | Guided end-to-end onboarding |
| `nightshift init` | Create a global or project config file |
| `nightshift config` | Show, edit, or validate config files |
| `nightshift run` | Execute configured tasks now |
| `nightshift preview` | Show upcoming runs |
| `nightshift budget` | Check token budget status |
| `nightshift task` | Browse and run tasks |
| `nightshift doctor` | Check environment health |
| `nightshift status` | View run history |
| `nightshift logs` | Stream or export logs |
| `nightshift stats` | Token usage statistics |
| `nightshift report` | Read run reports |
| `nightshift snapshot` | Capture usage snapshots |
| `nightshift busfactor` | Analyze ownership concentration |
| `nightshift daemon` | Manage the background daemon lifecycle |
| `nightshift install` | Install a launchd/systemd/cron service |
| `nightshift uninstall` | Remove the installed service |

## Bootstrap and Config

`nightshift setup` walks through provider setup, project selection, budget calibration, PATH setup, and daemon installation.

```bash
nightshift setup
```

`nightshift init` creates `nightshift.yaml` in the current directory by default. Use `--global` to create `~/.config/nightshift/config.yaml`, and `--force` to overwrite an existing file without prompting.

```bash
nightshift init
nightshift init --global
nightshift init --force
```

`nightshift config` shows the merged configuration from global and project files, plus environment overrides. `nightshift config set` writes to the project config when one exists, otherwise to the global config. Use `--global` to force the global file.

```bash
nightshift config
nightshift config get budget.max_percent
nightshift config set budget.max_percent 15
nightshift config set --global logging.level debug
nightshift config validate
```

| Subcommand | Description |
|------------|-------------|
| `config` | Show merged config and source paths |
| `config get KEY` | Read a nested value by key path |
| `config set KEY VALUE` | Update a value; use `--global` to force global config |
| `config validate` | Validate global, project, and merged config |

`nightshift config set` accepts booleans, integers, floats, and strings.

## Run Options

`nightshift run` shows a preflight summary, then prompts for confirmation in interactive terminals. Non-TTY contexts skip the prompt automatically.

```bash
nightshift run                              # Preflight + confirm + execute
nightshift run --yes                        # Skip confirmation
nightshift run --dry-run                    # Show preflight summary and exit
nightshift run --project ~/code/myapp       # Target a single project
nightshift run --task lint-fix              # Run a specific task
nightshift run --max-projects 3             # Process up to 3 projects
nightshift run --max-tasks 2                # Run up to 2 tasks per project
nightshift run --random-task                # Pick a random eligible task
nightshift run --ignore-budget              # Bypass budget checks
nightshift run --branch develop             # Base new feature branches on develop
nightshift run --timeout 45m                # Increase per-agent timeout
nightshift run --no-color                   # Disable ANSI colors
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show the preflight summary and exit without executing |
| `--yes`, `-y` | `false` | Skip the confirmation prompt |
| `--project`, `-p` | _(all configured)_ | Target a single project directory |
| `--task`, `-t` | _(auto-select)_ | Run a specific task by name |
| `--max-projects` | `1` | Max projects to process when `--project` is not set |
| `--max-tasks` | `1` | Max tasks per project when `--task` is not set |
| `--random-task` | `false` | Pick one random eligible task instead of the highest-scored task |
| `--ignore-budget` | `false` | Bypass budget checks with a warning |
| `--branch`, `-b` | _(current branch)_ | Base branch for new feature branches |
| `--timeout` | `30m` | Per-agent execution timeout |
| `--no-color` | `false` | Disable colored output |

`--random-task` and `--task` are mutually exclusive. When `--max-projects` or `--max-tasks` is omitted, Nightshift falls back to the values in `schedule.max_projects` and `schedule.max_tasks`.

## Daemon and Services

`nightshift daemon` manages the scheduler loop. `daemon start` backgrounds the process by default, `--foreground` keeps it in the current terminal, and `--timeout` defaults to 30m.

```bash
nightshift daemon start
nightshift daemon start --foreground
nightshift daemon start --timeout 45m
nightshift daemon status
nightshift daemon stop
```

| Subcommand | Description |
|------------|-------------|
| `daemon start` | Start the scheduler in the background by default |
| `daemon start --foreground` | Run the scheduler in the current terminal |
| `daemon start --timeout 45m` | Set the per-agent execution timeout |
| `daemon status` | Show whether the daemon is running |
| `daemon stop` | Stop the running daemon |

`nightshift install` installs the scheduler as a system service. If you do not pass an init system, Nightshift auto-detects one from the current platform.

```bash
nightshift install
nightshift install launchd
nightshift install systemd
nightshift install cron
nightshift uninstall
```

`launchd` targets macOS, `systemd` targets Linux, and `cron` works everywhere. `nightshift uninstall` removes the matching service entry if one is installed.

## Preview

```bash
nightshift preview                # Default view
nightshift preview -n 3           # Next 3 runs
nightshift preview --long          # Detailed prompts
nightshift preview --explain       # Budget and cooldown explanations
nightshift preview --plain         # Disable pager output
nightshift preview --json         # JSON output
nightshift preview --write ./dir   # Write prompts to files
```

## Budget

Budget commands accept `--provider` values of `claude`, `codex`, or `copilot`.

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget --provider copilot
nightshift budget snapshot --local-only
nightshift budget snapshot --provider codex
nightshift budget history -n 10
nightshift budget calibrate
```

| Command | Notes |
|---------|-------|
| `budget` | Show current budget status |
| `budget snapshot` | Capture a usage snapshot for calibration |
| `budget history` | Show recent snapshots |
| `budget calibrate` | Show inferred calibration status |

## Tasks

Task commands also accept `--provider` values of `claude`, `codex`, or `copilot` when running tasks.

```bash
nightshift task list
nightshift task list --category pr
nightshift task list --cost low --json
nightshift task show lint-fix
nightshift task show lint-fix --prompt-only
nightshift task run lint-fix --provider claude
nightshift task run lint-fix --provider copilot --dry-run
nightshift task run lint-fix --provider codex --timeout 45m
nightshift task run lint-fix --provider claude -p ~/code/myapp --branch develop
```

`nightshift task run` requires `--provider`. It accepts `--project`, `--dry-run`, `--timeout` (default 30m), and `--branch` for new feature branches.

## Reports and Diagnostics

```bash
nightshift status --today
nightshift logs --follow
nightshift stats
nightshift report --period last-night
nightshift snapshot --provider claude
nightshift busfactor .
nightshift doctor
```

## Shared Flags

| Flag | Scope | Description |
|------|-------|-------------|
| `--verbose` | Root command | Verbose output |
