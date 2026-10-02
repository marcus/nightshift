---
sidebar_position: 7
title: Scheduling
---

# Scheduling

Nightshift can run automatically on a schedule.

## Schedule Config

Nightshift supports either cron or interval scheduling. `schedule.cron` and `schedule.interval` are mutually exclusive.

```yaml
schedule:
  cron: "0 2 * * *"  # Every night at 2am
  # interval: "8h"   # Or run every 8 hours
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/Los_Angeles"
  max_projects: 3
  max_tasks: 2
```

`schedule.max_projects` and `schedule.max_tasks` act as defaults for `nightshift run` when the matching CLI flags are omitted.

## Daemon Mode

Run as a persistent background process:

```bash
nightshift daemon start
nightshift daemon start --foreground  # For debugging
nightshift daemon stop
nightshift daemon status
```

`nightshift daemon start` requires a schedule in config. Use `--foreground` when you want to watch the scheduler loop directly.

## System Service

Install as a system service for automatic startup:

```bash
# macOS (launchd)
nightshift install launchd

# Linux (systemd)
nightshift install systemd

# Universal (cron)
nightshift install cron

# Auto-detect based on the host OS
nightshift install

# Remove the installed service
nightshift uninstall
```

`nightshift install` uses the current binary and the configured schedule to create the service files for the active platform. It auto-detects launchd on macOS, systemd on Linux when available, and cron elsewhere.

## Manual Runs

Skip the scheduler and run immediately:

```bash
nightshift run                          # Preflight summary + confirm + execute
nightshift run --dry-run                # Show preflight summary, don't execute
nightshift run --yes                    # Skip confirmation prompt
nightshift run --project ~/code/myproject
nightshift run --task lint-fix
nightshift run --max-projects 3 --max-tasks 2  # Process more projects/tasks
nightshift run --ignore-budget          # Bypass budget limits
```

In interactive terminals, `nightshift run` shows a preflight summary and asks for confirmation before executing. Use `--yes` to skip the prompt (for example, in scripts). Non-TTY contexts auto-skip confirmation.
