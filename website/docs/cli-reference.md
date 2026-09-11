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
| `nightshift report` | Show what nightshift did |

## Configuration Commands

| Command | Description |
|---------|-------------|
| `nightshift config` | Show merged configuration |
| `nightshift config get` | Read a value by key path |
| `nightshift config set` | Write a value by key path |
| `nightshift config validate` | Validate configuration files |
| `nightshift init` | Create a config file from a template |
| `nightshift install` | Install a system service (launchd/systemd/cron) |
| `nightshift uninstall` | Remove the installed system service |

See [Configuration](/docs/configuration) for the config file format and available keys.

### `nightshift config`

With no subcommand, prints the configuration sources (global and project paths) and the merged effective configuration.

```bash
nightshift config                          # Show merged config
nightshift config get budget.max_percent
nightshift config get providers.claude.enabled
nightshift config set budget.max_percent 15
nightshift config set logging.level debug
nightshift config set providers.claude.enabled false
nightshift config set --global budget.weekly_tokens 700000
nightshift config validate
```

`config set` writes to the project config (`nightshift.yaml`) if one exists, otherwise to the global config (`~/.config/nightshift/config.yaml`). Values are parsed as bool, int, or float when they look like one; everything else is written as a string. Environment variables prefixed with `NIGHTSHIFT_` override config values for `config get` (for example `NIGHTSHIFT_BUDGET.MAX_PERCENT`).

| Subcommand | Flags |
|------------|-------|
| `config get KEY` | none |
| `config set KEY VALUE` | `--global`, `-g` — always write to global config instead of project config |
| `config validate` | none |

`config validate` checks the global config, the project config, and the merged result, and exits non-zero when any of them has errors.

### `nightshift init`

Creates a commented config file from a template. By default creates `nightshift.yaml` in the current directory; `--global` creates `~/.config/nightshift/config.yaml` instead. Prompts before overwriting an existing file unless `--force` is given.

```bash
nightshift init                 # Project config in ./nightshift.yaml
nightshift init --global        # Global config
nightshift init --global --force
```

| Flag | Default | Description |
|------|---------|-------------|
| `--global` | `false` | Create global config instead of project config |
| `--force`, `-f` | `false` | Overwrite existing config without prompting |

### `nightshift install` / `nightshift uninstall`

`install` generates and loads a system service that runs `nightshift run` on the schedule from your config (default: 2 AM daily):

- `launchd` — macOS (`~/Library/LaunchAgents/com.nightshift.agent.plist`)
- `systemd` — Linux, user units (`~/.config/systemd/user/nightshift.service` + `nightshift.timer`)
- `cron` — universal (managed crontab entry)

```bash
nightshift install              # Auto-detect launchd/systemd/cron
nightshift install launchd      # Explicit service type
nightshift install cron
nightshift uninstall            # Remove whichever service is installed
```

Neither command takes flags. See [Scheduling](/docs/scheduling) for schedule configuration.

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

| Flag | Default | Description |
|------|---------|-------------|
| `--runs`, `-n` | `3` | Number of upcoming runs to preview |
| `--project`, `-p` | | Preview only a specific project path |
| `--task`, `-t` | | Preview only a specific task type |
| `--long` | `false` | Show full prompts (default shows a truncated preview) |
| `--write` | | Write full prompts to a directory |
| `--explain` | `false` | Show budget and task-filter explanations |
| `--plain` | `false` | Disable gum pager output |
| `--json` | `false` | Output JSON (includes full prompts) |

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

| Subcommand | Flags |
|------------|-------|
| `task list` | `--category` (pr, analysis, options, safe, map, emergency), `--cost` (low, medium, high, veryhigh), `--json` |
| `task show <task-type>` | `--prompt-only`, `--json`, `--project`, `-p` |
| `task run <task-type>` | `--provider` (claude, codex, copilot), `--project`, `-p`, `--dry-run`, `--timeout` (default 30m), `--branch`, `-b` (base branch for new feature branches) |

See the [Task Reference](/docs/task-reference) for all built-in task types.

## Budget Commands

```bash
nightshift budget                 # Current status
nightshift budget --provider claude
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
```

| Subcommand | Flags |
|------------|-------|
| `budget` | `--provider`, `-p` (claude, codex, copilot) |
| `budget snapshot` | `--provider`, `-p`; `--local-only` — skip tmux scraping and store a local-only snapshot |
| `budget history` | `--provider`, `-p`; `-n` — number of snapshots to show (default 20) |
| `budget calibrate` | `--provider`, `-p` |

`budget snapshot` collects local token counts and — when tmux is installed and `calibrate_enabled: true` in config — scrapes the provider CLI's own usage display to infer the weekly budget. `budget calibrate` shows the inferred budget, its source, confidence, and sample count. See [Budget](/docs/budget) for details and the repo guide `docs/guides/provider-calibration.md` for the calibration workflow.

## Status, Logs, and Stats

```bash
nightshift status                 # Last 5 runs
nightshift status -n 20           # Last 20 runs
nightshift status --today         # Today's activity summary
```

```bash
nightshift logs                   # Last 50 lines
nightshift logs -f                # Follow
nightshift logs -n 200            # More lines
nightshift logs --level warn      # Minimum level
nightshift logs --component run   # Filter by component
nightshift logs --match "error"   # Filter by message substring
nightshift logs --summary         # Summary only
nightshift logs --export out.log  # Export to file
```

| `logs` flag | Default | Description |
|-------------|---------|-------------|
| `--tail`, `-n` | `50` | Number of log lines to show |
| `--follow`, `-f` | `false` | Follow log output |
| `--export`, `-e` | | Export logs to file |
| `--since` / `--until` | | Time range (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--level` | | Minimum log level (debug, info, warn, error) |
| `--component` | | Filter by component substring |
| `--match` | | Filter by message substring |
| `--summary` | `false` | Show summary only |
| `--raw` | `false` | Show raw log lines without formatting |
| `--no-color` | `false` | Disable ANSI colors |
| `--path` | | Override log directory |

```bash
nightshift stats                  # All time
nightshift stats -p last-7d       # Last 7 days
nightshift stats --json
```

| `stats` flag | Default | Description |
|------|---------|-------------|
| `--period`, `-p` | `all` | Time period: all, last-7d, last-30d, last-night |
| `--json` | `false` | Output as JSON |

## Daemon Commands

```bash
nightshift daemon start           # Start in background
nightshift daemon start -f        # Run in foreground
nightshift daemon stop            # Stop via SIGTERM
nightshift daemon status          # Check if running
```

| Subcommand | Flags |
|------------|-------|
| `daemon start` | `--foreground`, `-f` — run in foreground (don't daemonize); `--timeout` — per-agent execution timeout (default 30m) |
| `daemon stop` | none |
| `daemon status` | none |

The daemon runs the scheduler loop, executing tasks according to the configured schedule (cron or interval) and respecting time windows.

## Maintenance Commands

### `nightshift doctor`

Runs diagnostics on config, scheduling, service installation, daemon, provider CLIs, database health, budget readiness, snapshots, and tmux availability. Exits non-zero when any check fails.

```bash
nightshift doctor
```

Takes no flags. See [Troubleshooting](/docs/troubleshooting) for common issues it detects.

### `nightshift report`

Shows structured reports from recent runs — by default a polished overview of the last night, falling back to the most recent run when the default period is empty.

```bash
nightshift report                          # Overview of last night
nightshift report --report tasks           # Per-task breakdown
nightshift report --period last-7d         # Wider window
nightshift report --runs 10                # Include up to 10 runs
nightshift report --since 2026-09-01 --until 2026-09-08
nightshift report --format markdown        # Markdown output
nightshift report --format json            # JSON output
```

| Flag | Default | Description |
|------|---------|-------------|
| `--report`, `-r` | `overview` | Report type: overview, tasks, projects, budget, raw |
| `--period`, `-p` | `last-night` | Time period: last-night, last-run, last-24h, last-7d, today, yesterday, all |
| `--runs`, `-n` | `3` | Max runs to include (0 = all) |
| `--since` / `--until` | | Time range (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--format` | `fancy` | Output format: fancy, plain, markdown, json |
| `--no-color` | `false` | Disable ANSI colors |
| `--paths` | `false` | Include report/log file paths |
| `--max-items` | `5` | Max highlights per run |

### `nightshift busfactor`

Analyzes code ownership concentration in a git repository: bus factor (minimum contributors needed for 50% of commits), Herfindahl index, Gini coefficient, and a risk level.

```bash
nightshift busfactor                        # Current directory
nightshift busfactor ~/code/myapp           # Specific repo
nightshift busfactor --since 2026-01-01     # Limit date range
nightshift busfactor --file "*.go" --json   # Single file/pattern, JSON output
nightshift busfactor --save                 # Persist result to the database
```

| Flag | Default | Description |
|------|---------|-------------|
| `--path`, `-p` | current directory | Repository or directory path (or pass as positional argument) |
| `--json` | `false` | Output as JSON |
| `--since` / `--until` | | Date range (RFC3339 or YYYY-MM-DD) |
| `--file`, `-f` | | Analyze a specific file or pattern |
| `--save` | `false` | Save results to the database |
| `--db` | from config | Database path |

See the repo guide `docs/bus-factor.md` for how the metrics are computed, and the `bus-factor` task in the [Task Reference](/docs/task-reference) for the scheduled variant.

### `nightshift commit normalize`

Validates and rewrites a commit message into canonical Conventional Commits form (type prefix, lowercase type, subject length, wrapped body). The message is read from the positional argument, from `--file` (typically `.git/COMMIT_EDITMSG` via a commit-msg hook), or from stdin.

```bash
nightshift commit normalize "feat: add login"
nightshift commit normalize --file .git/COMMIT_EDITMSG
git log -1 --pretty=%B | nightshift commit normalize
nightshift commit normalize --check "Feat: Add login"   # Validate only
```

| Flag | Default | Description |
|------|---------|-------------|
| `--check`, `-c` | `false` | Only validate; do not rewrite (non-zero exit when non-conforming) |
| `--file`, `-f` | | Read the message from this file |

See the repo guide `docs/commit-messages.md` for the full formatting rules.

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Verbose output |
| `--version` | Print the nightshift version |

Provider selection (`--provider`) and timeouts (`--timeout`) are per-command flags — see the individual commands above.
