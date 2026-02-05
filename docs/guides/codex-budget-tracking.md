# Codex Budget Tracking

How nightshift tracks and budgets OpenAI Codex CLI usage.

## Data Sources

Nightshift uses two independent data sources for Codex budget management:

### 1. Rate Limits (Authoritative)

Codex session JSONL files contain `rate_limits` objects reported by OpenAI's API:

```json
{
  "rate_limits": {
    "primary": {
      "used_percent": 19.0,
      "window_minutes": 300,
      "resets_at": 1738800000
    },
    "secondary": {
      "used_percent": 31.0,
      "window_minutes": 10080,
      "resets_at": 1739145600
    }
  }
}
```

- **Primary**: 5-hour rolling window. Used for daily-mode budget decisions.
- **Secondary**: 7-day rolling window. Used for weekly-mode budget decisions.
- **`resets_at`**: Unix timestamp when the window resets.

Rate limits are the authoritative source for budget percentage calculations because they reflect OpenAI's actual accounting, including server-side prompt caching and any temporary generosity (e.g., model launch bonuses).

### 2. Local Token Counts (Measured)

Codex session files also contain `total_token_usage` events:

```json
{
  "type": "event_msg",
  "payload": {
    "type": "token_count",
    "info": {
      "total_token_usage": {
        "input_tokens": 67256790,
        "cached_input_tokens": 66140928,
        "output_tokens": 1023456,
        "reasoning_output_tokens": 487231,
        "total_tokens": 68767477
      }
    }
  }
}
```

**Important**: `total_token_usage` is cumulative within a session. The last event contains the session total. For sessions with multiple events, nightshift computes the delta between first and last events to get the session's contribution.

#### Billable Token Calculation

The raw `total_tokens` field includes `cached_input_tokens`, which do not count against rate limits. Nightshift computes billable tokens as:

```
billable = (input_tokens - cached_input_tokens) + output_tokens + reasoning_output_tokens
```

In practice, cached tokens can be 95%+ of total input tokens. A session showing 67M total tokens might have only 1.6M billable.

## Budget Calculation Flow

### Step 1: Resolve Weekly Budget

The calibrator infers the weekly budget from snapshot data:

```
inferred_budget = local_weekly_tokens / (scraped_weekly_pct / 100)
```

Example: If nightshift measures 500K billable tokens used this week and the rate limit says 31% weekly used, the inferred budget is `500K / 0.31 = 1.6M tokens/week`.

The calibrator takes the median of multiple samples, filters outliers via MAD (Median Absolute Deviation), and assigns a confidence level (low/medium/high) based on sample count and variance.

If no snapshots have both local tokens and scraped percentages, the calibrator falls back to the config value (`budget.weekly_tokens`).

### Step 2: Get Used Percentage

For the budget decision (how much nightshift can use), nightshift uses rate limit percentages:

- **Daily mode**: Primary rate limit `used_percent` (5h window)
- **Weekly mode**: Secondary rate limit `used_percent` (7-day window)

Token-based calculation is only used as a fallback when rate limit data is unavailable (e.g., no recent sessions with rate limit events).

### Step 3: Calculate Allowance

```
daily_budget = weekly_budget / 7
remaining = daily_budget * (1 - used_percent / 100)
nightshift_allowance = remaining * max_percent / 100 - reserve
```

### Step 4: Display Both

The budget display shows both data sources so the user can see the full picture:

```
[codex]
  Mode:         daily
  Weekly:       1.6M tokens (calibrated, high confidence, 7 samples)
  Daily:        228.6K tokens
  Used today:   43.4K (19.0%)
  Remaining:    185.1K tokens
  Rate limit:   19% primary (5h) · 31% weekly
  Local tokens: 451.9K today · 1.2M this week (billable)
  Reserve:      11.4K tokens
  Nightshift:   185.1K remaining x 80% max = 148.1K - 11.4K reserve = 136.7K available
```

The "Rate limit" line shows OpenAI's authoritative percentages. The "Local tokens" line shows measured billable usage. Discrepancies between these indicate differences in how OpenAI accounts tokens for rate limiting vs. what we measure from the API response.

## Why Rate Limits and Local Tokens Diverge

Several factors cause the rate limit percentage to imply different usage than our local billable token count:

1. **Server-side prompt caching**: OpenAI may apply additional caching beyond what `cached_input_tokens` reports. Our billable calculation may overcount.

2. **Model launch bonuses**: During new model launches, OpenAI sometimes increases rate limits without changing the percentages proportionally. The rate limit says 31% but the effective budget is larger than normal.

3. **Rate limit accounting differences**: OpenAI may weight tokens differently for rate limiting (e.g., reasoning tokens counted at a different rate, or a per-request overhead).

4. **Timing**: Rate limits are rolling windows. Local token counts are calendar-based (today's sessions, last 7 days of sessions). The windows don't align exactly.

Nightshift uses rate limits for budget decisions because they reflect what OpenAI actually enforces. Local tokens are shown for transparency and debugging.

## Session File Layout

Codex stores sessions at `~/.codex/sessions/YYYY/MM/DD/*.jsonl`. Each session is a single JSONL file containing interleaved events:

- `session_meta`: Session ID and metadata
- `event_msg` with `type: "token_count"`: Token usage and rate limits
- Other events: User messages, assistant responses, tool calls

Nightshift scans session files by date directory to efficiently sum token usage for "today" and "this week" queries.

## Tmux Scraping

When `calibrate_enabled: true` and `billing_mode: subscription`, nightshift can also scrape Codex's `/status` command via tmux to get the displayed rate limit percentages. This provides an additional cross-check and captures reset time strings (e.g., "resets 20:08 on 9 Feb").

The scraping flow:
1. Start Codex CLI in a tmux session
2. Send `/status` command
3. Capture screen output
4. Parse percentage and reset time from the output
5. Store as a snapshot

## Configuration

```yaml
budget:
  mode: daily              # daily or weekly
  weekly_tokens: 700000    # fallback budget if calibration unavailable
  calibrate_enabled: true  # enable tmux scraping for calibration
  billing_mode: subscription
  max_percent: 80          # nightshift uses up to 80% of remaining budget
  reserve_percent: 5       # hold 5% in reserve
  week_start_day: monday

providers:
  codex:
    enabled: true
    datapath: ~/.codex
```

## Snapshot Schema

Each snapshot stores:

| Field | Description |
|-------|-------------|
| `local_tokens` | Weekly billable tokens from session JSONL |
| `local_daily` | Today's billable tokens from session JSONL |
| `scraped_pct` | Weekly used percentage from tmux scrape |
| `inferred_budget` | `local_tokens / (scraped_pct / 100)` |
| `session_reset_time` | Scraped 5h window reset time string |
| `weekly_reset_time` | Scraped weekly reset time string |
