---
sidebar_position: 9
title: Integrations
---

# Integrations

## Execution providers

Nightshift supports Claude Code, Codex, and GitHub Copilot. They are enabled by built-in defaults:

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

Automatic `nightshift run` walks `providers.preference` and skips a provider when it is disabled,
its executable is unavailable, or its calculated allowance is exhausted. It does not test
authentication before selection. `run --ignore-budget` changes only the exhausted-budget
decision.

The generated `nightshift init --global` starter differs from these built-in defaults: it
explicitly prefers Claude and Codex, omits Copilot from the preference list, and enables the
Claude/Codex unattended permission-bypass settings.

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
codex
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

The npm install requires Node.js 22 or later. Copilot can also authenticate with
`COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, or `GITHUB_TOKEN`, or reuse an authenticated `gh` session.
Nightshift invokes non-interactive prompt mode and, when
`providers.copilot.dangerously_skip_permissions` is true, allows all tools and URLs.

If `copilot` is absent, automatic Nightshift runs fall back to `gh copilot`. Recent GitHub CLI
releases provide that command directly and download the current Copilot CLI when needed:

```bash
gh auth login --web
gh copilot
```

The old `github/gh-copilot` extension is retired and should not be installed. A Nightshift
compatibility bug remains in `task run --provider copilot`: when standalone `copilot` is absent,
that command still checks `gh extension list` and can reject the new built-in `gh copilot`.
Install standalone Copilot for direct task execution. Automatic `run` only checks for `gh` on
`PATH`, so it requires a recent GitHub CLI with the built-in command.

## Provider selection exceptions

- `preview` ignores `providers.preference`, chooses the first enabled provider in fixed Claude,
  Codex, Copilot order, and does not check executable availability.
- `task run --provider NAME` uses the requested provider even when it is disabled in
  configuration. It checks executable availability but bypasses budgets and automatic
  eligibility.
- Copilot usage is modeled as local premium-request counts under `providers.copilot.data_path`.
  The current agent execution path does not increment that counter, so budget output cannot be
  treated as authoritative Copilot account usage.

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

The readers can return the full file as context and extract task suggestions, conventions,
constraints, safety guidance, and action/tool restrictions as hints. However, the current CLI
does not construct or call the integration manager, so these files do not currently change
prompts or task selection through these configuration keys. Provider CLIs may still discover
their own instruction files independently.

## td task source

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true
```

The implemented reader runs `td list --format json` from each project and can add td usage
context. A missing or unconfigured td CLI is non-fatal. The current Nightshift commands do not
invoke this reader, so configuring it does not yet import td work.

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

The implemented reader uses `gh issue list` to read number, title, body, labels, and state.
Priority labels such as `critical`, `high`, `medium`, `low`, or `p0` through `p3` affect the
imported task priority. Authentication or repository failures are non-fatal. The current
Nightshift commands do not invoke this reader, so configuring it does not yet add GitHub issues
to task selection.

`task_sources[].file` is accepted by the configuration schema, but there is no file-source reader
in the current implementation.

## Generated reports

`reporting.morning_summary` defaults to `true` and writes local summaries after runs. The
`reporting.email` and `reporting.slack_webhook` keys are accepted by the configuration schema,
and notification helper implementations exist, but the current run finalizer does not call
notification dispatch.
