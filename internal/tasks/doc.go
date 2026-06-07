// Package tasks defines task structures, registration, and priority-based selection.
//
// Tasks are the units of work that nightshift executes on projects. Each task
// has a type, cost tier, risk level, and estimated token consumption. Tasks can
// come from built-in definitions, custom user definitions in nightshift.yaml,
// or external sources (td, GitHub issues).
//
// # Task Lifecycle
//
//  1. Register: Built-in tasks are registered at init time; custom tasks via
//     RegisterCustomTasksFromConfig
//  2. Select: The Selector picks the highest-priority tasks that fit within the
//     remaining budget, respecting cooldowns and per-project processing limits
//  3. Execute: Selected tasks are handed to the orchestrator for agent execution
//
// # Selection Criteria
//
// The Selector scores tasks based on priority, cost tier, cooldown status, and
// whether the task was mentioned in claude.md/agents.md or appears in external
// task sources. Use SelectTopN for the best N tasks or SelectRandom for a
// single random eligible task.
package tasks
