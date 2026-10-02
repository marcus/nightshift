---
sidebar_position: 1
slug: /intro
title: Introduction
---

# Nightshift

> It finds what you forgot to look for.

Nightshift is a Go CLI tool that runs AI-powered maintenance tasks on your codebase overnight, using your configured budget and one of the supported local provider CLIs: Claude Code, Codex, or GitHub Copilot. Wake up to a cleaner codebase without unexpected costs.

Your budget resets on the schedule you configure. Nightshift runs overnight to find dead code, doc drift, test gaps, security issues, and 20+ other things silently accumulating while you ship features.

Like a Roomba for your codebase. Runs overnight, worst case you close the PR.

## Key Principles

- **Everything is a PR** — Nightshift never writes directly to your primary branch. Don't like something? Close it. That's the whole rollback plan.
- **Budget-aware** — Uses the remaining budget for the current run, never exceeds your configured max (default 75%).
- **Multi-project** — Point it at your repos, it already knows what to look for.
- **Zero config defaults** — Works out of the box with sensible defaults. Customize when you need to.

## Quick Start

```bash
# Install
brew install marcus/tap/nightshift

# Interactive setup
nightshift setup

# Or bootstrap a project config directly
nightshift init

# Preview what it will do
nightshift preview

# Run immediately
nightshift run
```

## Next Steps

- [Installation](/docs/installation) — All installation methods and provider prerequisites
- [Quick Start](/docs/quick-start) — Get running in 2 minutes
- [Configuration](/docs/configuration) — Customize budgets, schedules, tasks, and integrations
- [Tasks](/docs/tasks) — Browse the 20+ built-in tasks
