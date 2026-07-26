---
sidebar_position: 9
title: Integrations
---

# Integrations

## Execution providers

Nightshift supports Claude Code, Codex, and GitHub Copilot. They are enabled by default and tried
in configured preference order:

```yaml
providers:
  preference: [claude, codex, copilot]
  claude:
    enabled: true
    data_path: ~/.claude
    dangerously_skip_permissions: false
  codex:
    enabled: true
    data_path: ~/.codex
    dangerously_bypass_approvals_and_sandbox: false
  copilot:
    enabled: true
    data_path: ~/.copilot
    dangerously_skip_permissions: false
```

Nightshift skips a provider when it is disabled, its executable is unavailable, or its allowance
is exhausted. `run --ignore-budget` changes only the exhausted-budget decision.

### Claude Code

Install and start the CLI once to authenticate:

```bash
npm install -g @anthropic-ai/claude-code
claude
```

Nightshift invokes `claude --print`. When
`providers.claude.dangerously_skip_permissions` is true it also passes
`--dangerously-skip-permissions`.

### Codex

```bash
npm install -g @openai/codex
codex --login
```

Nightshift invokes `codex exec` for headless work. Codex may use ChatGPT sign-in or
`OPENAI_API_KEY`. Autonomous Codex execution uses its approval-and-sandbox bypass behavior; if
that is inappropriate for your environment, remove Codex from `providers.preference` or set
`providers.codex.enabled: false`.

### GitHub Copilot

Standalone Copilot is preferred:

```bash
npm install -g @github/copilot
copilot login
```

Copilot can also authenticate with `COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, or `GITHUB_TOKEN`, or reuse
an authenticated `gh` session. Nightshift invokes non-interactive prompt mode and, when
`providers.copilot.dangerously_skip_permissions` is true, allows all tools and URLs.

If `copilot` is absent, Nightshift falls back to `gh` and requires the legacy Copilot extension
to appear in `gh extension list`:

```bash
gh auth login --web
gh extension install github/gh-copilot --force
gh extension list
```

The upstream extension is deprecated, so the standalone CLI is recommended.

## Project instruction files

These integrations default to enabled:

```yaml
integrations:
  claude_md: true
  agents_md: true
```

Nightshift looks in the project root, in order:

| Integration | Recognized filenames |
|-------------|----------------------|
| Claude context | `claude.md`, `CLAUDE.md`, `.claude.md` |
| Agent instructions | `AGENTS.md`, `agents.md`, `.agents.md` |

The full file becomes prompt context. Nightshift also extracts task suggestions, conventions,
constraints, safety guidance, and action/tool restrictions as hints. Task types mentioned in
these files receive a selection score bonus.

## td task source

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true
```

When `td` is on `PATH`, Nightshift runs `td list --format json` from each project. A missing or
unconfigured td CLI is non-fatal. `teach_agent: true` adds the core assign, inspect, comment, and
complete workflow to the agent context.

## GitHub issue source

```yaml
integrations:
  task_sources:
    - github_issues: true
```

This reader requires:

- `gh` on `PATH` and authenticated.
- A GitHub `origin` remote in the project.
- Open issues labeled `nightshift`.

Nightshift reads issue number, title, body, labels, and state with `gh issue list`. Priority labels
such as `critical`, `high`, `medium`, `low`, or `p0` through `p3` affect the imported task
priority. Authentication or repository failures are non-fatal and leave the source empty.

## Generated reports

`reporting.morning_summary` defaults to `true` and writes local summaries after runs. The
`reporting.email` and `reporting.slack_webhook` keys are accepted by the configuration schema,
but the current run implementation does not send those notifications.
