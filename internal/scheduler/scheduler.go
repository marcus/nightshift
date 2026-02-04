// Package scheduler handles time-based job scheduling.
// Supports cron expressions and one-time scheduled runs.
package scheduler

import "time"

// Scheduler manages scheduled nightshift runs.
type Scheduler struct {
	// TODO: Add fields for scheduled jobs
}

// New creates a scheduler.
func New() *Scheduler {
	// TODO: Implement
	return &Scheduler{}
}

// Schedule adds a job to run at the specified time.
func (s *Scheduler) Schedule(at time.Time, job func()) {
	// TODO: Implement
}

// ScheduleCron adds a recurring job using cron expression.
func (s *Scheduler) ScheduleCron(expr string, job func()) error {
	// TODO: Implement
	return nil
}
