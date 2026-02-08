package orchestrator

import "time"

// EventType classifies orchestrator lifecycle events.
type EventType int

const (
	EventTaskStart      EventType = iota // task execution begins
	EventPhaseStart                      // entering a phase (plan/implement/review)
	EventPhaseEnd                        // phase completed
	EventIterationStart                  // new iteration of the implement-review loop
	EventLog                             // internal log message
	EventTaskEnd                         // task execution finished
)

// Event carries data about an orchestrator lifecycle event.
type Event struct {
	Type      EventType
	Time      time.Time
	Phase     TaskStatus     // which phase: StatusPlanning, StatusExecuting, StatusReviewing
	Iteration int            // current iteration (1-based)
	MaxIter   int            // max iterations configured
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
