# Edge-Case Enumeration

A non-exhaustive catalog of plausible edge cases in nightshift, intended as
seeds for follow-up testing, defensive code review, and bug-triage tasks. No
fixes are proposed here — this document is enumeration only.

Recent fixes used as seeds:
- #20 — provider config YAML key serialization mismatch
- #21 — `--max-projects` interacted incorrectly with `processed-today` filter
- #46 — mobile navbar sidebar hidden behind secondary panel

Each entry is tagged with a rough **Severity** (S1 data loss / silent wrong
result, S2 user-visible failure, S3 cosmetic / minor) and **Likelihood**
(High/Med/Low).

---

## 1. CLI / Input

| # | Case | Sev | Lik |
|---|------|-----|-----|
| 1.1 | `--max-projects 0` — treated as "no limit" or "process none"? Ambiguous. | S2 | Med |
| 1.2 | `--max-projects` negative value — should error, may underflow slice indexing. | S2 | Low |
| 1.3 | `--max-projects` larger than available project count — must not panic on slice. | S2 | Med |
| 1.4 | Combination of `--max-projects` with other filters (processed-today, tags) — order of application matters (seed: #21). | S1 | High |
| 1.5 | Conflicting flags (e.g. dry-run + apply) — silently picks one. | S2 | Low |
| 1.6 | Unknown flags — cobra default behavior should error; verify subcommands. | S3 | Low |
| 1.7 | Flag value with leading/trailing whitespace from shell quoting. | S3 | Low |
| 1.8 | Non-UTF-8 arguments on macOS Terminal with custom locale. | S3 | Low |

## 2. Configuration / YAML

| # | Case | Sev | Lik |
|---|------|-----|-----|
| 2.1 | YAML key name mismatch between marshal and unmarshal (seed: #20). Re-audit every struct used in both directions. | S1 | High |
| 2.2 | Missing config file vs. empty config file vs. config with only comments. | S2 | Med |
| 2.3 | Config file present but unreadable (perms 000). | S2 | Low |
| 2.4 | Config file is a symlink to nonexistent target. | S2 | Low |
| 2.5 | Provider config with duplicate keys — YAML last-wins; user expects error? | S2 | Med |
| 2.6 | Unknown fields silently ignored — typo'd keys never flagged. | S2 | High |
| 2.7 | Provider list empty — does scheduler crash or no-op? | S2 | Med |
| 2.8 | Numeric fields with unit suffixes ("1m" vs "60") — inconsistent parsing. | S2 | Med |
| 2.9 | Path expansion: `~/...` vs absolute vs relative to CWD vs relative to config dir. | S2 | High |
| 2.10 | Concurrent writes to config (e.g. `nightshift setup` while another run reads). | S1 | Low |
| 2.11 | Config schema migration across versions — old field names from pre-v0.3.0. | S2 | Med |

## 3. Project Discovery & Filtering

| # | Case | Sev | Lik |
|---|------|-----|-----|
| 3.1 | Empty project list (no matches) — should exit cleanly with informative message, not error. | S3 | Med |
| 3.2 | Duplicate project entries (same path twice) — dedupe? process twice? | S2 | Med |
| 3.3 | Project path no longer exists on disk. | S2 | Med |
| 3.4 | Project path is a file, not a directory. | S2 | Low |
| 3.5 | Project path inside a git submodule / worktree. | S2 | Med |
| 3.6 | Symlinked project paths and loop detection. | S2 | Low |
| 3.7 | Very large project counts (>1000) — UI rendering, memory, scheduler fairness. | S2 | Low |
| 3.8 | Project name with spaces, unicode, emoji, or shell metacharacters. | S2 | Med |
| 3.9 | "processed-today" boundary across midnight / DST transition / system clock change. | S1 | High |
| 3.10 | "processed-today" when machine is in a different TZ than recorded timestamp. | S1 | Med |
| 3.11 | Filter ordering: max-projects applied before/after processed-today (seed: #21). | S1 | High |
| 3.12 | Project filtered out by one rule then re-included by another. | S2 | Low |

## 4. Runtime / Scheduler / Orchestrator

| # | Case | Sev | Lik |
|---|------|-----|-----|
| 4.1 | Two concurrent `nightshift` invocations — file/db lock, double-processing. | S1 | Med |
| 4.2 | Process killed mid-run (SIGKILL) — leftover state, lockfile, partial DB write. | S1 | High |
| 4.3 | Process killed gracefully (SIGINT/SIGTERM) — does it flush state? | S2 | High |
| 4.4 | tmux session pre-existing with same name — overwrite? attach? error? | S2 | Med |
| 4.5 | tmux not installed / wrong version. | S2 | Med |
| 4.6 | Long-running task hits per-project timeout — partial output captured? | S2 | High |
| 4.7 | Provider rate limit / 429 — retry policy and backoff bounds. | S2 | High |
| 4.8 | Provider auth token expired mid-session. | S2 | Med |
| 4.9 | Network drop mid-stream — does state record success or failure? | S1 | High |
| 4.10 | Disk full while writing snapshots / DB / logs. | S1 | Low |
| 4.11 | DB schema migration on first run after upgrade fails halfway. | S1 | Low |
| 4.12 | SQLite "database is locked" under concurrent writers. | S1 | Med |
| 4.13 | Clock skew (NTP jump backwards) during a run — ordering of timestamps. | S2 | Low |
| 4.14 | System suspend/resume mid-run (laptop closed). | S2 | High |
| 4.15 | Context cancellation propagation — leaked goroutines or zombie subprocesses. | S2 | Med |

## 5. Concurrency / Data Races

| # | Case | Sev | Lik |
|---|------|-----|-----|
| 5.1 | Shared maps mutated from multiple goroutines without sync (run with `-race`). | S1 | Med |
| 5.2 | Worker pool size of 0 or negative. | S2 | Low |
| 5.3 | Channel close-after-send panics on shutdown path. | S2 | Med |
| 5.4 | Reading state DB from web UI while CLI writer holds lock. | S2 | Med |
| 5.5 | Stats / trends aggregation while new rows are inserted (snapshot consistency). | S2 | Med |

## 6. Web UI / Mobile

| # | Case | Sev | Lik |
|---|------|-----|-----|
| 6.1 | Stacking-context regressions: secondary panels obscuring nav (seed: #46). | S2 | Med |
| 6.2 | Viewport widths between mobile/desktop breakpoints (e.g. iPad portrait). | S3 | High |
| 6.3 | Long project names overflow flex containers. | S3 | High |
| 6.4 | Empty states (no projects, no runs, no snapshots) — placeholder copy. | S3 | High |
| 6.5 | Realtime updates dropped when tab is backgrounded. | S2 | Med |
| 6.6 | Time displayed in server TZ vs browser TZ. | S2 | Med |
| 6.7 | Browser back/forward across SPA routes losing scroll/state. | S3 | Med |
| 6.8 | Auth-less local server bound to 0.0.0.0 by accident (security). | S1 | Low |
| 6.9 | XSS via project names rendered unescaped. | S1 | Low |
| 6.10 | Dark/light theme switching mid-session. | S3 | Low |

## 7. Error Handling / Reporting

| # | Case | Sev | Lik |
|---|------|-----|-----|
| 7.1 | Errors swallowed by `_ =` or by returning nil in deferred funcs. | S1 | Med |
| 7.2 | Wrapped errors lose original cause; `errors.Is/As` chains break. | S2 | Med |
| 7.3 | User-facing error messages include absolute paths / tokens. | S2 | Med |
| 7.4 | Panics in goroutines without recover — entire process exits. | S1 | Med |
| 7.5 | Logs at wrong level (errors logged as info or vice versa). | S3 | High |
| 7.6 | Repeated identical errors flood logs (no dedupe / sampling). | S3 | Med |

## 8. Release / Packaging

| # | Case | Sev | Lik |
|---|------|-----|-----|
| 8.1 | Pre-commit hook bypass (`--no-verify`) lands unformatted code. | S3 | Med |
| 8.2 | goreleaser dry-run divergence from real release. | S2 | Low |
| 8.3 | Version string baked at build vs read from VERSION file mismatch. | S3 | Low |
| 8.4 | Upgrade across breaking config change without migration notice. | S2 | Med |

---

## Suggested Follow-ups

1. Convert the High-likelihood S1 rows above into individual issues / tasks.
2. Add table-driven tests around filter ordering (3.11) and TZ boundaries (3.9, 3.10).
3. Run the suite under `go test -race ./...` in CI to catch §5.
4. Add a fuzz target for YAML config round-trip (§2.1, §2.6).
5. Audit web UI templates for unescaped interpolation (§6.9).
