# Privacy & Data Handling

This document describes what data nightshift collects, where it is stored,
and what leaves your machine.

## Local Storage

All persistent data lives under XDG-compliant paths:

| Data | Default path | Format | Retention |
|------|-------------|--------|-----------|
| Database | `~/.local/share/nightshift/nightshift.db` | SQLite (WAL mode) | Permanent |
| Logs | `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log` | JSON or text | 7 days (configurable) |
| Audit log | `~/.local/share/nightshift/audit/audit-YYYY-MM-DD.jsonl` | JSONL | Permanent (append-only, no automatic cleanup) |
| Summaries | `~/.local/share/nightshift/summaries/summary-YYYY-MM-DD.md` | Markdown | Permanent |
| Config | `~/.config/nightshift/config.yaml` | YAML | Permanent |

The database directory is created with `0700` permissions (owner-only access).

### What the database stores

- Project paths and execution history
- Task execution timestamps and assignments
- Run history (start/end times, project, tasks, tokens used, status, errors, provider, branch)
- Provider usage snapshots (token counts, daily/weekly usage, inferred budget)
- Bus-factor analysis results

### Provider data directories (read-only)

Nightshift reads — but never writes to — these provider CLI data directories
to track token usage locally:

- `~/.claude` — session history and `stats-cache.json`
- `~/.codex` — session JSONL files and rate-limit info
- `~/.copilot` — nightshift maintains a local request counter at `~/.copilot/nightshift-usage.json`

These paths are configurable via `providers.<name>.data_path` in config.

## External Transmission

Nightshift sends data externally **only** when you explicitly configure it.
Nothing is sent by default.

### AI provider CLIs

When nightshift runs a task, it invokes provider CLIs as subprocesses:

| Provider | Command | Data sent |
|----------|---------|-----------|
| Claude Code | `claude --print <prompt>` | Task prompt + selected file contents |
| Codex | `codex exec <prompt>` | Task prompt + selected file contents |
| Copilot | `gh copilot -- -p <prompt>` | Task prompt + selected file contents |

Each invocation is isolated — no session state persists between calls, and
no cross-project context is shared. The provider CLIs handle their own
authentication and network communication; nightshift does not transmit API
keys over the network itself.

Dangerous permission flags (`--dangerously-skip-permissions`,
`--dangerously-bypass-approvals-and-sandbox`, `--allow-all-tools`) default
to **false** and require explicit opt-in.

### Slack notifications (optional)

When `reporting.slack_webhook` is configured, nightshift posts morning
summaries containing: budget usage, completed task list, project counts,
and failed/skipped task info.

### Email notifications (optional)

When SMTP environment variables are set (`NIGHTSHIFT_SMTP_HOST`, etc.),
nightshift sends the same morning summary via email.

### GitHub integration (optional)

When enabled, nightshift uses the `gh` CLI to read issues (filtered by
label) and post completion comments. It relies on `gh`'s existing
authentication — nightshift does not handle GitHub tokens directly.

## Credential Handling

- **API keys** (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`) are read from
  environment variables only and are never written to disk.
- **Config file credential protection**: nightshift actively scans config
  files for credential patterns (`api_key:`, `secret:`, `sk-` prefixes)
  and rejects them.
- **Credential masking**: when credentials appear in log output, they are
  masked to show only the first 3 and last 3 characters.
- **SMTP credentials** (`NIGHTSHIFT_SMTP_USER`, `NIGHTSHIFT_SMTP_PASS`)
  are read from environment variables only.
- **Slack webhook URL** is stored in plaintext in config YAML — consider
  using an environment variable for sensitive deployments.

## Telemetry

Nightshift includes **zero** telemetry, analytics, crash reporting, or
phone-home functionality. All usage tracking is local-only, reading data
from provider CLI directories on disk.

## Deleting Your Data

```bash
# Remove all nightshift data
rm -rf ~/.local/share/nightshift

# Remove configuration
rm -rf ~/.config/nightshift

# Remove nightshift's copilot usage counter
rm -f ~/.copilot/nightshift-usage.json
```

Per-project config (`nightshift.yaml`) lives in each project directory.
