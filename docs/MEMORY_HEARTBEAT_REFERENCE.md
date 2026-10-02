# Memory and Heartbeat Artifact Reference

This guide defines purpose, ownership, cadence, and safety boundaries for repository-local memory artifacts.

Related docs:

- [WORKSPACE_WORKFLOW.md](./WORKSPACE_WORKFLOW.md)
- [FILE_REFERENCE.md](./FILE_REFERENCE.md)
- [../AGENTS.md](../AGENTS.md)

## Artifact Definitions

### `memory/YYYY-MM-DD.md`

Purpose:
- Raw daily chronology of decisions, outcomes, and relevant context.

Ownership and update expectations:
- Assistant-maintained during normal execution.
- Append as meaningful events happen.
- Keep entries concise and factual.

Usage cadence:
- Read today and yesterday at session start.
- Distill durable information into `MEMORY.md`.

Safe-usage boundaries:
- Do not use as a policy file.
- Do not store secrets or credentials.
- Keep as high-signal notes, not chat transcripts.

### `MEMORY.md`

Purpose:
- Curated long-term memory distilled from daily notes.

Ownership and update expectations:
- Shared human/assistant artifact.
- Update only when durable preferences, decisions, or lessons emerge.
- Remove stale or superseded items.

Usage cadence:
- Read in direct main-session workflows.
- Refresh periodically from recent daily notes.

Safe-usage boundaries:
- Do not load in shared/group contexts unless explicitly required.
- Keep stable and concise.
- Avoid mirroring sensitive private data across contexts.

### `memory/heartbeat-state.json`

Purpose:
- Minimal machine-readable heartbeat recency state (for example, last check timestamps).

Ownership and update expectations:
- Assistant-maintained operational metadata.
- Update timestamps after heartbeat checks complete.
- Keep schema compact and stable.

Usage cadence:
- Read before heartbeat checks decide what is due.
- Write only when state changes.

Safe-usage boundaries:
- Store timestamps/status flags only.
- Never store tokens, credentials, message bodies, or private conversation text.
- Treat as operational metadata, not user-facing narrative.

Minimal example:

```json
{
  "lastChecks": {
    "email": 1703275200,
    "calendar": 1703260800,
    "weather": null
  }
}
```

## Lifecycle Notes

- `memory/` can be created lazily when first needed.
- Daily notes are high-frequency and append-oriented.
- `MEMORY.md` is low-frequency and curated.
- `memory/heartbeat-state.json` should remain compact operational state.
