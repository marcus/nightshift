# Command Reference

This is a quick reference for every CLI command registered by nightshift, generated from the Cobra command definitions in `cmd/nightshift/commands/`. Run `nightshift <command> --help` for the full flag list.

## Global flags

| Flag | Description |
|------|-------------|
| `--verbose` | Enable verbose output (persistent, available on all commands) |

## Core execution

### `nightshift run`

Execute tasks. See [Run Lifecycle](run-lifecycle.md) for what happens from trigger to finished run.

| Flag | Description |
|------|-------------|
| `--dry-run` | Simulate execution without making changes |
| `-p, --project <path>` | Path to project directory |
| `-t, --task <name>` | Run a specific task by name |
| `--max-projects <n>` | Max projects to process per run (default 1; ignored with `--project`) |
| `--max-tasks <n>` | Max tasks to run per project (default 1; ignored with `--task`) |
| `--ignore-budget` | Bypass budget checks (use with caution) |
| `-y, --yes` | Skip confirmation prompt |
| `--random-task` | Pick a random task from eligible tasks |
| `-b, --branch <name>` | Base branch for new feature branches (defaults to current branch) |
| `--timeout <duration>` | Per-agent execution timeout |
| `--no-color` | Disable colored output |

### `nightshift preview`

Preview the next scheduled runs without executing anything.

| Flag | Description |
|------|-------------|
| `-n, --runs <n>` | Number of upcoming runs to preview (default 3) |
| `-p, --project <path>` | Preview only a specific project path |
| `-t, --task <type>` | Preview only a specific task type |
| `--long` | Show full prompts (default truncates) |
| `--write <dir>` | Write full prompts to a directory |
| `--explain` | Show budget and task-filter explanations |
| `--plain` | Disable gum pager output |
| `--json` | Output JSON (includes full prompts) |

## Tasks

### `nightshift task`

Manage and run tasks. See [Adding Tasks](adding-tasks.md) for the built-in task registry and how to add task types.

| Subcommand | Description |
|------------|-------------|
| `task list` | List available tasks with budget info (`--category`, `--cost`, `--json`) |
| `task show <task-type>` | Show task details and prompt (`--prompt-only`, `--json`, `-p, --project`) |
| `task run <task-type>` | Run a task immediately (`--provider`, `-p, --project`, `--dry-run`, `--timeout`, `-b, --branch`) |

`task run` requires `--provider <claude|codex|copilot>`.

## Budget and calibration

### `nightshift budget`

Show budget status. `--provider, -p` filters to one provider (claude, codex, copilot).

| Subcommand | Description |
|------------|-------------|
| `budget snapshot` | Capture a usage snapshot (`-p, --provider`, `--local-only` to skip tmux scraping) |
| `budget history` | Show recent budget snapshots (`-p, --provider`, `-n <count>`, default 20) |
| `budget calibrate` | Show inferred budget calibration status (`-p, --provider`) |

Budget snapshots combine local token counts with tmux-scraped usage percentages; see [Agent tmux Integration](agent-tmux-integration.md) and [Codex Budget Tracking](codex-budget-tracking.md). The standalone `cmd/provider-calibration` tool is covered in [Provider Calibration](provider-calibration.md).

### `nightshift busfactor`

Analyze code ownership concentration (bus factor) for a repository.

| Flag | Description |
|------|-------------|
| `-p, --path <dir>` | Repository or directory path |
| `--json` | Output as JSON |
| `--since` / `--until` | Date range (RFC3339 or YYYY-MM-DD) |
| `-f, --file <pattern>` | Analyze a specific file or pattern |
| `--save` | Save results to database |
| `--db <path>` | Database path (uses config if not set) |

## Reporting and status

### `nightshift report`

Show what nightshift did.

| Flag | Description |
|------|-------------|
| `-r, --report <type>` | `overview` (default), `tasks`, `projects`, `budget`, or `raw` |
| `-p, --period <period>` | `last-night` (default), `last-run`, `last-24h`, `last-7d`, `today`, `yesterday`, `all` |
| `-n, --runs <n>` | Max runs to include (0 = all, default 3) |
| `--since` / `--until` | Time filters (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339) |
| `--format <fmt>` | `fancy` (default), `plain`, `markdown`, `json` |
| `--no-color` | Disable ANSI colors |
| `--paths` | Include report/log file paths |
| `--max-items <n>` | Max highlights per run (default 5) |

### `nightshift stats`

Show aggregate statistics (`--json`, `-p, --period` for `all`, `last-7d`, `last-30d`, `last-night`).

### `nightshift status`

Show run history (`-n, --last <count>`, default 5; `--today` for today's activity summary).

### `nightshift logs`

View logs.

| Flag | Description |
|------|-------------|
| `-n, --tail <n>` | Number of log lines to show (default 50) |
| `-f, --follow` | Follow log output |
| `-e, --export <file>` | Export logs to file |
| `--since` / `--until` | Time filters |
| `--level <lvl>` | Minimum log level (debug\|info\|warn\|error) |
| `--component <str>` | Filter by component substring |
| `--match <str>` | Filter by message substring |
| `--summary` | Show summary only |
| `--raw` | Raw log lines without formatting |
| `--no-color` | Disable ANSI colors |
| `--path <dir>` | Override log directory |

## Setup and configuration

### `nightshift setup`

Interactive onboarding wizard (task presets and project configuration).

### `nightshift init`

Create a configuration file (`--global` for global config, `-f, --force` to overwrite without prompting).

### `nightshift config`

| Subcommand | Description |
|------------|-------------|
| `config get KEY` | Get a configuration value |
| `config set KEY VALUE` | Set a configuration value (`-g, --global` writes global config) |
| `config validate` | Validate the configuration file |

### `nightshift install [launchd|systemd|cron]` / `nightshift uninstall`

Install or remove the system service that triggers scheduled runs.

## Operations

### `nightshift daemon`

Manage the background daemon: `daemon start` (`-f, --foreground`, `--timeout`), `daemon stop`, `daemon status`.

### `nightshift doctor`

Check nightshift configuration and environment (credentials, providers, tmux availability).

### `nightshift commit`

Conventional Commits helpers. `commit normalize [MESSAGE]` rewrites a message to Conventional Commits format (`-c, --check` validates only; `-f, --file` reads the message from a file, as used by the commit-msg hook). See [Commit Messages](../commit-messages.md).
