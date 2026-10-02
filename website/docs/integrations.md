---
sidebar_position: 9
title: Integrations
---

# Integrations

Nightshift integrates with your existing development workflow for provider execution, task sourcing, and repo-level instruction files.

## Execution Providers

Nightshift can execute tasks with Claude Code, Codex, or GitHub Copilot. Automatic selection follows `providers.preference`, and `nightshift task run --provider ...` lets you target one provider directly.

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
```

### Claude Code

```bash
claude auth login
```

Nightshift uses the local `claude` CLI and reads usage data from `providers.claude.data_path` (default `~/.claude`).

### Codex

```bash
codex login
```

Nightshift uses the local `codex` CLI and reads usage data from `providers.codex.data_path` (default `~/.codex`).

### GitHub Copilot

```bash
gh auth login
gh extension install github/gh-copilot
gh copilot --help
```

Nightshift supports either a standalone `copilot` binary or `gh copilot`. It prefers standalone `copilot` when one is present in `PATH`, otherwise it falls back to `gh`, which must have the `gh-copilot` extension installed.

## GitHub Workflow

Nightshift works on branches and PRs instead of writing directly to your primary branch. In GitHub repos it can also source work from labeled issues when the GitHub task-source integration is enabled.

## td (Task Management)

Nightshift can source tasks from [td](https://td.haplab.com). When `teach_agent` is enabled, it also injects td workflow instructions into the agent prompt.

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true
```

Nightshift shells out to the local `td` CLI, reads tasks from the current project, and merges them into task selection.

## GitHub Issues

Nightshift can source open GitHub issues labeled `nightshift` through the `gh` CLI:

```yaml
integrations:
  task_sources:
    - github_issues: true
```

The repo must have a GitHub `origin`, `gh` must be installed, and issue access must already work in that checkout.

## CLAUDE.md And AGENTS.md

Nightshift reads project instruction files and folds them into prompt context before execution.

Supported filenames:

- `CLAUDE.md`, `claude.md`, `.claude.md`
- `AGENTS.md`, `agents.md`, `.agents.md`

Enable or disable them in config:

```yaml
integrations:
  claude_md: true
  agents_md: true
```

Use these files for repo-specific conventions, safety rules, workflow instructions, or architecture context you want Nightshift to inherit automatically.

## Common Integration Config

```yaml
integrations:
  claude_md: true
  agents_md: true
  task_sources:
    - td:
        enabled: true
        teach_agent: true
    - github_issues: true
```

After changing integrations, run:

```bash
nightshift config validate
nightshift preview --explain
```
