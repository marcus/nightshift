// Command provider-calibration analyzes AI provider token usage from session
// logs and produces a JSON report for budget calibration.
//
// It reads Codex session JSONL files and Claude JSONL session files, extracts
// per-session token counts, and aggregates them into weekly totals. The output
// can be used to calibrate nightshift's budget estimates against real-world
// provider usage.
//
// Usage:
//
//	provider-calibration [--dir <sessions-dir>] [--provider codex|claude] [--week YYYY-MM-DD]
//
// This is a standalone tool intended for manual budget tuning, not part of the
// nightly automated runs.
package main
