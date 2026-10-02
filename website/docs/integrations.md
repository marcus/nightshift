---
sidebar_position: 9
title: Integrations
---

# Integrations

Nightshift works best when the provider CLIs, your Git workflow, and your task sources are already part of your normal setup.

## Provider Prerequisites

Before Nightshift can run work, the provider must be:

- Installed and available in `PATH`
- Enabled in config under `providers.<name>.enabled`
- Authenticated in the provider's own CLI

`nightshift doctor` is the fastest way to confirm whether Nightshift can see the provider CLI, its data path, and its budget status.

## Claude Code

Nightshift can use the local Claude Code CLI as an execution provider.

```bash
claude
/login
```

Claude Code can be authenticated through a subscription login or API-backed credentials, depending on your local Claude setup.

Config example:

```yaml
providers:
  preference:
    - claude
    - codex
  claude:
    enabled: true
    data_path: "~/.claude"
```

## Codex

Nightshift also supports the OpenAI Codex CLI.

```bash
codex --login
```

Codex can run with either a ChatGPT login or API-backed credentials, depending on your local Codex setup.

Config example:

```yaml
providers:
  preference:
    - codex
    - claude
  codex:
    enabled: true
    data_path: "~/.codex"
```

## GitHub Copilot

Nightshift supports GitHub Copilot as a provider through either:

- `gh copilot` via the GitHub CLI extension
- The standalone `copilot` binary

GitHub CLI path:

```bash
gh auth login
gh extension install github/gh-copilot
gh extension list
```

Standalone path:

```bash
npm install -g @github/copilot
copilot --version
```

Nightshift prefers a standalone `copilot` binary when it is present in `PATH`; otherwise it falls back to `gh copilot`.

Config example:

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
  copilot:
    enabled: true
    data_path: "~/.copilot"
```

Budget note:

- GitHub Copilot does not expose authoritative token usage the way Claude Code and Codex do.
- Nightshift tracks Copilot usage locally in `~/.copilot/nightshift-usage.json`.
- Budget output for Copilot is therefore an approximation based on request counting, not a server-reported quota.

## GitHub and Pull Requests

Nightshift is designed around Git workflows. It works in your local checkout, creates branches, and reports results as branches or PR-ready changes.

GitHub-specific features such as issue sourcing require the `gh` CLI to be installed and authenticated:

```bash
gh auth login
```

## td Task Management

Nightshift can source work from [`td`](https://td.haplab.com).

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true
```

When `teach_agent: true` is enabled, Nightshift includes the local td workflow in the agent prompt context.

## GitHub Issues

Nightshift can source tasks from open GitHub issues labeled `nightshift`.

```yaml
integrations:
  task_sources:
    - github_issues: true
```

This integration uses the local `gh issue list` command in repositories whose `origin` remote points at GitHub.

## Project Instruction Files

Nightshift reads repo-local instruction files for extra project context:

- `CLAUDE.md`
- `AGENTS.md`

Enable or disable them in config:

```yaml
integrations:
  claude_md: true
  agents_md: true
```

These files are read from the project root and included as execution context when available.
