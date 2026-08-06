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
nightshift doctor
```

`nightshift doctor` checks config loading, scheduling, provider data paths, and
whether Nightshift can discover the local CLIs it depends on.

## Provider CLIs

Nightshift supports three local providers:

- `claude`
- `codex`
- `copilot`, or `gh` with the `gh-copilot` extension installed

Install and authenticate at least one provider before running Nightshift.

### Claude Code

```bash
claude
/login
```

Claude Code can use either a Claude subscription login or Anthropic API
credentials.

### Codex

```bash
codex --login
```

Codex can use a ChatGPT login or API key.

### GitHub Copilot

```bash
# Standalone Copilot CLI
npm install -g @github/copilot

# Or GitHub CLI + Copilot extension
gh extension install github/gh-copilot
```

Nightshift prefers a standalone `copilot` binary when it exists in `PATH`, then
falls back to `gh` if the `gh-copilot` extension is installed.

## Verify Provider Discovery

Use `nightshift doctor` to confirm Nightshift can see your configured provider
CLIs and data paths.

If you want to inspect the prompt Nightshift would generate for a task without
executing it, use a dry run:

```bash
nightshift task run lint-fix --provider claude --dry-run
```

Dry-run validates task lookup, config loading, provider selection, and prompt
generation. It does not execute the provider CLI, so it does not confirm auth or
remote account access.
