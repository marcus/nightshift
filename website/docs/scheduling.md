---
sidebar_position: 7
title: Scheduling
---

# Scheduling

Nightshift can run automatically on a schedule or be triggered manually when you want immediate execution.

## Schedule Configuration

Use cron or interval scheduling. Nightshift rejects configs that set both.

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

- `cron` schedules a specific time.
- `interval` repeats runs after a fixed duration.
- `window` restricts execution to a local time range.
- `max_projects` and `max_tasks` provide defaults for scheduled and manual runs when CLI flags are omitted.

If you want to bootstrap a schedule from scratch, run `nightshift setup` for the guided path, or `nightshift init` / `nightshift init --global` for a manual path. After editing the schedule, run `nightshift config validate`.

## Daemon Mode

Run Nightshift as a persistent background process:

```bash
nightshift daemon start
nightshift daemon start --foreground  # For debugging
nightshift daemon start --timeout 45m
nightshift daemon status
nightshift daemon stop
```

`nightshift daemon start` backgrounds the scheduler by default. `--foreground` keeps it in the current terminal, and `--timeout` defaults to 30m if you do not override it. The daemon requires a configured schedule. It writes its PID file to `~/.local/share/nightshift/nightshift.pid` and uses the scheduler loop to launch runs on schedule.

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

- `nightshift install` auto-detects the platform when you do not pass an init system.
- `launchd` targets macOS, `systemd` targets Linux, and `cron` works everywhere.
- `nightshift uninstall` removes the matching launchd, systemd, or cron entry if one is installed.

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
