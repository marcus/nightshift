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

Nightshift requires at least one AI provider CLI to be installed and authenticated. You can use Claude Code, Codex CLI, Ollama Cloud, or any combination.

### Claude Code CLI

Install and authenticate:

```bash
# Install
brew install marcus/tap/claude-code

# Authenticate
claude
/login
```

Claude Code login supports Claude.ai subscriptions or Anthropic Console credentials. You can also use an API key.

### Codex CLI

Install and authenticate:

```bash
# Install
brew install openai/codex/codex

# Authenticate
codex --login
```

Codex CLI supports signing in with a ChatGPT account, or you can use an API key directly.

### Ollama Cloud

No CLI installation needed — setup uses cookie authentication:

```bash
# Set up authentication
nightshift ollama auth
```

This creates `~/.ollama/cookies.txt`. To populate it:

1. Sign in to https://ollama.com/settings in your browser
2. Install a browser extension to export cookies:
   - Chrome/Firefox: ["EditThisCookie"](https://chrome.google.com/webstore/detail/editthiscookie/) or ["Get cookies.txt LOCALLY"](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/)
3. Export your ollama.com cookies in **Netscape format**
4. Paste the cookies into `~/.ollama/cookies.txt`

Verify setup:

```bash
nightshift budget --provider ollama
```
