# Nightshift

> Wake up to a cleaner codebase

![Nightshift logo](logo.png)

A Go CLI that runs overnight to perform AI-powered maintenance tasks on your codebase, using your remaining daily token budget from Claude Code/Codex subscriptions.

## Features

- **Budget-aware**: Uses remaining daily allotment, never exceeds configurable max (default 75%)
- **Multi-project support**: Works across multiple repos
- **Configurable tasks**: From auto-PRs to analysis reports
- **Great DX**: Built with bubbletea/lipgloss for a delightful CLI experience

## Installation

```bash
brew install marcus/tap/nightshift
```

Binary downloads are available on the GitHub releases page.

Manual install:

```bash
go install github.com/marcus/nightshift/cmd/nightshift@latest
```

## Quick Start

```bash
# Interactive setup (global config + snapshot + daemon)
nightshift setup

# Initialize config in current directory
nightshift init

# Run maintenance tasks
nightshift run

# Check status of last run
nightshift status
```

## Common CLI Usage

```bash
# Preview next scheduled runs with prompt previews
nightshift preview -n 3
nightshift preview --long
nightshift preview --write ./nightshift-prompts

# Guided global setup
nightshift setup

# Check environment and config health
nightshift doctor

# Launch the TUI
nightshift tui

# Budget status and calibration
nightshift budget --provider claude
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate

# Browse and inspect available tasks
nightshift task list
nightshift task list --category pr
nightshift task list --cost low --json

# Show task details and planning prompt
nightshift task show lint-fix
nightshift task show lint-fix --prompt-only

# Run a task immediately
nightshift task run lint-fix --provider claude
nightshift task run lint-fix --provider codex --dry-run
```

Useful flags:
- `nightshift run --dry-run` to simulate tasks without changes
- `nightshift run --project <path>` to target a single repo
- `nightshift run --task <task-type>` to run a specific task
- `nightshift status --today` to see today’s activity summary
- `nightshift daemon start --foreground` for debug
- `--category` — filter tasks by category (pr, analysis, options, safe, map, emergency)
- `--cost` — filter by cost tier (low, medium, high, veryhigh)
- `--prompt-only` — output just the raw prompt text for piping
- `--provider` — required for `task run`, choose claude or codex
- `--dry-run` — preview the prompt without executing
- `--timeout` — execution timeout (default 30m)

## Authentication (Subscriptions)

Nightshift relies on the local Claude Code and Codex CLIs. If you have subscriptions, you can sign in via the CLIs without API keys.

```bash
# Claude Code
claude
/login

# Codex
codex --login
```

Claude Code login supports Claude.ai subscriptions or Anthropic Console credentials. Codex CLI supports signing in with ChatGPT or an API key.

If you prefer API-based usage, you can authenticate those CLIs with API keys instead.

## Configuration

Nightshift uses YAML config files to define:

- Token budget limits
- Target repositories
- Task priorities
- Schedule preferences

Run `nightshift setup` to create/update the global config at `~/.config/nightshift/config.yaml`.

See [SPEC.md](docs/SPEC.md) for detailed configuration options.

Minimal example:

```yaml
schedule:
  cron: "0 2 * * *"

budget:
  mode: daily
  max_percent: 75
  reserve_percent: 5
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m

providers:
  claude:
    enabled: true
    data_path: "~/.claude"
  codex:
    enabled: true
    data_path: "~/.codex"

projects:
  - path: ~/code/sidecar
  - path: ~/code/td
```

Task selection:

```yaml
tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - bug-finder
  priorities:
    lint-fix: 1
    bug-finder: 2
```

## License

MIT - see [LICENSE](LICENSE) for details.
