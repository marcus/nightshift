---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

## Core Commands

| Command | Description |
|---------|-------------|
| `nightshift setup` | Guided global configuration |
| `nightshift init` | Create a configuration file |
| `nightshift run` | Execute scheduled tasks |
| `nightshift preview` | Show upcoming runs |
| `nightshift budget` | Check token budget status |
| `nightshift task` | Browse and run tasks |
| `nightshift config` | View and modify configuration |
| `nightshift doctor` | Check environment health |
| `nightshift status` | View run history |
| `nightshift report` | View reports from recent runs |
| `nightshift logs` | Stream or export logs |
| `nightshift stats` | Token usage statistics |
| `nightshift daemon` | Manage background daemon |
| `nightshift install` | Install system service (launchd/systemd/cron) |
| `nightshift uninstall` | Remove system service |
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

## Configuration Commands

```bash
nightshift config get budget.max_percent     # Read a value by key path
nightshift config set budget.max_percent 80  # Write a value by key path
nightshift config validate                   # Validate the configuration file
nightshift init                              # Create a new configuration file
```

| Subcommand | Description |
|------------|-------------|
| `config get KEY` | Get a configuration value by dotted key path |
| `config set KEY VALUE` | Set a configuration value by dotted key path |
| `config validate` | Validate the current configuration file |

`config get`/`config set` read and write project-local config by default. Use
`--global` (`-g`) to target the global config instead:

```bash
nightshift config set budget.max_percent 80 --global
```

## Daemon Commands

```bash
nightshift daemon start              # Start as a background process
nightshift daemon start --foreground # Run in the foreground (for debugging)
nightshift daemon stop               # Stop the running daemon
nightshift daemon status             # Show whether the daemon is running
```

| Subcommand | Description |
|------------|-------------|
| `daemon start` | Start the nightshift daemon |
| `daemon stop` | Stop the running daemon (sends SIGTERM) |
| `daemon status` | Check daemon status |

`daemon start` flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--foreground`, `-f` | `false` | Run in the foreground (don't daemonize) |
| `--timeout` | `30m` | Per-agent execution timeout |

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Verbose output |
| `--provider` | Select provider (claude, codex) |
| `--timeout` | Execution timeout (default 30m) |
