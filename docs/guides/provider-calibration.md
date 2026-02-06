# Provider Calibration Guide

This guide shows how to compare Claude and Codex token behavior on your own machine, using the same local session artifacts Nightshift reads for budgeting.

Use this whenever model behavior changes, provider limits shift, or you suspect your task cost assumptions are off.

## What This Solves

Nightshift currently mixes multiple measurements:

- Provider-reported usage percentages (authoritative for rate limits)
- Locally parsed token counters (from CLI session files)
- Task cost tiers (static ranges in task definitions)

When these drift, budget predictions can become too aggressive or too conservative. This guide gives you a repeatable way to re-check calibration.

## Tooling Added

The repo now includes a comparison tool:

- `cmd/provider-calibration`
- `scripts/provider-calibration` (wrapper)

It computes per-session and per-user-turn distributions for:

- Codex primary tokens: billable (`non-cached input + output + reasoning`)
- Codex alt tokens: raw (`input + cached input + output + reasoning`)
- Claude primary tokens: `input + output`
- Claude alt tokens: `input + output + cache_read + cache_creation`

And outputs cross-provider ratios plus a suggested multiplier.

## Prerequisites

- Local Claude and/or Codex session data exists.
- Nightshift repo is available locally.
- Go toolchain installed.

Optional but recommended:

- Use a repo filter (`--repo`) to avoid cross-project noise.
- Use Codex originator filter (`--codex-originator codex_cli_rs`) to avoid mixing desktop chat sessions with CLI runs.

## Quick Start

From repo root:

```bash
# Human-readable report
scripts/provider-calibration \
  --repo /absolute/path/to/your/repo \
  --codex-originator codex_cli_rs \
  --min-user-turns 2

# Machine-readable report
scripts/provider-calibration \
  --repo /absolute/path/to/your/repo \
  --codex-originator codex_cli_rs \
  --min-user-turns 2 \
  --json > /tmp/provider-calibration.json
```

Equivalent direct invocation:

```bash
go run ./cmd/provider-calibration \
  --repo /absolute/path/to/your/repo \
  --codex-originator codex_cli_rs \
  --min-user-turns 2
```

## Important Flags

- `--repo`: exact cwd match after path normalization. Strongly recommended.
- `--codex-originator`: set to `codex_cli_rs` for CLI-only Codex sessions.
- `--min-user-turns`: filters out tiny/stub sessions.
- `--json`: structured output for dashboards or historical tracking.

Paths (override only when needed):

- `--codex-sessions` default: `~/.codex/sessions`
- `--claude-projects` default: `~/.claude/projects`

## How To Interpret Output

The report provides ratios in two main families:

1. Per-session medians
- Good for coarse workload shape checks.
- Sensitive to long/short session artifacts.

2. Per-user-turn medians
- Better normalization when one provider tends to have longer sessions.
- Preferred for multiplier discussions.

Suggested multiplier uses:

- `codex_primary_per_user_turn / claude_alt_per_user_turn`

Reason: Claude `alt` includes cache-related fields, which is usually closer to subscription-accounting behavior than Claude `input+output` alone.

## Confidence Rules

Treat results as directional unless both providers have enough samples.

Minimum recommended:

- At least 20 sessions per provider for initial adjustment
- At least 50 sessions per provider for stable policy changes

If Codex or Claude sample count is low, do not change global defaults yet.

## Operational Process (Recommended)

Run this weekly (or after major model/provider changes):

1. Collect data
- Let Nightshift run normally for several days.
- Ensure both providers are actually used.

2. Generate report
- Run `scripts/provider-calibration ... --json`.

3. Compare trend over time
- Keep each JSON output with a date-stamped filename.
- Watch whether ratio trend is stable or drifting.

4. Decide action
- Stable ratio drift: adjust task cost assumptions.
- High variance/noisy ratio: collect more data, do not overfit.

## Applying Results in Nightshift Today

Nightshift does not yet expose a first-class runtime multiplier setting for provider task costs. Today, use calibration outputs to guide:

- Task cost-tier tuning in `internal/tasks/tasks.go`
- Budget policy conservatism (`max_percent`, `reserve_percent`)
- Provider preference strategy (e.g., choose provider order based on confidence)

## Keep It General For New Models

When new models/providers appear:

1. Re-run with same flags and repo scope.
2. Compare new run JSON vs previous baseline.
3. Validate sample sizes before changing policy.
4. If parser support is needed for new provider session formats, extend `cmd/provider-calibration/main.go` with a new collector and reuse the same summary/ratio functions.

## Troubleshooting

No Codex sessions found:

- Verify `--repo` matches the exact Codex session `cwd`.
- Remove `--codex-originator` temporarily to inspect mixed-origin sessions.

Ratios look extreme:

- Check sample count warnings.
- Raise `--min-user-turns` to filter tiny sessions.
- Confirm you are not mixing desktop interactive chats with automation sessions.

Slow run time:

- Narrow scope with `--repo`.
- Archive old session files if your local history is very large.

## Related Docs

- `docs/guides/codex-budget-tracking.md`
- `docs/user-guide.md` (Budget Calibration section)
