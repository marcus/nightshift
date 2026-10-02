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
  `chore`, `perf`, `build`, `ci`.
- **scope** — optional, e.g. `fix(api): ...`.
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

Add `--check` to validate only. The command exits non-zero when a message
cannot be normalized (missing/unknown type, capitalized or overlong subject).

## commit-msg hook

To enforce the rules locally, install the hook:

```sh
ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg
chmod +x scripts/commit-msg.sh
```

The hook normalizes your message file in place before the commit is created and
rejects messages that cannot be fixed automatically. Bypass it with
`git commit --no-verify`.
