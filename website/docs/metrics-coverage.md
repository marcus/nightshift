---
sidebar_position: 9
title: Metrics Coverage
---

# Metrics Coverage Analyzer

`nightshift metrics-coverage` statically scans a Go codebase for
instrumentation calls (logging, stats, metrics, telemetry) and reports
per-package coverage. It's a CI-friendly heuristic for spotting code that
ships with no observability.

## What counts as "instrumented"

A function is considered instrumented if its body contains at least one
call whose textual form matches one of the configured patterns. The
default pattern set looks for common call sites:

- Package prefixes: `log.`, `logger.`, `logging.`, `zerolog.`, `stats.`,
  `metrics.`, `otel.`, `telemetry.`, `trace.`, `prometheus.`,
  `observability.`
- Method suffixes: `.Info`, `.Infof`, `.Warn`, `.Warnf`, `.Error`,
  `.Errorf`, `.Debug`, `.Debugf`, `.Trace`, `.Tracef`, `.Record`,
  `.Emit`, `.Observe`, `.Inc`

Override the set entirely with one or more `--pattern` flags.

Test files, `vendor/`, `node_modules/`, `.git/`, and files marked with the
standard `Code generated ... DO NOT EDIT` comment are skipped by default.
Use `--include-tests` to include `*_test.go`.

## Usage

```bash
# Analyze the current directory
nightshift metrics-coverage

# Limit to a subtree, output JSON
nightshift metrics-coverage --path ./internal --format json

# CI gate: fail if overall coverage drops below 60%
nightshift metrics-coverage --min-coverage 60

# Exclude directories with glob patterns
nightshift metrics-coverage --exclude 'vendor/**' --exclude '**/mocks/**'

# Use custom patterns (e.g. an in-house metrics package)
nightshift metrics-coverage --pattern 'myco/obs.' --pattern '.RecordMetric'
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--path`, `-p` | current dir | Directory to analyze (recursively) |
| `--format` | `text` | Output format: `text` or `json` |
| `--min-coverage` | `0` | Fail if overall coverage is below this percent |
| `--exclude` | | Glob pattern of relative paths to skip (repeatable) |
| `--pattern` | _defaults_ | Override default instrumentation patterns (repeatable) |
| `--include-tests` | `false` | Include `*_test.go` files |
| `--max-gaps` | `10` | Cap on gaps listed per package (text only; `0` hides) |
| `--output`, `-o` | stdout | Write report to file |

## Output

Text output sorts packages from lowest to highest coverage and lists the
first uninstrumented functions per package with file and line:

```
Metrics Coverage Report
Root: /repo/internal

Overall: 142/210 functions instrumented (67.6%)

Per-package coverage (lowest first):
  budget (budget)        12/30   40.0%
  scheduler (scheduler)  18/25   72.0%
  ...

Uninstrumented functions:
  budget (budget):
    budget.go:88  (Manager).snapshot
    budget.go:124 normalize
    ...
```

JSON output is the full `OverallCoverage` struct — suitable for piping into
`jq` or attaching as a CI artifact.

## Interpretation

Coverage percentage is a rough signal, not a target. A low number on a
pure-data package (constants, struct definitions, getters) is fine; a low
number on a request-handling or job-orchestration package usually warrants
attention.

Useful workflows:

- **Spot dark code paths**: sort by lowest coverage; eyeball the gap list
  for any function that owns I/O, retries, or state transitions but has no
  logs.
- **Prevent regressions**: run with `--min-coverage` in CI on packages
  where you've already invested in observability.
- **Tune patterns**: if you have an in-house wrapper (e.g.
  `obs.RecordLatency`), pass `--pattern` so it counts.
