---
sidebar_position: 2
title: Installation
---

# Installation

## Homebrew (Recommended)

```bash
brew install marcus/tap/nightshift
```

## Binary Downloads

Pre-built binaries are available on the [GitHub releases page](https://github.com/marcus/nightshift/releases) for macOS and Linux (Intel and ARM).

## From Source

Requires Go 1.24+:

```bash
go install github.com/marcus/nightshift/cmd/nightshift@latest
```

Or build from the repository:

```bash
git clone https://github.com/marcus/nightshift.git
cd nightshift
go build -o nightshift ./cmd/nightshift
sudo mv nightshift /usr/local/bin/
```

## Verify Installation

```bash
nightshift --version
nightshift --help
```

## Provider Prerequisites

Nightshift needs at least one provider CLI available in `PATH`. It can run with Claude Code, Codex, or GitHub Copilot.

### Claude Code

Authenticate Claude Code before running `nightshift setup` or `nightshift doctor`:

```bash
claude auth login
claude --help
```

Nightshift reads local Claude usage data from `~/.claude` by default. Override that with `providers.claude.data_path` if needed.

### Codex

Codex works with either interactive login or an API key:

```bash
codex login
codex --help
```

API-key flow:

```bash
printenv OPENAI_API_KEY | codex login --with-api-key
```

Nightshift reads local Codex data from `~/.codex` by default.

### GitHub Copilot

Nightshift supports GitHub Copilot in two modes:

- Standalone `copilot` binary in `PATH`
- `gh copilot` via GitHub CLI

Typical setup:

```bash
gh auth login
gh extension install github/gh-copilot
gh copilot --help
```

Nightshift prefers a standalone `copilot` binary when one exists. If it does not find one, it falls back to `gh copilot`, which Nightshift detects through `gh extension list`. Copilot usage tracking is stored under `~/.copilot` by default.

## First Run

The onboarding wizard writes the global config, validates providers, captures a budget snapshot, previews the next run, and can install the daemon:

```bash
nightshift setup
```

If you want starter config files without the wizard:

```bash
nightshift init --global
nightshift init
nightshift config validate
```

## Verify Provider Access And Health

Use a dry run to confirm Nightshift can invoke the provider CLI you plan to use:

```bash
nightshift task run lint-fix --provider claude --dry-run
nightshift task run lint-fix --provider codex --dry-run
nightshift task run lint-fix --provider copilot --dry-run
```

Then run doctor to inspect config, data paths, usage, and snapshots for providers that are enabled in config:

```bash
nightshift doctor
```
