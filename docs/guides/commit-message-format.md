# Commit Message Format

Nightshift enforces a small subset of [Conventional Commits](https://www.conventionalcommits.org/)
via a `commit-msg` git hook.

## Format

```
<type>: <subject>

<optional body>
```

- **type** — one of `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`
- **subject** — a short, present-tense description; kept under 72 characters where possible
- **body** — optional, separated from the subject by a blank line

Examples:

```
feat: add budget snapshot command
fix: correct off-by-one in scheduler interval
docs: clarify install-hooks usage
```

## What the hook does

`scripts/commit-msg-normalize.sh` runs against your commit message before the commit is
finalized. It will:

- strip trailing whitespace from each line
- collapse runs of blank lines down to a single blank line
- lowercase the `type` prefix and insert a missing `:` / space after it (e.g. `fix add x` → `fix: add x`)
- reject the commit (non-zero exit) if the subject doesn't start with a recognized type
- warn (but not block) if the subject line exceeds 72 characters

Merge commits (`git merge --no-ff`), squash merges, and reverts (`git revert`) are exempt from
the type-prefix requirement — the hook checks git's commit-source argument and also recognizes
`Merge ...` / `Revert "..."` subjects, so these routine operations aren't blocked.

## Installing the hook

```bash
make install-hooks
# or directly:
scripts/install-hooks.sh
```

This is idempotent and installs both `scripts/pre-commit.sh` (as `.git/hooks/pre-commit`) and
`scripts/commit-msg.sh` (as `.git/hooks/commit-msg`).

## Bypassing in an emergency

```bash
git commit --no-verify
```

Use sparingly — this skips both the commit-msg and pre-commit hooks.
