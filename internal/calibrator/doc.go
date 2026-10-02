// Package calibrator infers provider subscription budgets from historical
// usage and feeds them back into budget planning.
//
// Rather than requiring the user to configure an exact weekly token budget,
// the calibrator mines the snapshot history collected by the [snapshots]
// package to estimate the effective subscription limit for each provider. It
// implements the [budget.BudgetSource] interface so its estimates flow
// transparently into the budget manager.
//
// [Calibrate] returns a [CalibrationResult] holding the inferred weekly
// budget plus a confidence level (none/low/medium/high), the sample count it
// was derived from, the variance across samples, and a human-readable source
// label. [GetBudget] wraps that result as a [budget.BudgetEstimate] for direct
// consumption by the budget package.
//
// A [Calibrator] is constructed with [New], taking the [db.DB] it reads
// snapshots from and the project [config.Config].
//
// [snapshots]: github.com/marcus/nightshift/internal/snapshots
// [db]: github.com/marcus/nightshift/internal/db
// [config]: github.com/marcus/nightshift/internal/config
package calibrator
