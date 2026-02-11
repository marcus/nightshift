---
sidebar_position: 6
title: Budget
---

# Budget Management

Nightshift is designed to use tokens you'd otherwise waste. It tracks your remaining budget and never exceeds your configured limits.

## Check Budget

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget --provider codex
nightshift budget --provider ollama
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `budget.mode` | string | `daily` | `daily` or `weekly` usage model |
| `budget.max_percent` | int | `75` | Max % of budget per run |
| `budget.reserve_percent` | int | `5` | Always keep this % in reserve |
| `budget.billing_mode` | string | `subscription` | `subscription` or `api` |
| `budget.calibrate_enabled` | bool | `true` | Enable subscription calibration via snapshots |
| `budget.snapshot_interval` | duration | `30m` | Automatic snapshot cadence |
| `budget.snapshot_retention_days` | int | `90` | Snapshot retention window |
| `budget.week_start_day` | string | `monday` | Week boundary for calibration |
| `budget.db_path` | string | `~/.local/share/nightshift/nightshift.db` | Override DB path |

## Budget Modes

### Daily Mode (recommended)

Each night uses up to `max_percent` of your daily budget (weekly / 7). Consistent, predictable usage.

### Weekly Mode

Uses `max_percent` of *remaining* weekly budget. With `aggressive_end_of_week: true`, spends more near week's end to avoid waste.

## Calibration

Nightshift infers subscription budgets by correlating local token counts with provider usage percentages.

Formula: `inferred_budget = local_tokens / (scraped_pct / 100)`

Confidence levels:
- **low**: 1–2 samples or high variance
- **medium**: stable signal across several snapshots
- **high**: consistent signal across a week or more

```bash
nightshift budget calibrate
```

Enable auto-calibration in config:

```yaml
budget:
  calibrate_enabled: true
  snapshot_interval: 30m
```

> Calibration uses tmux to scrape usage percentages. If tmux is unavailable, snapshots are local-only and budgets fall back to config values.

## API Billing

For API-billed accounts, set explicit token limits:

```yaml
budget:
  billing_mode: api
  weekly_tokens: 1000000
  per_provider:
    claude: 1000000
    codex: 500000
```

`weekly_tokens` and `per_provider` are authoritative for `billing_mode: api`. For subscription users, they act as a fallback until calibration has enough snapshots.

## Budget History

View past budget snapshots:

```bash
nightshift budget history -n 10
nightshift budget snapshot --local-only
```

## Morning Summary

After each run, Nightshift generates a summary at `~/.local/share/nightshift/summaries/nightshift-YYYY-MM-DD.md` covering budget usage, tasks completed, and suggested next steps.

## Safety

- `max_percent` (default 75%) caps how much budget a single run can use
- `reserve_percent` (default 5%) always keeps some budget available for your daytime work
- If budget is exhausted, Nightshift skips remaining tasks gracefully
