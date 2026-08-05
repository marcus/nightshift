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

## Observability and Diagnostics

### Run Status

`nightshift status` reads run history from the Nightshift database. It shows the
five most recent runs by default, or an aggregate for the current day.

```bash
nightshift status
nightshift status --last 10
nightshift status --today
```

| Flag | Default | Description |
|------|---------|-------------|
| `--last`, `-n` | `5` | Show the last N runs |
| `--today` | `false` | Show today's activity summary instead of individual runs |

If both flags are present, `--today` takes precedence.

### Logs

`nightshift logs` reads structured log files from the configured log directory.
It can filter, summarize, follow, or export matching entries.

```bash
nightshift logs
nightshift logs --tail 100 --level warn
nightshift logs --since today --component scheduler
nightshift logs --since "2025-02-10 22:00" --until "2025-02-11 06:00"
nightshift logs --summary --match timeout
nightshift logs --follow --raw
nightshift logs --export ./nightshift-logs.txt
```

| Flag | Default | Description |
|------|---------|-------------|
| `--tail`, `-n` | `50` | Limit output to the last N matching log lines |
| `--follow`, `-f` | `false` | Continue streaming new log entries |
| `--export`, `-e` | _(unset)_ | Export matching logs to a file |
| `--since` | _(unset)_ | Include entries at or after this time |
| `--until` | _(unset)_ | Include entries at or before this time |
| `--level` | _(unset)_ | Minimum level: `debug`, `info`, `warn`, or `error` |
| `--component` | _(unset)_ | Filter by a case-insensitive component substring |
| `--match` | _(unset)_ | Filter by a case-insensitive message substring |
| `--summary` | `false` | Show the matched-log summary without individual entries |
| `--raw` | `false` | Print original log lines without formatting |
| `--no-color` | `false` | Disable ANSI colors |
| `--path` | configured log path | Override the log directory; the fallback is `~/.local/share/nightshift/logs` |

`--since` and `--until` accept `now`, `today`, `yesterday`, `tomorrow`,
`YYYY-MM-DD`, `YYYY-MM-DD HH:MM`, `YYYY-MM-DD HH:MM:SS`, or RFC3339. Values
without an explicit offset use the local timezone. `--summary` cannot be
combined with `--follow`, and `--until` cannot be used while following logs.

### Aggregate Statistics

`nightshift stats` summarizes runs, task outcomes, tokens, budget projections,
and per-project activity. JSON output is available for scripts.

```bash
nightshift stats
nightshift stats --period last-7d
nightshift stats --period last-night --json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--period`, `-p` | `all` | Period: `all`, `last-7d`, `last-30d`, or `last-night` |
| `--json` | `false` | Output machine-readable JSON |

### Run Reports

`nightshift report` renders saved run reports. The default is a styled overview
of up to three runs from the last configured night window.

```bash
nightshift report
nightshift report --period last-run --report tasks
nightshift report --period last-7d --runs 0 --format plain
nightshift report --since yesterday --until now --report budget
nightshift report --period today --format json
nightshift report --period last-night --format markdown
```

| Flag | Default | Description |
|------|---------|-------------|
| `--report`, `-r` | `overview` | View: `overview`, `tasks`, `projects`, `budget`, or `raw` |
| `--period`, `-p` | `last-night` | Period: `last-night`, `last-run`, `last-24h`, `last-7d`, `today`, `yesterday`, or `all` |
| `--runs`, `-n` | `3` | Maximum runs to include; `0` includes all matching runs |
| `--since` | _(unset)_ | Override the period start time |
| `--until` | _(unset)_ | Override the period end time |
| `--format` | `fancy` | Output: `fancy`, `plain`, `markdown`, or `json` |
| `--no-color` | `false` | Disable ANSI colors in styled output |
| `--paths` | `false` | Include report and log file paths |
| `--max-items` | `5` | Maximum highlights shown per run |

Report time values accept the same forms as log times. Values without an
explicit offset use `schedule.window.timezone` when configured, otherwise the
local timezone. Setting either `--since` or `--until` overrides `--period`.
`plain` keeps the styled report layout but disables ANSI colors; `markdown` and
`json` emit dedicated machine-friendly representations.

### Bus Factor

`nightshift busfactor` uses Git history to report contributor concentration,
including bus factor, Herfindahl index, Gini coefficient, and a risk level.

```bash
nightshift busfactor
nightshift busfactor ~/code/myapp --since 2025-01-01
nightshift busfactor --path . --file "internal/**" --json
nightshift busfactor . --save
nightshift busfactor . --save --db ./nightshift.db
```

| Flag | Default | Description |
|------|---------|-------------|
| `[path]` | current directory | Repository or directory to analyze |
| `--path`, `-p` | _(unset)_ | Repository or directory; overrides the positional path |
| `--file`, `-f` | _(unset)_ | Restrict analysis to a file or pattern |
| `--since` | _(unset)_ | Include commits on or after an RFC3339 or `YYYY-MM-DD` date |
| `--until` | _(unset)_ | Include commits on or before an RFC3339 or `YYYY-MM-DD` date |
| `--json` | `false` | Output machine-readable JSON |
| `--save` | `false` | Save the analysis result to the database |
| `--db` | configured database | Override the database path used by `--save` |

`--path` takes precedence over the positional path. JSON mode writes the report
and exits without saving, so use `--save` with the default human-readable mode.

### Doctor

`nightshift doctor` checks configuration loading, database and state access,
scheduling, service and daemon status, enabled provider CLIs and data paths,
budget readiness, snapshots, and tmux availability.

```bash
nightshift doctor
nightshift --verbose doctor
```

The command has no command-specific flags. Warnings are reported without
failing the command; failed checks produce a non-zero exit status.

### Budget Snapshot

Usage snapshots combine local token counts with an optional provider percentage
scraped through tmux. Nightshift stores them for budget calibration and can infer
a weekly budget when both values are available. This is a `budget` subcommand;
there is no top-level `nightshift snapshot` command.

```bash
nightshift budget snapshot
nightshift budget snapshot --provider claude
nightshift budget snapshot --provider codex --local-only
```

| Flag | Default | Description |
|------|---------|-------------|
| `--provider`, `-p` | all enabled providers | Provider: `claude`, `codex`, or `copilot` |
| `--local-only` | `false` | Skip tmux scraping and store only local usage data |

Without `--local-only`, scraping also requires tmux,
`budget.calibrate_enabled: true`, and subscription billing mode. Local-only
snapshots remain available when scraping is disabled.

## Shared Flags

| Flag | Scope | Description |
|------|-------|-------------|
| `--verbose` | Root command | Verbose output |
