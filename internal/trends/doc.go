// Package trends analyzes historical snapshot data to surface usage patterns
// and predict near-term consumption.
//
// Working from the periodic samples collected by the [snapshots] package, an
// [Analyzer] builds a usage profile for a provider over a bounded lookback
// window and uses it to anticipate remaining demand. This feeds the budget
// package's predicted-usage reserve, so runs account for expected daytime
// consumption rather than only what has already been spent.
//
// [NewAnalyzer] constructs an analyzer bound to a [db.DB] with a default
// lookback. [BuildProfile] aggregates hourly averages into a [Profile] (per-
// hour averages, daily total, and the window length), and
// [PredictDaytimeUsage] estimates the tokens still expected to be consumed
// today given the current time and weekly budget.
//
// [snapshots]: github.com/marcus/nightshift/internal/snapshots
// [db]: github.com/marcus/nightshift/internal/db
package trends
