---
sidebar_position: 7
title: Scheduling
---

# Scheduling

Nightshift supports a persistent daemon and OS-managed scheduled runs. Configure one schedule
type before starting the daemon.

## Cron or interval

Use a five-field cron expression:

```yaml
schedule:
  cron: "0 2 * * *"
```

Or use a positive Go duration:

```yaml
schedule:
  interval: 8h
```

Do not set both. `nightshift config validate` rejects a config containing both values. The cron
parser accepts minute, hour, day of month, month, and day of week fields; it does not accept a
seconds field. Interval examples include `30m`, `8h`, and `24h`.

A schedule is optional for manual `run`, `task run`, and configuration commands. It is required
for `daemon start`.

## Execution windows

Restrict scheduler execution to a local or named timezone:

```yaml
schedule:
  cron: "0 * * * *"
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/Los_Angeles"
```

Times use `HH:MM`. The start is inclusive and the end is exclusive. When start is later than end,
the window crosses midnight. If `timezone` is omitted, Nightshift uses the machine's local
timezone. Invalid times and unknown IANA timezone names prevent scheduler creation.

The scheduler moves an out-of-window next-run preview to the next window start. At execution
time it also checks the window before running jobs.

## Project and task limits

```yaml
schedule:
  cron: "0 2 * * *"
  max_projects: 3
  max_tasks: 2
```

Positive `max_projects` and `max_tasks` values become defaults for `nightshift run` when the
matching CLI flags are omitted. With no positive config value, `run` defaults to one eligible
project and one task per project. `--project` ignores the project limit; `--task` ignores the
task limit. `--random-task` always selects exactly one task.

Task cooldowns and budget filters can reduce the actual count. Projects already processed today
are skipped during automatic selection; specifying `--task` bypasses that processed-today skip.

## Persistent daemon

```bash
nightshift daemon start
nightshift daemon status
nightshift daemon stop
```

`daemon start` detaches a child process. It waits for the next cron or interval time; it does not
run a task immediately. The daemon:

- Writes `~/.local/share/nightshift/nightshift.pid`.
- Runs scheduled work only inside the configured window.
- Selects providers by preference and remaining budget.
- Captures and prunes calibration snapshots in the background.
- Handles SIGINT and SIGTERM for graceful shutdown.

Use foreground mode to see failures directly:

```bash
nightshift daemon start --foreground --timeout 45m
```

`daemon status` reports the PID, schedule, window, and PID-file path. `daemon stop` sends SIGTERM,
waits up to ten seconds, and sends SIGKILL if the process does not exit.

## OS-managed service

`install` creates an OS scheduler entry that invokes `nightshift run`; it does not start the
persistent daemon.

```bash
nightshift install            # auto-detect
nightshift install launchd
nightshift install systemd
nightshift install cron
```

| Type | Installed artifact | Lifecycle |
|------|--------------------|-----------|
| launchd | `~/Library/LaunchAgents/com.nightshift.agent.plist` | Unloads any previous plist, writes it, then loads it |
| systemd | `~/.config/systemd/user/nightshift.service` and `nightshift.timer` | Reloads user units, then enables and starts the timer |
| cron | Managed `# nightshift managed cron entry` in the user's crontab | Replaces an older Nightshift entry |

With no type, macOS selects launchd. Linux selects systemd when `systemctl` exists and cron
otherwise; other systems select cron.

The service captures the resolved path of the Nightshift executable. Reinstall after moving or
upgrading a manually placed binary. It also runs non-interactively, so `run` confirmation is
automatically skipped.

Remove installed services with:

```bash
nightshift uninstall
```

`uninstall` checks launchd, systemd, and cron and removes every Nightshift installation it finds.
Use `daemon stop` separately for a daemon started with `daemon start`.

## Manual overrides

Manual runs bypass schedule timing and windows:

```bash
nightshift run --dry-run
nightshift run --yes
nightshift run --project ~/code/myproject --task lint-fix
nightshift run --max-projects 3 --max-tasks 2
nightshift run --ignore-budget
```

They still honor provider availability, task eligibility, cooldowns, and budget unless the
relevant explicit override applies. `--ignore-budget` does not enable a disabled provider or make
a missing executable available.
