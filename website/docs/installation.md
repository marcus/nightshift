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

## Prerequisites

- **Claude Code CLI** (`claude`) if you want Claude support
- **Codex CLI** (`codex`) if you want Codex support
- **GitHub Copilot** via either `gh` with the Copilot extension or the standalone `copilot` binary
- `tmux` if you want budget snapshots from live terminal usage
- `gh` if you plan to read GitHub issues as a task source

```bash
# Claude Code
claude
/login

# Codex
codex --login

# GitHub Copilot via gh
gh auth login
gh extension install github/gh-copilot

# Or install the standalone Copilot CLI
npm install -g @github/copilot
```

## After Install

Run the guided setup to create the global config, check provider availability, and optionally install a service:

```bash
nightshift setup
```

If you prefer to bootstrap by hand, create a config file directly and then validate it:

```bash
nightshift init --global
nightshift config validate
```
