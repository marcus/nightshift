// Package agents provides interfaces and implementations for spawning and
// driving AI coding agents.
//
// Unlike the [providers] package, which is concerned with tracking token usage
// and cost, agents are responsible for autonomously executing a task to
// completion: running a prompt against a CLI, handling timeouts, and returning
// structured output. The orchestrator uses agents to perform the implement and
// review phases of the plan-implement-review loop.
//
// The core abstraction is the [Agent] interface, implemented per backend:
//
//   - [ClaudeAgent] wraps the Claude Code CLI (claude --print).
//   - CodexAgent wraps the OpenAI Codex CLI.
//   - CopilotAgent wraps the GitHub Copilot CLI.
//
// Agents are constructed with functional options (e.g. [WithBinaryPath],
// [WithDangerouslySkipPermissions]) and report availability via Available so
// the orchestrator can skip backends that are not installed. Execution honours
// [DefaultTimeout] unless overridden by the caller's context.
//
// [providers]: github.com/marcus/nightshift/internal/providers
package agents
