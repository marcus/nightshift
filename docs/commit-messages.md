# Commit Messages

Nightshift uses [Conventional Commits](https://www.conventionalcommits.org/)
for all commit messages. This keeps the history readable and lets tooling
derive changelogs automatically.

## Format

```
<type>(<scope>)!: <subject>

<body>
```

- **type** — one of `feat`, `fix`, `docs`, `style`, `refactor`, `test`,
  `chore`, `perf`, `build`, `ci`, `revert`.
- **scope** — optional, e.g. `fix(api): ...`.
- **`!`** — optional breaking-change marker after the type or scope, e.g.
  `feat!:` or `feat(api)!:`. It is preserved as-is. (A `BREAKING CHANGE:`
  footer is also valid; body text is passed through unchanged apart from
  wrapping.)
- **subject** — lowercase, imperative mood, no trailing period, max 72 chars.
  The 72-char limit is checked after normalization, so a subject that only
  fits once its trailing period is trimmed is accepted.
- **body** — optional, wrapped at 72 columns (counted in characters, not
  bytes), separated from the subject by a blank line. A final paragraph made
  up entirely of trailer/footer lines (e.g. `Reviewed-by: ...`,
  `BREAKING CHANGE: ...`) is preserved verbatim rather than re-wrapped.

## The `commit normalize` command

Validate and reformat a message:

```sh
nightshift commit normalize "feat: add login screen"
nightshift commit normalize --file .git/COMMIT_EDITMSG
git log -1 --pretty=%B | nightshift commit normalize
```

Trivially fixable issues (whitespace, uppercase type or subject, trailing
period, body wrapping) are fixed automatically; messages that need a human
decision (missing/unknown type, missing or overlong subject) are rejected.
With `--file` the normalized message is written back to the file, otherwise it
is printed to stdout.

Add `--check` to validate only. A diff-style report is printed and the command
exits non-zero when the message is not in canonical form or cannot be
normalized.

## commit-msg hook

To enforce the rules locally, install the hook:

```sh
make install-hooks
# or manually:
ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg
```

The hook normalizes your message file in place before the commit is created and
rejects messages that cannot be fixed automatically. Bypass it with
`git commit --no-verify`.
