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

The npm package requires Node.js 22 or later. Homebrew and GitHub's install script are also
supported:

```bash
brew install copilot-cli
curl -fsSL https://gh.io/copilot-install | bash
```

An eligible Copilot plan and any required organization policy are prerequisites. Copilot can also
reuse an authenticated GitHub CLI token. See
[GitHub's Copilot CLI installation guide](https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/install-copilot-cli).

Recent GitHub CLI releases also provide a built-in `gh copilot` command that downloads and runs
the current standalone Copilot CLI when needed:

```bash
gh auth login --web
gh copilot
```

Do not install the retired `github/gh-copilot` extension; it stopped working in October 2025.
Nightshift's automatic `run` path can invoke the new built-in `gh copilot` command. The direct
`task run --provider copilot` availability check still looks for the retired extension when no
standalone `copilot` executable exists, so install standalone Copilot for that command.

## Provider discovery

Nightshift looks for `claude`, `codex`, and `copilot` on `PATH`. For Copilot it prefers
standalone `copilot`, then falls back to `gh`. Automatic `run` verifies only that `gh` exists
before invoking `gh copilot`, so an old GitHub CLI without the built-in command fails at
execution. `task run` additionally applies the stale extension-list check described above.
Provider `data_path` settings point to usage data and do not override executable paths.

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
