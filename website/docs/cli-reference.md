---
sidebar_position: 8
title: CLI Reference
---

# CLI Reference

Run `nightshift --help` or `nightshift COMMAND --help` for the version installed on your machine.

## Command overview

| Command | Purpose |
|---------|---------|
| `setup` | Run the interactive global onboarding wizard |
| `init` | Create a global or project configuration file |
| `config` | Show, read, write, or validate configuration |
| `run` | Plan and execute configured tasks immediately |
| `preview` | Preview future runs without changing state |
| `task` | List, inspect, or execute one task |
| `budget` | Show budget status, snapshots, and calibration |
| `daemon` | Start, stop, or inspect the persistent scheduler |
| `install` | Install an OS-managed scheduled service |
| `uninstall` | Remove installed launchd, systemd, and cron entries |
| `status` | Show recent run history or today's activity |
| `report` | Render structured run reports |
| `logs` | Read, filter, follow, or export logs |
| `stats` | Show aggregate run and token statistics |
| `doctor` | Diagnose configuration, providers, scheduling, database, and budget |
| `busfactor` | Analyze code ownership concentration |
| `completion` | Generate shell completion scripts |

## Configuration bootstrap

```bash
nightshift setup
nightshift init
nightshift init --global
nightshift init --global --force

nightshift config
nightshift config get budget.max_percent
nightshift config set budget.max_percent 60
nightshift config set providers.copilot.enabled true --global
nightshift config validate
```

`init` creates `nightshift.yaml` in the current directory. `init --global` creates
`~/.config/nightshift/config.yaml`. Without `--global`, `config set` writes to the current
project config when it exists and otherwise writes to the global config. `init --force` skips the
overwrite prompt.

`config` displays the effective configuration with built-in defaults. `config get` reads the
merged file/environment value for a key but returns `key not found` for a value supplied only by
a built-in default. `config set` accepts scalar values only and saves the change before printing
a validation warning, so run `config validate` after edits.

The generated files are opinionated starters, not dumps of built-in defaults. Both templates
include legacy task aliases. The global template also limits provider preference to Claude and
Codex and enables their unattended permission-bypass options. Review either file after creation
and use `nightshift task list` for accepted task slugs.

`config validate` validates each existing file and the merged configuration. It catches schema
rules such as mutually exclusive `schedule.cron` and `schedule.interval`, budget ranges, provider
preference names, task durations, and custom-task fields. It does not parse cron syntax,
`schedule.interval`, or execution-window values. Use `nightshift doctor`, `nightshift preview`,
or `nightshift daemon start --foreground` for scheduler parsing.

## `nightshift run`

`run` builds a preflight plan, showing the base branch, provider, budget, projects, and tasks.
It prompts before execution in an interactive terminal. Non-interactive invocations, including
cron and system services, skip the prompt automatically.

| Flag | Default | Behavior |
|------|---------|----------|
| `--dry-run` | `false` | Show the preflight plan and exit without execution |
| `--project`, `-p` | configured paths or current directory | Target one existing directory, load its `nightshift.yaml`, and ignore `--max-projects` |
| `--task`, `-t` | automatic selection | Run one built-in or configured custom task; ignores `--max-tasks`, task enablement, cooldown, estimated-cost filtering, and the processed-today skip |
| `--max-projects` | `1` | Maximum eligible projects; a positive `schedule.max_projects` supplies the default when the flag is omitted; explicit `0` is unlimited |
| `--max-tasks` | `1` | Maximum selected tasks per project; a positive `schedule.max_tasks` supplies the default when the flag is omitted |
| `--random-task` | `false` | Pick exactly one random eligible task; cannot be combined with `--task` |
| `--ignore-budget` | `false` | Permit selection even when a provider budget is exhausted |
| `--yes`, `-y` | `false` | Skip the interactive confirmation |
| `--branch`, `-b` | current branch of the first project | Set the base branch used for task branches |
| `--timeout` | `30m` | Set the per-agent execution timeout |
| `--no-color` | `false` | Disable colored output; `NO_COLOR` also works |

```bash
nightshift run --dry-run
nightshift run --yes --max-projects 3 --max-tasks 2
nightshift run -p ~/code/myapp -t docs-backfill
nightshift run --random-task
nightshift run --branch develop --timeout 45m
```

## `nightshift preview`

Preview requires a valid schedule. It never executes tasks or records project/task runs, although
it opens the configured database and `--write` deliberately creates prompt files.

| Flag | Default | Behavior |
|------|---------|----------|
| `--runs`, `-n` | `3` | Number of upcoming runs |
| `--project`, `-p` | all | Limit output to a project path |
| `--task`, `-t` | all | Limit output to a task type |
| `--long` | `false` | Show full prompts instead of truncated previews |
| `--write DIR` | unset | Write full prompts to a directory |
| `--explain` | `false` | Include budget and task-filter reasons |
| `--plain` | `false` | Disable the optional `gum` pager |
| `--json` | `false` | Emit JSON, including full prompts |

```bash
nightshift preview --explain
nightshift preview -n 5 --task lint-fix
nightshift preview --json
nightshift preview --write ./nightshift-prompts
```

Preview chooses the first enabled provider in the fixed order Claude, Codex, Copilot. It does not
use `providers.preference` or test the provider executable. Its task plan contains one task per
project per previewed run.

## `nightshift task`

```bash
nightshift task list
nightshift task list --category analysis --cost medium --json
nightshift task show docs-backfill
nightshift task show docs-backfill --prompt-only
nightshift task show docs-backfill --project ~/code/myapp --json
nightshift task run docs-backfill --provider copilot --project ~/code/myapp --dry-run
nightshift task run docs-backfill --provider codex --branch main --timeout 45m
```

`task list` accepts categories `pr`, `analysis`, `options`, `safe`, `map`, and `emergency`.
Its cost filter accepts `low`, `medium`, `high`, and `veryhigh`. `task run` requires one of
`claude`, `codex`, or `copilot`; its default project is the current directory and its default
timeout is `30m`.

The `task` subcommands currently expose built-in tasks only; they do not register `tasks.custom`
from configuration. Use `nightshift run --task CUSTOM_TYPE --project PATH` for a configured
custom task. `task run` is a direct execution path: it does not consult provider
enablement, provider preference, budgets, task enablement, cooldowns, or run history. Even
`--dry-run` checks that the requested provider executable is available. When standalone
`copilot` is absent, this direct path still uses a legacy `gh extension list` gate and can reject
the current built-in `gh copilot` command; install standalone Copilot for `task run`.

## `nightshift budget`

All budget commands accept `--provider`/`-p` with `claude`, `codex`, or `copilot`.

```bash
nightshift budget
nightshift budget --provider codex
nightshift budget snapshot
nightshift budget snapshot --provider claude --local-only
nightshift budget history -n 10
nightshift budget calibrate --provider copilot
```

`budget history` defaults to 20 snapshots. `snapshot --local-only` records provider data without
starting a tmux scraping session. `calibrate` reports inferred budget status; it does not consume
an agent run. Claude and Codex support tmux percentage scraping; Copilot snapshots are local-only.

## Daemon and installed services

The daemon is a persistent scheduler:

```bash
nightshift daemon start
nightshift daemon start --foreground --timeout 45m
nightshift daemon status
nightshift daemon stop
```

It loads global plus current-directory configuration once at startup and does not reload changed
files. At each scheduled time it walks every existing explicit `projects[].path`, skips projects
already processed that day, and selects up to five built-in tasks per project. The daemon does
not register `tasks.custom` and does not use `schedule.max_projects` or `schedule.max_tasks`;
those fields affect `nightshift run` only.

An installed service is OS-managed and invokes `nightshift run` at scheduled times:

```bash
nightshift install
nightshift install launchd
nightshift install systemd
nightshift install cron
nightshift uninstall
```

With no argument, `install` chooses launchd on macOS, systemd on Linux when `systemctl` is
available, and cron otherwise. `uninstall` looks for and removes all three Nightshift service
types.

For predictable installed-service timing, configure a simple five-field cron schedule before
installation. Cron preserves that expression; launchd uses only its numeric minute and hour;
systemd converts only its minute, hour, day-of-month, and month fields. Installed services ignore
execution windows. Interval configuration is fully supported by `daemon start`, but the service
generators do not translate it consistently: launchd and cron fall back to 02:00, while systemd
performs only a simple minute-style conversion. See [Scheduling](/docs/scheduling) for lifecycle
details.

## History, reports, logs, and diagnostics

```bash
nightshift status --last 10
nightshift status --today

nightshift report
nightshift report --report tasks --period last-7d --format markdown
nightshift report --since 2026-07-01 --until 2026-07-08 --runs 0 --paths

nightshift logs --tail 100
nightshift logs --follow
nightshift logs --since "2026-07-25 22:00" --level warn --component daemon
nightshift logs --match "budget exhausted" --summary
nightshift logs --export ./nightshift.log --raw --no-color

nightshift stats --period last-30d
nightshift stats --json
nightshift doctor
```

### `status`, `report`, `logs`, and `stats` flags

| Command | Flag | Default | Behavior |
|---------|------|---------|----------|
| `status` | `--last`, `-n` | `5` | Show the last N database run records |
| `status` | `--today` | `false` | Show today's activity summary instead |
| `report` | `--report`, `-r` | `overview` | Select `overview`, `tasks`, `projects`, `budget`, or `raw` |
| `report` | `--period`, `-p` | `last-night` | Select `last-night`, `last-run`, `last-24h`, `last-7d`, `today`, `yesterday`, or `all` |
| `report` | `--runs`, `-n` | `3` | Limit included runs; `0` means all matching runs |
| `report` | `--since`, `--until` | unset | Use explicit date, local date/time, or RFC3339 boundaries instead of `--period` |
| `report` | `--format` | `fancy` | Select `fancy`, `plain`, `markdown`, or `json` |
| `report` | `--no-color` | `false` | Disable ANSI color |
| `report` | `--paths` | `false` | Include report and log paths |
| `report` | `--max-items` | `5` | Limit highlights shown per run |
| `logs` | `--tail`, `-n` | `50` | Limit output to the last N matching lines; non-positive means unlimited |
| `logs` | `--follow`, `-f` | `false` | Stream appended entries; incompatible with `--summary` and `--until` |
| `logs` | `--export`, `-e` | unset | Write matching logs to a file |
| `logs` | `--since`, `--until` | unset | Apply date, local date/time, or RFC3339 boundaries |
| `logs` | `--level` | unset | Minimum `debug`, `info`, `warn`, or `error` level |
| `logs` | `--component`, `--match` | unset | Filter component or message by substring |
| `logs` | `--summary` | `false` | Show counts and range without entries |
| `logs` | `--raw` | `false` | Print raw stored log lines |
| `logs` | `--no-color` | `false` | Disable ANSI color |
| `logs` | `--path` | configured log path | Read a different log directory |
| `stats` | `--period`, `-p` | `all` | Select `all`, `last-7d`, `last-30d`, or `last-night` |
| `stats` | `--json` | `false` | Emit machine-readable JSON |

Report types are `overview`, `tasks`, `projects`, `budget`, and `raw`. Report formats are
`fancy`, `plain`, `markdown`, and `json`. `--period` accepts `last-night`, `last-run`,
`last-24h`, `last-7d`, `today`, `yesterday`, or `all`; `--since` and `--until` accept a date,
local date/time, or RFC3339 timestamp.

`logs --level` accepts `debug`, `info`, `warn`, or `error`. Statistics periods are `all`,
`last-7d`, `last-30d`, and `last-night`.

`doctor` parses the scheduler and checks enabled Claude/Codex executables, provider data paths,
database state, budget readiness, snapshots, tmux, daemon state, and the detected service files.
It does not test provider authentication or check whether standalone `copilot` or the current
built-in `gh copilot` command is available.

## Bus factor analysis

```bash
nightshift busfactor .
nightshift busfactor --path ~/code/myapp --since 2026-01-01 --json
nightshift busfactor . --file "internal/*.go" --save
nightshift busfactor . --db ~/.local/share/nightshift/nightshift.db
```

The optional positional path and `--path`/`-p` select the repository; the current directory is
the default. `--since` and `--until` accept `YYYY-MM-DD` or RFC3339 boundaries. `--file`/`-f`
limits analysis to a file or pattern, `--json` changes the output, `--save` stores the result,
and `--db` overrides the configured database path used by `--save`.

## Shell completion

Generate a completion script using Cobra's built-in subcommands:

```bash
nightshift completion bash
nightshift completion zsh
nightshift completion fish
nightshift completion powershell
```

## Global flags

`--verbose` is the only Nightshift-wide behavior flag. Cobra also provides `--help`, and the root
command provides `--version`. Provider, timeout, JSON, and formatting flags belong only to the
commands that list them.
