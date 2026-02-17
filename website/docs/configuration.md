---
sidebar_position: 4
title: Configuration
---

# Configuration

Nightshift uses YAML config files. Run `nightshift setup` for an interactive setup, or edit directly.

## Config Location

- **Global:** `~/.config/nightshift/config.yaml`
- **Per-project:** `nightshift.yaml` or `.nightshift.yaml` in the repo root

## Minimal Config

```yaml
schedule:
  cron: "0 2 * * *"

budget:
  mode: daily
  max_percent: 75
  reserve_percent: 5
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m

providers:
  preference:
    - claude
    - codex
    - ollama
  claude:
    enabled: true
    data_path: "~/.claude"
    dangerously_skip_permissions: true
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: true
  ollama:
    enabled: false
    data_path: "~/.ollama"

projects:
  - path: ~/code/sidecar
  - path: ~/code/td
```

## Schedule

Use cron syntax or interval-based scheduling:

```yaml
schedule:
  cron: "0 2 * * *"        # Every night at 2am
  # interval: "8h"         # Or run every 8 hours
```

## Budget

Control how much of your token budget Nightshift uses:

| Field | Default | Description |
|-------|---------|-------------|
| `mode` | `daily` | `daily` or `weekly` |
| `max_percent` | `75` | Max budget % to use per run |
| `reserve_percent` | `5` | Always keep this % available |
| `billing_mode` | `subscription` | `subscription` or `api` |
| `calibrate_enabled` | `true` | Auto-calibrate from local CLI data |

## Task Selection

Enable/disable tasks and set priorities:

```yaml
tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - bug-finder
  priorities:
    lint-fix: 1
    bug-finder: 2
  intervals:
    lint-fix: "24h"
    docs-backfill: "168h"
```

Each task has a default cooldown interval to prevent the same task from running too frequently on a project.

## Multi-Project Setup

```yaml
projects:
  - path: ~/code/project1
    priority: 1                # Higher priority = processed first
    tasks:
      - lint
      - docs
  - path: ~/code/project2
    priority: 2

  # Or use glob patterns
  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archived
```

## Safe Defaults

| Feature | Default | Override |
|---------|---------|----------|
| Read-only first run | Yes | `--enable-writes` |
| Max budget per run | 75% | `budget.max_percent` |
| Auto-push to remote | No | Manual only |
| Reserve budget | 5% | `budget.reserve_percent` |

## File Locations

| Type | Location |
|------|----------|
| Run logs | `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log` |
| Audit logs | `~/.local/share/nightshift/audit/audit-YYYY-MM-DD.jsonl` |
| Summaries | `~/.local/share/nightshift/summaries/` |
| Database | `~/.local/share/nightshift/nightshift.db` |
| PID file | `~/.local/share/nightshift/nightshift.pid` |

If `state/state.json` exists from older versions, Nightshift migrates it to the SQLite database and renames the file to `state.json.migrated`.

## Providers

Nightshift supports Claude Code, Codex, and Ollama Cloud as execution providers. It will use whichever has budget remaining, in the order specified by `preference`.

### Claude Code

```yaml
providers:
  claude:
    enabled: true
    data_path: "~/.claude"
    dangerously_skip_permissions: true
```

Authenticate via `/login` in the Claude Code CLI or use an API key.

### Codex

```yaml
providers:
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: true
```

Authenticate via `codex --login` or use an API key.

### Ollama Cloud

```yaml
providers:
  ollama:
    enabled: false
    data_path: "~/.ollama"
```

Ollama Cloud uses cookie-based authentication since it doesn't provide a public API for rate limiting.

**Setup steps:**

```bash
nightshift ollama auth
```

This creates `~/.ollama/cookies.txt`. To populate it:

1. Sign in to https://ollama.com/settings in your browser
2. Install a browser extension to export cookies:
   - Chrome/Firefox: ["EditThisCookie"](https://chrome.google.com/webstore/detail/editthiscookie/) or ["Get cookies.txt LOCALLY"](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/)
3. Export your ollama.com cookies in **Netscape format**
4. Paste the cookies into `~/.ollama/cookies.txt`

**Required cookies:**
- `__Secure-session` (or `__Secure-next-auth.session-token`)
- `aid`
- `cf_clearance`

**Verify setup:**

```bash
nightshift budget --provider ollama
```
