---
sidebar_position: 5
title: Tasks
---

# Tasks

Nightshift includes 59 built-in tasks organized by category. See the [Task Reference](/docs/task-reference) for a complete list.

## Browse Tasks

```bash
# List all tasks
nightshift task list

# Filter by category
nightshift task list --category pr
nightshift task list --category analysis

# Filter by cost tier
nightshift task list --cost low
nightshift task list --cost medium

# Show task details
nightshift task show lint-fix
nightshift task show lint-fix --prompt-only
```

## Categories

| Category | Description |
|----------|-------------|
| `pr` | Creates PRs with code changes |
| `analysis` | Produces analysis reports without code changes |
| `options` | Suggests improvements for human review |
| `safe` | Low-risk automated fixes |
| `map` | Codebase mapping and documentation |
| `emergency` | Critical issues (security vulnerabilities) |

## Cost Tiers

| Tier | Token Usage | Examples |
|------|-------------|----------|
| `low` | Minimal | lint-fix, docs-backfill |
| `medium` | Moderate | dead-code, test-gap |
| `high` | Significant | bug-finder, security-footgun |
| `veryhigh` | Large | migration-rehearsal, contract-fuzzer |

## Run a Single Task

```bash
# Dry run first
nightshift task run lint-fix --provider claude --dry-run

# Execute
nightshift task run lint-fix --provider claude
```

## Skill Grooming Task

Nightshift includes a built-in `skill-groom` task for keeping project-local skills aligned with the current codebase.

It is enabled by default. You can set its interval and priority:

```yaml
tasks:
  priorities:
    skill-groom: 2
  intervals:
    skill-groom: "168h"
```

To opt out:

```yaml
tasks:
  disabled:
    - skill-groom
```

`skill-groom` uses `README.md` as project context, checks `.claude/skills` and `.codex/skills`, and validates `SKILL.md` content against the Agent Skills format (starting docs lookup from `https://agentskills.io/llms.txt`).

## Custom Tasks

Define custom tasks in your config:

```yaml
tasks:
  custom:
    - type: pr-review
      name: "PR Review Session"
      description: |
        Review all open PRs. Fix obvious issues immediately.
        Create tasks for bigger problems.
      category: pr
      cost_tier: high
      risk_level: medium
      interval: "72h"
```

Custom tasks use the same scoring, cooldowns, and budget controls as built-in tasks. The `description` field becomes the agent prompt. Only `type`, `name`, and `description` are required — other fields have sensible defaults.

## Task Cooldowns

Each task has a default cooldown per project. After running `lint-fix` on `~/code/sidecar`, it won't run again on that project for 24 hours. Override with `tasks.intervals` in config.

Category defaults:

| Category | Default Interval |
|----------|-----------------:|
| PR | 7 days |
| Analysis | 3 days |
| Options | 7 days |
| Safe | 14 days |
| Map | 7 days |
| Emergency | 30 days |

Use `nightshift preview --explain` to see cooldown status, including which tasks are currently on cooldown and when they become eligible again. When all tasks for a project are on cooldown, the run is skipped.

## td Review Task

The `td-review` task runs a detailed review session over open td reviews. It:

- Reviews all open review tasks in the project
- Fixes obvious bugs immediately and creates td bug tasks for them
- Creates new td tasks with detailed descriptions for bigger issues
- Verifies changes have tests, creates tasks for missing coverage
- Uses subagents for reviews that can run in parallel
- Closes in-progress tasks once related bugs are resolved

This task is **disabled by default** and must be explicitly opted in:

```yaml
tasks:
  enabled:
    - td-review
```

Requires the td integration to be enabled (see [Integrations](/docs/integrations)).
