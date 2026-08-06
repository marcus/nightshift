# Nightshift - AI Agent Instructions

<!-- td-agent-instructions:start -->
<!-- td-agent-instructions:version=3 -->

## Working with td

td keeps task context durable across sessions. In a new context, run `td usage --new-session -q` to see current work.

Use your judgment about how much tracking a task needs. For substantive work: `td start <id>`, record progress with `td log`, hand off with `td handoff <id>`, then `td review <id>`.

Closing needs a review. Say who did it (default trusted mode; delegated/strict allow only the first):

- independent session: `td approve <id> --reason "..."`
- a sub-agent: `td approve <id> --reviewed-by "<who>"`
- you: `td approve <id> --self-review --reason "..."`

Prefer a reviewer with its own `TD_CONTEXT_ID`; never name one who did not review.

Run `td usage` or `td <command> --help`.

<!-- td-agent-instructions:end -->

## Purpose

Nightshift is a CLI tool that orchestrates AI coding agents (Claude Code, Codex, GitHub Copilot) to work on tasks overnight. It manages budgets, schedules runs, and coordinates parallel agent execution.

## Key Directories

- `cmd/nightshift/` - CLI entry point (cobra commands)
- `internal/config/` - Configuration loading
- `internal/budget/` - Cost tracking and limits
- `internal/scheduler/` - Time-based job scheduling
- `internal/providers/` - AI agent backends (Claude, Codex, Copilot)
- `internal/tasks/` - Task definitions and queue
- `internal/orchestrator/` - Agent coordination

## Commands

```bash
# Build
go build ./cmd/nightshift

# Run
./nightshift

# Test
go test ./...
```

## Conventions

- **Logging**: Hyper-concise messages. Include needed info, minimize words.
- **Style**: Standard Go (gofmt, govet). No magic, explicit is better.
- **Errors**: Wrap with context, don't swallow.
- **Tests**: Table-driven, in `_test.go` files alongside code.
- **Commits**: Conventional Commits (`<type>(<scope>): <subject>`). Allowed
  types: `feat fix docs style refactor test chore perf build ci`. Lowercase,
  imperative subject, max 72 chars. Install the commit-msg hook with
  `make install-hooks`; see `docs/commit-messages.md` for details.
