# Commit Messages

Nightshift uses [Conventional Commits](https://www.conventionalcommits.org/)
for all commit messages. This keeps the history readable and lets tooling
derive changelogs automatically.

## Format

```
<type>(<scope>): <subject>

<body>

<trailers>
```

- **type** — one of `feat`, `fix`, `docs`, `style`, `refactor`, `perf`,
  `test`, `build`, `ci`, `chore`, `revert`.
- **scope** — optional, e.g. `fix(api): ...`.
- **subject** — lowercase, imperative mood, no trailing period, max 72 chars.
- **body** — optional, wrapped at 72 columns, separated from the subject by a
  blank line.
- **trailers** — optional `Token: value` lines (e.g. `Signed-off-by:`); kept
  verbatim, one per line.

Examples:

```
feat(api): add retry with exponential backoff
fix: apply --max-projects limit after processed-today filter
docs: document the commit-msg hook
```

## The `commit normalize` command

Validate and reformat a message:

```sh
nightshift commit normalize "feat: add login screen"
nightshift commit normalize --file .git/COMMIT_EDITMSG
git log -1 --pretty=%B | nightshift commit normalize
```

The command fixes the trivially fixable (whitespace, type casing, trailing
period, body wrapping) and exits non-zero when a message cannot be normalized
(missing/unknown type, capitalized or overlong subject). Add `--check` to
validate only without printing the normalized message.

## commit-msg hook

To enforce the rules locally, install the hook:

```sh
make install-hooks
```

This also symlinks `scripts/commit-msg.sh` into `.git/hooks/commit-msg` (and
`scripts/pre-commit.sh` into `.git/hooks/pre-commit`). The commit-msg hook
normalizes your message file in place before the commit is created and rejects
messages that cannot be fixed automatically. Bypass it with
`git commit --no-verify`.

CI runs the same validation over every commit in a pull request, so a
non-conforming message fails the build even without the local hook.
