// Package orchestrator coordinates AI agents working on tasks.
//
// It implements the plan-implement-review loop for task execution. Each task
// goes through phases: planning, execution, and review. The orchestrator
// manages iterations, timeouts, git branch creation, and PR traceability.
//
// # Execution Loop
//
// For each task the orchestrator:
//  1. Creates a feature branch (docs/<task-type>-<slug>)
//  2. Runs the agent with a structured prompt
//  3. Reviews the result (optionally re-iterating)
//  4. Records the outcome (completed, abandoned, or failed)
//
// # Events
//
// The orchestrator emits lifecycle events (EventTaskStart, EventPhaseStart,
// etc.) that can be consumed via an EventHandler for live UI rendering.
//
// # Configuration
//
// Use Config to control max iterations, agent timeout, and other parameters.
// Options are applied via WithAgent, WithConfig, WithLogger, and
// WithEventHandler.
package orchestrator
