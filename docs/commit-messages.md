# Commit Message Convention

This project follows the [Conventional Commits](https://www.conventionalcommits.org/)
specification. All commits are expected to match the shape below and are
enforced by a `commit-msg` hook (see [Installing the hook](#installing-the-hook)).

## Format

```
<type>(<scope>): <subject>

<body wrapped to 72 columns>

<Trailers>
```

- **type** *(required)* — lowercase, one of the allowed types listed below.
- **scope** *(optional)* — lowercase, one of the allowed scopes. Omit the
  parentheses entirely when there is no scope.
- **subject** *(required)* — imperative mood (`add`, not `added`/`adds`), no
  trailing period, hard limit of **72 characters**. The first letter is
  lower-cased.
- **body** *(optional)* — separated from the subject by a blank line, hard-wrapped
  to **72 columns**. Paragraphs are separated by blank lines.
- **trailers** *(optional)* — separated from the body by a blank line, in
  `Key: value` form. `Nightshift-*` trailers required for task tracking are
  sorted to the end automatically.

A trailing `!` after the type/scope marks a breaking change
(e.g. `feat(api)!: drop v1`).

## Allowed types

| Type       | Use for                                                       |
|------------|---------------------------------------------------------------|
| `feat`     | A new feature                                                 |
| `fix`      | A bug fix                                                     |
| `docs`     | Documentation only                                            |
| `style`    | Formatting, whitespace, semicolons — no code change           |
| `refactor` | Code change that neither fixes a bug nor adds a feature       |
| `perf`     | A change that improves performance                            |
| `test`     | Adding or correcting tests                                    |
| `build`    | Build system or external dependencies                         |
| `ci`       | CI configuration files and scripts                            |
| `chore`    | Repetitive tasks, tooling, repo maintenance                   |
| `revert`   | Reverting a previous commit                                   |

## Allowed scopes

`commits`, `config`, `scheduler`, `providers`, `agents`, `cli`, `docs`,
`website`, `db`, `reporting`, `security`, `setup`, `orchestrator`, `budget`.

Unknown scopes are not rejected outright (new subsystems appear before their
scope is registered), but tooling will warn. Keep scope names lowercase and
short.

## Examples

Well-formed:

```
feat(commits): add message normalizer

Parses raw commit messages into type/scope/subject/body/trailers and
rewrites them to the canonical format. Idempotent on already-conforming
messages.

Nightshift-Task: commit-normalize
Nightshift-Ref: https://github.com/marcus/nightshift
```

Needs normalization (the normalizer rewrites this to the form above):

```
FEAT(Commits): Added message normalizer.
```

The validator reports:

- `unknown_type` / `missing_type` — type missing or not in the allowed list
- `oversize_subject` — subject longer than 72 characters
- `non_imperative` — leading verb is past tense, gerund, or third person
- `trailing_period` — subject ends with a period
- `unwrapped_body` — a body line longer than 72 columns

## Programmatic use

```go
import "github.com/marcus/nightshift/internal/commits"

normalized, err := commits.Normalize(raw)        // auto-fix
violations := commits.Validate(raw)              // report
ok := commits.Conforms(raw)                      // bool
```

## Installing the hook

```sh
scripts/install-commit-msg.sh
```

This writes `.git/hooks/commit-msg`, which runs `cmd/commit-lint` on the staged
message and rejects the commit if it does not conform. Re-run the script after a
fresh clone.
