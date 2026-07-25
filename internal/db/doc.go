// Package db provides SQLite-backed persistence for nightshift state and
// usage snapshots.
//
// It is the storage layer shared by several subsystems — [state] for run
// history and task assignments, [snapshots] for periodic usage samples, and
// [stats]/[trends] for read-only aggregation — all of which operate against a
// single database file.
//
// The entry point is [Open], which opens (or creates) the database at the
// given path, applies connection pragmas, and runs any pending [Migrate]
// migrations inside transactions. [DefaultPath] returns the conventional
// database location. Callers obtain the underlying handle through [DB.SQL]
// when they need direct query access; [DB.Close] releases the connection.
//
// Schema evolution is handled by the [Migration] type and the version tracking
// exposed by [CurrentVersion].
//
// [state]: github.com/marcus/nightshift/internal/state
// [snapshots]: github.com/marcus/nightshift/internal/snapshots
// [stats]: github.com/marcus/nightshift/internal/stats
// [trends]: github.com/marcus/nightshift/internal/trends
package db
