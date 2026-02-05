# Nightshift User Guide

Nightshift is a Go CLI tool that runs AI-powered maintenance tasks on your codebase overnight, using your remaining daily token budget from Claude Code or Codex subscriptions. Wake up to a cleaner codebase without unexpected costs.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage](#usage)
- [Monitoring & Visibility](#monitoring--visibility)
- [Integrations](#integrations)
- [Troubleshooting](#troubleshooting)

---

## Installation

### Prerequisites

- Go 1.23 or later
- Claude Code CLI (`claude`) and/or Codex CLI (`codex`) installed
- API keys configured via environment variables:
  - `ANTHROPIC_API_KEY` for Claude
  - `OPENAI_API_KEY` for Codex

### Build from Source

```bash
git clone https://github.com/marcus/nightshift.git
cd nightshift
go build -o nightshift ./cmd/nightshift
```

Move the binary to your PATH:

```bash
sudo mv nightshift /usr/local/bin/
```

### Verify Installation

```bash
nightshift --version
nightshift --help
```

---

## Quick Start

### 1. Initialize Configuration

For an interactive global setup (recommended):

```bash
nightshift setup
```

Create a project config in your repository:

```bash
cd ~/code/myproject
nightshift init
```

Or create a global config for all projects:

```bash
nightshift init --global
```

### 2. Run a Dry-Run

See what Nightshift would do without making changes:

```bash
nightshift run --dry-run
```

### 3. Run Tasks

Execute tasks manually:

```bash
nightshift run
```

### 4. Install as System Service

Set up automatic overnight runs:

```bash
# macOS
nightshift install launchd

# Linux
nightshift install systemd

# Universal (cron)
nightshift install cron
```

---

## Run Lifecycle

For a detailed walkthrough of what happens in a scheduled run (logging, reports, provider selection, etc.), see:

- `docs/guides/run-lifecycle.md`

---

## Configuration

### Config File Locations

| Type | Location |
|------|----------|
| Global | `~/.config/nightshift/config.yaml` |
| Project | `nightshift.yaml` or `.nightshift.yaml` in project root |

Project configs override global settings.

### Basic Configuration

```yaml
# Schedule when Nightshift runs
schedule:
  cron: "0 2 * * *"           # Run at 2 AM daily
  # OR
  interval: 4h                 # Run every 4 hours

  window:                      # Optional: only run during these hours
    start: "22:00"
    end: "06:00"
    timezone: "America/Denver"

# Budget controls
budget:
  mode: daily                  # daily | weekly
  max_percent: 75              # Max % of budget per run
  reserve_percent: 5           # Always keep this % in reserve

# Enable/disable task types
tasks:
  enabled:
    - lint
    - docs
    - security
    - dead-code
  disabled:
    - idea-generator
```

### Task Cooldowns

Tasks have a minimum interval between runs per project to avoid redundant work. Category defaults:

| Category | Default Interval |
|----------|-----------------|
| PR | 7 days |
| Analysis | 3 days |
| Options | 7 days |
| Safe | 14 days |
| Map | 7 days |
| Emergency | 30 days |

Override intervals per task in your config with `tasks.intervals` using duration strings:

```yaml
tasks:
  intervals:
    lint-fix: "24h"
    docs-backfill: "168h"
    bug-finder: "72h"
```

Use `nightshift preview --explain` to see cooldown status, including which tasks are currently on cooldown and when they become eligible again. When all tasks for a project are on cooldown, the run is skipped with a diagnostic message.

### Budget Controls

Budget controls apply globally unless overridden per provider.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `budget.mode` | string | `daily` | `daily` or `weekly` usage model. |
| `budget.max_percent` | int | `75` | Max % of budget per run. |
| `budget.reserve_percent` | int | `5` | Always keep this % in reserve. |
| `budget.billing_mode` | string | `subscription` | `subscription` or `api`. |
| `budget.calibrate_enabled` | bool | `true` | Enable subscription calibration via snapshots. |
| `budget.snapshot_interval` | duration | `30m` | Automatic snapshot cadence. |
| `budget.snapshot_retention_days` | int | `90` | Snapshot retention window. |
| `budget.week_start_day` | string | `monday` | Week boundary for calibration. |
| `budget.db_path` | string | `~/.local/share/nightshift/nightshift.db` | Override DB path. |

### Example: Subscription Users

```yaml
budget:
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m
  snapshot_retention_days: 90
  week_start_day: monday
```

### Example: API Billing

```yaml
budget:
  billing_mode: api
  weekly_tokens: 1000000
  per_provider:
    claude: 1000000
    codex: 500000
```

### Budget Modes

**Daily Mode** (recommended for most users):
- Each night uses up to `max_percent` of your daily budget (weekly / 7)
- Consistent, predictable usage

**Weekly Mode**:
- Uses `max_percent` of *remaining* weekly budget
- With `aggressive_end_of_week: true`, spends more near week's end to avoid waste

### Provider Configuration

```yaml
providers:
  preference:
    - claude
    - codex
  claude:
    enabled: true
    data_path: "~/.claude"     # Where Claude Code stores usage data
    dangerously_skip_permissions: true
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: true

budget:
  weekly_tokens: 700000        # Fallback if provider doesn't expose limits
  per_provider:
    claude: 700000
    codex: 500000
```

`weekly_tokens` and `per_provider` are authoritative for `billing_mode: api`. For subscription users, they act as a fallback until calibration has enough snapshots.

### Multi-Project Setup

```yaml
# In global config (~/.config/nightshift/config.yaml)
projects:
  - path: ~/code/project1
    priority: 1                # Higher priority = processed first
    tasks:
      - lint
      - docs
  - path: ~/code/project2
    priority: 2

  # Or use glob patterns
  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archived
```

---

## Usage

### Manual Execution

```bash
# Run all enabled tasks
nightshift run

# Dry-run (show what would happen)
nightshift run --dry-run

# Run for specific project
nightshift run --project ~/code/myproject

# Run specific task type
nightshift run --task lint
```

### Preview Upcoming Runs

See what Nightshift would run next without changing state:

```bash
# Preview the next 3 scheduled runs
nightshift preview -n 3

# Show full prompts
nightshift preview --long

# Show budget and task-filter explanations
nightshift preview --explain

# Disable gum pager output
nightshift preview --plain

# Emit JSON (full prompts included)
nightshift preview --json

# Write prompts to files
nightshift preview --write ./nightshift-prompts
```

If `gum` is available, Nightshift pipes preview output through the gum pager (it will attempt a Homebrew install if missing). Use `--plain` to force direct output.

### Daemon Mode

Run Nightshift as a background daemon that executes on schedule:

```bash
# Start the daemon
nightshift daemon start

# Check status
nightshift daemon status

# Stop the daemon
nightshift daemon stop

# Run in foreground (for debugging)
nightshift daemon start --foreground
```

### System Service

For automatic overnight runs, install as a system service:

```bash
# macOS (launchd)
nightshift install launchd

# Linux (systemd)
nightshift install systemd

# Any platform (cron)
nightshift install cron
```

---

## Uninstalling

1. Remove the system service.

```bash
nightshift uninstall
```

2. Remove configs and data (optional).

```bash
rm -rf ~/.config/nightshift ~/.local/share/nightshift
```

3. Remove the binary from your PATH.

```bash
rm "$(which nightshift)"
```

---

## Monitoring & Visibility

### Check Run History

```bash
# Show last 5 runs
nightshift status

# Show last 10 runs
nightshift status --last 10

# Today's summary
nightshift status --today
```

Example output:

```
Last 5 Runs
===========

2024-01-15 02:30  myproject     3 tasks  45.2K tokens  completed
2024-01-14 02:30  myproject     2 tasks  23.1K tokens  completed
2024-01-14 02:35  library       1 task   12.4K tokens  completed
```

### View Logs

```bash
# Show recent logs
nightshift logs

# Show last 100 lines
nightshift logs --tail 100

# Stream logs in real-time
nightshift logs --follow

# Export logs to file
nightshift logs --export nightshift-logs.txt
```

### Check Budget

```bash
# Show all providers
nightshift budget

# Show specific provider
nightshift budget --provider claude
```

Example output (subscription):

```
Budget Status (mode: weekly)
================================

[claude]
  Mode:         weekly
  Weekly:       700.0K tokens (calibrated, high confidence, 8 samples)
  Used:         315.0K (45.0%)
  Remaining:    385.0K tokens
  Days left:    4
  Daytime:      50.0K tokens reserved
  Reserve:      35.0K tokens
  Nightshift:   50.0K tokens available
  Progress:     [#############-----------------] 45.0%
```

Example output (API billing):

```
Budget Status (mode: weekly)
================================

[codex]
  Mode:         weekly
  Weekly:       1.0M tokens (api, high confidence)
  Used:         120.0K (12.0%)
  Remaining:    880.0K tokens
  Days left:    5
  Reserve:      50.0K tokens
  Nightshift:   125.0K tokens available
  Progress:     [###---------------------------] 12.0%
```

### Budget Calibration

Nightshift infers subscription budgets by correlating local token counts with provider usage percentages.

Formula: `inferred_budget = local_tokens / (scraped_pct / 100)`.

Confidence levels:
- `low`: 1-2 samples or high variance.
- `medium`: stable signal across several snapshots.
- `high`: consistent signal across a week or more.

> **Note**: Calibration uses tmux to scrape usage percentages. If tmux is unavailable, snapshots are local-only and budgets fall back to config values.

### Snapshot History

```bash
# Capture a snapshot now
nightshift budget snapshot

# Capture a local-only snapshot (skip tmux)
nightshift budget snapshot --local-only

# Show recent snapshots
nightshift budget history -n 20

# Show calibration status
nightshift budget calibrate
```

### Morning Summary

After each run, Nightshift generates a summary at:
`~/.local/share/nightshift/summaries/nightshift-YYYY-MM-DD.md`

```markdown
# Nightshift Summary - 2024-01-15

## Budget
- Started with: 100,000 tokens
- Used: 45,234 tokens (45%)
- Remaining: 54,766 tokens

## Tasks Completed
- [PR #123] Fixed 12 linting issues in myproject
- [Report] Found 3 dead code blocks in library

## What's Next?
- Review PR #123 in myproject
- Consider removing dead code in library
```

## Integrations

### claude.md / agents.md

Nightshift reads these files from your project root to understand context:

- **claude.md**: Project description, coding conventions, task hints
- **AGENTS.md**: Agent behavior preferences, tool restrictions

Tasks mentioned in these files get a priority bonus (+2).

### GitHub Issues

Label issues with `nightshift` to have them picked up:

```yaml
integrations:
  task_sources:
    - github_issues
```

Nightshift will:
1. Fetch issues with the `nightshift` label
2. Process them as tasks (priority bonus +3)
3. Comment progress on the issue

### td Task Management

Integrate with [td](https://github.com/marcus/td) for task tracking:

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true   # Include `td usage --new-session` + core workflow in prompts
```

---

## Troubleshooting

### Common Issues

**"Something feels off"**
- Run `nightshift doctor` to check config, schedule, and provider health

**"No config file found"**
```bash
nightshift init           # Create project config
nightshift init --global  # Create global config
```

**"Insufficient budget"**
- Check current budget: `nightshift budget`
- Increase `max_percent` in config
- Wait for budget reset (check reset time in output)

**"Calibration confidence is low"**
- Run `nightshift budget snapshot` a few times to collect samples
- Ensure tmux is installed so usage percentages are available
- Keep snapshots running for at least a few days

**"tmux not found"**
- Install tmux or set `budget.billing_mode: api` if you pay per token

**"Week boundary looks wrong"**
- Set `budget.week_start_day` to `monday` or `sunday`

**"Provider not available"**
- Ensure Claude/Codex CLI is installed and in PATH
- Check API key environment variables are set

### Debug Mode

Enable verbose logging:

```bash
nightshift run --verbose
```

Or set log level in config:

```yaml
logging:
  level: debug    # debug | info | warn | error
```

### Safe Defaults

Nightshift includes safety features:

| Feature | Default | Override |
|---------|---------|----------|
| Read-only first run | Yes | `--enable-writes` |
| Max budget per run | 75% | `budget.max_percent` |
| Auto-push to remote | No | Manual only |
| Reserve budget | 5% | `budget.reserve_percent` |

### File Locations

| Type | Location |
|------|----------|
| Run logs | `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log` |
| Audit logs | `~/.local/share/nightshift/audit/audit-YYYY-MM-DD.jsonl` |
| Summaries | `~/.local/share/nightshift/summaries/` |
| Database | `~/.local/share/nightshift/nightshift.db` |
| PID file | `~/.local/share/nightshift/nightshift.pid` |

If `state/state.json` exists from older versions, Nightshift migrates it to the SQLite database and renames the file to `state.json.migrated`.

### Getting Help

```bash
nightshift --help
nightshift <command> --help
```

Report issues: https://github.com/marcus/nightshift/issues
