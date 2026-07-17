---
sidebar_position: 2
title: Installation
---

# Installation

Nightshift runs local AI-provider CLIs, so install and authenticate at least one provider before the first real run. Git is required for project work. `tmux` is optional but required for subscription-usage scraping and calibration snapshots.

## Homebrew (Recommended)

```bash
brew install marcus/tap/nightshift
```

## Binary Downloads

Pre-built binaries are available on the [GitHub releases page](https://github.com/marcus/nightshift/releases) for macOS and Linux (Intel and ARM).

## From Source

Requires Go 1.24 or newer:

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

Nightshift can use Claude Code, Codex, and GitHub Copilot. Install and authenticate any provider you enable before running `nightshift setup` or `nightshift run`. Provider binaries must be on `PATH`.

### Claude Code

```bash
# Node.js 18+ installation
npm install -g @anthropic-ai/claude-code

# Start the interactive client, then choose an Anthropic Console,
# Claude Pro/Max, Bedrock, or Vertex authentication path.
claude
/login
```

Claude's current setup options are documented in [Set up Claude Code](https://docs.anthropic.com/en/docs/claude-code/getting-started). Nightshift executes `claude --print`; `providers.claude.data_path` defaults to `~/.claude` for local usage data.

### Codex

```bash
npm install -g @openai/codex

# ChatGPT subscription sign-in
codex login

# API-key sign-in (usage is billed through the API account)
printenv OPENAI_API_KEY | codex login --with-api-key

# Verify the saved method
codex login status
```

Nightshift executes `codex exec`. The browser-based `codex login` flow uses ChatGPT access; piping a key with `--with-api-key` uses API billing. Local session data defaults to `~/.codex`.

### GitHub Copilot

The standalone agentic CLI is the recommended path. It requires an eligible Copilot account; npm installation currently requires Node.js 22 or newer.

```bash
npm install -g @github/copilot
copilot login
```

GitHub also supports installing/running the current Copilot CLI through recent GitHub CLI releases:

```bash
gh auth login
gh copilot -- --version
```

Nightshift prefers a `copilot` executable and otherwise constructs a `gh copilot -- ...` command. However, this release's availability probe for the `gh` path checks `gh extension list` for `gh-copilot`. GitHub [retired the legacy extension](https://github.blog/changelog/2025-09-25-upcoming-deprecation-of-gh-copilot-cli-extension/) in October 2025, so a current built-in `gh copilot` command alone may not pass that probe. Use the standalone `copilot` executable for reliable operation until Nightshift's probe is updated.

See GitHub's [Copilot CLI installation](https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/install-copilot-cli) and [authentication](https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/authenticate-copilot-cli) guides. Nightshift's usage provider reads `~/.copilot/nightshift-usage.json` by default, but the current execution path does not increment that request counter automatically.

## Guided Setup

Run the onboarding wizard after provider authentication:

```bash
nightshift setup
```

The wizard creates or updates the global config, selects projects and tasks, configures budget and safety choices, takes an initial snapshot, previews the schedule, and can install the service.

## Manual Setup

Create either a project config in the current directory or the global config, then inspect and validate the merged result:

```bash
nightshift init --global       # ~/.config/nightshift/config.yaml
# or: nightshift init          # ./nightshift.yaml
nightshift config
nightshift config validate
nightshift doctor
nightshift preview
```

`preview` and the daemon require either `schedule.cron` or `schedule.interval`. A manual `nightshift run` does not require a schedule.
