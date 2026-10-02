# Commit Message Guide

Nightshift now standardizes future commits around a small conventional format that matches most recent history:

```text
<type>(<optional-scope>): <imperative summary>
```

Examples:

- `fix(tasks): standardize commit message template`
- `feat(run): add preflight summary display`
- `docs: add commit message guide`

## Rules

- Use one of: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `release`, `revert`, `style`, `test`
- Keep the subject under 72 characters
- Use an imperative summary
- Do not end the subject with a period
- Add a body only when it adds useful context

## Nightshift Trailers

When a change is made by Nightshift or one of its agents, include both trailers together:

```text
Nightshift-Task: <task-id>
Nightshift-Ref: https://github.com/marcus/nightshift
```

Leave a blank line before the trailers.

## Local Setup

Run:

```bash
make install-hooks
```

This installs:

- `.git/hooks/pre-commit` for formatting, vet, and build checks
- `.git/hooks/commit-msg` for commit subject and Nightshift trailer validation
- `.gitmessage` as the local commit template for this repository

The hook is intentionally forward-looking. It does not rewrite or validate old history.
