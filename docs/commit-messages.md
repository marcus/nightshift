# Commit Messages

Nightshift uses [Conventional Commits](https://www.conventionalcommits.org/)
for all commit messages. This keeps the history readable and lets tooling
derive changelogs automatically.

## Format

```
<type>[(<scope>)][!]: <subject>

<body>
```

- **type** — one of `feat`, `fix`, `docs`, `style`, `refactor`, `test`,
  `chore`, `perf`, `build`, `ci`.
- **scope** — optional, e.g. `fix(api): ...`.
- **!** — optional breaking-change marker, placed immediately before the colon,
  e.g. `feat!:` or `feat(api)!:` (Conventional Commits 1.0.0).
- **subject** — lowercase, imperative mood, no trailing period, max 72 chars.
- **body** — optional, wrapped at 72 columns, separated from the subject by a
  blank line.

## The `commit normalize` command

Validate and reformat a message:

```sh
nightshift commit normalize "feat: add login screen"
nightshift commit normalize --file .git/COMMIT_EDITMSG
git log -1 --pretty=%B | nightshift commit normalize
```

By default the normalized message is written to stdout. Add `--check` to
**validate only**: nothing is written to stdout (besides an `ok` confirmation),
and the command exits non-zero when a message cannot be normalized (missing or
unknown type, capitalized or overlong subject).

```sh
# non-zero exit if the message does not conform
nightshift commit normalize --check --file .git/COMMIT_EDITMSG
```

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
