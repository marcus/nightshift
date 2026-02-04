# Nightshift - AI Agent Instructions

## Purpose

Nightshift is a CLI tool that orchestrates AI coding agents (Claude Code, Codex) to work on tasks overnight. It manages budgets, schedules runs, and coordinates parallel agent execution.

## Key Directories

- `cmd/nightshift/` - CLI entry point (cobra commands)
- `internal/config/` - Configuration loading
- `internal/budget/` - Cost tracking and limits
- `internal/scheduler/` - Time-based job scheduling
- `internal/providers/` - AI agent backends (Claude, Codex)
- `internal/tasks/` - Task definitions and queue
- `internal/orchestrator/` - Agent coordination
- `internal/ui/` - Terminal UI (bubbletea)

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
