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
| `nightshift completion` | Generate shell completion for bash, fish, PowerShell, or zsh |
| `nightshift help [command]` | Show command help |

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

For supported configuration fields, `nightshift config set` accepts booleans, integers, and strings. It does not type-check the key against the configuration schema before writing, and the current schema has no floating-point fields. The command writes the value before validating the merged configuration; a validation warning does not roll the edit back.

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

`--random-task` and `--task` are mutually exclusive. When `--max-projects` or `--max-tasks` is omitted, a positive `schedule.max_projects` or `schedule.max_tasks` value replaces the flag's default. `--project` loads that directory's project config and targets only that directory. A manual run ignores the configured schedule and window.

## Daemon and Services

`nightshift daemon` manages the persistent scheduler loop. `daemon start` backgrounds the process by default, `--foreground` keeps it in the current terminal, and `--timeout` defaults to 30m per agent.

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

`nightshift install` installs a scheduled system job. If you do not pass an init system, Nightshift auto-detects one from the current platform.

```bash
nightshift install
nightshift install launchd
nightshift install systemd
nightshift install cron
nightshift uninstall
```

`launchd` targets macOS, `systemd` targets Linux, and `cron` works everywhere. `nightshift uninstall` removes the matching service entry if one is installed.

Installed services execute `nightshift run` directly; they are separate from the persistent daemon. See [Scheduling](/docs/scheduling) for schedule-conversion limitations.

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

| Flag | Default | Description |
|------|---------|-------------|
| `--runs`, `-n` | `3` | Number of upcoming runs; must be positive |
| `--project`, `-p` | configured projects/current directory | Preview one project and load its project config |
| `--task`, `-t` | auto-select | Preview one task type |
| `--long` | `false` | Show full prompts instead of the 400-character preview |
| `--write DIR` | none | Write full prompts to files in `DIR` |
| `--explain` | `false` | Include budget and task-filter diagnostics |
| `--plain` | `false` | Do not use the optional gum pager |
| `--json` | `false` | Emit structured output with full prompts and diagnostics |

Preview requires a schedule and at least one project. It does not run tasks or update task state, although `--write` creates files.

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

`budget snapshot` accepts `--provider`, `-p` and `--local-only`. `budget history` accepts `--provider`, `-p` and `--n`, `-n` (default 20). `budget calibrate` accepts `--provider`, `-p`. Without a provider filter, each command processes all enabled providers.

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

| Command | Flags |
|---------|-------|
| `task list` | `--category` (`pr`, `analysis`, `options`, `safe`, `map`, `emergency`), `--cost` (`low`, `medium`, `high`, `veryhigh`), `--json` |
| `task show TASK` | `--project`, `-p`; `--prompt-only`; `--json` |
| `task run TASK` | required `--provider`; `--project`, `-p`; `--dry-run`; `--timeout` (30m); `--branch`, `-b` |

Direct task execution validates the project path and provider CLI but does not check the budget allowance or scheduled cooldown.

## Reports, Logs, Status, and Statistics

```bash
nightshift status --today
nightshift logs --follow
nightshift stats
nightshift report --period last-night
nightshift budget snapshot --provider claude
nightshift busfactor .
nightshift doctor
```

### `report`

| Flag | Default | Values/meaning |
|------|---------|----------------|
| `--report`, `-r` | `overview` | `overview`, `tasks`, `projects`, `budget`, or `raw` |
| `--period`, `-p` | `last-night` | `last-night`, `last-run`, `last-24h`, `last-7d`, `today`, `yesterday`, or `all` |
| `--runs`, `-n` | `3` | Maximum runs; `0` means all |
| `--since` / `--until` | none | `YYYY-MM-DD`, `YYYY-MM-DD HH:MM`, or RFC3339 bounds |
| `--format` | `fancy` | `fancy`, `plain`, `markdown`, or `json` |
| `--no-color` | `false` | Disable ANSI colors |
| `--paths` | `false` | Include report and log paths |
| `--max-items` | `5` | Highlights shown per run |

### `logs`

| Flag | Default | Description |
|------|---------|-------------|
| `--tail`, `-n` | `50` | Recent matching lines |
| `--follow`, `-f` | `false` | Continue streaming |
| `--export`, `-e` | none | Write selected logs to a file |
| `--since` / `--until` | none | Time bounds in the report date formats |
| `--level` | none | Minimum `debug`, `info`, `warn`, or `error` level |
| `--component` | none | Component substring filter |
| `--match` | none | Message substring filter |
| `--summary` | `false` | Print counts only |
| `--raw` | `false` | Print stored lines without formatting |
| `--no-color` | `false` | Disable ANSI colors |
| `--path` | configured log path | Override the log directory |

### `status` and `stats`

`status --last N` (`-n`, default 5) shows recent run records; `status --today` switches to today's activity summary. `stats --period` (`-p`) accepts `all`, `last-7d`, `last-30d`, or `last-night`; `stats --json` produces machine-readable output.

### `busfactor`

`busfactor [path]` analyzes a Git repository, defaulting to the current directory. `--path`, `-p` is the flag equivalent. Use `--since`/`--until` with RFC3339 or `YYYY-MM-DD`, `--file`/`-f` for a path or pattern, `--json` for structured output, `--save` to store the result, and `--db` to override the configured database.

`doctor` has no command-specific flags. It checks config, schedule, provider paths and binaries, database health, snapshots, tmux, and budget readiness.

## Setup, Init, and Completion

- `setup` runs the interactive end-to-end wizard and has no command-specific flags.
- `init` creates `./nightshift.yaml`; `--global` targets the global path and `--force`, `-f` overwrites without prompting.
- `completion bash|fish|powershell|zsh` emits Cobra's completion script. Run the selected subcommand's `--help` for shell-specific installation instructions.

## Shared Flags

| Flag | Scope | Description |
|------|-------|-------------|
| `--verbose` | Root command | Verbose output |
| `--version`, `-v` | Root command | Print the Nightshift version |
| `--help`, `-h` | Any command | Print contextual help |
