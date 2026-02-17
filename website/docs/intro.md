---
sidebar_position: 1
slug: /intro
title: Introduction
---

# Nightshift

> It finds what you forgot to look for.

Nightshift is a Go CLI tool that runs AI-powered maintenance tasks on your codebase overnight, using your remaining daily token budget from Claude Code, Codex, or Ollama Cloud. Wake up to a cleaner codebase without unexpected costs.

Your tokens get reset every week — you might as well use them. Nightshift runs overnight to find dead code, doc drift, test gaps, security issues, and 20+ other things silently accumulating while you ship features.

Like a Roomba for your codebase. Runs overnight, worst case you close the PR.

## Key Principles

- **Everything is a PR** — Nightshift never writes directly to your primary branch. Don't like something? Close it. That's the whole rollback plan.
- **Budget-aware** — Uses remaining daily allotment, never exceeds your configured max (default 75%).
- **Multi-project** — Point it at your repos, it already knows what to look for.
- **Zero config defaults** — Works out of the box with sensible defaults. Customize when you need to.

## Quick Start

```bash
# Install
brew install marcus/tap/nightshift

# Interactive setup
nightshift setup

# Preview what it will do
nightshift preview

# Run immediately
nightshift run
```

## Next Steps

- [Installation](/docs/installation) — All installation methods
- [Quick Start](/docs/quick-start) — Get running in 2 minutes
- [Configuration](/docs/configuration) — Customize budgets, schedules, and tasks
- [Tasks](/docs/tasks) — Browse the 20+ built-in tasks
