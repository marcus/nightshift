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
| `nightshift config` | Manage configuration (get/set/validate) |
| `nightshift run` | Execute scheduled tasks |
| `nightshift preview` | Show upcoming runs |
| `nightshift budget` | Check token budget status |
| `nightshift task` | Browse and run tasks |
| `nightshift report` | Show what nightshift did |
| `nightshift doctor` | Check environment health |
| `nightshift status` | View run history |
| `nightshift logs` | Stream or export logs |
| `nightshift stats` | Token usage statistics |
| `nightshift busfactor` | Analyze code ownership concentration |
| `nightshift commit` | Conventional Commits helpers |
| `nightshift daemon` | Background scheduler |
| `nightshift install` | Install system service |
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

### `task list`

| Flag | Default | Description |
|------|---------|-------------|
| `--category` | | Filter by category (pr, analysis, options, safe, map, emergency) |
| `--cost` | | Filter by cost tier (low, medium, high, veryhigh) |
| `--json` | `false` | Output as JSON |

### `task show <task-type>`

| Flag | Default | Description |
|------|---------|-------------|
| `--prompt-only` | `false` | Output only the raw prompt text |
| `--json` | `false` | Output as JSON |
| `--project`, `-p` | | Project directory (used in prompt context) |

### `task run <task-type>`

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | | Provider to run against (claude, codex, copilot) |
| `--project`, `-p` | | Project directory to run in |
| `--dry-run` | `false` | Show prompt without executing |
| `--timeout` | `30m` | Execution timeout |
| `--branch`, `-b` | current branch | Base branch for new feature branches |

## Budget Commands

```bash
nightshift budget                 # Current status
nightshift budget --provider claude
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
```

### `budget snapshot`

Capture a usage snapshot for budget calibration. Collects local token counts and (optionally) scrapes the provider's usage display via tmux. When both local tokens and a scraped percentage are available, nightshift infers the weekly budget.

| Flag | Default | Description |
|------|---------|-------------|
| `--provider`, `-p` | | Provider to snapshot (claude, codex, copilot) |
| `--local-only` | `false` | Skip tmux scraping and store a local-only snapshot |

### `budget history`

Show recent usage snapshots for budget calibration.

| Flag | Default | Description |
|------|---------|-------------|
| `--provider`, `-p` | | Provider to show history for (claude, codex, copilot) |
| `--n` | `20` | Number of snapshots to show |

### `budget calibrate`

Show inferred budget calibration status for providers.

| Flag | Default | Description |
|------|---------|-------------|
| `--provider`, `-p` | | Provider to calibrate (claude, codex, copilot) |

## Commit Commands

```bash
nightshift commit normalize "feat: add login"
nightshift commit normalize --file .git/COMMIT_EDITMSG
git log -1 --pretty=%B | nightshift commit normalize
```

### `commit normalize [MESSAGE]`

Validate and rewrite a commit message into canonical Conventional Commits form (type prefix, lowercase type, subject length, wrapped body). The message is read from the positional argument, from the file passed via `--file` (typically `.git/COMMIT_EDITMSG`, used by a commit-msg hook), or from stdin when neither is given.

| Flag | Default | Description |
|------|---------|-------------|
| `--check`, `-c` | `false` | Only validate; do not rewrite (non-zero exit when the message does not conform) |
| `--file`, `-f` | | Read the message from this file |

## Bus Factor

```bash
nightshift busfactor                     # Analyze current directory
nightshift busfactor ~/code/myapp        # Analyze a specific repository
nightshift busfactor --file '*.go' --json
nightshift busfactor --since 2026-01-01 --until 2026-06-30
```

### `busfactor [path]`

Analyze code ownership concentration. Reports the bus factor (minimum contributors needed for 50% of commits), the Herfindahl index, the Gini coefficient, and an overall risk level.

| Flag | Default | Description |
|------|---------|-------------|
| `--path`, `-p` | current directory | Repository or directory path |
| `--json` | `false` | Output as JSON |
| `--since` | | Start date (RFC3339 or YYYY-MM-DD) |
| `--until` | | End date (RFC3339 or YYYY-MM-DD) |
| `--file`, `-f` | | Analyze a specific file or pattern |
| `--save` | `false` | Save results to database |
| `--db` | config | Database path |

## Config Commands

```bash
nightshift config                        # Show merged configuration
nightshift config get budget.max_percent
nightshift config set budget.max_percent 15
nightshift config set logging.level debug --global
nightshift config validate
```

### `config get KEY`

Get a specific configuration value by key path (e.g. `budget.max_percent`, `providers.claude.enabled`).

### `config set KEY VALUE`

Set a configuration value by key path. Writes to the project config if it exists, otherwise to global config.

| Flag | Default | Description |
|------|---------|-------------|
| `--global`, `-g` | `false` | Write to global config instead of project config |

### `config validate`

Validate the current configuration, checking both global and project configs for errors.

## Init

```bash
nightshift init            # Create nightshift.yaml in the current directory
nightshift init --global   # Create ~/.config/nightshift/config.yaml
nightshift init --force    # Overwrite existing config without prompting
```

| Flag | Default | Description |
|------|---------|-------------|
| `--global` | `false` | Create global config instead of project config |
| `--force`, `-f` | `false` | Overwrite existing config without prompting |

## Daemon Commands

```bash
nightshift daemon start    # Start the background daemon
nightshift daemon stop     # Stop the running daemon (SIGTERM)
nightshift daemon status   # Check if the daemon is running
```

The daemon runs the scheduler loop, executing tasks according to the configured schedule (cron or interval) and respecting time windows.

### `daemon start`

| Flag | Default | Description |
|------|---------|-------------|
| `--foreground`, `-f` | `false` | Run in foreground (don't daemonize) |
| `--timeout` | | Per-agent execution timeout |

## Service Management

```bash
nightshift install          # Auto-detect init system
nightshift install launchd  # macOS (~/Library/LaunchAgents plist)
nightshift install systemd  # Linux (user systemd unit)
nightshift install cron     # Universal (crontab entry)
nightshift uninstall        # Remove the installed service
```

## Report

```bash
nightshift report                          # Overview of last night
nightshift report --report tasks           # Per-task breakdown
nightshift report --period last-7d         # Wider window
nightshift report --format markdown        # Markdown output
nightshift report --since 2026-09-01 --until 2026-09-18
```

View structured reports from recent nightshift runs. If the default period has no reports, falls back to the most recent run.

| Flag | Default | Description |
|------|---------|-------------|
| `--report`, `-r` | `overview` | Report type: overview \| tasks \| projects \| budget \| raw |
| `--period`, `-p` | `last-night` | Time period: last-night \| last-run \| last-24h \| last-7d \| today \| yesterday \| all |
| `--runs`, `-n` | `3` | Max runs to include (0 = all) |
| `--since` | | Start time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--until` | | End time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--format` | `fancy` | Output format: fancy \| plain \| markdown \| json |
| `--no-color` | `false` | Disable ANSI colors |
| `--paths` | `false` | Include report/log file paths |
| `--max-items` | `5` | Max highlights per run |

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Verbose output |

Provider selection (`--provider`) and timeouts (`--timeout`) are per-command flags — see the command sections above.
