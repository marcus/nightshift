// Package reporting generates run reports and morning summaries for nightshift.
//
// After a nightshift run completes, this package produces structured JSON
// results (RunResults) and human-readable markdown reports. Reports are saved
// to ~/.local/share/nightshift/reports/ and can optionally be sent via email
// or Slack webhook.
//
// # Report Types
//
//   - Run report: Markdown summary of a single run (tasks completed, failed, skipped)
//   - Run results: Structured JSON with per-task metrics and output references
//   - Morning summary: Aggregated summary of the most recent run, suitable for
//     notification delivery
//
// # File Naming
//
// Reports are named with timestamps: run-2006-01-02-150405.md and
// run-2006-01-02-150405.json.
package reporting
