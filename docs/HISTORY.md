# Nightshift: Historical Context

This document is a curated summary of the historical context of the Nightshift
codebase: how the project came to exist, which components were built (and when),
what major decisions shaped the architecture, and which recent fixes have
stabilized the system. It complements `README.md` (user-facing intro) and
`docs/permissions-auth-surface-map.md` (current security/auth surface).

Commits are referenced by short SHA. To inspect any one in detail:

```bash
git show <sha>
```

---

## 1. Project Origin

Nightshift began in the first quarter of the project's life as a Go CLI that
runs coding agents overnight against your own repositories — turning otherwise
idle weekly token allotments into background dead-code/doc-drift/test-gap/etc.
work. The earliest commits scaffold the project and lay the design contract:

- [`628f01d`] chore: initialize git repo with Go gitignore
- [`be0865f`] docs: add README and MIT license
- [`a84934f`] docs: add technical specification (SPEC.md)
- [`3233791`] chore: scaffold Go project structure
- [`8dffb29`] docs: incorporate review notes into SPEC.md

Two design choices set the tone for everything that followed:

1. **Branch/PR-only output.** Nightshift never writes directly to a primary
   branch; every run lands in a feature branch / PR so users can close what
   they don't like. This is the "whole rollback plan" mentioned in
   `README.md`.
2. **Budget-aware execution.** From day one the agent was meant to live inside
   weekly token allotments, deferring work rather than blowing past quota.

---

## 2. Component-by-Component Timeline

### 2.1 Configuration (`internal/config`)

- [`91de755`] feat(config): implement YAML config loading and validation
  — initial schema, validation, defaults.
- [`ba73293`] chore(config): change default `max_percent` from 10 → 75.
  Codified the user expectation that idle quota should mostly be used.
- [`99bcd82`] feat: add `CustomTaskConfig` struct — first-class user-defined
  tasks alongside built-ins.
- [`ee98bd8`] feat: add custom task config validation.
- [`615e2b9`] fix(#19): config `max_projects`, budget day-boundary, codex
  fallback flags — stabilization pass on long-standing UX issues.
- [`519821c`] fix: serialize provider config with correct YAML key names
  (fixes #20) — round-trip bug surfaced once the setup wizard started writing
  config back to disk.

### 2.2 Logging & State (`internal/logging`, `internal/state`)

- [`81f95f1`] feat(logging): structured logging with rotation.
- [`b6f3a80`] feat(state): run history and task tracking.
- [`8220d43`] feat: task cooldown / interval system — prevents the same task
  from being picked every night.

### 2.3 CLI Surface (`cmd/nightshift`, `internal/agents`, `internal/orchestrator`)

The CLI is structured around cobra. Commands landed roughly in the order a
user would touch them:

- [`4e3e9b1`] feat(cli): cobra command structure
- [`ae85ac7`] feat(cli): `init`
- [`bfe2ca5`] feat(cli): `run`
- [`20f7479`] feat(cli): `daemon`
- [`274afae`] feat(cli): `status` / `logs`
- [`7486bd9`] feat(cli): `budget`
- [`1da2620`] feat(cli): `config`
- [`ce58dc8`] feat(cli): `install` / `uninstall`
- [`ccef144`] feat(commands): `task list/show/run` subcommands
- [`9c56de6`] feat: `nightshift stats` (with JSON output)

Run-UX work over several iterations:

- [`7eaba8e`] `--max-projects` / `--max-tasks` flags
- [`d6ab73a`] `--ignore-budget` flag
- [`bcbb941`] preflight summary display
- [`bd8cd82`] confirmation prompt + `--yes` flag
- [`43cb05d`] `--dry-run` shows preflight and exits
- [`85480af`] `--random-task` flag
- [`0b33782`] live colored output in interactive mode
- [`4c0a915`] `--timeout` flag for `run` and `daemon`

### 2.4 Providers (`internal/providers`)

Provider support grew from Claude → Codex → Copilot, with parsing fidelity
improving in passes:

- [`7ceda39`] / [`cbfe759`] feat(providers): Claude data parser
- [`9b4e553`] feat(providers): Codex data parser
- [`d6c21dd`] feat(providers): parse Codex session JSONL for daily usage
- [`63463f4`] feat(providers): parse and store reset times from scraped output
- [`63a297d`] fix(providers): compute **billable** tokens excluding cached input
- [`0b2fec0`] / [`d88b989`] / [`3430191`] direct JSONL scanning for Claude,
  then revert to stats-cache as primary with JSONL as fallback (a deliberate
  walk-back after stability issues).
- [`caa932d`] Add GitHub Copilot CLI support with monthly budget tracking
- [`3337893`] Fix copilot cli integration (PR #39)
- [`c703999`] fix: resolve lint warnings in copilot provider and helpers

### 2.5 Budget (`internal/budget`)

- [`5d6779e`] feat(budget): initial calculation algorithm
- [`cf8c770`] feat(budget): wire Codex daily token usage in
- [`b9f7cdf`] fix: budget codex daily mode uses token-based calculation over
  5h window
- [`e5c7c88`] fix(budget): improve display clarity
- [`24f54cd`] feat(budget): display reset times from scraped snapshots
- [`b9b07b4`] fix: show budget period label (today/this week) on usage bar
- [`d78cb36`] fix: use week-aware avg daily usage in budget projection

### 2.6 Tasks (`internal/tasks`)

- [`b486f26`] feat(tasks): task definitions and cost estimation
- [`a9ab729`] feat(tasks): task selection and priority scoring
- [`59ecb25`] feat: `RegisterCustom`, `IsCustom`, `ClearCustom`
- [`93a34c7`] feat: wire custom task registration into `run` and `preview`
- [`5baae99`] feat: add `SelectRandom` method to task selector
- [`073c05a`] feat(tasks): detailed agent instructions for PII Exposure Scanner

### 2.7 Agents & Orchestrator (`internal/agents`, `internal/orchestrator`)

- [`367ac50`] / [`b6e8084`] Claude Code CLI adapter
- [`2f57b0d`] Codex CLI adapter
- [`a492d86`] / [`f98c1ca`] plan-implement-review loop
- [`8c85a3a`] fix: switch codex non-interactive mode to `exec` (community PR
  #11, brandon93s)
- [`b13164a`] fix: capture partial output on timeout and kill process groups

### 2.8 Scheduler (`internal/scheduler`)

- [`c4a831f`] feat(scheduler): cron and time-window scheduling
- [`82baf56`] Clarification of budgets for nighttime vs daytime

### 2.9 Reporting & Stats (`internal/reporting`, `internal/stats`)

- [`6162ec8`] feat(reporting): morning summary generator
- [`5226c88`] feat: run reports + provider preference
- [`cd5c7b7`] feat: task detail rows in report overview
- [`91c5f7a`] feat: show PR links prominently
- [`f36e30a`] feat: extract PR URLs from agent output
- [`93d616e`] feat: polish report summary card (duration, success rate)
- [`9c0b25b`] feat: stats computation
- [`512d11b`] fix: derive stats run count and dates from report files
- [`74875ec`] fix: show "exhausted" for past projected timestamps

### 2.10 Setup & Calibration (`internal/setup`, `internal/calibrator`)

- [`903dfd5`] feat: setup wizard
- [`a487ad1`] feat: improve setup UX
- [`e3170de`] Add provider calibration tooling and run_history provider
  tracking
- [`3348c51`] fix: stats budget projection uses calibrator instead of
  snapshot inference

### 2.11 Security & Safety (`internal/security`)

- [`af56947`] feat(security): security and safety features
- [`f070c93`] fix: guard sensitive paths (community PR #14, davemac)
- [`5e80e71`] fix: block task run in sensitive directories (home, root, tmp)
- [`f30eee7`] feat: enforce branch+PR prompts and dangerous-flags defaults
- [`8eb2cb5`] feat: inject nightshift run metadata into PRs and commits
- See also: `docs/permissions-auth-surface-map.md` for the current
  permissions/auth landscape, derived from the code above.

### 2.12 Projects (`internal/projects`)

- [`d0f5c19`] feat(projects): multi-project discovery and budget allocation
- [`18815f1`] feat: branch selection ability (community PR #12)
- [`9237154`] fix: apply `--max-projects` limit **after** processed-today
  filter (fixes #21) — the off-by-one of project filtering.

### 2.13 Bus-Factor Analyzer (`internal/analysis`)

- [`7f24d07`] feat: implement bus-factor analyzer for code ownership
  concentration (merged via PR #4)
- [`1f7ecc8`] fix: correct markdown rendering of recommendations
- See also: `docs/bus-factor.md`.

### 2.14 Website & Docs

- [`3cb3bf2`] feat: add Docusaurus website with warm nighttime theme
- [`939c50f`] docs: comprehensive task reference page for all 59 built-in
  tasks
- [`ebbee00`] docs: remove auto-generated implementation docs (cleanup)
- [`91c25ba`] docs: update guides, website docs, README

### 2.15 TUI (removed)

A Bubbletea-based TUI was implemented and later removed once the CLI matured
and live colored output covered most needs:

- [`057d616`] feat(ui): implement bubbletea TUI for monitoring
- [`aba9798`] Remove TUI and related docs

---

## 3. Release Milestones

| Version | Commit       | Notes                                                |
|---------|--------------|------------------------------------------------------|
| v0.2.0  | [`d051ca8`]  | Run UX, preflight, dry-run, confirmation, flags     |
| v0.3.0  | [`c59dcb9`]  | Bus-factor, security hardening, backward-compat tests|
| v0.3.1  | (migration: `docs/MIGRATION-v0.3.0-to-v0.3.1.md`)   |                                              |
| v0.3.2  | [`0b5e905`]  | Codex exec fix, sensitive-path guards               |
| v0.3.3  | [`1e89a6c`]  | Branch selection, gofmt fixes                       |
| v0.3.4  | [`2ebd805`]  | Provider config serialization, max-projects filter, |
|         |              | mobile navbar fix, pre-commit hook                  |

The release process (goreleaser, version bumping, tag/push) is documented in
the `nightshift-release` skill — see CLAUDE-CODE skill metadata, not a file in
this repo.

---

## 4. Recent Stabilization Pass (around v0.3.4)

The most recent commits on `main` are a focused stabilization pass rather than
new features. They sit at the top of `git log --first-parent main`:

- [`68bbf11`] **fix: mobile navbar sidebar hidden behind secondary panel**
  (#46) — Docusaurus site z-index/layering on small viewports.
- [`9237154`] **fix: apply `--max-projects` limit after processed-today
  filter** (#45, fixes #21) — limit was being applied before filtering, so
  users could end up with zero projects to run.
- [`13720ae`] **Add pre-commit hook for gofmt/vet/build** (#44) — keep
  lint-fix churn out of `main` by failing the commit locally.
- [`519821c`] **fix: serialize provider config with correct YAML key names**
  (#43, fixes #20) — write path didn't match the read schema.
- [`615e2b9`] **fix(#19): config `max_projects`, budget day-boundary, codex
  fallback flags** (#42) — bundled config/budget/provider polish.
- [`c703999`] **fix: resolve lint warnings in copilot provider and helpers**
  (#38) — long tail of the Copilot integration landing.
- [`6d34c71`] **refactor: replace `WriteString(fmt.Sprintf(...))` with
  `fmt.Fprintf`** (#41) — cosmetic but caught by linters.

A series of `codex/lint-fix-*` and `chore/lint-fix-*` branches in
`git branch -a` are the staging areas for this work; the squashed result is
what shows up on `main`.

---

## 5. Architectural Decisions Worth Knowing

These aren't documented in any single ADR file but are visible in the commit
history above:

1. **Provider abstraction with explicit billable-token semantics.**
   `63a297d` cemented the rule that cached input tokens are excluded from
   billing math; all later budget code assumes that contract.
2. **Stats-cache primary, JSONL fallback** for Claude token counting.
   `0b2fec0 → d88b989 → 3430191` walked from "JSONL first" back to
   "stats-cache first," because the JSONL path was slower and noisier than
   the cache in practice.
3. **Branch + PR is mandatory.** `f30eee7` made branch/PR prompts and
   dangerous-flag defaults part of orchestrator behavior, not an opt-in.
   This is enforced at the agent layer so all providers inherit it.
4. **Sensitive-directory guard.** `5e80e71` + `f070c93` block runs in
   `$HOME`, `/`, `/tmp` — accidental "run in the wrong directory" is the
   single largest blast-radius failure mode for an unattended overnight
   agent.
5. **Calibrator is the source of truth for projection.** `3348c51` moved
   stats projection off scraped snapshots onto the calibrator, since
   snapshots lag and the calibrator already integrates run history.

---

## 6. Cross-References

- `README.md` — user-facing project intro, install/quick-start.
- `docs/permissions-auth-surface-map.md` — present-day permissions/auth
  surface, derived from `internal/security`, the orchestrator, and provider
  adapters above.
- `docs/bus-factor.md` — bus-factor analyzer output format and methodology
  (see §2.13).
- `docs/MIGRATION-v0.3.0-to-v0.3.1.md` — the only formal migration guide so
  far.
- `docs/guides/` — task-author / user guides.
- `docs/implemented/` — historical implementation notes for landed features.
- `docs/deprecated/` — features removed but worth preserving context for.

---

*This file is hand-curated and will drift over time. When in doubt, trust
`git log --first-parent main` over this document.*
