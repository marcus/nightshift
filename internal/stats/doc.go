// Package stats computes aggregate statistics from nightshift's run and usage
// data.
//
// It is a read-only aggregation layer that pulls together several data
// sources — saved report JSONs, the run history held by [state], periodic
// usage [snapshots], and the projects table — and folds them into a single
// [StatsResult] via [Stats.Compute].
//
// A [Stats] instance is created with [New], or with [NewWithBudgetSource] when
// a calibrated [budget.BudgetSource] should inform the projections. The
// package surfaces several supporting types: [BudgetProjection] estimates how
// many days of budget remain and whether exhaustion precedes the subscription
// reset; [ProjectStats] summarizes per-project activity; and [Duration] wraps
// [time.Duration] for clean JSON serialization as integer seconds.
//
// [state]: github.com/marcus/nightshift/internal/state
// [snapshots]: github.com/marcus/nightshift/internal/snapshots
// [budget]: github.com/marcus/nightshift/internal/budget
package stats
