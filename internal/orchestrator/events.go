package orchestrator

import "time"

// EventType classifies orchestrator lifecycle events.
type EventType int

const (
	// EventTaskStart indicates that task execution has begun.
	EventTaskStart EventType = iota
	// EventPhaseStart indicates entry into a phase (plan/implement/review).
	EventPhaseStart
	// EventPhaseEnd indicates that a phase has completed.
	EventPhaseEnd
	// EventIterationStart indicates a new iteration of the implement-review loop.
	EventIterationStart
	// EventLog carries an internal log message.
	EventLog
	// EventTaskEnd indicates that task execution has finished.
	EventTaskEnd
)

// Event carries data about an orchestrator lifecycle event.
type Event struct {
	Type      EventType
	Time      time.Time
	Phase     TaskStatus // which phase: StatusPlanning, StatusExecuting, StatusReviewing
	Iteration int        // current iteration (1-based)
	MaxIter   int        // max iterations configured
	TaskID    string
	TaskTitle string
	Message   string         // human-readable message
	Level     string         // "info", "warn", "error"
	Fields    map[string]any // structured fields
	Status    TaskStatus     // for EventTaskEnd: final status
	Duration  time.Duration  // for EventPhaseEnd/EventTaskEnd: elapsed time
	Error     string         // error message if applicable
}

// EventHandler is a callback that receives orchestrator events.
type EventHandler func(Event)
