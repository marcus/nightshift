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

Do not set both. `nightshift config validate` rejects a config containing both values, but it does
not parse either value. `nightshift doctor`, `nightshift preview`, and `nightshift daemon start`
construct the scheduler and catch malformed values. The cron parser accepts minute, hour, day of
month, month, and day of week fields; it does not accept a seconds field. Interval examples
include `30m`, `8h`, and `24h`, and the duration must be positive.

A schedule is optional for manual `run`, `task run`, and configuration commands. It is required
for `preview` and `daemon start`.

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

The scheduler moves an out-of-window next-run preview to the next window start. For an interval
daemon that becomes the next timer firing. Cron jobs are different: the cron engine still fires
only on matching cron times, and an out-of-window firing is skipped rather than deferred to the
displayed window start.

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

The persistent daemon does not use these two fields. It visits every existing explicit
`projects[].path` in configuration order and selects up to five tasks per project.

Task cooldowns and budget filters can reduce the actual count. Projects already processed today
are skipped during automatic selection; specifying `--task` bypasses that processed-today skip.

## Persistent daemon

```bash
nightshift daemon start
nightshift daemon status
nightshift daemon stop
```

`daemon start` loads global plus current-directory configuration, detaches a child process, and
waits for the next cron or interval time; it does not run a task immediately. Configuration is
not reloaded while the process is running. The daemon:

- Writes `~/.local/share/nightshift/nightshift.pid`.
- Runs scheduled work only inside the configured window.
- Selects providers by preference and remaining budget.
- Processes only existing explicit `projects[].path` entries (or the current directory when none
  are configured); project patterns and per-project config merging are not used.
- Captures Claude and Codex snapshots and prunes old snapshots in the background.
- Handles SIGINT and SIGTERM for graceful shutdown.

Use foreground mode to see failures directly:

```bash
nightshift daemon start --foreground --timeout 45m
```

`daemon status` reports the PID, schedule, window, and PID-file path. `daemon stop` sends SIGTERM,
waits up to ten seconds, and sends SIGKILL if the process does not exit.

## OS-managed service

`install` creates an OS scheduler entry that invokes `nightshift run`; it does not start the
persistent daemon. Installed jobs therefore load the global config plus any `nightshift.yaml`
visible from the service's working directory, and they bypass execution windows because manual
`run` does not enforce the scheduler window.

```bash
nightshift install            # auto-detect
nightshift install launchd
nightshift install systemd
nightshift install cron
```

| Type | Installed artifact | Lifecycle |
|------|--------------------|-----------|
| launchd | `~/Library/LaunchAgents/com.nightshift.agent.plist` | Uses numeric hour/minute from `schedule.cron`, unloads an old plist, writes, and loads |
| systemd | `~/.config/systemd/user/nightshift.service` and `nightshift.timer` | Converts part of the schedule to `OnCalendar`, reloads user units, enables, and starts |
| cron | Managed `# nightshift managed cron entry` in the user's crontab | Preserves `schedule.cron` and replaces an older Nightshift entry |

With no type, macOS selects launchd. Linux selects systemd when `systemctl` exists and cron
otherwise; other systems select cron.

The service captures the resolved path of the Nightshift executable. Reinstall after moving or
upgrading a manually placed binary. It also runs non-interactively, so `run` confirmation is
automatically skipped.

The three generators do not implement identical schedule semantics:

- Use a simple five-field cron schedule for predictable installed-service behavior.
- launchd reads only numeric minute and hour. It ignores day, month, weekday, interval, and
  execution-window constraints; without cron it uses 02:00.
- systemd maps interval text with a simple minute-oriented conversion and drops cron weekday.
  Complex cron or non-minute intervals may produce an invalid or unintended timer.
- cron preserves a configured cron expression, ignores interval and execution-window settings,
  and uses `0 2 * * *` when cron is empty.
- If loading configuration fails, the standalone `install` command silently installs the 02:00
  fallback instead of returning the configuration error. Run `config validate` and `doctor`
  first.

Use the persistent daemon when interval or execution-window semantics matter.

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

Automatic `run` still honors provider enablement, executable availability, task enablement,
cooldowns, and budget. `--task` bypasses task enablement, cooldown, cost filtering, and the
processed-today check. `--ignore-budget` bypasses budget exhaustion but does not enable a
disabled provider or make a missing executable available.

`task run` is more direct: its explicit provider and task bypass provider enablement, preference,
budgets, task enablement, cooldowns, and run history. It still requires the executable and a
non-sensitive project path.
