# Provider budget telemetry and selection

## Summary

Nightshift preview and preflight output drifted from real provider state in three ways: preview ignored configured provider preference order, weekly reserve logic zeroed out valid Codex nightly allowances, and Claude local telemetry fell back to stale or misleading local files. The fix set restored deterministic provider ordering, corrected weekly reserve math, and added a safe manual override path for Claude usage when local telemetry is unavailable.

## Symptom(s)

- `nightshift preview -p /Users/dallascrilley/Code/cohost-ai-studio --explain` showed `Provider: claude (preview picks first enabled: claude -> codex)` even when config preference was `codex, claude`.
- Codex reported healthy remaining quota in the Codex UI, but Nightshift computed `0 available` for nightly runs.
- Claude UI showed about `49% used` for the current week, but Nightshift reported absurd values like `262.3% used`, `749.3% used`, and negative weekly budget bases.
- Preview/preflight selected different providers than expected and sometimes filtered out all tasks despite healthy provider budgets.

## Reproduction steps

1. Configure provider preference as Codex first in `~/.config/nightshift/config.yaml`:
   ```yaml
   providers:
     preference:
       - codex
       - claude
   ```
2. Run preview before the fix:
   ```bash
   nightshift preview -p /Users/dallascrilley/Code/cohost-ai-studio --explain
   ```
3. Observe any of the following broken behaviors:
   - preview summary still says Claude is chosen first
   - Codex shows `0 available` despite weekly quota remaining
   - Claude shows impossible `used_percent` values from JSONL fallback
4. Compare with live provider UIs:
   - Codex TUI showed roughly `60% left` weekly / `41% used`
   - Claude usage pane showed roughly `49% used`

## Root cause

### 1. Preview provider order was hardcoded
`cmd/nightshift/commands/preview.go` used fixed provider checks (`claude`, then `codex`, then `copilot`) instead of reusing `providerPreference(cfg)`. That made preview output disagree with actual configured order.

### 2. Weekly reserve was subtracted from the full remaining weekly pool
`internal/budget/budget.go` computed a nightly allowance from a per-day slice of the remaining weekly budget, but then `applyReserve` subtracted reserve against the entire remaining weekly budget. For realistic weekly settings, this wiped out valid nightly Codex allowances.

### 3. Claude JSONL fallback was not a safe source for subscription usage
`internal/providers/claude.go` fell back from stale `~/.claude/stats-cache.json` to raw session JSONL scanning. On this machine:
- `~/.claude/stats-cache.json` had `lastComputedDate=2026-02-16`
- JSONL totals did not match Claude’s current subscription usage UI
That produced bogus weekly percentages and negative base budget calculations.

### 4. Claude had no reliable fresh local machine-readable usage source
Opening Claude’s Usage pane did not produce a parseable fresh snapshot in `~/.claude/history.jsonl`. Since the local telemetry source was stale, the safe path was to support a manual override instead of scraping undocumented OAuth internals.

## Fix implemented

### Preview/provider selection
- Updated `cmd/nightshift/commands/preview.go` so both `collectProviderBudgets` and `previewProvider` iterate through `providerPreference(cfg)`.
- Updated `cmd/nightshift/commands/preview_output.go` text to say preview respects configured provider preference order.
- Added tests in `cmd/nightshift/commands/run_test.go`:
  - `TestPreviewProvider_UsesPreferenceOrder`
  - `TestCollectProviderBudgets_UsesPreferenceOrder`

### Weekly reserve math
- Updated `internal/budget/budget.go` so weekly reserve is applied to tonight’s per-day slice of remaining budget rather than the entire remaining weekly pool.
- Added/updated tests in `internal/budget/budget_test.go` to cover:
  - weekly reserve behavior
  - real-world Codex-like weekly usage with positive nightly allowance

### Claude telemetry safety
- Updated `internal/providers/claude.go` to treat stale `stats-cache.json` as an error instead of using unsafe JSONL fallback for weekly subscription usage.
- Added optional fallback parsing from `history.jsonl` only if a real Usage screen snapshot exists.
- Added provider tests in `internal/providers/claude_test.go` for:
  - stale cache error handling
  - history-based usage parsing when available

### Manual Claude override
- Added `used_percent_override` to `internal/config/config.go` `ProviderConfig`.
- Added validation and `GetProviderUsedPercentOverride` helper.
- Updated `internal/budget/budget.go` to honor config usage overrides before provider telemetry and label the source as `config-override`.
- Added regression tests in:
  - `internal/config/config_test.go`
  - `internal/budget/budget_test.go`

### Operational config used during recovery
Global config was adjusted to restore sane operation for `/Users/dallascrilley/Code/cohost-ai-studio`:
- Codex preferred first
- Codex weekly budget tuned to allow low-risk nightly work
- Claude re-enabled with:
  - `budget.per_provider.claude: 2000000`
  - `providers.claude.used_percent_override: 49`

## Verification evidence

Commands run during the fix:

```bash
# Verify config + budget changes
cd /Users/dallascrilley/Code/dallas-plugin-marketplace/nightshift
go test ./internal/config ./internal/budget

go test ./internal/providers

go test ./cmd/nightshift/commands -run 'TestPreviewProvider_UsesPreferenceOrder|TestCollectProviderBudgets_UsesPreferenceOrder|TestSelectProvider_PreferenceOrder|TestSelectProvider_FallbackOnBudget'

# Verify patched CLI build output
/tmp/nightshift-bin preview -p /Users/dallascrilley/Code/cohost-ai-studio --plain --explain
```

Observed post-fix results:
- `go test ./internal/config ./internal/budget` -> pass
- `go test ./internal/providers` -> pass
- preview now shows provider order honoring config
- preview now shows realistic Codex nightly allowance instead of `0 available`
- preview now shows Claude using `49.0% used` from `config-override` with a `2.0M` weekly cap
- preview selected a low-risk task (`Documentation Backfiller`) for `cohost-ai-studio`

Representative fixed preview output:
- `codex: 53.7K available (41.0% used, weekly=850.0K, source=config)`
- `claude: 109.3K available (49.0% used, weekly=2.0M, source=config)`

## Prevention / guardrails

- Reuse `providerPreference(cfg)` everywhere provider order matters; do not duplicate hardcoded provider order in preview code.
- In weekly mode, apply reserve to the same time-slice used for nightly allowance computation.
- Do not use Claude JSONL token scanning as a source of truth for subscription/quota percentages when `stats-cache.json` is stale.
- Prefer failing safe over fabricated telemetry when local provider data is stale or approximate.
- Support explicit config overrides for provider usage when telemetry is unavailable.
- Keep targeted regression tests around:
  - provider preference order
  - weekly reserve math
  - stale Claude cache handling
  - manual usage overrides

## Related files, issues, PRs

Primary code files:
- `cmd/nightshift/commands/preview.go`
- `cmd/nightshift/commands/preview_output.go`
- `cmd/nightshift/commands/run_test.go`
- `internal/budget/budget.go`
- `internal/budget/budget_test.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/providers/claude.go`
- `internal/providers/claude_test.go`

Relevant operational files:
- `~/.config/nightshift/config.yaml`
- `/Users/dallascrilley/Code/cohost-ai-studio/nightshift.yaml`
- `~/.claude/stats-cache.json`
- `~/.claude/history.jsonl`

Related branch context:
- current local recovery work on `main` in `/Users/dallascrilley/Code/dallas-plugin-marketplace/nightshift`
- recent stable baseline commit: `2ebd805` (`Bump version to v0.3.4`)
