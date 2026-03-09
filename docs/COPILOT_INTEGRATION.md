# Copilot Integration

Nightshift supports [GitHub Copilot](https://github.com/features/copilot) as a provider alongside Claude Code and Codex CLI. This document covers setup, billing, configuration, and known limitations.

## Prerequisites

- **GitHub Copilot subscription** (any tier — Free, Pro, Pro+, Business, or Enterprise)
- **GitHub CLI** (`gh`) installed and authenticated, OR the standalone `copilot` binary
- The `gh-copilot` extension: `gh extension install github/gh-copilot`

Verify your installation:

```bash
# Using gh (recommended)
gh copilot --version

# Or standalone binary
copilot --version
```

## Configuration

Add Copilot to your `nightshift.toml`:

```toml
[providers.copilot]
enabled = true
copilot_plan = "pro"  # Sets monthly PRU limit automatically

# Or set an explicit monthly limit:
# monthly_limit = 500
```

### Plan Presets

Set `copilot_plan` to automatically configure your monthly premium request (PRU) limit:

| Plan | `copilot_plan` value | PRUs/month | Monthly cost |
|------|---------------------|-----------|-------------|
| Free | `"free"` | 50 | $0 |
| Pro | `"pro"` | 300 | $10 |
| Pro+ | `"pro_plus"` | 1,500 | $39 |
| Business | `"business"` | 300/user | $19/user |
| Enterprise | `"enterprise"` | 1,000/user | $39/user |

Resolution order: `monthly_limit` (explicit) > `copilot_plan` (preset) > `weekly_tokens * 4` (fallback).

### Provider Preference

Control which provider nightshift tries first:

```toml
[providers]
preference = ["copilot", "claude", "codex"]  # Try Copilot first
```

### Permissions

By default, Copilot runs in safe mode. To allow autonomous operation (required for most nightshift tasks):

```toml
[providers.copilot]
dangerously_skip_permissions = true  # Passes --allow-all-tools --allow-all-urls
```

## How It Works

Nightshift spawns the Copilot CLI in non-interactive mode:

```
gh copilot -- -p "<task prompt>" --no-ask-user --silent [--allow-all-tools --allow-all-urls]
```

The agent receives a structured prompt (plan/implement/review), executes in the project directory, and returns output. Nightshift then parses the result, creates a branch, commits changes, and opens a PR.

## Budget & Premium Requests (PRUs)

### How PRUs Work

GitHub Copilot bills using **Premium Request Units (PRUs)**, not tokens. Each request to the AI costs a number of PRUs based on the model used:

| Model | PRU Multiplier | Best for |
|-------|---------------|----------|
| GPT-4.1 | 0x (free) | Lint fixes, formatting, simple refactors |
| GPT-4.1 Mini | 0.25x | Quick suggestions |
| Claude Haiku | 0.25x | Fast, simple tasks |
| Gemini Flash | 0.25x | Fast, simple tasks |
| GPT-4o | 1x (standard) | General coding tasks |
| Claude Sonnet | 1x (standard) | General coding tasks |
| Gemini Pro | 1x (standard) | General coding tasks |
| Claude Opus 4 | 10x | Complex bug-finding, security analysis |
| o1 | 10x | Complex reasoning |
| o3-mini | 0.33x | Reasoning on a budget |
| o3-pro | 50x | Maximum capability (use sparingly) |

### Budget-Aware Usage

Nightshift tracks Copilot requests in `~/.copilot/nightshift-usage.json`. The counter resets monthly on the 1st at 00:00:00 UTC, matching GitHub's billing cycle.

Check your budget status:

```bash
nightshift budget --provider copilot
```

### Cost Optimization Tips

1. **Use cheap models for cheap tasks** — GPT-4.1 costs 0 PRUs. Perfect for lint-fix, format, and simple refactoring tasks.
2. **Reserve expensive models** — Claude Opus (10x) and o3-pro (50x) should only be used for complex tasks like bug-finding or security analysis.
3. **Set `max_percent`** — Limit how much of your monthly budget nightshift can use per run:
   ```toml
   [budget]
   max_percent = 50  # Never use more than 50% of remaining budget
   ```
4. **Use `reserve_percent`** — Keep a safety buffer for interactive Copilot usage:
   ```toml
   [budget]
   reserve_percent = 25  # Always keep 25% for interactive use
   ```

## Limitations

### No External Usage Visibility

Nightshift can only track requests it makes itself. It cannot see PRUs consumed through:
- VS Code / IDE Copilot usage
- GitHub.com Copilot chat
- Copilot in CLI interactive mode
- Other tools using your Copilot quota

This means budget calculations are estimates. Set `reserve_percent` higher if you use Copilot heavily outside nightshift.

### No Server-Side Quota API

GitHub does not currently expose a "remaining PRUs" API endpoint. Nightshift tracks usage locally, which means:
- Reinstalling or clearing `~/.copilot/nightshift-usage.json` resets the counter
- Usage across multiple machines is not aggregated
- The counter may undercount actual usage

### Model Selection

Nightshift does not currently select models per-task. The Copilot CLI chooses the model based on the task complexity. Future versions may add task-to-model mapping to optimize PRU spend.

## Troubleshooting

### "copilot agent not configured"

The provider's `Execute()` method requires an agent. This is wired automatically when using `nightshift run` or `nightshift task run --provider copilot`. If you see this in custom integrations, call `SetAgent()` on the Copilot provider.

### "copilot CLI not found in PATH"

Install the GitHub CLI and Copilot extension:

```bash
# Install gh CLI
brew install gh  # macOS
# or see https://cli.github.com/

# Install copilot extension
gh auth login
gh extension install github/gh-copilot
```

### High PRU Usage

If nightshift is consuming too many PRUs:
1. Check `nightshift budget --provider copilot` for current usage
2. Lower `max_percent` in config
3. Increase `reserve_percent`
4. Disable expensive task types (e.g., `bug-finder`) and keep cheap ones (`lint-fix`, `format`)

## Related

- [README](../README.md) — General nightshift setup
- [GitHub Copilot Plans](https://github.com/features/copilot#pricing) — Official pricing
- [Premium Requests Documentation](https://docs.github.com/en/copilot/managing-copilot/monitoring-usage-and-entitlements/about-premium-requests) — GitHub's PRU documentation
