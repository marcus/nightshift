---
sidebar_position: 9
title: Integrations
---

# Integrations

Nightshift integrates with local provider CLIs and includes readers for repository instructions and external task sources.

## AI Providers

Nightshift supports three execution providers. For an ordinary `nightshift run`, it walks `providers.preference` (default `claude -> codex -> copilot`) and selects the first provider that is enabled, has a discoverable CLI, and has a positive calculated allowance. Missing or exhausted providers are skipped. `--ignore-budget` keeps the order but permits an exhausted provider; `nightshift task run` requires an explicit provider and does not perform this budget selection.

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

When the last key is true, Nightshift adds Claude's `--dangerously-skip-permissions` flag. The configured default is false.

### Codex

Nightshift supports OpenAI's Codex CLI as an alternative provider:

```bash
codex login
```

Nightshift looks for the `codex` binary on `PATH`.

Relevant config keys:

- `providers.codex.enabled`
- `providers.codex.data_path`
- `providers.codex.dangerously_bypass_approvals_and_sandbox`

Nightshift runs Codex through `codex exec`. Although the config default for the dangerous bypass key is false, the current headless Codex agent preserves its own bypass-enabled default when the key is false because the config cannot distinguish "unset" from an explicit false. Disable the Codex provider if that execution mode is unacceptable.

### GitHub Copilot

Nightshift supports GitHub Copilot through either the standalone `copilot` binary or `gh copilot`.

```bash
# Recommended standalone binary
npm install -g @github/copilot
copilot login

# GitHub CLI route
gh auth login
gh copilot -- --version
```

Nightshift prefers the standalone `copilot` binary when it is available and otherwise constructs `gh copilot -- ...`. Its current `gh` availability probe also requires `gh extension list` to contain `gh-copilot`. GitHub has [retired the legacy extension](https://github.blog/changelog/2025-09-25-upcoming-deprecation-of-gh-copilot-cli-extension/), so use the standalone binary for reliable operation with this release.

Copilot's usage provider models usage as request counts rather than tokens and reads `providers.copilot.data_path/nightshift-usage.json` (default `~/.copilot/nightshift-usage.json`). The counter resets on the first of the month at 00:00 UTC, and budget code approximates the configured weekly value as a monthly request limit by multiplying it by four. The current execution path does not call the counter's increment method, so this file is not an authoritative record of Nightshift or external Copilot use and normally remains empty unless another caller updates it.

If you use `gh copilot`, authenticate with `gh auth login` first.

Relevant config keys:

- `providers.copilot.enabled`
- `providers.copilot.data_path`
- `providers.copilot.dangerously_skip_permissions`

When the last key is true, Nightshift adds `--allow-all-tools --allow-all-urls`. The configured default is false.

## td (Task Management)

Nightshift's td reader runs `td list --format json` in the project directory. It accepts either a top-level JSON array or an object containing `tasks`, imports every returned record without status or label filtering, and maps subject, description, labels, owner, status, and priority into Nightshift's integration model. Text priorities map to 100 (`critical`/`urgent`), 75 (`high`), 50 (`medium`/`normal` or unknown), and 25 (`low`). Numeric strings are preserved as integers.

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true   # Include td usage + core workflow in prompts
```

If `td` is missing, not configured for the repository, or returns an error, the reader quietly produces no result. When `teach_agent` is enabled, its aggregated context includes assign/complete workflow notes.

## CLAUDE.md / AGENTS.md

The instruction readers look only in the project root. Claude candidates are `claude.md`, `CLAUDE.md`, then `.claude.md`; agent candidates are `AGENTS.md`, `agents.md`, then `.agents.md`. They return the full file as context and classify bullets under recognized headings as hints.

## GitHub Issues

Source tasks from GitHub issues by enabling the GitHub task source:

```yaml
integrations:
  task_sources:
    - github_issues: true
```

The GitHub reader first requires `gh` on `PATH` and an `origin` remote whose URL contains `github.com`. It then runs:

```bash
gh issue list --label nightshift --json number,title,body,labels,state --state open
```

Only open issues with the hard-coded `nightshift` label are imported. Issue IDs become `gh-NUMBER`; title, body, state, and labels are retained. Priority is inferred from labels containing `critical`/`urgent` or `p0` (100), `high` or `p1` (75), `medium` or `p2` (50), and `low` or `p3` (25), defaulting to 50. Authentication or query failures are treated as no result. The label is not configurable in the current schema.

## Current Wiring Limitation

These readers and their aggregation model are implemented, but the current `run`, `daemon`, and `preview` command paths do not call the integration manager. Enabling td, GitHub issues, `CLAUDE.md`, or `AGENTS.md` therefore does not currently add external work items or scoring bonuses to scheduled CLI task selection. The `file` task-source field is present in the config schema but has no reader. Treat these settings as forward-compatible until the execution path is wired to them.
