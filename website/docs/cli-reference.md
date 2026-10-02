---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

## Core Commands

| Command | Description |
|---------|-------------|
| `nightshift setup` | Guided global configuration |
| `nightshift run` | Execute scheduled tasks |
| `nightshift preview` | Show upcoming runs |
| `nightshift budget` | Check token budget status |
| `nightshift task` | Browse and run tasks |
| `nightshift doctor` | Check environment health |
| `nightshift status` | View run history |
| `nightshift logs` | Stream or export logs |
| `nightshift stats` | Token usage statistics |
| `nightshift daemon` | Background scheduler |
| `nightshift config` | View and modify configuration |
| `nightshift init` | Create a nightshift.yaml configuration file |
| `nightshift install` | Install a system service (launchd/systemd/cron) |
| `nightshift uninstall` | Remove the installed system service |
| `nightshift report` | Show structured reports from recent runs |
| `nightshift busfactor` | Analyze code ownership concentration |

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
nightshift run --project ~/code/myapp   # Target specific project (ignores --max-projects)
nightshift run --task lint-fix          # Run specific task (ignores --max-tasks)
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show preflight summary and exit without executing |
| `--yes`, `-y` | `false` | Skip confirmation prompt |
| `--max-projects` | `1` | Max projects to process (ignored when `--project` is set) |
| `--max-tasks` | `1` | Max tasks per project (ignored when `--task` is set) |
| `--random-task` | `false` | Pick a random task from eligible tasks instead of the highest-scored one |
| `--ignore-budget` | `false` | Bypass budget checks with a warning |
| `--project`, `-p` | | Target a specific project directory |
| `--task`, `-t` | | Run a specific task by name |

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

```bash
nightshift budget                 # Current status
nightshift budget --provider claude
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
```

## Config Commands

`nightshift config` shows the merged configuration (global + project) when run without a subcommand.

```bash
nightshift config                                     # Show merged configuration
nightshift config get budget.max_percent              # Get a value by key path
nightshift config get providers.claude.enabled
nightshift config set budget.max_percent 15           # Set a value (writes project config, or global if none exists)
nightshift config set logging.level debug --global    # Force write to global config
nightshift config validate                             # Validate global, project, and merged configs
```

| Flag | Command | Description |
|------|---------|-------------|
| `--global`, `-g` | `config set` | Write to global config instead of project config |

## Project Setup Commands

```bash
nightshift init                          # Create nightshift.yaml in the current directory
nightshift init --global                 # Create global config at ~/.config/nightshift/config.yaml
nightshift init --force                  # Overwrite an existing config without prompting

nightshift install                       # Install a system service, auto-detecting the init system
nightshift install launchd               # macOS: install a LaunchAgent
nightshift install systemd               # Linux: install a user systemd unit
nightshift install cron                  # Universal: install a crontab entry
nightshift uninstall                     # Remove the installed system service
```

| Flag | Command | Default | Description |
|------|---------|---------|-------------|
| `--global` | `init` | `false` | Create global config instead of project config |
| `--force`, `-f` | `init` | `false` | Overwrite existing config without prompting |

## Reporting Commands

```bash
nightshift report                        # Overview of last night's runs
nightshift report --report tasks         # Report type: overview | tasks | projects | budget | raw
nightshift report --period last-7d       # Time period: last-night | last-run | last-24h | last-7d | today | yesterday | all
nightshift report --since 2024-01-01 --until 2024-01-07
nightshift report --format json          # Output format: fancy | plain | markdown | json
nightshift report --paths                # Include report/log file paths

nightshift busfactor                     # Analyze bus factor for the current directory
nightshift busfactor ~/code/myproject    # Analyze a specific repository or directory
nightshift busfactor --since 2024-01-01 --until 2024-06-01
nightshift busfactor --file "internal/**/*.go"
nightshift busfactor --json
nightshift busfactor --save              # Persist results to the database
```

See [docs/bus-factor.md](https://github.com/marcus/nightshift/blob/main/docs/bus-factor.md) for details on the bus factor, Herfindahl index, and Gini coefficient metrics.

| Flag | Command | Default | Description |
|------|---------|---------|-------------|
| `--report`, `-r` | `report` | `overview` | Report type: overview \| tasks \| projects \| budget \| raw |
| `--period`, `-p` | `report` | `last-night` | Time period: last-night \| last-run \| last-24h \| last-7d \| today \| yesterday \| all |
| `--runs`, `-n` | `report` | `3` | Max runs to include (0 = all) |
| `--since` | `report` | | Start time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--until` | `report` | | End time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--format` | `report` | `fancy` | Output format: fancy \| plain \| markdown \| json |
| `--no-color` | `report` | `false` | Disable ANSI colors |
| `--paths` | `report` | `false` | Include report/log file paths |
| `--max-items` | `report` | `5` | Max highlights per run |
| `--path`, `-p` | `busfactor` | | Repository or directory path (can also be passed as the first argument) |
| `--json` | `busfactor` | `false` | Output as JSON |
| `--since` | `busfactor` | | Start date (RFC3339 or YYYY-MM-DD) |
| `--until` | `busfactor` | | End date (RFC3339 or YYYY-MM-DD) |
| `--file`, `-f` | `busfactor` | | Analyze specific file or pattern |
| `--save` | `busfactor` | `false` | Save results to database |
| `--db` | `busfactor` | | Database path (uses config if not set) |

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Verbose output |
| `--provider` | Select provider (claude, codex) |
| `--timeout` | Execution timeout (default 30m) |
