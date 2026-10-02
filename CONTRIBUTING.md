# Contributing to Nightshift

Thanks for taking the time to contribute. This document captures the
conventions that keep history readable and reviews short.

## Getting Started

1. Install Go 1.21+.
2. Clone this repository and run `make deps` to fetch modules.
3. Install the pre-commit hook: `make install-hooks`.
4. Build the binary: `make build` (produces `./nightshift`).

## Branching

- Branch from `main`.
- Name branches with a short type prefix: `feat/...`, `fix/...`,
  `docs/...`, `chore/...`, `refactor/...`.
- Keep branches focused: one logical change per branch.

## Commits

- Use [Conventional Commits](https://www.conventionalcommits.org/):
  `type(optional-scope): imperative subject`.
- Allowed types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`,
  `perf`, `build`, `ci`, `revert`.
- Subject line in the imperative mood, <= 72 characters, no trailing
  period.
- Wrap the body at 72 columns. Explain the *why*, not the *what*.

## Code Style

- Run `gofmt` before committing (the pre-commit hook enforces this).
- Run `go vet ./...` and address findings.
- Keep logs hyper-concise: include needed info, minimize words (see
  [AGENTS.md](AGENTS.md)).
- Wrap errors with context (`fmt.Errorf("do thing: %w", err)`); never
  silently drop them.
- Tests live in `_test.go` files alongside code and prefer table-driven
  style.

## Running Checks

```bash
make test          # unit tests
make test-race     # race detector
make coverage      # coverage report
make lint          # golangci-lint (install separately)
make check         # test + lint
```

## Pull Requests

Before opening a PR:

- [ ] Rebase on the latest `main`.
- [ ] Run `make check` and confirm it passes.
- [ ] Update docs affected by the change (README, ARCHITECTURE,
      inline doc comments).
- [ ] Fill out the PR description: summary, motivation, test plan.

Small, reviewable PRs land faster than large ones. If a change grows
beyond ~400 lines of diff, consider splitting it.

## Reporting Issues

Open a GitHub issue with:

- A short descriptive title.
- Steps to reproduce or a minimal code sample.
- Expected vs. actual behavior.
- Relevant environment details (OS, Go version, provider CLIs in use).

## License

By contributing, you agree that your contributions will be licensed
under the [MIT License](LICENSE).
