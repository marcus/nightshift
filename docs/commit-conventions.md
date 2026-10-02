# Commit Message Conventions

Nightshift uses [Conventional Commits](https://www.conventionalcommits.org/) for
all commit messages. A local git hook normalizes and validates each message,
so most contributors never need to think about formatting beyond following the
template.

## Format

```
<type>(<scope>)!: <subject>

<body — optional, wrap at 72 chars>

<trailers — optional>
```

- **type** *(required)* — one of: `feat`, `fix`, `docs`, `style`, `refactor`,
  `perf`, `test`, `build`, `ci`, `chore`, `revert`.
- **scope** *(optional)* — the area touched (`parser`, `daemon`, `config`).
- **`!`** *(optional)* — appended to the type/scope to mark a breaking change.
- **subject** *(required)* — imperative, no trailing period, ≤ 72 characters
  including the type prefix.
- **body** *(optional)* — separated from the subject by a blank line. Explain
  *why* the change is needed.
- **trailers** *(optional)* — `Key: value` lines after a blank line. Common
  trailers: `Co-Authored-By`, `Fixes`, `Refs`, `Signed-off-by`,
  `Nightshift-Task`, `Nightshift-Ref`.

## Examples

Valid:

```
feat(parser): allow optional trailing comma
```

```
fix: handle empty config without panicking

When NIGHTSHIFT_CONFIG points at an empty file the daemon crashed on
startup. Treat empty as "use defaults".

Fixes: #123
```

```
refactor(daemon)!: replace polling with channel-based notify

BREAKING CHANGE: the `--poll-interval` flag is removed. Configure
notifications via `notify.channel` in the config file instead.
```

Invalid (and how the normalizer reacts):

| Message                                | What happens                                      |
| -------------------------------------- | ------------------------------------------------- |
| `Fix typo`                             | Rejected — missing type prefix.                   |
| `FEAT: add thing`                      | Normalized → `feat: add thing` (lowercased type). |
| `feat:   add   thing  `                | Normalized — extra whitespace trimmed.            |
| `wibble: do stuff`                     | Rejected — `wibble` is not an allowed type.       |
| `feat: ` *(empty description)*         | Rejected — subject has no description.            |
| 90-character `feat: …` subject          | Rejected — exceeds 72-char limit.                 |

## Skip rules

The normalizer intentionally leaves these alone:

- Merge commits (`Merge branch …`, `Merge pull request …`).
- Fixup / squash / amend commits (`fixup!`, `squash!`, `amend!`).
- Revert commits (`Revert "…"`).

## Enabling the hook

Run once per clone:

```sh
scripts/setup-hooks.sh
```

This sets `core.hooksPath=.githooks` and `commit.template=.gitmessage`. After
that, every `git commit` runs through `scripts/commit-normalizer.sh`.

To bypass for a single commit:

```sh
COMMIT_NORMALIZER=0 git commit -m "..."
```

To disable entirely:

```sh
git config --unset core.hooksPath
```

## Testing the normalizer

```sh
tests/commit-normalizer/run-tests.sh
```

Fixture-driven; each case lives under `tests/commit-normalizer/cases/`.
