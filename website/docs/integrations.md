---
sidebar_position: 9
title: Integrations
---

# Integrations

Nightshift integrates with your existing development workflow.

## Claude Code

Nightshift uses the Claude Code CLI to execute tasks. Authenticate via subscription or API key:

```bash
claude
/login
```

## Codex

Nightshift supports OpenAI's Codex CLI as an alternative provider:

```bash
codex --login
```

## Ollama Cloud

Nightshift supports Ollama Cloud as a third provider. Since Ollama Cloud doesn't provide a public API for rate limiting, authentication uses cookies:

```bash
nightshift ollama auth
```

This creates `~/.ollama/cookies.txt`. Populate it with your ollama.com browser cookies:

1. Sign in to https://ollama.com/settings
2. Export cookies via browser extension (Netscape format):
   - ["EditThisCookie"](https://chrome.google.com/webstore/detail/editthiscookie/)
   - ["Get cookies.txt LOCALLY"](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/)
3. Paste cookies into `~/.ollama/cookies.txt`

**Required cookies:**
- `__Secure-session` (or `__Secure-next-auth.session-token`)
- `aid`
- `cf_clearance`

Verify setup:
```bash
nightshift budget --provider ollama
```

## GitHub

All output is PR-based. Nightshift creates branches and pull requests for its findings.

## td (Task Management)

Nightshift can source tasks from [td](https://td.haplab.com) — task management for AI-assisted development. Tasks tagged with `nightshift` in td will be picked up automatically.

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true   # Include td usage + core workflow in prompts
```

## CLAUDE.md / AGENTS.md

Nightshift reads project-level instruction files to understand context when executing tasks. Place a `CLAUDE.md` or `AGENTS.md` in your repo root to give Nightshift project-specific guidance. Tasks mentioned in these files get a priority bonus (+2).

## GitHub Issues

Source tasks from GitHub issues labeled with `nightshift`:

```yaml
integrations:
  github_issues:
    enabled: true
    label: "nightshift"
```
