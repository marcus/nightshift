# Nightshift Technical Specification

## Overview

Nightshift is a Go CLI tool that runs overnight to perform AI-powered maintenance tasks on your codebase. It uses remaining daily token budget from Claude Code or Codex subscriptions (derived from local usage data), ensuring you wake up to a cleaner codebase without unexpected costs.

### Goals

- **Budget-aware execution**: Never exceed configured limits; default max 75% of daily budget
- **Multi-project support**: Manage one or many repositories
- **Configurable tasks**: From automated PRs to analysis reports
- **Great DX**: Delightful CLI experience
- **Morning surprise**: Like a gift waiting when you wake up

## Architecture

```
                              +-----------------+
                              |  System Timer   |
                              |  (launchd/      |
                              |   systemd/cron) |
                              +--------+--------+
                                       |
                                       v
+------------------------------------------------------------------------------+
|                            Nightshift Daemon                                  |
|                                                                               |
|  +----------------+    +------------------+    +-------------------+          |
|  |   Scheduler    |--->|  Budget Manager  |--->|  Task Selector    |          |
|  | (cron expr,    |    | (check remaining |    | (priority, stale- |          |
|  |  interval,     |    |  budget, calc    |    |  ness, cost est)  |          |
|  |  time window)  |    |  allowance)      |    |                   |          |
|  +----------------+    +------------------+    +---------+---------+          |
|                                                         |                     |
|                                                         v                     |
|                        +------------------+    +-------------------+          |
|                        |     Logger       |<---|  Agent Spawner    |          |
|                        | (structured logs,|    | (plan->implement  |          |
|                        |  morning report) |    |  ->review loop)   |          |
|                        +------------------+    +-------------------+          |
+------------------------------------------------------------------------------+
                                       |
                                       v
                        +-----------------------------+
                        |     External Agents         |
                        | (claude-code, codex-cli)    |
                        +-----------------------------+
```

### Flow

1. System timer (launchd/systemd) wakes daemon at configured interval
2. Budget Manager checks remaining budget via provider usage data (local caches)
3. If within threshold, Task Selector picks next task
4. Agent Spawner executes: plan -> implement -> review (max 3 iterations)
5. Logger records results; generates morning summary
6. Daemon sleeps until next interval

## Configuration Schema

Configuration file: `nightshift.yaml` (per-project) or `~/.config/nightshift/config.yaml` (global)

```yaml
# Schedule configuration
schedule:
  cron: "0 2 * * *"           # Run at 2 AM daily (cron expression)
  interval: 1h                 # Alternative: run every N hours
  # cron and interval are mutually exclusive; specifying both is an error
  window:                      # Optional time window constraint
    start: "22:00"             # Only run between these hours
    end: "06:00"
    timezone: "America/Denver"
# Budget configuration
#
# How budget modes work:
# - daily: Each night uses up to max_percent of that day's budget (weekly/7).
#          Example: 75% daily = use up to 75% of your daily allotment each night.
# - weekly: Each night uses up to max_percent of the REMAINING weekly budget.
#           Example: 75% weekly on day 1 = 75% of full week. On day 7 = 75% of what's left.
#           With aggressive_end_of_week, this ramps up as the week ends to avoid waste.
#
budget:
  mode: daily                  # daily | weekly (see explanation above)
  max_percent: 75              # Max % of budget to use per run (default: 75)
  aggressive_end_of_week: false # Weekly mode: ramp up spending in last 2 days
  reserve_percent: 5           # Always keep this % in reserve (never touch)
  weekly_tokens: 700000        # Fallback weekly budget (used if provider doesn't expose limits)
  per_provider:                # Optional per-provider overrides
    claude: 700000
    codex: 500000

# Provider configuration (how to track usage)
providers:
  claude:
    enabled: true
    data_path: "~/.claude"     # Path to Claude Code data directory
    # File format details: see claude-code-metadata skill
    # stats-cache.json for aggregates
    # projects/<path>/<session>.jsonl for per-message usage
  codex:
    enabled: true
    data_path: "~/.codex"      # Path to Codex data directory
    # File format parsing reference: ~/code/sidecar/
    # sessions/<year>/<month>/<day>/*.jsonl for session data
    # rate_limits.primary.used_percent for daily tracking
    # rate_limits.secondary.used_percent for weekly tracking

# Projects to manage (or single project mode if omitted)
projects:
  - path: ~/code/myproject
    priority: 1
    tasks:
      - lint
      - docs
  - path: ~/code/library
    priority: 2
    config: .nightshift.yaml   # Per-project override

# Task configuration
tasks:
  enabled:
    - lint
    - docs
    - security
    - test-gaps
    - dead-code
  priorities:
    lint: 1
    security: 2
    docs: 3
  disabled:
    - idea-generator          # Explicitly disable

# Integration points
integrations:
  claude_md: true              # Read claude.md for project context
  agents_md: true              # Read agents.md for agent behavior hints
  task_sources:
    - td:                      # td task management (https://github.com/marcus/td)
        enabled: true
        teach_agent: true      # Include brief td usage in agent prompts
        # Agent learns: td usage --new-session, td start, td log, td handoff, td review
        # AGENTS.md should already have td instructions, but this ensures it
    - github_issues            # GitHub issues with "nightshift" label
    - file: TODO.md            # Custom file

# Logging and reporting
logging:
  level: info                  # debug | info | warn | error
  path: ~/.local/share/nightshift/logs
  format: json                 # json | text

reporting:
  morning_summary: true
  email: null                  # Optional email notification
  slack_webhook: null          # Optional Slack notification
```

## Budget Management Algorithm

### Data Sources

**Claude Code** (`~/.claude/`):
- `stats-cache.json`: Daily aggregates with `messageCount`, `sessionCount`, `toolCallCount`, `tokensByModel`
- `projects/<path>/<session>.jsonl`: Per-message token usage in `usage` field
  - `inputTokens`, `outputTokens`, `cacheReadInputTokens`, `cacheCreationInputTokens`
  - Note: `costUSD` is always 0; requires external Anthropic pricing table
  - Budget baseline: use `budget.per_provider.claude` or `budget.weekly_tokens`

**Codex** (`~/.codex/`):
- `sessions/<year>/<month>/<day>/*.jsonl`: Session data with token_count events
- `sqlite/codex-dev.db`: SQLite with `automations` table (rrule scheduling)
- Rate limits in session JSONL (use directly for budget):
```json
{
  "rate_limits": {
    "primary": { "used_percent": 34.0, "window_minutes": 300, "resets_at": 1769896359 },
    "secondary": { "used_percent": 10.0, "window_minutes": 10080, "resets_at": 1770483159 }
  }
}
```

### Budget Calculation

**Daily Mode**: You get 1/7 of your weekly budget each day. Nightshift takes up to `max_percent` of whatever you haven't used today.

**Weekly Mode**: Nightshift looks at your remaining weekly budget and takes up to `max_percent` of what's left divided by remaining days. With `aggressive_end_of_week`, it spends more in the final days to avoid wasted budget.

**Used Percent Source**:
- Claude Code: derive `used_percent` from `stats-cache.json` aggregates (daily uses current date; weekly uses last 7 days) vs configured weekly budget.
- Codex: use `rate_limits.primary.used_percent` for daily mode and `rate_limits.secondary.used_percent` for weekly mode.

```
weekly_budget = provider_limit_tokens if known else config.budget.per_provider[provider] or config.budget.weekly_tokens

# Daily mode
daily_budget = weekly_budget / 7
available_today = daily_budget * (1 - used_percent / 100)
nightshift_allowance = min(available_today * max_percent / 100, available_today)
budget_base = daily_budget

# Weekly mode
remaining_days = days_until_weekly_reset
remaining_weekly = weekly_budget * (1 - used_percent / 100)
if aggressive_end_of_week && remaining_days <= 2:
    multiplier = 3 - remaining_days  # 2x on day before, 3x on last day
else:
    multiplier = 1
nightshift_allowance = (remaining_weekly / remaining_days) * max_percent / 100 * multiplier
budget_base = remaining_weekly

# Reserve enforcement
reserve_amount = budget_base * reserve_percent / 100
final_allowance = max(0, nightshift_allowance - reserve_amount)
```

### Cost Estimation per Task Type

| Task Category | Estimated Token Cost | Risk Level |
|--------------|---------------------|------------|
| lint, docs backfill | Low (10-50k) | Low |
| security scan, dead code | Medium (50-150k) | Low |
| refactor, PR creation | High (150-500k) | Medium |
| migration rehearsal | Very High (500k+) | High |

## Task Categories

### 1. "It's done - here's the PR"

Fully formed, review-ready artifacts. Deterministic, minimal diff, explainable rationale.

- Linter fixes
- Bug finder & fixer
- Auto DRY refactoring
- API contract verification
- Backward-compatibility checks
- Build time optimization
- Documentation backfiller
- Commit message normalizer
- Changelog synthesizer
- Release note drafter
- ADR drafter

### 2. "Here's what I found"

Completed analysis with conclusions, no code touched. Morning briefing: skimmable, ranked, opinionated.

- Doc drift detector
- Semantic diff explainer
- Dead code detector
- Dependency risk scanner
- Test gap finder
- Test flakiness analyzer
- Logging quality auditor
- Metrics coverage analyzer
- Performance regression spotter
- Cost attribution estimator
- Security foot-gun finder
- PII exposure scanner
- Privacy policy consistency checker
- Schema evolution advisor
- Event taxonomy normalizer
- Roadmap entropy detector
- Bus-factor analyzer
- Knowledge silo detector

### 3. "Here are options - what do you want to do?"

Surfaces judgment calls, tradeoffs, design forks. Crisp options, explicit tradeoffs, cost of inaction.

- Task groomer
- Guide/skill improver
- Idea generator
- Tech-debt classifier
- "Why does this exist?" annotator
- Edge-case enumerator
- Error-message improver
- SLO/SLA candidate suggester
- UX copy sharpener
- Accessibility linting (non-checkbox)
- "Should this be a service?" advisor
- Ownership boundary suggester
- Oncall load estimator

### 4. "I tried it safely"

Required execution/simulation but left no lasting side effects. Rehearsal, not action.

- Migration rehearsal runner
- Integration contract fuzzer
- Golden-path recorder
- Performance profiling runs
- Allocation/hot-path profiling

### 5. "Here's the map"

Pure context laid out cleanly. Slow-burn value that changes reasoning speed.

- Visibility instrumentor
- Repo topology visualizer
- Permissions/auth surface mapper
- Data lifecycle tracer
- Feature flag lifecycle monitor
- CI signal-to-noise scorer
- Historical context summarizer

### 6. "For when things go sideways"

Artifacts you hope to never need. Age quietly until they save you hours.

- Runbook generator
- Rollback plan generator
- Incident postmortem draft generator

## Task Selection Logic

```
1. Filter: enabled tasks only
2. Filter: tasks within budget estimate
3. Score each task:
   score = base_priority
         + staleness_bonus (days since last run * 0.1)
         + context_bonus (mentioned in claude.md/agents.md: +2)
         + task_source_bonus (in td/github issues: +3)
4. Sort by score descending
5. Select top task that fits remaining budget
6. Mark task as assigned (prevent duplicate selection)
```

## Agent Orchestration

### Plan -> Implement -> Review Loop

```
┌─────────────────────────────────────────────────────────┐
│                    Orchestrator                         │
│                                                         │
│  ┌─────────┐    ┌─────────────┐    ┌──────────┐        │
│  │  Plan   │───>│  Implement  │───>│  Review  │        │
│  │  Agent  │    │    Agent    │    │  Agent   │        │
│  └─────────┘    └─────────────┘    └────┬─────┘        │
│                                          │              │
│                                    ┌─────┴─────┐       │
│                                    │  Pass?    │       │
│                                    └─────┬─────┘       │
│                                    Yes   │   No        │
│                              ┌───────────┴──────┐      │
│                              │                  │      │
│                              v                  v      │
│                         ┌────────┐      ┌───────────┐  │
│                         │ Commit │      │ Iteration │  │
│                         │ Result │      │ < 3?      │  │
│                         └────────┘      └─────┬─────┘  │
│                                         Yes   │  No    │
│                                         ┌─────┴────┐   │
│                                         │          │   │
│                                         v          v   │
│                                   (back to    Abandon  │
│                                   Implement)   Task    │
└─────────────────────────────────────────────────────────┘
```

### Sub-agent Spawning Strategy

- Use `claude-code --print` or `codex-cli` for headless execution
- Pass context via stdin or temp files
- Capture structured output (JSON preferred)
- Timeout: 30 minutes per agent invocation
- Max 3 review iterations before abandoning task

## CLI Commands

```bash
# Initialize configuration
nightshift init [--global]              # Create config file (project or global)

# Run tasks
nightshift run [--dry-run] [--project PATH] [--task TASK]
                                        # Execute tasks (or simulate with --dry-run)

# Daemon management
nightshift daemon start                 # Start background daemon
nightshift daemon stop                  # Stop daemon
nightshift daemon status                # Check daemon status

# Status and history
nightshift status [--last N]            # Show last N runs (default: 5)
nightshift status --today               # Today's activity summary

# Configuration
nightshift config                       # Show current config
nightshift config get KEY               # Get specific value
nightshift config set KEY VALUE         # Set value
nightshift config validate              # Validate config file

# System integration
nightshift install [launchd|systemd|cron]
                                        # Generate and install system service
nightshift uninstall                    # Remove system service

# Logs
nightshift logs [--tail N] [--follow]   # View logs
nightshift logs --export FILE           # Export logs to file

# Budget
nightshift budget                       # Show current budget status
nightshift budget --provider claude     # Show specific provider status
```

## Multi-Project Support

### Configuration Hierarchy

1. **Global config**: `~/.config/nightshift/config.yaml`
   - Default settings for all projects
   - List of managed projects

2. **Per-project override**: `.nightshift.yaml` in project root
   - Overrides global settings for this project
   - Can enable/disable specific tasks

### Project Discovery

```yaml
# In global config
projects:
  # Explicit list
  - path: ~/code/project1
  - path: ~/code/project2

  # Or glob pattern
  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archived
```

### Iteration Strategy

- Process projects in priority order
- Split budget proportionally or by priority weight
- Skip project if already processed today
- Track per-project state in `~/.local/share/nightshift/state/`

## Integration Points

### Reading claude.md

```go
// Look for claude.md in project root
// Extract: project description, coding conventions, task hints
// Use for: context in agent prompts, task prioritization
```

### Reading agents.md

```go
// Look for agents.md or AGENTS.md
// Extract: agent behavior preferences, tool restrictions
// Use for: configuring agent behavior, safety constraints
```

### Task List Integration

**td (task management)**:
```bash
td list --format json  # Get tasks
td assign TASK_ID      # Mark as assigned
td complete TASK_ID    # Mark as done
```

**GitHub Issues**:
```bash
gh issue list --label "nightshift" --json number,title,body
gh issue comment NUMBER --body "Working on this..."
```

## Logging and Reporting

### Log Format

```json
{
  "timestamp": "2024-01-15T02:30:00Z",
  "level": "info",
  "component": "task_selector",
  "message": "Selected task",
  "task": "dead-code-detector",
  "project": "/Users/dev/myproject",
  "budget_used": 15234,
  "budget_remaining": 84766
}
```

### Log Location

- Default: `~/.local/share/nightshift/logs/`
- Format: `nightshift-YYYY-MM-DD.log`
- Rotation: 7 days default

### Morning Summary

Generated at end of each night's run:

```markdown
# Nightshift Summary - 2024-01-15

## Budget
- Started with: 100,000 tokens
- Used: 45,234 tokens (45%)
- Remaining: 54,766 tokens

## Projects Processed
1. **myproject** (3 tasks)
2. **library** (1 task)

## Tasks Completed
- [PR #123] Fixed 12 linting issues in myproject
- [Report] Found 3 dead code blocks in library
- [Analysis] Test coverage gaps identified (see report)

## Tasks Skipped (insufficient budget)
- migration-rehearsal (estimated: 200k tokens)

## What's Next?
- Review PR #123 in myproject
- Consider removing dead code in library (see report)
```

## Security Considerations

### No Credentials in Config

- Never store API keys in config files
- Use environment variables: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`
- Use system keychain for sensitive values

### Sandboxed Execution

- Run agents with minimal permissions
- No network access unless explicitly required
- Temporary directory for working files
- Clean up after each task

### Audit Logging

- Log all agent invocations
- Log all file modifications
- Log all git operations
- Immutable log format (append-only)

### Safe Defaults

- Read-only mode by default for first run
- Require explicit `--enable-writes` for first time
- Max 75% budget prevents runaway costs
- No auto-push to remote repositories

## Future Considerations

### Web Dashboard

- Real-time status monitoring
- Historical analytics
- Budget visualization
- Task management UI

### Team/Shared Configs

- Shared team configuration
- Role-based task assignment
- Aggregate reporting across team

### More Providers

- Support for additional AI providers
- Custom provider adapters
- Unified budget tracking across providers

### Enhanced Task Types

- Custom task definitions
- Plugin architecture for tasks
- Community task repository
