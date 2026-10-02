---
sidebar_position: 3
title: Quick Start
---

# Quick Start

Get running in 2 minutes.

## 1. Install

```bash
brew install marcus/tap/nightshift
```

## 2. Setup

Run the guided setup to configure providers, projects, budget, and daemon/service options:

```bash
nightshift setup
```

If you want to create a config file directly instead of using the wizard, run:

```bash
nightshift init
nightshift init --global
```

## 3. Preview

See what Nightshift will do before it does anything:

```bash
nightshift preview
nightshift budget
```

## 4. Run

Execute tasks manually (or let the scheduler handle it):

```bash
nightshift run
```

You'll see a preflight summary showing what will run, then a confirmation prompt. Use `--dry-run` to preview without executing, or `--yes` to skip the prompt:

```bash
nightshift run --dry-run    # Preview only
nightshift run --yes        # Skip confirmation
```

## 5. Check Results

Review what happened:

```bash
nightshift status --today
```

Everything lands as a branch or PR. Merge what surprises you, close the rest.
