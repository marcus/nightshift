# Nightshift

> It finds what you forgot to look for.

**[nightshift.haplab.com](https://nightshift.haplab.com)** · [Docs](https://nightshift.haplab.com/docs/intro) · [Quick Start](https://nightshift.haplab.com/docs/quick-start) · [CLI Reference](https://nightshift.haplab.com/docs/cli-reference)

![Nightshift logo](logo.png)

Your budget resets on the schedule you configure, so you might as well use the idle time. Nightshift runs overnight to find dead code, doc drift, test gaps, security issues, and 20+ other things silently accumulating while you ship features. Like a Roomba for your codebase - runs overnight, worst case you close the PR.

Everything lands as a branch or PR. It never writes directly to your primary branch. Don't like something? Close it. That's the whole rollback plan.

## Features

- **Budget-aware**: Uses the remaining configured budget, never exceeds configurable max (default 75%)
- **Multi-project**: Point it at your repos, it already knows what to look for
- **Zero risk**: Everything is a PR — merge what surprises you, close the rest
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

This walks you through provider configuration, project selection, budget calibration, and daemon setup. It also checks whether Claude Code, Codex, and Copilot CLIs are available. Once complete you can preview what nightshift will do:

```bash
nightshift preview
nightshift budget
```

Or kick off a run immediately:

```bash
nightshift run
```

If you want to create config files directly instead of using the wizard, use:

```bash
nightshift init
nightshift init --global
nightshift config validate
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

# Create or inspect config directly
nightshift init
nightshift config
nightshift config validate

# Check environment and config health
nightshift doctor

# Budget status and calibration
nightshift budget --provider claude
nightshift budget --provider codex
nightshift budget --provider copilot
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
nightshift task run lint-fix --provider copilot --dry-run

# Manage the daemon and service
nightshift daemon start
nightshift daemon start --foreground
nightshift daemon status
nightshift install
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
| `--max-projects` | `1` | Max projects to process (falls back to `schedule.max_projects` when set in config) |
| `--max-tasks` | `1` | Max tasks per project (falls back to `schedule.max_tasks` when set in config) |
| `--random-task` | `false` | Pick a random task from eligible tasks instead of the highest-scored one |
| `--ignore-budget` | `false` | Bypass budget checks (use with caution) |
| `--yes`, `-y` | `false` | Skip the confirmation prompt |
| `--branch`, `-b` | _(current branch)_ | Base branch for new feature branches and metadata |
| `--timeout` | `30m` | Per-agent execution timeout |
| `--no-color` | `false` | Disable colored output |

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
```

Other useful flags:
- `nightshift status --today` to see today's activity summary
- `nightshift daemon start --foreground` for debug
- `--category` — filter tasks by category (pr, analysis, options, safe, map, emergency)
- `--cost` — filter by cost tier (low, medium, high, veryhigh)
- `--prompt-only` — output just the raw prompt text for piping
- `--provider` — required for `task run`, choose claude, codex, or copilot
- `--dry-run` — preview the prompt without executing
- `--timeout` — execution timeout (default 30m)

## Authentication (Subscriptions)

Nightshift supports three AI providers:
- **Claude Code** - Anthropic's Claude via local CLI
- **Codex** - OpenAI's Codex via local CLI
- **GitHub Copilot** - via either `gh` with the Copilot extension or a standalone `copilot` binary

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
# Via gh
gh auth login
gh extension install github/gh-copilot

# Or standalone
npm install -g @github/copilot
```

Requires a GitHub Copilot subscription. See the [Integrations docs](https://nightshift.haplab.com/docs/integrations) for task-source setup and the [Installation docs](https://nightshift.haplab.com/docs/installation) for provider prerequisites.

If you prefer API-based usage, you can authenticate Claude and Codex CLIs with API keys instead.

## Configuration

Full guide: [Configuration docs](https://nightshift.haplab.com/docs/configuration) · [Budget docs](https://nightshift.haplab.com/docs/budget) · [Scheduling docs](https://nightshift.haplab.com/docs/scheduling) · [Tasks docs](https://nightshift.haplab.com/docs/tasks)

Nightshift uses YAML config files to define:

- Token budget limits
- Target repositories
- Task priorities
- Schedule preferences

Run `nightshift setup` to create or update the global config at `~/.config/nightshift/config.yaml`. Run `nightshift init` to create a project-local `nightshift.yaml`, and `nightshift config validate` to check both files.

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
  weekly_tokens: 700000

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

projects:
  - path: ~/code/sidecar
  - path: ~/code/td

integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true
    - github_issues: true
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
- **gofmt** — flags any staged `.go` files that need formatting
- **go vet** — catches common correctness issues
- **go build** — ensures the project compiles

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
