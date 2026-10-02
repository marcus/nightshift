# docs-backfill — Documentation Backfiller

`docs-backfill` is the built-in Nightshift task that generates **missing**
documentation. It is one of the catalog entries registered in
`internal/tasks/tasks.go`:

```go
TaskDocsBackfill: {
    Type:            TaskDocsBackfill,        // "docs-backfill"
    Category:        CategoryPR,              // produces a PR
    Name:            "Documentation Backfiller",
    Description:     "Generate missing documentation",
    CostTier:        CostLow,                 // 10–50k tokens
    RiskLevel:       RiskLow,
    DefaultInterval: 168 * time.Hour,         // 7 days
},
```

Like every Nightshift task, it lands its work as a branch or PR — it never
writes to the primary branch. Don't like the result? Close the PR; that is the
entire rollback plan.

## What it does

The task looks for documentation that is genuinely absent and adds it, grounded
in the actual code and existing docs rather than inventing APIs, types, or
exported symbols that do not exist. In scope:

- **Top-level docs** — a project `README.md`, `CONTRIBUTING.md`, `LICENSE`
  notes, and similar foundational files when they are missing.
- **Package godoc** — Go `// Package ...` comments for packages that lack one,
  so `go doc ./...` renders useful output.
- **Guides** — task and feature guides under `docs/guides/`.
- **Architecture and design notes** — package maps and design docs under
  `docs/implemented/`.

Out of scope: rewriting or duplicating documentation that already exists, and
inventing exported symbols, configuration keys, or CLI flags. When existing
docs need changes, a separate change should edit them in place rather than
forking a parallel copy.

## What counts as "missing"

A doc is a candidate when:

1. **It does not exist.** There is no file at the conventional path (e.g. no
   `CONTRIBUTING.md` at the repo root, or no `// Package` comment on a Go
   package).
2. **It is not redundant.** Adding it would not duplicate content already
   present elsewhere. For Go packages this matters in particular: if a package
   already documents itself with a `// Package` comment in a source file, the
   task must not also drop a `doc.go` next to it — that creates a duplicate
   package comment.
3. **It can be grounded.** The content can be written from the real codebase
   (actual package responsibilities, real symbols, real config keys) rather
   than speculation.

When in doubt, the task prefers a smaller, clearly-correct addition over a
large speculative one.

## File-selection heuristics

- Operate per project (Nightshift resolves the target project the same way as
  every other task).
- Prefer the conventional locations: repo root for `README.md` /
  `CONTRIBUTING.md`, `docs/guides/` for guides, `docs/implemented/` for
  architecture and design notes.
- For Go packages, read the existing `// Package` comments and source before
  writing; only add a package comment where one is absent.
- Keep additions consistent in voice and structure with the surrounding docs
  (see existing entries under `docs/guides/` for the house style).

## Scheduling and cooldown

`docs-backfill` runs on the standard Nightshift schedule and is subject to the
usual task-selection rules in `internal/tasks`:

- **Default cooldown: 168h (7 days).** Once `docs-backfill` has run against a
  project, it will not be selected again for that project until the interval
  elapses. Tasks that have never run (or whose interval is `<= 0`) are always
  eligible.
- The selector also applies priority scoring and the staleness bonus, so a
  project that has never had a docs-backfill run ranks higher than one that
  was just covered.

Override the interval per task in `nightshift.yaml`:

```yaml
tasks:
  enabled:
    - docs-backfill
  intervals:
    docs-backfill: "336h"   # extend to every two weeks
```

Disable it by listing it under `tasks.disabled`.

## Safe by design

The output model is the same as every other Nightshift task:

- Work happens on a **branch**, never the primary branch.
- Changes are committed and submitted as a **PR** with traceable trailers:
  ```
  Nightshift-Task: docs-backfill
  Nightshift-Ref: https://github.com/marcus/nightshift
  ```
- Reviewers merge what surprises them and close the rest.

Because documentation changes are low-risk (`RiskLevel: RiskLow`) and
typically low-cost (`CostTier: CostLow`), `docs-backfill` is well suited to
running unattended overnight.

## Related

- [`adding-tasks.md`](adding-tasks.md) — how built-in tasks (including this
  one) are defined and registered
- [Architecture](../implemented/architecture.md) — package map, including the
  `internal/tasks` registry that defines `docs-backfill`
- [`run-lifecycle.md`](run-lifecycle.md) — the run lifecycle and PR output model
