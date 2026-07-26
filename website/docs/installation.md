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
# macOS or Linux installer
curl -fsSL https://chatgpt.com/codex/install.sh | sh

# Alternatives
npm install -g @openai/codex
brew install --cask codex

codex
```

Choose ChatGPT sign-in on first launch, or configure API-key authentication. See the
[official Codex CLI README](https://github.com/openai/codex#readme).

### GitHub Copilot CLI

The standalone `copilot` executable is Nightshift's preferred Copilot mode:

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

GitHub has retired that extension in favor of the standalone CLI, so prefer `copilot` for new
installations. The fallback remains documented only because Nightshift retains a compatibility
execution path.

## Provider discovery

Nightshift looks for `claude`, `codex`, and `copilot` on `PATH`. For Copilot it prefers
standalone `copilot`, then falls back to `gh`. `task run` verifies the retired extension through
`gh extension list`, but automatic `run` selection only verifies that `gh` exists; without the
extension, execution fails later. Provider `data_path` settings point to usage data and do not
override executable paths.

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
scheduled service. Its environment screen treats any `gh` executable as Copilot availability and
does not test authentication, so verify standalone `copilot` and its login directly before
unattended use.
