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
| `nightshift report` | View structured run reports |
| `nightshift daemon` | Background scheduler |
| `nightshift config` | View and modify configuration |
| `nightshift init` | Create a config file |
| `nightshift commit` | Conventional Commits tools |
| `nightshift busfactor` | Code ownership analysis |
| `nightshift install` | Install a system service |
| `nightshift uninstall` | Remove the system service |

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

### `budget snapshot`

Capture a usage snapshot for budget calibration. Collects local token counts
(Claude: `stats-cache.json`; Codex: session JSONL files) and optionally scrapes
the CLI's usage display via tmux to get the provider's own usage percentage.
When both are available, nightshift infers the weekly budget:
`budget = local_tokens / (scraped% / 100)`. Tmux scraping requires tmux
installed and `calibrate_enabled: true` in the config.

See [the provider calibration guide](https://github.com/marcus/nightshift/blob/main/docs/guides/provider-calibration.md) for details.

| Flag | Default | Description |
|------|---------|-------------|
| `--local-only` | `false` | Skip tmux scraping and store a local-only snapshot |
| `--provider`, `-p` | | Provider to snapshot (claude, codex, copilot) |

### `budget history`

Show recent usage snapshots for budget calibration.

| Flag | Default | Description |
|------|---------|-------------|
| `--n`, `-n` | `20` | Number of snapshots to show |
| `--provider`, `-p` | | Provider to show history for |

### `budget calibrate`

Show inferred budget calibration status for providers. Accepts `--provider`/`-p`.

## Commit Commands

Tools for working with Conventional Commits messages. See
[docs/commit-messages.md](https://github.com/marcus/nightshift/blob/main/docs/commit-messages.md)
for the full rule set.

### `commit normalize`

Validate and rewrite a commit message into canonical Conventional Commits form
(type prefix, lowercase type, subject length, wrapped body). The message is
read from a positional argument, from a file passed via `--file` (typically
`.git/COMMIT_EDITMSG` from a commit-msg hook), or from stdin when neither is given.

```bash
nightshift commit normalize "feat: add login"
nightshift commit normalize --file .git/COMMIT_EDITMSG
git log -1 --pretty=%B | nightshift commit normalize
```

| Flag | Default | Description |
|------|---------|-------------|
| `--check`, `-c` | `false` | Only validate; do not rewrite (non-zero exit on non-conforming messages) |
| `--file`, `-f` | | Read the message from this file |

## Busfactor Command

Analyze code ownership concentration in a repository or directory. The bus
factor measures how many key contributors are critical to project continuity.

Metrics reported: bus factor (minimum contributors for 50% of commits),
Herfindahl index (0 = diverse, 1 = concentrated), Gini coefficient
(0 = equal, 1 = unequal), and an overall risk level.

```bash
nightshift busfactor                     # Analyze current directory
nightshift busfactor ~/code/myapp        # Analyze a specific repo
nightshift busfactor --file '*.go'       # Limit to a file pattern
nightshift busfactor --since 2026-01-01 --json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--db` | config | Database path |
| `--file`, `-f` | | Analyze a specific file or pattern |
| `--json` | `false` | Output as JSON |
| `--path`, `-p` | | Repository or directory path |
| `--save` | `false` | Save results to the database |
| `--since` | | Start date (RFC3339 or YYYY-MM-DD) |
| `--until` | | End date (RFC3339 or YYYY-MM-DD) |

## Config Commands

View and modify nightshift configuration. The bare `nightshift config`
command shows the current configuration merged from global and project configs.

```bash
nightshift config                          # Show merged config
nightshift config get budget.max_percent   # Read one value
nightshift config set budget.max_percent 15
nightshift config set logging.level debug
nightshift config set providers.claude.enabled false
nightshift config validate                 # Check for errors
```

| Subcommand | Description |
|------------|-------------|
| `config get KEY` | Get a value by key path |
| `config set KEY VALUE` | Set a value by key path (writes to project config if it exists; `--global`/`-g` forces global config) |
| `config validate` | Validate global and project configs |

## Init Command

Initialize a new nightshift configuration file. By default creates
`nightshift.yaml` in the current directory; use `--global` to create the
global config at `~/.config/nightshift/config.yaml`.

```bash
nightshift init            # Create nightshift.yaml here
nightshift init --global   # Create the global config
nightshift init --force    # Overwrite without prompting
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force`, `-f` | `false` | Overwrite an existing config without prompting |
| `--global` | `false` | Create the global config instead of a project config |

## Doctor Command

Run diagnostics to detect configuration and environment issues. Checks config,
scheduling, providers, database health, and budget readiness.

```bash
nightshift doctor
```

## Daemon Commands

Start, stop, or check the nightshift background daemon. The daemon runs the
scheduler loop, executing tasks according to the configured schedule (cron or
interval) and respecting time windows.

```bash
nightshift daemon start           # Start in the background
nightshift daemon start -f        # Run in the foreground
nightshift daemon status          # Is it running?
nightshift daemon stop            # Stop via SIGTERM
```

| Subcommand | Flags |
|------------|-------|
| `daemon start` | `--foreground`, `-f` (run in foreground); `--timeout` (per-agent execution timeout, default 30m) |
| `daemon status` | — |
| `daemon stop` | — |

## Install / Uninstall Commands

Generate and install a system service for nightshift, or remove it.
`install` supports `launchd` (macOS, creates a LaunchAgents plist), `systemd`
(Linux, creates a user unit), and `cron` (universal, creates a crontab entry);
with no argument it auto-detects based on OS.

```bash
nightshift install            # Auto-detect init system
nightshift install launchd    # macOS only
nightshift install systemd    # Linux only
nightshift install cron       # Any OS
nightshift uninstall          # Remove the installed service
```

## Logs Command

View nightshift logs. Displays recent log entries; use `--follow` to stream
in real time. Logs live in `~/.local/share/nightshift/logs/`.

```bash
nightshift logs                        # Last 50 entries
nightshift logs -n 200                 # More entries
nightshift logs --follow               # Stream
nightshift logs --level warn           # Warnings and above
nightshift logs --component scheduler  # Filter by component
nightshift logs --since 2026-08-01 --summary
nightshift logs --export logs.txt      # Export to file
```

| Flag | Default | Description |
|------|---------|-------------|
| `--component` | | Filter by component substring |
| `--export`, `-e` | | Export logs to a file |
| `--follow`, `-f` | `false` | Follow log output |
| `--level` | | Minimum log level (debug\|info\|warn\|error) |
| `--match` | | Filter by message substring |
| `--no-color` | `false` | Disable ANSI colors |
| `--path` | | Override log directory |
| `--raw` | `false` | Show raw log lines without formatting |
| `--since` | | Start time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--summary` | `false` | Show a summary only |
| `--tail`, `-n` | `50` | Number of log lines to show |
| `--until` | | End time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |

## Report Command

View structured reports from recent nightshift runs. By default shows a
polished overview of what happened during the last night.

```bash
nightshift report                          # Last night, fancy output
nightshift report --period last-24h        # Different window
nightshift report --report tasks           # Focus on task outcomes
nightshift report --report budget --json   # Machine-readable
nightshift report --runs 10 --format markdown
```

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `fancy` | Output format: fancy \| plain \| markdown \| json |
| `--max-items` | `5` | Max highlights per run |
| `--no-color` | `false` | Disable ANSI colors |
| `--paths` | `false` | Include report/log file paths |
| `--period`, `-p` | `last-night` | last-night \| last-run \| last-24h \| last-7d \| today \| yesterday \| all |
| `--report`, `-r` | `overview` | Report type: overview \| tasks \| projects \| budget \| raw |
| `--runs`, `-n` | `3` | Max runs to include (0 = all) |
| `--since` | | Start time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--until` | | End time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |

## Status Command

Display nightshift run history and activity. Shows the last N runs
(default 5) or today's activity summary.

```bash
nightshift status           # Last 5 runs
nightshift status -n 20     # Last 20 runs
nightshift status --today   # Today's activity summary
```

| Flag | Default | Description |
|------|---------|-------------|
| `--last`, `-n` | `5` | Show the last N runs |
| `--today` | `false` | Show today's activity summary |

## Stats Command

Display aggregate statistics from all nightshift runs: run counts, task
outcomes, token usage, budget projections, and per-project breakdowns.

```bash
nightshift stats                  # All time
nightshift stats --period last-7d
nightshift stats --json           # Machine-readable
```

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output as JSON |
| `--period`, `-p` | `all` | all \| last-7d \| last-30d \| last-night |

## Setup Command

Interactive onboarding wizard that configures Nightshift end-to-end. Creates or
updates the global config, validates providers, runs a snapshot, previews the
next run, and optionally installs/enables the daemon.

```bash
nightshift setup
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Verbose output |
| `--provider` | Select provider (claude, codex) |
| `--timeout` | Execution timeout (default 30m) |
