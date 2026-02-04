// Package scheduler handles time-based job scheduling.
// Supports cron expressions, intervals, and time window constraints.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/marcusvorwaller/nightshift/internal/config"
	"github.com/robfig/cron/v3"
)

// Errors for scheduler operations.
var (
	ErrInvalidCron     = errors.New("invalid cron expression")
	ErrInvalidInterval = errors.New("invalid interval duration")
	ErrInvalidWindow   = errors.New("invalid time window")
	ErrInvalidTimezone = errors.New("invalid timezone")
	ErrNoSchedule      = errors.New("no schedule configured (need cron or interval)")
	ErrAlreadyRunning  = errors.New("scheduler already running")
	ErrNotRunning      = errors.New("scheduler not running")
)

// Job is a function to execute on schedule.
type Job func(ctx context.Context) error

// Scheduler manages scheduled nightshift runs.
type Scheduler struct {
	mu sync.RWMutex

	// Configuration
	cronExpr string
	interval time.Duration
	window   *Window
	location *time.Location

	// Runtime state
	cron     *cron.Cron
	jobs     []Job
	running  bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	nextRun  time.Time
	entryID  cron.EntryID
}

// Window represents a time window constraint.
type Window struct {
	Start    TimeOfDay // Start time (e.g., 22:00)
	End      TimeOfDay // End time (e.g., 06:00)
	Location *time.Location
}

// TimeOfDay represents a time within a day (hour and minute).
type TimeOfDay struct {
	Hour   int
	Minute int
}

// ParseTimeOfDay parses a time string like "22:00" or "06:30".
func ParseTimeOfDay(s string) (TimeOfDay, error) {
	var h, m int
	n, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil || n != 2 {
		return TimeOfDay{}, fmt.Errorf("invalid time format %q: expected HH:MM", s)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return TimeOfDay{}, fmt.Errorf("invalid time %q: hour must be 0-23, minute 0-59", s)
	}
	return TimeOfDay{Hour: h, Minute: m}, nil
}

// String returns the time in HH:MM format.
func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// Minutes returns the time as minutes since midnight.
func (t TimeOfDay) Minutes() int {
	return t.Hour*60 + t.Minute
}

// New creates an empty scheduler.
func New() *Scheduler {
	return &Scheduler{
		location: time.Local,
	}
}

// NewFromConfig creates a scheduler from configuration.
func NewFromConfig(cfg *config.ScheduleConfig) (*Scheduler, error) {
	s := &Scheduler{
		location: time.Local,
	}

	// Parse cron expression
	if cfg.Cron != "" {
		if err := s.SetCron(cfg.Cron); err != nil {
			return nil, fmt.Errorf("parsing cron: %w", err)
		}
	}

	// Parse interval
	if cfg.Interval != "" {
		d, err := time.ParseDuration(cfg.Interval)
		if err != nil {
			return nil, fmt.Errorf("parsing interval: %w", err)
		}
		if err := s.SetInterval(d); err != nil {
			return nil, err
		}
	}

	// Validate that we have at least one schedule type
	if cfg.Cron == "" && cfg.Interval == "" {
		return nil, ErrNoSchedule
	}

	// Parse window if present
	if cfg.Window != nil {
		if err := s.SetWindow(cfg.Window); err != nil {
			return nil, fmt.Errorf("parsing window: %w", err)
		}
	}

	return s, nil
}

// SetCron sets the cron expression for scheduling.
func (s *Scheduler) SetCron(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCron, err)
	}
	s.mu.Lock()
	s.cronExpr = expr
	s.interval = 0 // Clear interval if setting cron
	s.mu.Unlock()
	return nil
}

// SetInterval sets the interval for scheduling.
func (s *Scheduler) SetInterval(d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("%w: duration must be positive", ErrInvalidInterval)
	}
	s.mu.Lock()
	s.interval = d
	s.cronExpr = "" // Clear cron if setting interval
	s.mu.Unlock()
	return nil
}

// SetWindow sets the time window constraint.
func (s *Scheduler) SetWindow(cfg *config.WindowConfig) error {
	start, err := ParseTimeOfDay(cfg.Start)
	if err != nil {
		return fmt.Errorf("%w: start: %v", ErrInvalidWindow, err)
	}
	end, err := ParseTimeOfDay(cfg.End)
	if err != nil {
		return fmt.Errorf("%w: end: %v", ErrInvalidWindow, err)
	}

	loc := time.Local
	if cfg.Timezone != "" {
		loc, err = time.LoadLocation(cfg.Timezone)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidTimezone, err)
		}
	}

	s.mu.Lock()
	s.window = &Window{Start: start, End: end, Location: loc}
	s.location = loc
	s.mu.Unlock()
	return nil
}

// AddJob adds a job to be executed on schedule.
func (s *Scheduler) AddJob(job Job) {
	s.mu.Lock()
	s.jobs = append(s.jobs, job)
	s.mu.Unlock()
}

// Start begins the scheduler loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}

	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})

	if s.cronExpr != "" {
		// Cron-based scheduling
		s.cron = cron.New(cron.WithLocation(s.location))
		entryID, err := s.cron.AddFunc(s.cronExpr, func() {
			s.runJobs(ctx)
		})
		if err != nil {
			s.running = false
			s.mu.Unlock()
			return fmt.Errorf("adding cron job: %w", err)
		}
		s.entryID = entryID
		s.cron.Start()
		s.updateNextRunLocked()
		s.mu.Unlock()

		// Wait for stop signal
		go func() {
			<-s.stopCh
			s.cron.Stop()
			close(s.doneCh)
		}()
	} else if s.interval > 0 {
		// Interval-based scheduling
		s.updateNextRunLocked()
		s.mu.Unlock()

		go s.intervalLoop(ctx)
	} else {
		s.running = false
		s.mu.Unlock()
		return ErrNoSchedule
	}

	return nil
}

// intervalLoop runs the interval-based scheduling loop.
func (s *Scheduler) intervalLoop(ctx context.Context) {
	defer close(s.doneCh)

	timer := time.NewTimer(time.Until(s.NextRun()))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-timer.C:
			s.runJobs(ctx)
			s.updateNextRun()
			timer.Reset(time.Until(s.NextRun()))
		}
	}
}

// runJobs executes all registered jobs if within the time window.
func (s *Scheduler) runJobs(ctx context.Context) {
	now := time.Now().In(s.location)

	// Check time window
	if !s.IsInWindow(now) {
		return
	}

	s.mu.RLock()
	jobs := make([]Job, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.RUnlock()

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			_ = job(ctx) // Errors handled by job itself
		}
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return ErrNotRunning
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	// Wait for scheduler to stop
	<-s.doneCh
	return nil
}

// NextRun returns the next scheduled run time.
func (s *Scheduler) NextRun() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextRun
}

// updateNextRun calculates and stores the next run time.
func (s *Scheduler) updateNextRun() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateNextRunLocked()
}

// updateNextRunLocked calculates and stores the next run time.
// Must be called while holding the write lock.
func (s *Scheduler) updateNextRunLocked() {
	now := time.Now().In(s.location)

	if s.cronExpr != "" && s.cron != nil {
		entry := s.cron.Entry(s.entryID)
		s.nextRun = entry.Next
	} else if s.interval > 0 {
		s.nextRun = now.Add(s.interval)
	}

	// Adjust for window if needed (use internal check, no lock)
	if s.window != nil && !s.window.Contains(s.nextRun) {
		s.nextRun = s.nextWindowStartLocked(s.nextRun)
	}
}

// IsInWindow checks if the given time is within the allowed execution window.
// Returns true if no window is configured.
func (s *Scheduler) IsInWindow(t time.Time) bool {
	s.mu.RLock()
	window := s.window
	s.mu.RUnlock()

	if window == nil {
		return true
	}

	return window.Contains(t)
}

// Contains checks if a time falls within the window.
func (w *Window) Contains(t time.Time) bool {
	// Convert to window's timezone
	t = t.In(w.Location)
	current := TimeOfDay{Hour: t.Hour(), Minute: t.Minute()}
	currentMins := current.Minutes()
	startMins := w.Start.Minutes()
	endMins := w.End.Minutes()

	// Handle overnight windows (e.g., 22:00-06:00)
	if startMins > endMins {
		// Window spans midnight
		// Valid if after start OR before end
		return currentMins >= startMins || currentMins < endMins
	}

	// Normal window (e.g., 09:00-17:00)
	return currentMins >= startMins && currentMins < endMins
}

// nextWindowStart returns the next time the window starts after t.
func (s *Scheduler) nextWindowStart(t time.Time) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextWindowStartLocked(t)
}

// nextWindowStartLocked returns the next time the window starts after t.
// Must be called while holding the lock.
func (s *Scheduler) nextWindowStartLocked(t time.Time) time.Time {
	if s.window == nil {
		return t
	}

	t = t.In(s.window.Location)
	today := time.Date(t.Year(), t.Month(), t.Day(), s.window.Start.Hour, s.window.Start.Minute, 0, 0, s.window.Location)

	// If window start has passed today, use tomorrow
	if t.After(today) || t.Equal(today) {
		today = today.AddDate(0, 0, 1)
	}

	return today
}

// IsRunning returns whether the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// ScheduleCron adds a recurring job using cron expression.
// Deprecated: Use SetCron and AddJob instead.
func (s *Scheduler) ScheduleCron(expr string, job func()) error {
	if err := s.SetCron(expr); err != nil {
		return err
	}
	s.AddJob(func(ctx context.Context) error {
		job()
		return nil
	})
	return nil
}

// ScheduleInterval adds a recurring job using interval.
func (s *Scheduler) ScheduleInterval(d time.Duration, job func()) error {
	if err := s.SetInterval(d); err != nil {
		return err
	}
	s.AddJob(func(ctx context.Context) error {
		job()
		return nil
	})
	return nil
}

// Schedule adds a one-time job to run at the specified time.
func (s *Scheduler) Schedule(at time.Time, job func()) {
	s.AddJob(func(ctx context.Context) error {
		if time.Now().After(at) {
			job()
		}
		return nil
	})
}
