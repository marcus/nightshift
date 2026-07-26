---
sidebar_position: 10
title: Troubleshooting
---

# Troubleshooting

Start with:

```bash
nightshift doctor
nightshift config
nightshift config validate
```

`doctor` checks configuration, scheduler parsing, enabled Claude/Codex executables, provider
data, database health, snapshots, and budget readiness. It does not test authentication or check
the Copilot executable. `config` shows which global and current-project files exist before
printing the merged configuration.

## Configuration is missing or loaded from the wrong place

Create the intended file:

```bash
nightshift init --global   # ~/.config/nightshift/config.yaml
nightshift init            # ./nightshift.yaml
```

Only `nightshift.yaml` is recognized as a project config. A file named `.nightshift.yaml` is not
loaded. Run commands from the project root or use `run --project PATH` so Nightshift can load that
project's file.

Configured `projects[].path` entries select directories to work on, but Nightshift does not
automatically load a separate `nightshift.yaml` from each of those directories. Project
`pattern`, `exclude`, `priority`, `tasks`, and `config` fields are also not used by the current
`run`, `preview`, or daemon path resolver.

`config set` writes to the project file only when it already exists in the current directory.
Use `--global` when you intend to change the global file:

```bash
nightshift config set logging.level debug --global
```

## The schedule is invalid

Typical errors are:

- Both `schedule.cron` and `schedule.interval` are set.
- Cron does not have the supported five-field form.
- An interval is not a positive Go duration.
- A window time is not valid `HH:MM`.
- A window timezone is not an installed IANA timezone.

Keep one schedule type and validate:

```yaml
schedule:
  interval: 8h
  window:
    start: "22:00"
    end: "06:00"
    timezone: "America/Los_Angeles"
```

```bash
nightshift config validate
nightshift doctor
nightshift preview --explain
```

`config validate` catches the case where both schedule types are set, but it does not parse cron,
interval, window time, or timezone values. `doctor`, `preview`, and `daemon start` perform that
runtime validation. If `daemon start` says no schedule is configured, add `cron` or `interval`.
Manual runs do not require a schedule.

## Nothing ran

Use a dry run with explanations:

```bash
nightshift preview --explain
nightshift run --dry-run --no-color
```

Common skip reasons include:

- The project was already processed today.
- Every enabled task is on cooldown.
- No task fits the remaining token allowance.
- All preferred providers are disabled, unavailable, or out of budget.
- A configured project path does not exist.

If configuration contains only project patterns, add explicit `path` entries; current execution
commands do not invoke pattern discovery.

To inspect a task independently of automatic selection:

```bash
nightshift task show docs-backfill --project .
nightshift task run docs-backfill --provider claude --project . --dry-run
```

## Budget is exhausted

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget history -n 10
```

Check `budget.max_percent`, `reserve_percent`, `weekly_tokens`, `per_provider`, and the reported
reset boundary. For a deliberate one-off manual run:

```bash
nightshift run --ignore-budget
```

That flag bypasses the exhausted-budget decision, but not provider enablement, executable
discovery, or task validation.

## Calibration is missing or low-confidence

Subscription calibration for Claude and Codex needs local usage data plus provider usage
percentages:

```bash
nightshift budget snapshot --provider claude
nightshift budget history --provider claude
nightshift budget calibrate --provider claude
```

Install `tmux`, enable `budget.calibrate_enabled`, and collect several snapshots over time.
`snapshot --local-only` records token counts but cannot by itself infer a budget from a provider
percentage. Verify the provider `data_path` when Nightshift cannot find local session data.
Copilot does not support tmux percentage scraping; its snapshots are local-only.

If you are billed by API token rather than a subscription, avoid scraping and set explicit
limits:

```yaml
budget:
  billing_mode: api
  weekly_tokens: 700000
  per_provider:
    codex: 500000
```

API billing disables calibration after configuration is loaded.

## A provider is unavailable or unauthenticated

Check executable discovery first:

```bash
command -v claude
command -v codex
command -v copilot
command -v gh
nightshift doctor
```

Then authenticate the provider directly:

```bash
claude                 # Complete Claude Code login
codex                  # Choose ChatGPT sign-in or API-key setup
copilot login
gh auth status         # Legacy gh copilot fallback
gh extension list
```

Nightshift prefers standalone `copilot`. The direct `task run` path checks `gh extension list`
before using the retired fallback, but automatic `run` selection checks only whether `gh` exists
and can fail later when the extension is missing. Install standalone Copilot for unattended use.
A provider `data_path` is not an executable path; changing it will not fix a missing CLI.

For a service-only PATH problem, compare the interactive executable path with the PATH captured
by launchd, systemd, or cron. Nightshift supplements service PATH with common local bin
directories, but a custom installation directory must still be available. Reinstall the service
after correcting PATH:

```bash
nightshift uninstall
nightshift install
```

## An installed service runs at the wrong time

Installed launchd, systemd, and cron jobs do not share all persistent-daemon scheduling
semantics. Use a simple five-field cron expression before `nightshift install`:

```yaml
schedule:
  cron: "0 2 * * *"
```

launchd reads only numeric minute/hour, systemd performs a limited conversion, and cron preserves
the expression. All three ignore execution windows, and launchd/cron fall back to 02:00 for an
interval-only config. Use `nightshift daemon start` when intervals or windows matter.

The standalone installer also falls back to 02:00 if configuration loading fails. Run these
before reinstalling:

```bash
nightshift config validate
nightshift doctor
nightshift uninstall
nightshift install
```

## Daemon lifecycle problems

```bash
nightshift daemon status
nightshift logs --component daemon --level warn --tail 100
nightshift daemon start --foreground
```

The PID file is `~/.local/share/nightshift/nightshift.pid`. A foreground start is the fastest way
to expose config, database, provider, and scheduler errors. `nightshift uninstall` removes
OS-managed scheduled services; it does not replace `nightshift daemon stop` for a detached daemon.

## Inspect logs and reports

```bash
nightshift logs --since "2026-07-25 22:00" --level warn
nightshift logs --match "no provider" --summary
nightshift report --period last-run --format plain --paths
nightshift status --today
```

Set `logging.level: debug` or `NIGHTSHIFT_LOG_LEVEL=debug` for more detail.

Report reproducible problems at
[github.com/marcus/nightshift/issues](https://github.com/marcus/nightshift/issues) with the
Nightshift version, `doctor` output, relevant config with secrets removed, and filtered logs.
