// Package tmux scrapes tmux sessions to detect running AI agent processes and
// extract real-time usage data.
//
// When the tmux-based scraper is enabled (the default for calibration), this
// package launches provider CLIs inside tmux sessions, sends the /usage command,
// and parses the output to extract usage percentages and reset times. This
// provides authoritative usage data that supplements or replaces local file-based
// estimates.
//
// # Scraping Flow
//
//  1. Start a new tmux session
//  2. Launch the provider CLI (claude or codex)
//  3. Send the /usage command
//  4. Capture and parse the output
//  5. Kill the tmux session
//
// # Usage
//
// Call ScrapeClaudeUsage or ScrapeCodexUsage with a context. The returned
// UsageResult contains the parsed weekly percentage and reset times.
package tmux
