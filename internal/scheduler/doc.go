// Package scheduler handles time-based job scheduling for nightshift runs.
//
// A [Scheduler] decides when registered [Job]s execute, supporting three
// complementary ways to express timing: cron expressions, fixed intervals,
// and one-shot [Schedule] calls. Execution can additionally be gated by a time
// window via [IsInWindow] so runs only happen during allowed hours.
//
// Schedulers are typically built from configuration with [NewFromConfig], or
// assembled manually with [New] plus [SetCron]/[SetInterval] and [AddJob].
// Inspection helpers — [NextRun], [NextRuns], [IsRunning] — support dry-run
// planning and status reporting without starting the loop.
//
// The package defines typed errors for malformed configuration
// ([ErrInvalidCron], [ErrInvalidInterval], [ErrInvalidWindow],
// [ErrInvalidTimezone]) and for lifecycle misuse ([ErrNoSchedule],
// [ErrAlreadyRunning], [ErrNotRunning]).
package scheduler
