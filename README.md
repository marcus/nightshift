# Nightshift

> Wake up to a cleaner codebase

![Nightshift logo](logo.png)

A Go CLI that runs overnight to perform AI-powered maintenance tasks on your codebase, using your remaining daily token budget from Claude Code/Codex subscriptions.

## Features

- **Budget-aware**: Uses remaining daily allotment, never exceeds configurable max (default 10%)
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
# Initialize config in current directory
nightshift init

# Run maintenance tasks
nightshift run

# Check status of last run
nightshift status
```

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

Nightshift uses a YAML config file (`nightshift.yaml`) to define:

- Token budget limits
- Target repositories
- Task priorities
- Schedule preferences

See [SPEC.md](docs/SPEC.md) for detailed configuration options.

## License

MIT - see [LICENSE](LICENSE) for details.
