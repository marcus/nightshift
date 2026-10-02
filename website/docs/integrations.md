---
sidebar_position: 9
title: Integrations
---

# Integrations

Nightshift integrates with your existing development workflow.

## Claude Code

Nightshift uses the Claude Code CLI to execute tasks. Authenticate via subscription or API key:

```bash
claude
/login
```

## Codex

Nightshift supports OpenAI's Codex CLI as an alternative provider:

```bash
codex --login
```

## td (Task Management)

Nightshift can source tasks from [td](https://td.haplab.com) - task management for AI-assisted development.

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true   # Include td usage + core workflow in prompts
```

Nightshift reads tasks from `td list --format json` in each project directory. It does not filter by tag; every task returned by `td` is considered during scoring and selection. When `teach_agent` is enabled, Nightshift also adds td usage guidance to the agent prompt.

## CLAUDE.md / AGENTS.md

Nightshift reads project-level instruction files to understand context when executing tasks. Place a `CLAUDE.md` or `AGENTS.md` in your repo root to give Nightshift project-specific guidance. Tasks mentioned in these files get a priority bonus (+2).

## GitHub Issues

Source tasks from GitHub issues labeled with `nightshift`:

```yaml
integrations:
  task_sources:
    - github_issues: true
```

Nightshift reads open issues from `gh issue list --label nightshift --state open` in GitHub repositories. The label is currently hard-coded to `nightshift`, so the config does not expose a separate label field.
