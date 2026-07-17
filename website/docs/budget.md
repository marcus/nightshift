---
sidebar_position: 6
title: Budget
---

# Budget Management

Nightshift converts provider usage into an allowance for each run. Use `nightshift budget` to see the resolved budget source, current usage, reserve deductions, predicted daytime use, and final allowance.

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget --provider codex
nightshift budget --provider copilot
```

## Configuration

| Option | Default | Implemented behavior |
|--------|---------|----------------------|
| `budget.mode` | `daily` | `daily` or `weekly` allowance calculation |
| `budget.max_percent` | `75` | Percentage applied to the available base; validation accepts `0` through `100`, and `0` is normalized back to `75` during calculation |
| `budget.reserve_percent` | `5` | Percentage subtracted after applying `max_percent`; accepts `0` through `100` |
| `budget.weekly_tokens` | `700000` | Fallback weekly token budget |
| `budget.per_provider` | none | Provider-specific value replacing `weekly_tokens` |
| `budget.aggressive_end_of_week` | `false` | Enables the weekly-mode end-of-period multiplier |
| `budget.billing_mode` | `subscription` | `subscription` uses calibration when available; `api` uses configured values |
| `budget.calibrate_enabled` | `true` | Enables tmux scraping and calibrated estimates for subscription mode |
| `budget.snapshot_interval` | `30m` | Daemon snapshot cadence; an invalid or non-positive duration disables the loop with a warning |
| `budget.snapshot_retention_days` | `90` | Prune age; `0` disables pruning |
| `budget.week_start_day` | `monday` | Groups calibration snapshots by Monday or Sunday week start |
| `budget.db_path` | `~/.local/share/nightshift/nightshift.db` | Snapshot, state, and report database |

## Allowance Calculations

Nightshift first resolves a weekly budget from API/config values or calibration, then asks the provider for its used percentage.

### Daily mode

```text
daily_budget = weekly_budget / 7
available_today = daily_budget * (1 - used_percent / 100)
pre_reserve_allowance = available_today * max_percent / 100
reserve = daily_budget * reserve_percent / 100
allowance = max(0, pre_reserve_allowance - reserve - predicted_daytime_usage)
```

The percent is applied to what remains today, not to the entire weekly budget. Integer conversion truncates fractional tokens.

### Weekly mode

```text
remaining_weekly = weekly_budget * (1 - used_percent / 100)
pre_reserve_allowance = (remaining_weekly / remaining_days)
                        * max_percent / 100
                        * end_of_period_multiplier
reserve = remaining_weekly * reserve_percent / 100
allowance = max(0, pre_reserve_allowance - reserve - predicted_daytime_usage)
```

For Claude, `remaining_days` assumes a Sunday reset (Sunday itself returns seven days). Codex uses the secondary rate-limit reset timestamp when present and otherwise falls back to seven days. Copilot uses its next monthly reset and otherwise falls back to 30 days.

With `aggressive_end_of_week: true`, the current multiplier is `1x` when two days remain and `2x` when one day remains. It does not increase the allowance earlier in the week.

### Reserve and daytime forecast

`reserve_percent` is a deduction from the mode's current budget base on every calculation; it is not a separately tracked account balance. After that deduction, Nightshift estimates remaining daytime usage from hourly snapshot averages and subtracts the estimate. The lookback uses `snapshot_retention_days`, capped at 30 days; a non-positive value falls back to 14 days. With no usable snapshot history, the daytime deduction is zero.

An allowance at or below zero causes automatic provider selection to try the next provider, then skip the run if none remain. `nightshift run --ignore-budget` is the explicit manual override. Direct `nightshift task run ... --provider ...` does not consult the budget manager.

## Provider Usage Sources

- Claude daily and weekly percentages use local `stats-cache.json` token totals, falling back to session JSONL scanning. The daily denominator is the resolved weekly budget divided by seven.
- Codex daily mode prefers today's local billable-token total and falls back to the primary (five-hour) rate limit. Weekly mode prefers the secondary rate-limit percentage and falls back to local weekly tokens.
- Copilot uses a local request-counter model. Both modes are estimates, and the current agent execution path does not increment the counter automatically; see [Integrations](/docs/integrations).

## Subscription Calibration

Snapshots combine local usage with a provider percentage scraped from an interactive CLI inside tmux:

```text
inferred_weekly_budget = local_weekly_usage / (scraped_percent / 100)
```

```bash
nightshift budget snapshot
nightshift budget snapshot --provider claude
nightshift budget snapshot --local-only
nightshift budget history -n 10
nightshift budget calibrate
```

`budget snapshot` can collect Claude, Codex, or Copilot local data. tmux scraping exists only for Claude and Codex; Copilot snapshots are local-only. The daemon takes an immediate snapshot and then repeats at `snapshot_interval`, but its automatic loop currently covers only enabled Claude and Codex providers.

Calibration uses samples from the configured week with local usage greater than zero and scraped percentages from 10% through 95%. It removes median-absolute-deviation outliers when at least three samples exist, takes the median inferred budget, and rounds to the nearest 1,000 tokens. If the current week has no samples, it tries the previous week before falling back to config.

Confidence is:

- `none`: no usable calibrated samples
- `low`: one or two samples, or higher variance
- `medium`: three to five samples with coefficient of variation at most 15%, or more samples at most 15%
- `high`: more than five samples with coefficient of variation at most 10%

If tmux is absent or scraping fails, the snapshot is still stored as local-only and cannot contribute a percentage-based calibration sample. Install tmux and authenticate the provider CLI, or use API mode with explicit values.

## API Billing

API mode disables calibration after configuration is loaded and treats configured budgets as authoritative:

```yaml
budget:
  billing_mode: api
  weekly_tokens: 1000000
  per_provider:
    claude: 1000000
    codex: 500000
```

`per_provider` wins for a matching provider; otherwise `weekly_tokens` is used. API mode does not fetch provider billing limits from a remote API.

## History and Reports

`nightshift budget history` reads stored snapshots (20 per provider by default). `snapshot_retention_days` controls the daemon's daily prune job. A run summary is written to `~/.local/share/nightshift/summaries/summary-YYYY-MM-DD.md`; structured per-run reports are under `~/.local/share/nightshift/reports/`.
