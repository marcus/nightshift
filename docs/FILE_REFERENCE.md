# File Reference

Quick reference for important root files and memory artifacts in this repository.

## Root Files

- `AGENTS.md`
Purpose: primary operating contract for task selection, workflow, and safety boundaries.
Ownership/edit expectations: keep current with workflow changes.
Used when: at the start of every implementation session.

- `README.md`
Purpose: project overview, command usage, and documentation navigation.
Ownership/edit expectations: keep user-facing usage and internal doc links accurate.
Used when: onboarding to repository behavior and CLI flows.

- `CODEX.md`
Purpose: Codex-specific project notes.
Ownership/edit expectations: maintain concise, project-local guidance.
Used when: Codex runs need repository-specific constraints.

- `BOOTSTRAP.md` (if present)
Purpose: first-run onboarding marker.
Ownership/edit expectations: temporary lifecycle signal; remove once onboarding is complete.
Used when: setting up a brand-new workspace context.

## Memory Files

For lifecycle and safety details, see [MEMORY_HEARTBEAT_REFERENCE.md](./MEMORY_HEARTBEAT_REFERENCE.md).

- `memory/YYYY-MM-DD.md`
Purpose: raw daily chronology of decisions and events.
Ownership/edit expectations: append-oriented daily notes.
Used when: loading recent context at session start.

- `MEMORY.md`
Purpose: curated long-term memory distilled from daily notes.
Ownership/edit expectations: maintain high-signal durable context.
Used when: main-session continuity and preference recall.

- `memory/heartbeat-state.json`
Purpose: machine-readable heartbeat check recency state (timestamps/status flags).
Ownership/edit expectations: assistant-maintained operational metadata; keep compact.
Used when: heartbeat polls decide whether checks are due and avoid duplicate checks.

## Additional Docs

- [WORKSPACE_WORKFLOW.md](./WORKSPACE_WORKFLOW.md): first-run and recurring session workflow.
- [FILE_REFERENCE.md](./FILE_REFERENCE.md): this ownership and usage reference.
- [MEMORY_HEARTBEAT_REFERENCE.md](./MEMORY_HEARTBEAT_REFERENCE.md): memory/heartbeat lifecycle and safety boundaries.
