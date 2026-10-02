// Package state manages the persistent state of nightshift runs.
//
// Backed by the [db] package, it records run history and in-flight task
// assignments so that nightshift can reason about recency across restarts.
// This state underpins three behaviours: staleness calculation (how long since
// a task type last ran for a project), duplicate-run prevention (skip work
// already done today), and task-assignment tracking (knowing what is
// currently in progress).
//
// A [State] manager is created with [New] over a [db.DB]. Per-project
// recency is tracked through [ProjectState] and queried with helpers such as
// [DaysSinceLastRun] and [LastRun]. Completed runs are recorded as [RunRecord]
// entries via [AddRunRecord].
//
// In-flight work is modelled by [AssignedTask]. Assignments are added, cleared
// individually ([ClearAssigned]), cleared in bulk ([ClearAllAssigned], e.g. on
// daemon restart), and pruned by age ([ClearStaleAssignments]) to recover from
// abandoned tasks.
//
// [db]: github.com/marcus/nightshift/internal/db
package state
