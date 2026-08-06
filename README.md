# Nightshift

> It finds what you forgot to look for.

**[nightshift.haplab.com](https://nightshift.haplab.com)** · [Docs](https://nightshift.haplab.com/docs/intro) · [Quick Start](https://nightshift.haplab.com/docs/quick-start) · [CLI Reference](https://nightshift.haplab.com/docs/cli-reference)

![Nightshift logo](logo.png)

Your tokens get reset every week, you might as well use them. Nightshift runs overnight to find dead code, doc drift, test gaps, security issues, and 20+ other things silently accumulating while you ship features. Like a Roomba for your codebase - runs overnight, worst case you close the PR.

Everything lands as a branch or PR. It never writes directly to your primary branch. Don't like something? Close it. That's the whole rollback plan.

## Features

- **Budget-aware**: Uses remaining daily allotment, never exceeds configurable max (default 75%)
- **Multi-project**: Point it at your repos, it already knows what to look for
- **Zero risk**: Everything is a PR - merge what surprises you, close the rest
- **Great DX**: Thoughtful CLI defaults with clear output and reports

## Installation

Full guide: [Installation docs](https://nightshift.haplab.com/docs/installation)

```bash
brew install marcus/tap/nightshift
```

Binary downloads are available on the GitHub releases page.

Manual install:

```bash
go install github.com/marcus/nightshift/cmd/nightshift@latest
```

## Getting Started

Full guide: [Quick Start docs](https://nightshift.haplab.com/docs/quick-start)

After installing, run the guided setup:

```bash
nightshift setup
```

This walks you through provider configuration, project selection, budget calibration, and daemon setup. It covers Claude, Codex, and Copilot, and checks for the provider CLIs on `PATH`. If you want a manual flow instead, use `nightshift init` for a project config, `nightshift init --global` for `~/.config/nightshift/config.yaml`, and `nightshift config validate` to verify the result.

Once complete you can preview what nightshift will do:

```bash
nightshift preview
nightshift budget
```

Or kick off a run immediately:

```bash
nightshift run
```

## Common CLI Usage

Full reference: [CLI Reference docs](https://nightshift.haplab.com/docs/cli-reference)

```bash
# Preview next scheduled runs with prompt previews
nightshift preview -n 3
nightshift preview --long
nightshift preview --explain
nightshift preview --plain
nightshift preview --json
nightshift preview --write ./nightshift-prompts

# Guided global setup
nightshift setup

# Bootstrap or inspect config
nightshift init
nightshift init --global
nightshift config
nightshift config get budget.max_percent
nightshift config set budget.max_percent 15
nightshift config validate

# Check environment and config health
nightshift doctor

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
nightshift task show skill-groom
nightshift task show lint-fix --prompt-only

# Run a task immediately
nightshift task run lint-fix --provider claude
nightshift task run skill-groom --provider codex --dry-run
nightshift task run lint-fix --provider codex --dry-run

# Manage the scheduler and service lifecycle
nightshift daemon start
nightshift daemon start --foreground
nightshift daemon start --timeout 45m
nightshift daemon status
nightshift daemon stop
nightshift install
nightshift install launchd
nightshift install systemd
nightshift install cron
nightshift uninstall
```

If `gum` is available, preview output is shown through the gum pager. Use `--plain` to disable.

### `nightshift run`

Before executing, `nightshift run` displays a **preflight summary** showing the
selected provider, budget status, projects, and planned tasks. In interactive
terminals you are prompted for confirmation; in non-TTY environments (cron,
daemon, CI) confirmation is auto-skipped.

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show preflight summary and exit without executing |
| `--project`, `-p` | _(all configured)_ | Target a single project directory |
| `--task`, `-t` | _(auto-select)_ | Run a specific task by name |
| `--max-projects` | `1` | Max projects to process (ignored when `--project` is set) |
| `--max-tasks` | `1` | Max tasks per project (ignored when `--task` is set) |
| `--random-task` | `false` | Pick a random task from eligible tasks instead of the highest-scored one |
| `--ignore-budget` | `false` | Bypass budget checks (use with caution) |
| `--branch`, `-b` | _(current branch)_ | Base branch for new feature branches |
| `--timeout` | `30m` | Per-agent execution timeout |
| `--yes`, `-y` | `false` | Skip the confirmation prompt |

```bash
# Interactive run with preflight summary + confirmation prompt
nightshift run

# Non-interactive: skip confirmation
nightshift run --yes

# Dry-run: show preflight summary and exit
nightshift run --dry-run

# Process up to 3 projects, 2 tasks each
nightshift run --max-projects 3 --max-tasks 2

# Pick a random eligible task
nightshift run --random-task

# Bypass budget limits (shows warning)
nightshift run --ignore-budget

# Target a specific project and task directly
nightshift run -p ./my-project -t lint-fix

# Base new branches off develop
nightshift run --branch develop

# Give agents more time
nightshift run --timeout 45m
```

Other useful flags:
- `nightshift status --today` to see today's activity summary
- `nightshift daemon start --foreground` for debug
- `--category` - filter tasks by category (pr, analysis, options, safe, map, emergency)
- `--cost` - filter by cost tier (low, medium, high, veryhigh)
- `--prompt-only` - output just the raw prompt text for piping
- `--provider` - required for `task run`, choose claude, codex, or copilot
- `--dry-run` - preview the prompt without executing
- `--timeout` - execution timeout (default 30m)

## Provider Setup

Nightshift supports three AI providers:
- **Claude Code** - Anthropic's Claude via local CLI
- **Codex** - OpenAI's Codex via local CLI
- **GitHub Copilot** - GitHub's Copilot via the standalone `copilot` binary or `gh copilot`

### Claude Code

```bash
claude
/login
```

Supports Claude.ai subscriptions or Anthropic Console credentials.

### Codex

```bash
codex --login
```

Supports signing in with ChatGPT or an API key.

### GitHub Copilot

```bash
# Standalone binary
npm install -g @github/copilot
# or
curl -fsSL https://gh.io/copilot-install | bash
# GitHub CLI extension
gh extension install github/gh-copilot
```

Nightshift prefers the standalone `copilot` binary when it exists, otherwise it falls back to `gh copilot`. `nightshift setup` checks for both and uses whichever is on `PATH`.

If you use `gh copilot`, authenticate with `gh auth login` first.

If you prefer API-based usage, you can authenticate Claude and Codex CLIs with API keys instead.

## Configuration

Full guide: [Configuration docs](https://nightshift.haplab.com/docs/configuration) · [Budget docs](https://nightshift.haplab.com/docs/budget) · [Scheduling docs](https://nightshift.haplab.com/docs/scheduling) · [Tasks docs](https://nightshift.haplab.com/docs/tasks)

Nightshift uses YAML config files to define:

- Token budget limits
- Target repositories
- Task priorities
- Schedule preferences

Use `nightshift setup` for guided onboarding. Use `nightshift init` to create a project config in the current directory, or `nightshift init --global` to create `~/.config/nightshift/config.yaml`. `nightshift config` shows the merged configuration, `nightshift config get KEY` reads a value, `nightshift config set KEY VALUE` writes a value, and `nightshift config validate` checks both the global and project files.

See the [full configuration docs](https://nightshift.haplab.com/docs/configuration) for detailed options.

Minimal example:

```yaml
schedule:
  cron: "0 2 * * *"
  max_projects: 1
  max_tasks: 1

budget:
  mode: daily
  max_percent: 75
  reserve_percent: 5
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m

providers:
  preference:
    - claude
    - codex
    - copilot
  claude:
    enabled: true
    data_path: "~/.claude"
    dangerously_skip_permissions: true
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: true
  copilot:
    enabled: true
    data_path: "~/.copilot"
    dangerously_skip_permissions: false

projects:
  - path: ~/code/sidecar
  - path: ~/code/td
```

Default provider order is `claude -> codex -> copilot`. Nightshift uses whichever provider has budget remaining, unless you override the provider or disable one in config.

Task selection:

```yaml
tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - bug-finder
  priorities:
    lint-fix: 1
    skill-groom: 2
    bug-finder: 2
  intervals:
    lint-fix: "24h"
    skill-groom: "168h"
    docs-backfill: "168h"
```

Each task has a default cooldown interval to prevent the same task from running too frequently on a project (e.g., 24h for lint-fix, 7d for docs-backfill). Override per-task with `tasks.intervals`.

`skill-groom` is enabled by default. Add it to `tasks.disabled` if you want to opt out. It updates project-local skills under `.claude/skills` and `.codex/skills` using `README.md` as project context and starts Agent Skills docs lookup from `https://agentskills.io/llms.txt`.

## Development

### Pre-commit hooks

Install the git pre-commit hook to catch formatting and vet issues before pushing:

```bash
make install-hooks
```

This symlinks `scripts/pre-commit.sh` into `.git/hooks/pre-commit`. The hook runs:
- **gofmt** - flags any staged `.go` files that need formatting
- **go vet** - catches common correctness issues
- **go build** - ensures the project compiles

To bypass in a pinch: `git commit --no-verify`

## Uninstalling

```bash
# Remove the system service
nightshift uninstall

# Remove configs and data (optional)
rm -rf ~/.config/nightshift ~/.local/share/nightshift

# Remove the binary
rm "$(which nightshift)"
```

## License

MIT - see [LICENSE](LICENSE) for details.
