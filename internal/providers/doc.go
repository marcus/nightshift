// Package providers defines the abstraction over the AI coding backends that
// nightshift tracks for usage, cost, and budgeting.
//
// Where the [agents] package executes prompts, providers are the accounting
// layer: each implementation reports how many tokens have been consumed, at
// what cost, and how close a subscription is to its limit. This is the data
// the [budget], [calibrator], [stats], and [trends] packages build on.
//
// The built-in backends wrap their respective CLIs:
//
//   - [Claude] wraps Claude Code, reading usage primarily from its
//     stats-cache.json with a JSONL fallback.
//   - Codex wraps the OpenAI Codex CLI.
//   - Copilot wraps GitHub Copilot, which is metered by monthly request limits
//     rather than weekly tokens.
//
// Common functionality — daily/weekly usage ([GetTodayUsage], [GetWeeklyUsage]),
// used-percentage against a budget ([GetUsedPercent]), daily statistics, and
// per-token cost ([Cost]) — is exposed uniformly so callers can treat providers
// interchangeably.
//
// [agents]: github.com/marcus/nightshift/internal/agents
// [budget]: github.com/marcus/nightshift/internal/budget
package providers
