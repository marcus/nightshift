# Budget Inference & Usage Tracking System

## Goal
Replace the hardcoded `weekly_tokens` guess with a calibration system that **infers** the real subscription budget by correlating local token counts with scraped `/usage` percentages. Consolidate all nightshift state into a single SQLite database. Continue working for API/pay-per-token users via historical token tracking.

## Assumptions & Non-Goals
- Assumes local token counts reflect provider-side weekly counters closely enough for calibration.
- Assumes `/usage` (Claude) and `/status` (Codex) represent current-week usage in percent.
- Non-goal: perfectly matching provider week boundaries on day one. If provider week start differs, we will allow configuration and later inference.
- Non-goal: storing full raw tmux output by default (optional debug field only).

**Core formula (subscription users):** `total_budget = local_tokens / (scraped_pct / 100)`

Example: If local files show 315K tokens used and `/usage` reports 45%, then total budget = 315K / 0.45 = 700K tokens/week.

**API token users:** Budget is known deterministically from token counts x pricing. No calibration needed -- use config `weekly_tokens` directly and track actual spend from local session data.

## Risks & Mitigations
- **Week boundary mismatch** (provider resets on a different day/time): introduce a configurable week start, store computed `week_start` per snapshot, and allow calibrator to filter by that boundary.
- **Low-signal snapshots** (very low or near-100% usage): filter `scraped_pct` outside a safe range to avoid unstable inference.
- **tmux not installed or blocked**: degrade to local-only snapshots and fall back to config budgets.
- **Multi-process DB contention**: enable WAL + `busy_timeout` and keep writes short; snapshot inserts are single-row.
- **Data drift** if local counters under/over-report: use median + variance checks and confidence gating.

---

## Phase 1: Unified SQLite Database + Migration System

Replace the JSON state file (`~/.local/share/nightshift/state/state.json`) and snapshot storage with a single SQLite database at `~/.local/share/nightshift/nightshift.db`.

**Dependency:** `modernc.org/sqlite` (pure Go, no CGo) added to `go.mod`

### Database Manager

**New file:** `internal/db/db.go` - Central database manager

```go
type DB struct {
    sql *sql.DB
    path string
}

// Open(dbPath) - expands ~, creates parent dir, opens/creates DB,
//                enables WAL mode, sets busy_timeout, enables foreign_keys,
//                runs pending migrations
// Close()
// SQL() *sql.DB - raw access for packages that need it
```

Open once per process and share the handle across packages/commands to reduce SQLite contention.

### Migration System

**New file:** `internal/db/migrations.go` - Simple, functional schema migration

The migration system uses a `schema_version` table to track which migrations have been applied. Migrations are numbered sequentially and run in order. Each migration is a single SQL string executed in a transaction.

```go
// Migration represents a single schema change.
type Migration struct {
    Version     int
    Description string
    SQL         string    // DDL/DML to execute
}

// migrations is the ordered list of all schema migrations.
var migrations = []Migration{
    {
        Version:     1,
        Description: "initial schema: projects, task_history, assigned_tasks, run_history, snapshots",
        SQL:         migration001SQL,  // const with CREATE TABLE statements
    },
    // Future migrations added here as new entries
    // {Version: 2, Description: "add foo column to bar", SQL: "ALTER TABLE bar ADD COLUMN foo TEXT;"},
}

// Migrate runs all pending migrations inside transactions.
// Creates schema_version table if it doesn't exist.
// Algorithm:
//   1. CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at DATETIME)
//   2. SELECT MAX(version) FROM schema_version → currentVersion
//   3. For each migration where Version > currentVersion:
//      a. BEGIN
//      b. Execute migration.SQL
//      c. INSERT INTO schema_version (version, applied_at) VALUES (?, CURRENT_TIMESTAMP)
//      d. COMMIT
//   4. Log each migration applied
func Migrate(db *sql.DB) error

// CurrentVersion returns the current schema version (0 if fresh DB).
func CurrentVersion(db *sql.DB) (int, error)
```

The `Open()` function calls `Migrate()` automatically, so the DB is always up-to-date when opened. Adding a new migration is just appending to the `migrations` slice.

### Schema (migration 001)

```sql
CREATE TABLE projects (
    path        TEXT PRIMARY KEY,
    last_run    DATETIME,
    run_count   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE task_history (
    project_path TEXT NOT NULL,
    task_type    TEXT NOT NULL,
    last_run     DATETIME NOT NULL,
    PRIMARY KEY (project_path, task_type)
);

CREATE TABLE assigned_tasks (
    task_id     TEXT PRIMARY KEY,
    project     TEXT NOT NULL,
    task_type   TEXT NOT NULL,
    assigned_at DATETIME NOT NULL
);

CREATE TABLE run_history (
    id          TEXT PRIMARY KEY,
    start_time  DATETIME NOT NULL,
    end_time    DATETIME,
    project     TEXT NOT NULL,
    tasks       TEXT NOT NULL,          -- JSON array
    tokens_used INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL,          -- success, failed, partial
    error       TEXT
);

CREATE TABLE snapshots (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    provider        TEXT NOT NULL,
    timestamp       DATETIME NOT NULL,
    week_start      DATE NOT NULL,      -- computed from configured week start + local TZ
    local_tokens    INTEGER NOT NULL,
    local_daily     INTEGER NOT NULL DEFAULT 0,
    scraped_pct     REAL,               -- NULL if tmux scrape unavailable
    inferred_budget INTEGER,            -- local_tokens / (scraped_pct/100)
    day_of_week     INTEGER NOT NULL,
    hour_of_day     INTEGER NOT NULL,
    week_number     INTEGER NOT NULL,   -- derived from configured week start
    year            INTEGER NOT NULL    -- derived from configured week start
);

CREATE INDEX idx_snapshots_provider_time ON snapshots(provider, timestamp DESC);
CREATE INDEX idx_snapshots_provider_week ON snapshots(provider, week_start);
CREATE INDEX idx_run_history_time ON run_history(start_time DESC);
```

**New file:** `internal/db/db_test.go`
- Test fresh DB creation runs all migrations
- Test idempotency (Open twice is safe)
- Test migration versioning (add migration, reopen, only new migration runs)
- Test CurrentVersion()

### State Rewrite

**Rewrite:** `internal/state/state.go`
- Same public API (RecordProjectRun, RecordTaskRun, WasProcessedToday, StalenessBonus, MarkAssigned, AddRunRecord, etc.)
- Implementation changes from JSON read/write to SQLite queries
- Constructor: `New(db *db.DB)` instead of `New(stateDir string)`
- Remove: JSON marshaling, atomic file writes, in-memory StateData struct

### JSON-to-SQLite Data Migration

**New file:** `internal/db/import.go` - One-time state.json import

On first `Open()`, if `~/.local/share/nightshift/state/state.json` exists:
1. If the DB already has any state rows, skip import (avoid duplicates)
2. Parse the JSON into the old StateData struct
3. Insert all projects, task_history, assigned_tasks, run_history rows
4. Commit transaction, then rename `state.json` to `state.json.migrated`
5. Log: "migrated N projects, M run records from state.json"

This runs after schema migrations, inside a transaction. If import fails, the JSON file is left untouched for retry.

---

## Phase 2: tmux Scraping Package

**New file:** `internal/tmux/tmux.go`
- `Session` struct wrapping tmux lifecycle (create, send-keys, capture-pane, kill)
- `SessionOption` funcs: `WithWorkDir`, `WithSize`, `WithRunner`
- `WaitForPattern()` - polls capture-pane until regex match or timeout
- `CommandRunner` interface for testability (mock shell execution)

**New file:** `internal/tmux/scraper.go`
- `UsageResult` struct: `Provider`, `WeeklyPct`, `ScrapedAt`, `RawOutput`
- `ScrapeClaudeUsage(ctx)` - starts Claude in tmux, sends `/usage`, parses output
- `ScrapeCodexUsage(ctx)` - starts Codex in tmux, sends `/status`, parses output
- ANSI stripping, trust prompt handling, percentage extraction per `docs/agent-tmux-integration.md`
- Graceful failure: returns error (callers degrade to local-only data)
- Use a dedicated, detached tmux session name (e.g. `nightshift-usage-claude`) and always clean up, even on timeout
- Hard timeout (e.g. 15s) and small capture buffer to avoid large output reads
- Check tmux availability up front (`exec.LookPath("tmux")`), return a typed error for clearer CLI output

**New file:** `internal/tmux/tmux_test.go`
- Mock `CommandRunner` for unit tests
- Table-driven parser tests with sample `/usage` and `/status` outputs

---

## Phase 3: Snapshot Collection

**New file:** `internal/snapshots/collector.go`
- `Collector` struct: takes `*db.DB`, Claude/Codex providers, tmux scraper, budget config (week start + retention)
- `TakeSnapshot(ctx, provider)`:
  1. Read local token counts from providers (stats-cache.json / session JSONL)
  2. Attempt tmux scrape for usage % (non-fatal if fails)
  3. If `scraped_pct` is outside 0-100, discard it (store NULL)
  4. If both available, compute `inferred_budget = local_tokens / (scraped_pct / 100)`
  5. Compute `week_start`, `day_of_week`, `hour_of_day`, `week_number`, `year` from configured week start (default: local week start)
  6. Insert into `snapshots` table
- `GetLatest(provider, n)`, `GetSinceWeekStart(provider)` - query helpers
- `GetHourlyAverages(provider, lookbackDays)` - for trend analysis
- `Prune(retentionDays)` - cleanup old data

---

## Phase 4: Calibrator

**New file:** `internal/calibrator/calibrator.go`

**`CalibrationResult` struct:**
- `InferredBudget int64` - best estimate of weekly token budget
- `Confidence string` - "none" / "low" / "medium" / "high"
- `SampleCount int`, `Variance float64`
- `Source string` - "calibrated", "api" (known), or "config" (fallback)

**`Calibrator` struct with methods:**
- `Calibrate(provider)` - runs inference:
  1. **If billing_mode=api**: return config `weekly_tokens` directly, confidence="high", source="api". No calibration needed -- API users know their budget.
  2. **If calibrate_enabled=false**: return config `weekly_tokens`, confidence="none", source="config"
  3. Get snapshots from current week where `scraped_pct` is non-null and in a safe range (default 10-95%) and `local_tokens > 0`
  4. Compute `inferred_budget = local_tokens / (scraped_pct / 100)` for each
  5. If sample count >= 3, filter outliers using MAD (median absolute deviation); otherwise skip filtering
  6. Take median of remaining values (round to nearest 1K tokens)
  7. Score confidence by sample count + coefficient of variation (stddev/median):
     - none: 0 samples
     - low: 1-2 samples or high variance
     - medium: 3-5 samples and CV <= 0.15
     - high: 6+ samples and CV <= 0.10
  8. Fallback to config `weekly_tokens` if no calibration data
- `GetBudget(provider)` - converts `CalibrationResult` into `budget.BudgetEstimate` for the budget manager

**Weekly reset handling:** Only uses current-week snapshots for calibration, based on configured week start (default: local week start). Confidence resets to "none" at start of new week.

### API Token User Path

For users on pay-per-token plans (`billing_mode: api`):
- Budget is deterministic: `weekly_tokens` in config is their actual budget in tokens (or they set a dollar cap and we convert via known pricing)
- `GetUsedPercent()` from local token counts is accurate -- no scraping needed
- Snapshots are still collected (local-only, no tmux scrape) for trend analysis and history
- The tmux scraping goroutine is skipped entirely
- CLI output shows "api" source instead of "calibrated"

---

## Phase 5: Config & Budget Manager Integration

**Modify:** `internal/config/config.go`
- Add to `BudgetConfig`:
  - `BillingMode string` (default: "subscription") -- "subscription" or "api"
  - `CalibrateEnabled bool` (default: true)
  - `SnapshotInterval string` (default: "30m")
  - `SnapshotRetentionDays int` (default: 90)
  - `WeekStartDay string` (default: "monday") -- used for snapshot grouping/calibration windows
  - `DBPath string` (optional override, default: `~/.local/share/nightshift/nightshift.db`)
- Add defaults in `setDefaults()`, validation in `Validate()` (validate BillingMode is "subscription" or "api", WeekStartDay is "monday" or "sunday")
- When `billing_mode: api`, calibration is implicitly disabled (config `weekly_tokens` is authoritative)

**Modify:** `internal/budget/budget.go`
- Add `BudgetEstimate` struct: `WeeklyTokens int64`, `Source string`, `Confidence string`, `SampleCount int`, `Variance float64`
- Add `BudgetSource` interface: `GetBudget(provider string) (BudgetEstimate, error)`
- Add `WithBudgetSource(bs BudgetSource)` option to `NewManager`
- In `CalculateAllowance()`: use `BudgetSource` if available (and non-zero), else fall back to `cfg.GetProviderBudget()`
- Add `BudgetSource string`, `BudgetConfidence string`, `BudgetSampleCount int` fields to `AllowanceResult`
- Fully backward-compatible: existing callers work unchanged

---

## Phase 6: Daemon with Daytime Auto-Snapshots

**Modify:** `cmd/nightshift/commands/daemon.go`

The daemon runs 24/7. Snapshots are a **separate scheduled job** that runs on its own interval, independent of the main task schedule. This means snapshots collect during the day when the user is coding, building calibration data for nightshift's overnight runs. If `snapshot_interval` is `0` or invalid, auto-snapshots are disabled and only manual snapshots run.

In `runDaemonLoop()`:
```go
// Open shared SQLite DB
database, _ := db.Open(cfg.DBPath())

// Existing: main task scheduler (cron-based, runs overnight)
sched.AddJob(func(ctx) { runScheduledTasks(ctx, cfg, database, log) })

// NEW: snapshot scheduler (separate ticker, runs all day)
snapshotInterval, _ := time.ParseDuration(cfg.Budget.SnapshotInterval) // default 30m
if snapshotInterval > 0 {
    go func() {
        // Take one immediately on startup
        takeSnapshot(ctx, cfg, database, log)
        ticker := time.NewTicker(snapshotInterval)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done(): return
            case <-ticker.C:
                takeSnapshot(ctx, cfg, database, log)
            }
        }
    }()
}

// NEW: snapshot pruning (daily)
go func() {
    pruneTicker := time.NewTicker(24 * time.Hour)
    defer pruneTicker.Stop()
    for {
        select {
        case <-ctx.Done(): return
        case <-pruneTicker.C:
            pruneSnapshots(ctx, cfg, database, log)
        }
    }
}()
```

`takeSnapshot()`:
1. Creates `snapshots.Collector` with DB and providers
2. Calls `collector.TakeSnapshot(ctx, "claude")` and `collector.TakeSnapshot(ctx, "codex")`
3. If `billing_mode: api` or `calibrate_enabled=false`, skip tmux scraping -- collect local token data only
4. If `billing_mode: subscription` and `calibrate_enabled=true`, attempt tmux scrape for usage %
5. Logs result (scraped % if available, or "local-only")
6. Non-fatal errors logged as warnings

`pruneSnapshots()`:
- Calls `collector.Prune(cfg.Budget.SnapshotRetentionDays)` and logs rows deleted

`runScheduledTasks()`:
- Wire calibrator into budget calculation: `budget.WithBudgetSource(calibrator)`
- Use calibrated budget instead of config-hardcoded value

---

## Phase 7: CLI Enhancements

**Modify:** `cmd/nightshift/commands/budget.go`
- Open shared SQLite DB, wire calibrator into `runBudget()`
- Enhanced output shows calibrated budget, confidence level, sample count:
  ```
  [claude]
    Mode:         weekly
    Weekly:       700.0K tokens (calibrated, high confidence, 8 samples)
    Used:         315.0K (45.0%)
    Remaining:    385.0K tokens
    Days left:    4
    Reserve:      35.0K tokens
    Nightshift:   50.0K tokens available
    Progress:     [#############-----------------] 45.0%
  ```

**New file:** `cmd/nightshift/commands/snapshot.go`
- `nightshift budget snapshot [--provider claude|codex] [--local-only]` - manually trigger a snapshot (local-only skips tmux)
- `nightshift budget history [-n 20] [--provider claude|codex]` - show recent snapshots in table format
- `nightshift budget calibrate [--provider claude|codex]` - show calibration status, inferred budget, confidence, variance

---

## Phase 8: Trend Analysis

**New file:** `internal/trends/analyzer.go`
- `BuildProfile(provider, lookbackDays)` - aggregates hourly usage patterns from snapshots
- `PredictDaytimeUsage(provider, calibratedBudget)` - estimates user's daytime consumption
- `SafeNightshiftBudget` calculation: remaining - predicted_daytime - reserve
- Integrate into CLI budget display and budget allowance calculation
- Works for both subscription and API users (both have local token snapshots)

---

## Phase 9: Update User Guide

**Modify:** `docs/user-guide.md`

Update the following sections to reflect new features:

- **Configuration > Budget Controls**: Document `billing_mode` (subscription vs api), `calibrate_enabled`, `snapshot_interval`, `snapshot_retention_days`, `week_start_day`, `db_path`. Show example config for both subscription and API users.
- Add `snapshot_retention_days` and `week_start_day` to the same section.
- **Configuration > Provider Configuration**: Update the `weekly_tokens` description -- explain that for API users this is authoritative, for subscription users it's a fallback that gets replaced by calibration.
- **Monitoring > Check Budget**: Update example output to show calibration data (confidence, sample count, source). Show both subscription and API example outputs.
- **New section: Budget Calibration**: Explain how calibration works (snapshot collection, tmux scraping, inference formula). How confidence builds over time. What "low/medium/high" means. How to manually trigger `nightshift budget snapshot`.
- **New section: Snapshot History**: Document `nightshift budget history` and `nightshift budget calibrate` commands.
- **Troubleshooting > File Locations**: Update state location from `state/state.json` to `nightshift.db`. Document that old state.json is auto-migrated.
- **Troubleshooting**: Add entries for "Calibration confidence is low" (run `nightshift budget snapshot` a few times, ensure tmux is available), "tmux not found" (install tmux, or set `billing_mode: api` if pay-per-token), and "Week boundary looks wrong" (adjust `week_start_day`).

- Add an "uninstalling" section to help users who want to remove nightshift from their systems
---

## Files Modified
| File | Change |
|------|--------|
| `internal/state/state.go` | Rewrite: JSON file -> SQLite backend, same public API |
| `internal/config/config.go` | Add BillingMode, CalibrateEnabled, SnapshotInterval, SnapshotRetentionDays, WeekStartDay, DBPath |
| `internal/budget/budget.go` | Add BudgetEstimate + BudgetSource interface, WithBudgetSource option |
| `cmd/nightshift/commands/budget.go` | Enhanced display with calibration data |
| `cmd/nightshift/commands/daemon.go` | Open shared DB, add snapshot ticker, wire calibrator |
| `cmd/nightshift/commands/run.go` | Use shared DB for state + calibrator |
| `cmd/nightshift/commands/status.go` | Use shared DB for state queries |
| `docs/user-guide.md` | Document calibration, snapshots, billing_mode, new commands |
| `go.mod` | Add `modernc.org/sqlite` |

## New Files
| File | Purpose |
|------|---------|
| `internal/db/db.go` | Central SQLite DB manager (open, migrate, close) |
| `internal/db/db_test.go` | DB lifecycle and migration tests |
| `internal/db/migrations.go` | Versioned schema migrations |
| `internal/db/import.go` | One-time state.json -> SQLite import |
| `internal/tmux/tmux.go` | tmux session management |
| `internal/tmux/scraper.go` | Claude/Codex usage scraping |
| `internal/tmux/tmux_test.go` | Tests with mock runner |
| `internal/snapshots/collector.go` | Snapshot collection and queries |
| `internal/snapshots/collector_test.go` | Collector tests |
| `internal/calibrator/calibrator.go` | Budget inference engine |
| `internal/calibrator/calibrator_test.go` | Calibrator tests |
| `internal/trends/analyzer.go` | Usage pattern analysis |
| `internal/trends/analyzer_test.go` | Trend tests |
| `cmd/nightshift/commands/snapshot.go` | CLI subcommands (snapshot, history, calibrate) |

## Verification
1. `go build ./...` - compiles cleanly
2. `go test ./...` - all tests pass (including migrated state tests, migration tests)
3. **Migration system**: open fresh DB -> all tables created. Open again -> no-op. Add migration 002 -> only 002 runs.
4. **JSON import**: place a state.json, open DB -> data imported, file renamed to .migrated
5. **Subscription user**: start daemon, wait 1hr, `nightshift budget history -n 5` shows auto-collected snapshots with scraped %
6. **API user**: set `billing_mode: api`, `nightshift budget` shows source="api", high confidence, no tmux scraping
7. `nightshift budget snapshot` - manually triggers snapshot
8. `nightshift budget calibrate` - shows inferred budget and confidence
9. `nightshift budget` - shows enhanced output with calibration data
10. After 3+ snapshots with tmux scrape: confidence should reach "medium" or "high"
11. `nightshift status` / `nightshift run --dry-run` - work correctly with SQLite-backed state
12. `docs/user-guide.md` - accurately reflects all new features, both user types documented
13. `calibrate_enabled=false` or `snapshot_interval=0` disables tmux scraping/auto-snapshots but manual local-only snapshots still work
14. `snapshot_retention_days` prunes old rows on schedule
15. `week_start_day` shifts the calibration window as expected
