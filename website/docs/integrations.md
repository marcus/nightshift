---
sidebar_position: 9
title: Integrations
---

# Integrations

Nightshift plugs into the tools you already use: local coding-agent CLIs,
repository instruction files, and optional task sources.

## Provider CLIs

Nightshift executes tasks through one of three local provider CLIs:

- Claude Code via `claude`
- Codex via `codex`
- GitHub Copilot via standalone `copilot`, or `gh` with `gh-copilot`

Set the preferred order in config:

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
```

Use `nightshift doctor` to verify Nightshift can discover those CLIs and their
usage data paths.

## Git Workflow

Nightshift works in normal Git repositories and produces branches or PR-ready
changes instead of writing directly to your primary branch.

## td Task Sourcing

Nightshift can source work from [td](https://td.haplab.com):

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true
```

When `teach_agent` is enabled, td workflow instructions are included in the
generated agent prompt.

## Instruction Files

Nightshift reads project instruction files and folds them into agent context.
Supported filenames:

- `CLAUDE.md`, `claude.md`, `.claude.md`
- `AGENTS.md`, `agents.md`, `.agents.md`

Enable or disable those readers in config:

```yaml
integrations:
  claude_md: true
  agents_md: true
```

Tasks mentioned in those files receive a priority bonus during task selection.

## Other Task Sources

Nightshift also supports GitHub issue and file-based task sources through the
same `integrations.task_sources` list:

```yaml
integrations:
  task_sources:
    - github_issues: true
    - file: TODO.md
```
