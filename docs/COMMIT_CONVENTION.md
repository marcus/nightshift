# Commit Convention

Nightshift uses a normalized Conventional Commits format.

## Format

```text
type(scope)!: lowercase description

wrapped body text

Trailer: value
```

The type is required. The scope is optional. The `!` marker is optional and marks a breaking change. The description is lowercase, has no terminal punctuation, and the complete subject is at most 72 characters.

Allowed types are `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, and `test`.

The normalizer infers `feat` from `add`, `implement`, `introduce`, or `support` subjects. It infers `fix` from `fix`, `bug`, `repair`, `resolve`, or `handle` subjects. It maps common first words to the other allowed types and uses `chore` by default.

The body starts after one blank line. Body paragraphs wrap at 72 characters. Trailers start after one blank line and remain at the end. The normalizer preserves body content that it can repair without data loss.

## Trailers

Trailers remain at the end of the message. The normalizer preserves the first trailer for each case-insensitive token and removes later duplicates. It preserves trailer values, including `Nightshift-Task` and `Nightshift-Ref`.

Breaking changes can use either `!` in the subject or a `BREAKING CHANGE` trailer. The normalizer adds `!` when a breaking trailer exists.

## Examples

```text
feat(cli): add commit message normalization

Normalize Git commit messages before they enter project history.

Nightshift-Task: commit-normalize
Nightshift-Ref: https://github.com/marcus/nightshift
```

```text
fix!: remove the legacy config format

BREAKING CHANGE: migrate existing config files before upgrading.
```

## Installation and bypass

Install both the `pre-commit` and `commit-msg` hooks with:

```bash
make install-hooks
```

The `commit-msg` hook normalizes the message file atomically. Use `nightshift commit-msg --check <file>` to validate a message without changing it.

Use `git commit --no-verify` to bypass both hooks for an exceptional commit.
