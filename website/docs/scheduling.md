---
sidebar_position: 7
title: Scheduling
---

# Scheduling

Nightshift can run automatically on a schedule or be triggered manually when you want immediate execution.

## Schedule Configuration

Use cron or interval scheduling. Configuration validation rejects a file that sets both. It does not reject a file that sets neither, but `preview` and `daemon start` do because the scheduler requires one.

```yaml
schedule:
  cron: "0 2 * * *"  # Every night at 2am
  # interval: "8h"   # Or run every 8 hours
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/Denver"
  max_projects: 1
  max_tasks: 1
```

- `cron` is a standard five-field expression (`minute hour day-of-month month day-of-week`); seconds are not accepted.
- `interval` is a positive Go duration such as `30m`, `8h`, or `24h`.
- `window` restricts the persistent scheduler to a time range. Start is inclusive and end is exclusive; a start later than end spans midnight.
- `window.timezone` must be an IANA name such as `America/Denver`. An empty value uses the machine's local zone.
- `max_projects` and `max_tasks` override the `nightshift run` defaults only when they are positive and the corresponding CLI flag was not passed.

If you want to bootstrap a schedule from scratch, run `nightshift setup` for the guided path, or `nightshift init` / `nightshift init --global` for a manual path. After editing the schedule, run `nightshift config validate` and `nightshift preview`.

For the interval scheduler, an occurrence outside the window is delayed to the next window start. Cron jobs still fire only at their cron times; a cron trigger outside the window is skipped rather than replayed later. `preview` adjusts displayed out-of-window candidates to the next window start, so use a cron expression that already falls inside the window when you need preview and daemon timing to agree exactly.

## Daemon Mode

Run Nightshift as a persistent background process:

```bash
nightshift daemon start
nightshift daemon start --foreground  # For debugging
nightshift daemon start --timeout 45m
nightshift daemon status
nightshift daemon stop
```

`nightshift daemon start` backgrounds the scheduler by spawning the same command with `--foreground`. `--foreground` keeps it in the current terminal, and `--timeout` defaults to 30m per agent. The daemon requires a configured schedule, writes `~/.local/share/nightshift/nightshift.pid`, removes a stale PID file on `daemon stop`, and escalates from SIGTERM to SIGKILL after ten seconds if necessary.

The persistent daemon resolves only `projects[].path` entries, processes every resolved project not already processed that day, and currently selects up to five tasks per project. It does not apply `schedule.max_projects` or `schedule.max_tasks`. It also runs the automatic snapshot and retention loops described in [Budget Management](/docs/budget).

## Service Lifecycle

Install Nightshift as a system service for automatic startup:

```bash
# Auto-detect the init system
nightshift install

# macOS (launchd)
nightshift install launchd

# Linux (systemd)
nightshift install systemd

# Universal (cron)
nightshift install cron

# Remove the installed service
nightshift uninstall
```

- `nightshift install` auto-detects launchd on macOS, systemd on Linux when `systemctl` is present, and cron otherwise.
- All three installers schedule `nightshift run` directly; they do not start the persistent daemon. This means configured execution windows are not enforced by installed service jobs.
- launchd extracts only literal minute/hour values from `schedule.cron`; without a cron value it uses 02:00. It captures the current `PATH` and writes stdout/stderr under `~/.local/share/nightshift/logs/`.
- systemd creates a user oneshot service and timer. Its cron conversion ignores day-of-week, and its interval conversion is only suitable for simple minute values such as `30m`.
- cron installs the configured cron expression or `0 2 * * *` if none exists. It does not translate `schedule.interval`.
- `nightshift uninstall` checks and removes launchd, systemd, and cron installations; it returns an error if none are found.

Because service generators accept a narrower schedule subset than the persistent scheduler, prefer `daemon start` when intervals, windows, or complex cron expressions matter. Inspect generated service behavior after installation.

## Preview Scheduled Work

```bash
nightshift preview
nightshift preview --runs 5
nightshift preview --project ~/code/myproject
nightshift preview --task lint-fix
nightshift preview --long
nightshift preview --explain
nightshift preview --json
nightshift preview --write ./nightshift-prompts
```

Preview requires a schedule and at least one project (the current directory is the fallback). It does not execute tasks or mark state, although `--write` creates prompt files. It simulates cooldowns across the requested future runs. Provider display uses the first enabled provider in the fixed order Claude, Codex, Copilot; unlike real execution, it does not use `providers.preference` or CLI availability.

## Manual Runs

Skip the scheduler and run immediately:

```bash
nightshift run                          # Preflight summary + confirm + execute
nightshift run --dry-run                # Show preflight summary and exit
nightshift run --yes                    # Skip confirmation prompt
nightshift run --project ~/code/myproject
nightshift run --task lint-fix
nightshift run --max-projects 3 --max-tasks 2  # Process more projects/tasks
nightshift run --random-task            # Pick a random eligible task
nightshift run --ignore-budget          # Bypass budget limits
nightshift run --branch develop         # Base new branches on develop
nightshift run --timeout 45m            # Increase per-agent timeout
nightshift run --no-color               # Disable ANSI colors
```

`nightshift run` shows a preflight summary before executing. In interactive terminals you get a confirmation prompt; `--yes` skips it. Non-TTY contexts such as cron, daemons, and CI skip confirmation automatically.

`--random-task` is mutually exclusive with `--task`. When `--max-projects` or `--max-tasks` is omitted, Nightshift falls back to the values in `schedule.max_projects` and `schedule.max_tasks`. `--branch` defaults to the current branch, and `--timeout` defaults to 30m.

Manual runs ignore `schedule.cron`, `schedule.interval`, and `schedule.window`. `--project` targets exactly one directory and ignores `--max-projects`; `--task` targets exactly one task and ignores `--max-tasks`. `--ignore-budget` bypasses provider allowance checks. Direct `nightshift task run TASK --provider PROVIDER` also runs immediately and does not consult the budget manager.
