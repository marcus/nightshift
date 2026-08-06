# Nightshift

> It finds what you forgot to look for.

**[nightshift.haplab.com](https://nightshift.haplab.com)** · [Docs](https://nightshift.haplab.com/docs/intro) · [Quick Start](https://nightshift.haplab.com/docs/quick-start) · [CLI Reference](https://nightshift.haplab.com/docs/cli-reference)

![Nightshift logo](logo.png)

Your tokens get reset every week, you might as well use them. Nightshift runs overnight to find dead code, doc drift, test gaps, security issues, and 20+ other things silently accumulating while you ship features. Like a Roomba for your codebase — runs overnight, worst case you close the PR.

Everything lands as a branch or PR. It never writes directly to your primary branch. Don't like something? Close it. That's the whole rollback plan.

## Features

- **Budget-aware**: Uses remaining daily allotment, never exceeds configurable max (default 75%)
- **Multi-project**: Point it at your repos, it already knows what to look for
- **Multi-provider**: Runs with Claude Code, Codex, or GitHub Copilot
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

Nightshift needs at least one provider CLI available in `PATH`:

- Claude Code via `claude`
- Codex via `codex`
- GitHub Copilot via standalone `copilot` or `gh copilot`

## Getting Started

Full guide: [Quick Start docs](https://nightshift.haplab.com/docs/quick-start)

After installing, run the guided setup:

```bash
nightshift setup
```

This walks you through provider configuration, project selection, budget calibration, and daemon setup. Once complete you can preview what nightshift will do:

```bash
nightshift preview
nightshift budget
```

Or kick off a run immediately:

```bash
nightshift run
```

If you prefer starter config files instead of the wizard:

```bash
nightshift init --global
nightshift init
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

# Guided setup and config inspection
nightshift setup
nightshift init --global
nightshift config
nightshift config get providers.preference
nightshift config set budget.max_percent 60 --global
nightshift config validate

# Check environment and config health
nightshift doctor

# Budget status and calibration
nightshift budget --provider claude
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
nightshift task run docs-backfill --provider copilot --dry-run

# Inspect logs, reports, and stats
nightshift logs --tail 100
nightshift logs --follow --component daemon
nightshift report --report overview --period last-night
nightshift report --report tasks --period last-7d --format markdown
nightshift stats --period last-7d

# Analyze repository ownership concentration
nightshift busfactor .

# Background execution
nightshift install
nightshift daemon start --foreground
nightshift daemon status
nightshift daemon stop
nightshift uninstall
```

If `gum` is available, preview output is shown through the gum pager. Use `--plain` to disable.

### `nightshift run`

Before executing, `nightshift run` displays a **preflight summary** showing the selected provider, budget status, projects, and planned tasks. In interactive terminals you are prompted for confirmation; in non-TTY environments (cron, daemon, CI) confirmation is auto-skipped.

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show preflight summary and exit without executing |
| `--project`, `-p` | _(all configured)_ | Target a single project directory |
| `--task`, `-t` | _(auto-select)_ | Run a specific task by name |
| `--max-projects` | `1` | Max projects to process (ignored when `--project` is set) |
| `--max-tasks` | `1` | Max tasks per project (ignored when `--task` is set) |
| `--random-task` | `false` | Pick a random task from eligible tasks instead of the highest-scored one |
| `--ignore-budget` | `false` | Bypass budget checks (use with caution) |
| `--yes`, `-y` | `false` | Skip the confirmation prompt |
| `--branch`, `-b` | current branch | Base branch for new feature branches |
| `--timeout` | `30m` | Per-agent execution timeout |

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

# Use a different base branch for generated work
nightshift run --branch develop
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
- **Codex** - OpenAI's GPT via local CLI
- **GitHub Copilot** - Copilot CLI via standalone `copilot` or `gh copilot`

### Claude Code

```bash
claude auth login
```

Supports Claude.ai subscriptions or Anthropic Console credentials.

### Codex

```bash
codex login
```

Supports interactive sign-in or API-key login:

```bash
printenv OPENAI_API_KEY | codex login --with-api-key
```

### GitHub Copilot

```bash
gh auth login
gh extension install github/gh-copilot
gh copilot --help
```

Nightshift prefers a standalone `copilot` binary when one is present in `PATH`. If not, it falls back to `gh copilot`, which Nightshift detects through `gh extension list`.

To verify provider CLI access without executing a task:

```bash
nightshift task run lint-fix --provider claude --dry-run
nightshift task run lint-fix --provider codex --dry-run
nightshift task run lint-fix --provider copilot --dry-run
```

Then run `nightshift doctor` to inspect config, data paths, usage, and snapshots for the providers you have enabled in config.

If you prefer API-based usage, you can authenticate Claude and Codex CLIs with API keys instead.

## Configuration

Full guide: [Configuration docs](https://nightshift.haplab.com/docs/configuration) · [Budget docs](https://nightshift.haplab.com/docs/budget) · [Scheduling docs](https://nightshift.haplab.com/docs/scheduling) · [Tasks docs](https://nightshift.haplab.com/docs/tasks)

Nightshift uses YAML config files to define:

- Token budget limits
- Target repositories
- Task priorities
- Schedule preferences

Create or update config with either:

```bash
nightshift setup
nightshift init --global
nightshift init
```

Runtime layering is:

1. Built-in defaults
2. Global config: `~/.config/nightshift/config.yaml`
3. Project config: `nightshift.yaml`
4. Environment overrides such as `NIGHTSHIFT_BUDGET_MAX_PERCENT`

Inspect and edit config from the CLI:

```bash
nightshift config
nightshift config get budget.max_percent
nightshift config set providers.copilot.enabled true --global
nightshift config validate
```

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
  preference:
    - claude
    - codex
    - copilot
  claude:
    enabled: true
    data_path: "~/.claude"
    dangerously_skip_permissions: false
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: false
  copilot:
    enabled: false
    data_path: "~/.copilot"
    dangerously_skip_permissions: false

projects:
  - path: ~/code/sidecar
  - path: ~/code/td

integrations:
  claude_md: true
  agents_md: true
  task_sources:
    - td:
        enabled: true
        teach_agent: true
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
