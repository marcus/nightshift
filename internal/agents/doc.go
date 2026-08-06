// Package agents provides interfaces and implementations for spawning AI agents.
//
// Unlike providers (which track usage and billing), agents are responsible for
// the actual execution of coding tasks. Each agent wraps a CLI tool and handles
// prompt construction, process lifecycle, timeout management, and output parsing.
//
// # Supported Agents
//
//   - ClaudeAgent: Executes tasks via the Claude Code CLI ("claude --print")
//   - CodexAgent:  Executes tasks via the OpenAI Codex CLI ("codex exec")
//   - CopilotAgent: Executes tasks via GitHub Copilot CLI ("gh copilot" or standalone "copilot")
//
// # Agent Interface
//
// All agents implement the Agent interface, which provides Name() and Execute()
// methods. The Execute method accepts ExecuteOptions (prompt, working directory,
// files, timeout) and returns an ExecuteResult with the agent's output, exit code,
// duration, and optional parsed JSON.
//
// # Testing
//
// Agents accept a CommandRunner option for mocking in tests. Use WithRunner
// (or the provider-specific variant) to inject a test double.
package agents
