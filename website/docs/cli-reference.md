---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

Nightshift commands are grouped around setup, run planning, task execution,
budget visibility, and operations.

## Top-Level Commands

| Command | Purpose |
|---------|---------|
| `nightshift init` | Create a project or global config file |
| `nightshift setup` | Run the guided onboarding wizard |
| `nightshift config` | View, update, or validate configuration |
| `nightshift run` | Execute scheduled tasks immediately |
| `nightshift preview` | Preview upcoming runs and prompts |
| `nightshift budget` | Show provider budget status and calibration data |
| `nightshift task` | List, inspect, and run task definitions |
| `nightshift doctor` | Check config, service, provider, and budget health |
| `nightshift completion` | Generate shell completion scripts |
| `nightshift help` | Show help for any command |
| `nightshift status` | Show run history and today's activity |
| `nightshift logs` | View, filter, follow, or export logs |
| `nightshift stats` | Show aggregate usage and run statistics |
| `nightshift daemon` | Start, stop, or inspect the background scheduler |
| `nightshift install` | Install a launchd, systemd, or cron service |
| `nightshift uninstall` | Remove the installed service |
| `nightshift report` | Summarize recent run reports |
| `nightshift busfactor` | Analyze contributor ownership concentration |

## Configuration

`nightshift init` writes a starter config. By default it creates
`nightshift.yaml` in the current directory; `--global` writes
`~/.config/nightshift/config.yaml`.

```bash
nightshift init                 # Create ./nightshift.yaml
nightshift init --global        # Create global config
nightshift init --force         # Overwrite without prompting
nightshift setup                # Guided global setup
```

| Command or Flag | Description |
|-----------------|-------------|
| `init --global` | Create the global config instead of a project config |
| `init --force`, `-f` | Overwrite an existing config without confirmation |
| `setup` | Walk through providers, projects, budget, and daemon setup |

`nightshift config` shows the merged global and project config. Use subcommands
for scripting or quick edits.

```bash
nightshift config
nightshift config get budget.max_percent
nightshift config set providers.claude.enabled false
nightshift config set logging.level debug --global
nightshift config validate
```

| Command or Flag | Description |
|-----------------|-------------|
| `config get KEY` | Print a config value by key path |
| `config set KEY VALUE` | Write a value to project config when present, otherwise global config |
| `config set --global`, `-g` | Force writes to global config |
| `config validate` | Validate global and project config files |

## Run

`nightshift run` shows a preflight summary before executing, then prompts for
confirmation in interactive terminals. Non-interactive contexts such as cron,
daemon, or CI auto-confirm.

```bash
nightshift run                          # Preflight + confirm + execute
nightshift run --yes                    # Skip confirmation
nightshift run --dry-run                # Show preflight, do not execute
nightshift run --max-projects 3         # Process up to 3 projects
nightshift run --max-tasks 2            # Run up to 2 tasks per project
nightshift run --random-task            # Pick one random eligible task
nightshift run --ignore-budget          # Bypass budget limits
nightshift run -p ~/code/myapp -t lint-fix
nightshift run --branch develop         # Use develop as the base branch
nightshift run --timeout 45m --no-color
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show preflight summary and exit without executing |
| `--yes`, `-y` | `false` | Skip the confirmation prompt |
| `--project`, `-p` | Configured projects | Target one project directory |
| `--task`, `-t` | Auto-select | Run one task by name |
| `--max-projects` | `1` | Max projects to process, ignored when `--project` is set |
| `--max-tasks` | `1` | Max tasks per project, ignored when `--task` is set |
| `--random-task` | `false` | Pick one random eligible task, mutually exclusive with `--task` |
| `--ignore-budget` | `false` | Bypass budget checks with a warning |
| `--branch`, `-b` | Current branch | Base branch for new feature branches |
| `--timeout` | `30m` | Per-agent execution timeout |
| `--no-color` | `false` | Disable colored output |

## Preview

`nightshift preview` computes upcoming scheduled runs without executing tasks or
modifying state.

```bash
nightshift preview
nightshift preview -n 3
nightshift preview --project ~/code/myapp --task docs-backfill
nightshift preview --long
nightshift preview --explain
nightshift preview --plain
nightshift preview --json
nightshift preview --write ./nightshift-prompts
```

| Flag | Default | Description |
|------|---------|-------------|
| `--runs`, `-n` | `3` | Number of upcoming runs to preview |
| `--project`, `-p` | All configured projects | Preview one project path |
| `--task`, `-t` | All eligible tasks | Preview one task type |
| `--long` | `false` | Show full prompts instead of truncated previews |
| `--write DIR` | | Write full prompts to a directory |
| `--explain` | `false` | Include budget and task-filter explanations |
| `--plain` | `false` | Disable gum pager output |
| `--json` | `false` | Output JSON, including full prompts |

## Tasks

Use `nightshift task` to explore built-in and custom tasks, inspect prompt
context, or run one task immediately.

```bash
nightshift task list
nightshift task list --category pr
nightshift task list --cost low --json
nightshift task show lint-fix
nightshift task show lint-fix --json --project ~/code/myapp
nightshift task show lint-fix --prompt-only
nightshift task run lint-fix --provider claude
nightshift task run docs-backfill --provider codex --project ~/code/myapp
nightshift task run lint-fix --provider codex --branch main --timeout 45m
nightshift task run lint-fix --provider copilot --dry-run
```

| Command or Flag | Description |
|-----------------|-------------|
| `task list` | List task type, name, category, cost, token range, and risk |
| `task list --category` | Filter by `pr`, `analysis`, `options`, `safe`, `map`, or `emergency` |
| `task list --cost` | Filter by `low`, `medium`, `high`, or `veryhigh` |
| `task list --json` | Output task list as JSON |
| `task show TASK` | Show metadata and the planning prompt |
| `task show --prompt-only` | Print only the raw prompt text |
| `task show --json` | Output task details and prompt as JSON |
| `task show --project`, `-p` | Use a project directory in prompt context |
| `task run TASK --provider` | Run with `claude`, `codex`, or `copilot`; required |
| `task run --project`, `-p` | Run from a specific project directory |
| `task run --dry-run` | Show the prompt without executing |
| `task run --timeout` | Execution timeout, default `30m` |
| `task run --branch`, `-b` | Base branch for generated feature branches |

## Budget

`nightshift budget` reports remaining allowance by provider. Snapshot and
calibration subcommands help infer subscription budgets from local usage and
optional tmux scraping.

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget snapshot --provider codex
nightshift budget snapshot --local-only
nightshift budget history --provider claude -n 10
nightshift budget calibrate --provider copilot
```

| Command or Flag | Description |
|-----------------|-------------|
| `budget --provider`, `-p` | Show one provider: `claude`, `codex`, or `copilot` |
| `budget snapshot` | Capture provider usage for calibration |
| `budget snapshot --provider`, `-p` | Snapshot one provider |
| `budget snapshot --local-only` | Skip tmux scraping and store local-only usage |
| `budget history` | Show recent usage snapshots |
| `budget history --provider`, `-p` | Show history for one provider |
| `budget history -n` | Number of snapshots to show, default `20` |
| `budget calibrate` | Show inferred budget calibration status |
| `budget calibrate --provider`, `-p` | Show calibration for one provider |

## Operations

`doctor`, `status`, `logs`, `stats`, and `daemon` cover day-to-day operations.

```bash
nightshift doctor
nightshift status
nightshift status --today
nightshift status -n 20
nightshift logs
nightshift logs --follow --level warn
nightshift logs --since "2026-04-25 22:00" --until "2026-04-26 08:00"
nightshift logs --component run --match "budget" --summary
nightshift logs --export ./nightshift.log --raw
nightshift stats
nightshift stats --period last-7d --json
nightshift daemon start
nightshift daemon start --foreground --timeout 45m
nightshift daemon status
nightshift daemon stop
```

| Command or Flag | Description |
|-----------------|-------------|
| `doctor` | Check config, DB, schedule, service, daemon, CLIs, budget, snapshots, and tmux |
| `status --last`, `-n` | Show last N runs, default `5` |
| `status --today` | Show today's run count, outcomes, tokens, projects, and tasks |
| `logs --tail`, `-n` | Number of log lines to show, default `50` |
| `logs --follow`, `-f` | Stream new log output |
| `logs --export`, `-e` | Export matching logs to a file |
| `logs --since`, `--until` | Filter by `YYYY-MM-DD`, `YYYY-MM-DD HH:MM`, or RFC3339 time |
| `logs --level` | Minimum level: `debug`, `info`, `warn`, or `error` |
| `logs --component` | Filter by component substring |
| `logs --match` | Filter by message substring |
| `logs --summary` | Show summary only; cannot be used with `--follow` |
| `logs --raw` | Print raw log lines without formatting |
| `logs --no-color` | Disable ANSI colors |
| `logs --path` | Override the log directory |
| `stats --period`, `-p` | Use `all`, `last-7d`, `last-30d`, or `last-night` |
| `stats --json` | Output aggregate stats as JSON |
| `daemon start --foreground`, `-f` | Run scheduler in the foreground |
| `daemon start --timeout` | Per-agent execution timeout, default `30m` |
| `daemon status` | Show whether the daemon PID is running |
| `daemon stop` | Send SIGTERM to the running daemon |

## Reports and Analysis

`nightshift report` reads structured run reports. `nightshift busfactor`
analyzes git history to find contributor concentration risk.

```bash
nightshift report
nightshift report --period last-run
nightshift report --report tasks --format markdown
nightshift report --since 2026-04-01 --until 2026-04-26 --runs 0
nightshift report --format json --paths --max-items 10

nightshift busfactor
nightshift busfactor ~/code/myapp
nightshift busfactor --path ~/code/myapp --since 2026-01-01
nightshift busfactor --file "internal/**" --json
nightshift busfactor --save --db ~/.local/share/nightshift/nightshift.db
```

| Command or Flag | Description |
|-----------------|-------------|
| `report --report`, `-r` | Report type: `overview`, `tasks`, `projects`, `budget`, or `raw` |
| `report --period`, `-p` | Period: `last-night`, `last-run`, `last-24h`, `last-7d`, `today`, `yesterday`, or `all` |
| `report --runs`, `-n` | Max runs to include, default `3`; `0` means all |
| `report --since`, `--until` | Explicit time range using date, local datetime, or RFC3339 |
| `report --format` | Output format: `fancy`, `plain`, `markdown`, or `json` |
| `report --no-color` | Disable ANSI colors |
| `report --paths` | Include report and log file paths |
| `report --max-items` | Max highlights per run, default `5` |
| `busfactor [path]` | Analyze a git repository, default current directory |
| `busfactor --path`, `-p` | Repository or directory path |
| `busfactor --json` | Output results as JSON |
| `busfactor --since`, `--until` | Restrict commit history by RFC3339 or `YYYY-MM-DD` |
| `busfactor --file`, `-f` | Analyze a specific file or pattern |
| `busfactor --save` | Save results to the configured database |
| `busfactor --db` | Database path to use when saving |

## Service Installation

`nightshift install` creates a scheduled user service. If the service type is
omitted, Nightshift auto-detects launchd on macOS, systemd on Linux when
available, and cron otherwise.

```bash
nightshift install
nightshift install launchd
nightshift install systemd
nightshift install cron
nightshift uninstall
```

| Command | Description |
|---------|-------------|
| `install [launchd|systemd|cron]` | Install and enable a scheduled service |
| `uninstall` | Remove the installed launchd, systemd, or cron service |

## Shell Completion

`nightshift completion` prints shell completion scripts generated by Cobra.
Use the shell-specific subcommand for one-time loading or installation.

```bash
source <(nightshift completion bash)
source <(nightshift completion zsh)
nightshift completion fish | source
nightshift completion powershell | Out-String | Invoke-Expression

nightshift completion zsh > "$(brew --prefix)/share/zsh/site-functions/_nightshift"
nightshift completion bash --no-descriptions
nightshift help run
```

| Command or Flag | Description |
|-----------------|-------------|
| `completion bash` | Generate Bash completions |
| `completion zsh` | Generate Zsh completions |
| `completion fish` | Generate Fish completions |
| `completion powershell` | Generate PowerShell completions |
| `completion SHELL --no-descriptions` | Disable completion descriptions when supported by the shell subcommand |
| `help COMMAND` | Show command-specific help |

## Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Enable verbose output |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Print the Nightshift version |
