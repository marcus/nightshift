# Contributing to Nightshift

Thanks for helping improve Nightshift! This guide covers the local development
loop, the checks your changes must pass, and the commit/PR conventions the
project uses.

## Prerequisites

- **Go** 1.24 or newer (see `go.mod`)

Optional but recommended:

- **golangci-lint** — for `make lint` (install with
  `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`)
- **gum** — used by the CLI's preview pager (the CLI degrades gracefully without it)

## Getting the code

```bash
git clone https://github.com/marcus/nightshift.git
cd nightshift
make deps          # go mod download && go mod tidy
```

## Development loop

All common tasks are wrapped as `make` targets:

| Command | What it does |
|---------|--------------|
| `make build` | Build the `nightshift` binary |
| `make test` | Run the full test suite (`go test ./...`) |
| `make test-verbose` | Run tests with `-v` |
| `make test-race` | Run tests with the race detector (`go test -race ./...`) |
| `make coverage` | Run tests with a coverage report (`go tool cover -func`) |
| `make coverage-html` | Generate an HTML coverage report |
| `make lint` | Run `golangci-lint` (if installed) |
| `make check` | Run `test` + `lint` |
| `make install` | Build and install the binary to your Go bin directory |
| `make calibrate-providers` | Run the provider-calibration diagnostic tool |
| `make help` | List all available targets |

Typical flow before opening a PR:

```bash
make build
make test-race
make lint
```

## Pre-commit hook

Install the git pre-commit hook to catch formatting and build issues before
pushing:

```bash
make install-hooks
```

This symlinks `scripts/pre-commit.sh` into `.git/hooks/pre-commit`. On every
commit it runs:

- **gofmt** — flags any staged `.go` files that need formatting
- **go vet** — catches common correctness issues
- **go build** — ensures the whole module compiles

To bypass the hook in a pinch:

```bash
git commit --no-verify
```

## Code style

Follow standard Go conventions (the project enforces `gofmt` and `go vet`):

- Run `gofmt -w .` before committing (the hook checks this for you).
- Wrap errors with context rather than swallowing them.
- Write table-driven tests in `_test.go` files alongside the code they cover.
- Keep log messages hyper-concise but informative.

## Branching and pull requests

1. **Never commit directly to `main`.** Branch off the latest `main`:

   ```bash
   git checkout main
   git pull
   git checkout -b <your-branch>
   ```

   Use a descriptive branch name, e.g. `docs/backfill-missing-docs`,
   `fix/budget-overflow`, or `feat/new-task`.

2. Make your changes, keeping commits focused.
3. Push your branch and **open a pull request** against `marcus/nightshift`
   `main`. Describe what changed and why; link any relevant issue.
4. Ensure CI and the pre-commit checks pass. Address review feedback with
   additional commits (or a tidy rebase if the maintainer prefers).

## Commit trailers

Every commit should include the Nightshift task trailers so it can be traced
back to the work item that produced it. Add them to your commit message:

```
Fix budget rollover at midnight boundaries

Nightshift-Task: budget-rollover
Nightshift-Ref: https://github.com/marcus/nightshift
```

- `Nightshift-Task` — the short id/slug of the task this change belongs to.
- `Nightshift-Ref` — the canonical project URL (`https://github.com/marcus/nightshift`).

You can add trailers interactively in your editor, or pass them on the command line:

```bash
git commit -m "Fix budget rollover at midnight boundaries" \
  -m "Nightshift-Task: budget-rollover" \
  -m "Nightshift-Ref: https://github.com/marcus/nightshift"
```

## Reporting issues

Found a bug or have a feature idea? Please open an issue on
[github.com/marcus/nightshift/issues](https://github.com/marcus/nightshift/issues)
with a clear description and, for bugs, steps to reproduce.

## License

By contributing you agree that your contributions will be licensed under the
project's [MIT license](LICENSE).
