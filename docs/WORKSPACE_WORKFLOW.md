# Workspace Workflow

This guide defines the minimal operating workflow for this repository workspace.

For memory and heartbeat boundaries, see [MEMORY_HEARTBEAT_REFERENCE.md](./MEMORY_HEARTBEAT_REFERENCE.md).

## First Run (Bootstrap)

Use this flow when the workspace is brand new:

1. Read [AGENTS.md](../AGENTS.md).
2. Review [README.md](../README.md) for repository context.
3. If `BOOTSTRAP.md` exists, complete onboarding guidance there.
4. Remove `BOOTSTRAP.md` when onboarding is complete.

## Every Session

Start each session with this order:

1. Read [AGENTS.md](../AGENTS.md) for operating and safety rules.
2. Read [README.md](../README.md) for current docs map and run context.
3. Read `memory/YYYY-MM-DD.md` for today and yesterday when present.
4. In direct main sessions, also read `MEMORY.md` when present.

## Memory Lifecycle

Use three context layers:

- Daily notes: `memory/YYYY-MM-DD.md`
- Curated long-term memory: `MEMORY.md`
- Heartbeat operational state: `memory/heartbeat-state.json`

Lifecycle pattern:

1. Capture decisions and outcomes in daily notes as they happen.
2. For heartbeat polls, read `memory/heartbeat-state.json` before checks and update it after state changes.
3. Periodically distill durable lessons into `MEMORY.md`.
4. Keep `MEMORY.md` concise by removing stale items.

## Safety Boundaries

Default behavior:

- Local read/analysis actions can proceed without extra approval.
- External side effects require explicit user approval.
- Avoid destructive commands unless explicitly requested.

## Operating Principle

Write durable context to files quickly. Session memory is transient; workspace files are persistent.
