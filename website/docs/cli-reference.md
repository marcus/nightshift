---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

## Core Commands

| Command | Description |
|---------|-------------|
| `nightshift setup` | Guided global configuration |
| `nightshift init` | Create project or global configuration files |
| `nightshift run` | Execute scheduled tasks |
| `nightshift preview` | Show upcoming runs |
| `nightshift budget` | Check token budget status |
| `nightshift task` | Browse and run tasks |
| `nightshift config` | View, update, and validate configuration |
| `nightshift doctor` | Check environment health |
| `nightshift status` | View run history |
| `nightshift logs` | Stream or export logs |
| `nightshift stats` | Token usage statistics |
| `nightshift report` | Show structured run reports |
| `nightshift busfactor` | Analyze code ownership concentration |
| `nightshift daemon` | Background scheduler |
| `nightshift install` | Install a launchd, systemd, or cron service |
| `nightshift uninstall` | Remove the installed system service |
| `nightshift completion` | Generate shell completion scripts |
| `nightshift help` | Show command help |

## Setup and Configuration Commands

```bash
nightshift setup                  # Guided onboarding wizard
nightshift init                   # Create ./nightshift.yaml
nightshift init --global          # Create ~/.config/nightshift/config.yaml
nightshift init --force           # Overwrite an existing config without prompting

nightshift config                 # Show merged global + project config
nightshift config get budget.max_percent
nightshift config get providers.claude.enabled
nightshift config set budget.max_percent 75
nightshift config set logging.level debug --global
nightshift config validate
```

`nightshift init` is the quick file generator. `nightshift setup` is the guided wizard for provider configuration, project selection, budget calibration, and daemon setup.

`nightshift config set` writes to the project config when one exists; otherwise it writes to the global config. Use `--global` to force a global write.

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
nightshift task show lint-fix --project ~/code/myapp
nightshift task run lint-fix --provider claude --project ~/code/myapp
nightshift task run lint-fix --provider codex --dry-run
nightshift task run lint-fix --provider copilot --timeout 45m
nightshift task run lint-fix --provider codex --branch main
```

## Budget Commands

```bash
nightshift budget                 # Current status
nightshift budget --provider claude
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
```

## Reports, Logs, and Stats

```bash
nightshift status                 # Last 5 runs
nightshift status --today         # Today's activity summary
nightshift status -n 10

nightshift report                 # Polished overview for last night
nightshift report --period last-run
nightshift report --period last-7d --runs 0
nightshift report --since "2026-05-01" --until "2026-05-04 09:00"
nightshift report --report tasks --format markdown
nightshift report --format json --paths

nightshift logs                   # Last 50 log lines
nightshift logs --follow
nightshift logs --level warn --component orchestrator
nightshift logs --since "2026-05-01" --summary
nightshift logs --export ./nightshift-logs.jsonl

nightshift stats                  # Aggregate statistics
nightshift stats --period last-7d
nightshift stats --json
```

## Daemon and Service Commands

```bash
nightshift daemon start
nightshift daemon start --foreground
nightshift daemon stop
nightshift daemon status

nightshift install                # Auto-detect launchd, systemd, or cron
nightshift install launchd        # macOS LaunchAgent
nightshift install systemd        # Linux user service + timer
nightshift install cron           # Managed crontab entry
nightshift uninstall              # Remove the installed service
```

`nightshift install` uses the configured schedule when available and falls back to a daily 2 AM run when no config exists.

## Analysis Commands

```bash
nightshift busfactor
nightshift busfactor ~/code/myapp
nightshift busfactor --path ~/code/myapp --json
nightshift busfactor --since 2026-01-01 --until 2026-05-01
nightshift busfactor --file internal/orchestrator --save
nightshift busfactor --db ~/.local/share/nightshift/nightshift.db
```

`busfactor` requires a Git repository. It reports contributor concentration, Herfindahl index, Gini coefficient, and risk level.

## Help and Completion

Nightshift includes Cobra's built-in help and completion commands:

```bash
nightshift --help
nightshift run --help
nightshift help task run
nightshift --version

nightshift completion bash
nightshift completion zsh
nightshift completion fish
nightshift completion powershell
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Verbose output |
| `--help`, `-h` | Show help for a command |
| `--version`, `-v` | Print the Nightshift version |

Provider and timeout flags are command-specific. For example, `--provider` and `--timeout` are used by `nightshift task run`.
