// Package logging provides structured logging with file rotation for
// nightshift.
//
// It wraps zerolog with nightshift-specific conventions: JSON or text output,
// a configurable level, and date-based log-file naming with retention so old
// logs are pruned automatically. The package exposes both a process-wide
// global logger and per-component loggers for emitting scoped output.
//
// [Init] configures and installs the global logger from a [Config]; the
// package-level [Info], [Warn], [Error], and [Debug] functions then log to it.
// For component-scoped output, [Component] returns a [*Logger] that tags every
// record, and [Get] returns the global logger directly. Each [Logger] supports
// formatted and context-field variants (e.g. DebugCtx).
//
// [New] constructs a standalone logger when a non-global instance is required;
// [Logger.Close] flushes and closes the underlying file.
package logging
