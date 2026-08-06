# Commit Messages

Nightshift uses [Conventional Commits](https://www.conventionalcommits.org/)
for all commit messages. This keeps the history readable and lets tooling
derive changelogs automatically.

## Format

```
<type>(<scope>): <subject>

<body>
```

- **type** — one of `feat`, `fix`, `docs`, `style`, `refactor`, `test`,
  `chore`, `perf`, `build`, `ci`, `revert`.
- **scope** — optional, e.g. `fix(api): ...`.
- **subject** — lowercase, imperative mood, no trailing period, max 72 chars.
- **body** — optional, wrapped at 72 columns, separated from the subject by a
  blank line.

A `!` after the type (or scope) denotes a breaking change, e.g.
`feat(api)!: change response shape`.

## The `commit normalize` command

Validate and reformat a message:

```sh
nightshift commit normalize "feat: add login screen"
nightshift commit normalize --file .git/COMMIT_EDITMSG
git log -1 --pretty=%B | nightshift commit normalize
```

Add `--check` (`-c`) to validate only. With `--check` the command prints
nothing on success and exits non-zero when a message cannot be normalized
(missing/unknown type, capitalized or overlong subject) — useful in CI:

```sh
nightshift commit normalize --check --file .git/COMMIT_EDITMSG && echo ok
```

## commit-msg hook

To enforce the rules locally, install the hook:

```sh
make install-hooks
# or manually:
ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg
```

The hook normalizes your message file in place before the commit is created and
rejects messages that cannot be fixed automatically, printing the expected
format and allowed types on failure. Bypass it with `git commit --no-verify`.
