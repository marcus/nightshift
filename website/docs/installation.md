---
sidebar_position: 2
title: Installation
---

# Installation

## Install Nightshift

### Homebrew

```bash
brew install marcus/tap/nightshift
```

### Binary release

Download a macOS or Linux archive from the
[GitHub releases page](https://github.com/marcus/nightshift/releases), place the executable in a
directory on `PATH`, and verify it:

```bash
nightshift --version
nightshift --help
```

### From source

Nightshift requires Go 1.24 or later:

```bash
go install github.com/marcus/nightshift/cmd/nightshift@latest
```

The resulting binary is normally placed in `$GOBIN` or `$GOPATH/bin`. Ensure that directory is
on `PATH`.

## Install at least one provider

Nightshift executes work through a local provider CLI. Install and authenticate one or more of
Claude Code, Codex, or GitHub Copilot before running setup.

### Claude Code

Claude Code requires Node.js 18 or later for the npm installation:

```bash
npm install -g @anthropic-ai/claude-code
claude
```

Complete the browser login after starting `claude`. Claude.ai plans, Anthropic Console accounts,
and supported enterprise providers can authenticate the CLI. See
[Anthropic's setup guide](https://docs.anthropic.com/en/docs/claude-code/getting-started).

### Codex

```bash
npm install -g @openai/codex
codex --login
```

Choose ChatGPT sign-in, or supply `OPENAI_API_KEY` for API-billed use. See
[OpenAI's Codex CLI setup](https://help.openai.com/en/articles/11381614-api-codex-cli-and-sign-in-with-chatgpt).

### GitHub Copilot CLI

The standalone `copilot` executable is Nightshift's preferred Copilot mode. The npm package
requires Node.js 22 or later:

```bash
npm install -g @github/copilot
copilot login
```

Homebrew and GitHub's install script are also supported:

```bash
brew install copilot-cli
curl -fsSL https://gh.io/copilot-install | bash
```

An eligible Copilot plan and any required organization policy are prerequisites. Copilot can also
reuse an authenticated GitHub CLI token. See
[GitHub's Copilot CLI installation guide](https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/install-copilot-cli).

Nightshift also recognizes the legacy `gh copilot` extension when standalone `copilot` is not
installed:

```bash
gh auth login --web
gh extension install github/gh-copilot --force
gh extension list
```

GitHub has deprecated that extension in favor of the standalone CLI, so prefer `copilot` for new
installations. The fallback remains documented because the current Nightshift implementation
detects `github/gh-copilot` in `gh extension list`.

## Provider discovery

Nightshift looks for `claude`, `codex`, and `copilot` on `PATH`. For Copilot it prefers
standalone `copilot`, then falls back to `gh` plus the installed extension. Provider
`data_path` settings point to usage data and do not override executable paths.

When launched by a daemon or service, Nightshift also searches existing directories at:

- `~/.local/bin`
- `~/go/bin`
- `~/.cargo/bin`
- `~/.npm-global/bin`
- `/usr/local/bin`
- `/opt/homebrew/bin`
- `$GOPATH/bin`, when `GOPATH` is set

## Finish setup

```bash
nightshift setup
nightshift doctor
nightshift preview --explain
```

The wizard creates or updates the global config, checks provider availability, adds projects,
configures budget and safety options, captures a snapshot, previews a run, and can install a
scheduled service.
