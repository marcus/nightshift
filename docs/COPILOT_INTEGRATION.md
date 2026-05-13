# GitHub Copilot Integration

Nightshift can run tasks through GitHub Copilot CLI in the same provider pipeline as Claude Code and Codex. Copilot is selected when it is enabled, appears in `providers.preference`, its CLI is available, and the budget manager reports remaining allowance.

## Installation

Nightshift supports either the standalone Copilot CLI or the GitHub CLI Copilot extension.

Standalone:

```bash
npm install -g @github/copilot
```

GitHub CLI extension:

```bash
gh extension install github/gh-copilot
```

If both `copilot` and `gh` are on `PATH`, Nightshift prefers the standalone `copilot` binary. Otherwise it falls back to `gh copilot`.

## Authentication

Copilot requires a GitHub account with an active Copilot subscription. Authenticate through GitHub CLI before using Nightshift:

```bash
gh auth login
gh auth status
```

Then confirm Copilot is available:

```bash
gh copilot -- --version
# or, for standalone installs:
copilot --version
```

## Configuration

Copilot is enabled by default in Nightshift's config defaults, and the default provider preference order is `claude`, `codex`, then `copilot`. Add it explicitly when you want Copilot in your configured order:

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
  copilot:
    enabled: true
    data_path: "~/.copilot"
    dangerously_skip_permissions: false
```

`providers.copilot.data_path` stores Nightshift's local request counter at `nightshift-usage.json`. The file is separate from GitHub's own account usage data.

## Execution Model

Nightshift runs Copilot non-interactively with a prompt and disables user questions:

```bash
copilot -p "<prompt>" --no-ask-user --silent
```

When using GitHub CLI passthrough, the equivalent command is:

```bash
gh copilot -- -p "<prompt>" --no-ask-user --silent
```

If `providers.copilot.dangerously_skip_permissions` is true, Nightshift also passes:

```bash
--allow-all-tools --allow-all-urls
```

Leave this false for interactive or exploratory use. Set it only for unattended cron, daemon, or CI runs where Copilot must not stop for tool or URL prompts.

## Running a Task

Run Copilot directly for a single task:

```bash
nightshift task run docs-backfill --provider copilot
nightshift task run lint-fix --provider copilot --dry-run
```

For scheduled or immediate runs, include Copilot in `providers.preference` and run:

```bash
nightshift run
nightshift run --yes
```

Nightshift chooses the first enabled provider in preference order that has a CLI available and enough remaining budget.

## Budget Tracking

GitHub Copilot does not expose authoritative local usage percentages to Nightshift. Nightshift tracks Copilot conservatively by counting successful requests it sends through the Copilot provider:

- Each Nightshift Copilot execution counts as one premium request.
- The counter resets monthly on the first day of the month at 00:00:00 UTC.
- The counter only includes Copilot usage made through Nightshift.
- Token costs are reported as zero because Copilot uses request limits rather than per-token pricing.

Nightshift's budget manager still uses `budget.weekly_tokens` or `budget.per_provider.copilot` as the configured allowance input. Internally it approximates a monthly Copilot request limit as four times the configured weekly provider budget, then applies the same daily or weekly budget mode, max percent, reserve percent, and daytime usage reserve logic as other providers.

Example request-limit-oriented configuration:

```yaml
budget:
  mode: weekly
  max_percent: 75
  reserve_percent: 5
  per_provider:
    copilot: 75
```

With Copilot, treat this value as a request allowance rather than a token allowance.

## Limitations

- GitHub does not provide Nightshift with a local authoritative remaining-quota API.
- Usage outside Nightshift is not counted in `~/.copilot/nightshift-usage.json`.
- Daily and weekly Copilot percentages are estimates derived from monthly request tracking.
- The low-level provider adapter in `internal/providers/copilot.go` is used for budget tracking; task execution uses `internal/agents/copilot.go`.

## Troubleshooting

`copilot CLI not found in PATH`: Install either `copilot` or `gh`, then make sure the binary is visible to the shell that starts Nightshift.

`copilot CLI not found in PATH (install via 'gh' or standalone)`: For `gh` mode, install the `github/gh-copilot` extension and verify `gh extension list` includes it.

Copilot waits for permission: Set `providers.copilot.dangerously_skip_permissions: true` for unattended runs, or run interactively and approve the prompt.

Budget shows unexpected request counts: Inspect or remove `~/.copilot/nightshift-usage.json`. Nightshift will recreate it for the current UTC month.

Related docs:

- [Technical specification](SPEC.md)
- [Run lifecycle](guides/run-lifecycle.md)
- [Codex budget tracking](guides/codex-budget-tracking.md)
- [Provider calibration](guides/provider-calibration.md)
