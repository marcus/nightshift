// Package tasks defines task structures and loading from various sources.
// Tasks can come from GitHub issues, local files, or inline definitions.
package tasks

// Task represents a unit of work for an AI agent.
type Task struct {
	ID          string
	Title       string
	Description string
	Priority    int
	// TODO: Add more fields (labels, assignee, source, etc.)
}

// Queue holds tasks to be processed.
type Queue struct {
	// TODO: Add fields
}

// NewQueue creates an empty task queue.
func NewQueue() *Queue {
	// TODO: Implement
	return &Queue{}
}

// Add queues a task.
func (q *Queue) Add(t Task) {
	// TODO: Implement
}

// Next returns the highest priority task.
func (q *Queue) Next() *Task {
	// TODO: Implement
	return nil
}
