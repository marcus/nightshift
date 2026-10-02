// Package orchestrator is the run-orchestration core of nightshift.
//
// It coordinates AI agents working on tasks by driving the
// plan-implement-review loop to convergence: an agent plans, an agent
// implements, and a review pass decides whether another iteration is needed,
// up to [DefaultMaxIterations]. Per-agent work is bounded by
// [DefaultAgentTimeout].
//
// A run is configured with [Config] (iteration limit, per-agent timeout, and
// working directory). Lifecycle progress is published as [Event] values to
// registered [EventHandler] callbacks, allowing the CLI and other observers to
// react to phase transitions, task start/end, and errors without coupling to
// the orchestration internals.
//
// The package also offers a few git- and PR-oriented helpers used while
// orchestrating runs: [CurrentBranch] resolves the active branch in a working
// directory, [ExtractPRURL] recovers a GitHub PR URL from agent output, and
// [ParseMetadataBlock] reads the structured metadata comment embedded in a PR
// body.
package orchestrator
