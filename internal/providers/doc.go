// Package providers defines interfaces and implementations for tracking AI
// coding agent usage and costs.
//
// Each provider knows how to read usage data from the agent's local data
// directory (stats-cache.json, JSONL session files, rate-limit endpoints) and
// report percentage of budget consumed.
//
// # Supported Providers
//
//   - Claude:  Reads from ~/.claude/stats-cache.json and JSONL session files
//   - Codex:   Reads from ~/.codex JSONL session files and rate-limit data
//   - Copilot: Tracks usage via local request counting (GitHub API does not
//     expose usage)
//
// # Usage Tracking
//
// Providers implement methods like GetUsedPercent, GetWeeklyUsage, and
// GetUsageBreakdown that the budget package uses to calculate remaining
// allowance for nightly runs.
//
// # Calibration
//
// The GetUsedPercent method accepts a mode ("daily" or "weekly") and a budget
// ceiling in tokens. Some providers also support tmux-based scraping for
// authoritative usage data when API data is unavailable.
package providers
