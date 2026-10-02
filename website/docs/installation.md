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

## Shell Completion

Nightshift can generate completion scripts for bash, zsh, fish, and PowerShell:

```bash
nightshift completion bash
nightshift completion zsh
nightshift completion fish
nightshift completion powershell
```

Start a new shell after writing a completion file so the new completions are loaded.

### Bash

The generated bash script depends on the `bash-completion` package.

```bash
# Current shell
source <(nightshift completion bash)

# Persistent install on Linux
nightshift completion bash > /etc/bash_completion.d/nightshift

# Persistent install on macOS
nightshift completion bash > $(brew --prefix)/etc/bash_completion.d/nightshift
```

### Zsh

If completion is not already enabled, add `autoload -U compinit; compinit` to your `~/.zshrc` once.

```bash
# Current shell
source <(nightshift completion zsh)

# Persistent install on Linux
nightshift completion zsh > "${fpath[1]}/_nightshift"

# Persistent install on macOS
nightshift completion zsh > $(brew --prefix)/share/zsh/site-functions/_nightshift
```

### Fish

```bash
# Current shell
nightshift completion fish | source

# Persistent install
nightshift completion fish > ~/.config/fish/completions/nightshift.fish
```

### PowerShell

```powershell
# Current shell
nightshift completion powershell | Out-String | Invoke-Expression
```

To make the PowerShell completion persistent, add the output of that command to your PowerShell profile.

## Prerequisites

- **Claude Code CLI** (`claude`) and/or **Codex CLI** (`codex`) installed
- Authenticated via subscription login or API keys:

```bash
# Claude Code
claude
/login

# Codex
codex --login
```
