// Package orchestrator coordinates AI agents working on tasks.
// Handles task assignment, parallel execution, and result aggregation.
package orchestrator

import (
	"github.com/marcusvorwaller/nightshift/internal/budget"
	"github.com/marcusvorwaller/nightshift/internal/providers"
	"github.com/marcusvorwaller/nightshift/internal/tasks"
)

// Orchestrator manages agent execution.
type Orchestrator struct {
	providers []providers.Provider
	budget    *budget.Tracker
	queue     *tasks.Queue
	// TODO: Add fields for concurrency control, results, etc.
}

// New creates an orchestrator.
func New(providers []providers.Provider, budget *budget.Tracker, queue *tasks.Queue) *Orchestrator {
	return &Orchestrator{
		providers: providers,
		budget:    budget,
		queue:     queue,
	}
}

// Run processes tasks until queue empty or budget exhausted.
func (o *Orchestrator) Run() error {
	// TODO: Implement main orchestration loop
	return nil
}
