---
sidebar_position: 9
title: Integrations
---

# Integrations

Nightshift integrates with your existing development workflow.

## AI Providers

Nightshift supports three execution providers. It uses the first enabled provider in `providers.preference`, which defaults to `claude -> codex -> copilot`.

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
```

### Claude Code

Nightshift uses the Claude Code CLI to execute tasks. Authenticate via subscription or API key:

```bash
claude
/login
```

Nightshift looks for the `claude` binary on `PATH`.

Relevant config keys:

- `providers.claude.enabled`
- `providers.claude.data_path`
- `providers.claude.dangerously_skip_permissions`

### Codex

Nightshift supports OpenAI's Codex CLI as an alternative provider:

```bash
codex --login
```

Nightshift looks for the `codex` binary on `PATH`.

Relevant config keys:

- `providers.codex.enabled`
- `providers.codex.data_path`
- `providers.codex.dangerously_bypass_approvals_and_sandbox`

### GitHub Copilot

Nightshift supports GitHub Copilot through either the standalone `copilot` binary or `gh copilot`.

```bash
# Standalone binary
npm install -g @github/copilot
# or
curl -fsSL https://gh.io/copilot-install | bash
# GitHub CLI extension
gh extension install github/gh-copilot
```

Nightshift prefers the standalone `copilot` binary when it is available and falls back to `gh copilot`. Copilot usage is tracked by request count instead of token usage. Nightshift stores that tracking data under `providers.copilot.data_path` (default: `~/.copilot`).

If you use `gh copilot`, authenticate with `gh auth login` first.

Relevant config keys:

- `providers.copilot.enabled`
- `providers.copilot.data_path`
- `providers.copilot.dangerously_skip_permissions`

## GitHub

All output is PR-based. Nightshift creates branches and pull requests for its findings.

## td (Task Management)

Nightshift can source tasks from [td](https://td.haplab.com) - task management for AI-assisted development. The td reader imports every task from `td list --format json`; there is no tag filter.

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true   # Include td usage + core workflow in prompts
```

When `teach_agent` is enabled, Nightshift adds td workflow notes to the agent prompt.

## CLAUDE.md / AGENTS.md

Nightshift reads project-level instruction files to understand context when executing tasks. Place a `CLAUDE.md` or `AGENTS.md` in your repo root to give Nightshift project-specific guidance. Tasks mentioned in these files get a priority bonus (+2).

## GitHub Issues

Source tasks from GitHub issues by enabling the GitHub task source:

```yaml
integrations:
  task_sources:
    - github_issues: true
```

Nightshift reads open issues with the hard-coded `nightshift` label via `gh issue list --label nightshift --state open`. The label is not configurable in Nightshift's current schema.
