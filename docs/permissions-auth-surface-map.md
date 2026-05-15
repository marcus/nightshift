# Nightshift Permissions & Auth Surface Map

A read-only audit of every permission gate, credential, sandbox restriction,
and audit-logged operation in the nightshift codebase. Each entry cites
`file:line` so reviewers can jump directly to the source.

Generated as part of the `permissions-mapper` task. See the
[Identified Gaps](#7-identified-gaps--hardening-recommendations) section for
discrepancies between the documented model and the code that ships today.

---

## 1. Safety Modes & Gates

The `security` package defines a safety model centered on a `SafetyMode` and a
`Manager` that gates write, push, and budget operations.

| Concept              | Type / Constant                                                        | Source                                  |
| -------------------- | ---------------------------------------------------------------------- | --------------------------------------- |
| Safety mode (enum)   | `SafetyMode` with `ModeReadOnly`, `ModeNormal`                         | `internal/security/security.go:15-22`   |
| Default config       | `DefaultConfig()` — read-only, writes off, push off, network off, 75%  | `internal/security/security.go:36-47`   |
| Security manager     | `Manager` aggregates config + credentials + audit                      | `internal/security/security.go:50-79`   |
| First-run gate       | `IsFirstRun()` / `MarkFirstRunComplete()`                              | `internal/security/security.go:82-98`   |
| Write gate           | `ValidateWriteAccess()` — blocks first-run + read-only without writes  | `internal/security/security.go:101-114` |
| Push gate            | `ValidateGitPush()` — blocks unless `AllowGitPush`                     | `internal/security/security.go:117-126` |
| Budget gate          | `ValidateBudgetSpend(currentPercent)` vs `MaxBudgetPercent`            | `internal/security/security.go:129-138` |
| Pre-execution gate   | `CheckPreExecution(op)` runs credential + write + push checks per op   | `internal/security/security.go:180-206` |
| Project-path guard   | `ValidateProjectPath(path)` rejects `/`, `/tmp`, `/var`, `/etc`, `/usr`, `$HOME` | `internal/security/security.go:212-238` |

### Default policy summary

- Mode: `ModeReadOnly`
- Writes: disabled (`EnableWrites=false`)
- Git push: disabled (`AllowGitPush=false`)
- Network for agents: disabled (`AllowNetworkAgent=false`)
- Budget cap: `MaxBudgetPercent=75`
- Audit log dir: `~/.local/share/nightshift/audit` (mode `0700`)
- First-run sentinel: `~/.local/share/nightshift/.first_run_complete`

---

## 2. Operation Types Matrix

`Operation` records the action being performed and is fed into
`CheckPreExecution`. Each `OperationType` triggers a different combination of
gates and audit events.

| Operation Type   | Constant         | Write gate                | Push gate                 | Audit event           |
| ---------------- | ---------------- | ------------------------- | ------------------------- | --------------------- |
| Agent invoke     | `OpAgentInvoke`  | no                        | no                        | `agent_start`         |
| File read        | `OpFileRead`     | no                        | no                        | `file_read`           |
| File write       | `OpFileWrite`    | `ValidateWriteAccess()`   | no                        | `file_write`          |
| Git commit       | `OpGitCommit`    | `ValidateWriteAccess()`   | no                        | `git_commit`          |
| Git push         | `OpGitPush`      | no                        | `ValidateGitPush()`       | `git_push`            |
| Network call     | `OpNetworkCall`  | no                        | no (no gate today)        | passthrough enum      |

References:

- Operation type constants: `internal/security/security.go:243-250`
- `Operation` struct: `internal/security/security.go:252-260`
- Gate routing in `CheckPreExecution`: `internal/security/security.go:180-206`
- Operation→audit event mapping: `internal/security/audit.go:157-172`

> Note: `OpNetworkCall` exists but is not currently checked against
> `Config.AllowNetworkAgent` (see Gaps §7.3).

---

## 3. CLI Permission Flags

Permission-affecting CLI flags surfaced today.

| Flag (CLI)                | Subcommand                | Purpose                                                  | Source                                  |
| ------------------------- | ------------------------- | -------------------------------------------------------- | --------------------------------------- |
| `--dry-run`               | `run`, `task run`         | Preflight only; no execution                             | `cmd/nightshift/commands/run.go:106`, `cmd/nightshift/commands/task.go:71` |
| `--ignore-budget`         | `run`                     | Bypass budget checks                                     | `cmd/nightshift/commands/run.go:111`    |
| `--yes / -y`              | `run`                     | Skip interactive confirmation                            | `cmd/nightshift/commands/run.go:112`    |
| `--max-projects N`        | `run`                     | Cap projects per run (default 1)                         | `cmd/nightshift/commands/run.go:109`    |
| `--max-tasks N`           | `run`                     | Cap tasks per project (default 1)                        | `cmd/nightshift/commands/run.go:110`    |
| `--random-task`           | `run`                     | Pick a random task                                       | `cmd/nightshift/commands/run.go:113`    |
| `--branch`                | `run`, `task run`         | Base branch for new feature branches                     | `cmd/nightshift/commands/run.go:114`, `cmd/nightshift/commands/task.go:73` |
| `--timeout`               | `run`, `task run`         | Per-agent execution timeout                              | `cmd/nightshift/commands/run.go:116`, `cmd/nightshift/commands/task.go:72` |
| `--provider`              | `task run` (required)     | Force provider selection                                 | `cmd/nightshift/commands/task.go:69-74` |
| `--force / -f`            | `init`                    | Overwrite existing config without prompt                 | `cmd/nightshift/commands/init.go:35`    |

### Flags documented in spec/user-guide but NOT wired into the CLI

The README and historical spec advertise `--enable-writes`, `--allow-push`, and
a `--max-budget-percent` flag, all of which appear only in error messages
emitted by `internal/security/security.go`:

| Documented flag         | Doc reference                                | Status                          |
| ----------------------- | -------------------------------------------- | ------------------------------- |
| `--enable-writes`       | `docs/deprecated/user-guide.md:654`, `website/docs/configuration.md:111` | Not registered in any cobra command (see Gaps §7.1). Error message at `internal/security/security.go:106`. |
| `--allow-push`          | implicit via error message                   | Not registered. Error at `internal/security/security.go:122`. |
| `--max-budget-percent`  | task description only                        | Not registered.                 |

---

## 4. Credentials & Env Vars by Provider/Integration

### 4.1 AI provider credentials (security package)

| Credential           | Env var               | Required?                            | Source                                  |
| -------------------- | --------------------- | ------------------------------------ | --------------------------------------- |
| Anthropic API key    | `ANTHROPIC_API_KEY`   | At least one of Anthropic/OpenAI     | `internal/security/credentials.go:14`   |
| OpenAI API key       | `OPENAI_API_KEY`      | At least one of Anthropic/OpenAI     | `internal/security/credentials.go:15`   |

Validation:

- Required check: `CredentialManager.ValidateRequired()` errors when both are
  missing — `internal/security/credentials.go:41-51`.
- All-credential status (for `doctor`): `ValidateAll()` —
  `internal/security/credentials.go:54-81`.
- Per-key existence: `HasAnthropicKey()` / `HasOpenAIKey()` —
  `internal/security/credentials.go:84-91`.
- Format check: `ValidateCredentialFormat(name, value)` —
  `internal/security/credentials.go:203-222` (expects `sk-ant-*` / `sk-*`).
- Config-leak scan: `CheckConfigForCredentials(content)` /
  `EnsureNoCredentialsInFile(path)` —
  `internal/security/credentials.go:100-148`.

### 4.2 Provider CLIs (auth handled out-of-band by the upstream CLI)

`nightshift` shells out to the provider CLI; auth lives in the CLI's own
state/login flow rather than env vars enforced by nightshift.

| Provider | Binary                                       | Default data path                  | Source                                  |
| -------- | -------------------------------------------- | ---------------------------------- | --------------------------------------- |
| Claude   | `claude`                                     | `~/.claude` (override `data_path`) | `internal/agents/claude.go:97-109`, `internal/providers/claude.go:84-97` |
| Codex    | `codex`                                      | `~/.codex`                         | `internal/agents/codex.go:54-66`, `internal/config/config.go:158`        |
| Copilot  | `gh` (preferred) or standalone `copilot`     | `~/.copilot`                       | `internal/agents/copilot.go:65-76`, `cmd/nightshift/commands/helpers.go:72-95`, `internal/config/config.go:159` |

### 4.3 Provider permission-bypass flags (per-provider config)

Stored in `nightshift.yaml` (defaults all `false` after the security audit
hardening — see `internal/config/config.go:253-262`).

| Setting                                                 | Effect                                                           | Source                                  |
| ------------------------------------------------------- | ---------------------------------------------------------------- | --------------------------------------- |
| `providers.claude.dangerously_skip_permissions`         | Adds `--dangerously-skip-permissions` to every `claude` invocation | `internal/agents/claude.go:131-134`     |
| `providers.codex.dangerously_bypass_approvals_and_sandbox` | Adds `--dangerously-bypass-approvals-and-sandbox` to `codex exec`  | `internal/agents/codex.go:88-92`, `cmd/nightshift/commands/helpers.go:48-69` |
| `providers.copilot.dangerously_skip_permissions`        | Adds `--allow-all-tools --allow-all-urls` to copilot invocation  | `internal/agents/copilot.go:104-115`    |

> CodexAgent's *constructor* still defaults `bypassPerm=true`
> (`internal/agents/codex.go:55-66`); the safe default is preserved at the
> CLI/config layer (helpers.go:48-69, config.go:258).

### 4.4 Reporting / notification credentials

| Service     | Env vars                                                                                          | Source                                  |
| ----------- | ------------------------------------------------------------------------------------------------- | --------------------------------------- |
| SMTP email  | `NIGHTSHIFT_SMTP_HOST`, `NIGHTSHIFT_SMTP_PORT`, `NIGHTSHIFT_SMTP_USER`, `NIGHTSHIFT_SMTP_PASS`, `NIGHTSHIFT_SMTP_FROM` | `internal/reporting/summary.go:314-348` |
| Slack       | `reporting.slack_webhook` config field (URL is the credential)                                    | `internal/config/config.go:142`, `internal/reporting/summary.go:296-388` |

### 4.5 Integration auth (delegated to external CLIs)

| Integration | Binary | Auth source                                                                       | Source                                  |
| ----------- | ------ | --------------------------------------------------------------------------------- | --------------------------------------- |
| GitHub      | `gh`   | `gh auth` token managed by the user; nightshift only `LookPath`s `gh`             | `internal/integrations/github.go:48-91` |
| td          | `td`   | td CLI's own auth/state; nightshift only `LookPath`s `td`                         | `internal/integrations/td.go:45-83`     |

### 4.6 Generic env binding via viper

`bindEnvVars()` exposes any `NIGHTSHIFT_<KEY>` env var as a config override via
`AutomaticEnv` plus explicit bindings for `BUDGET_MAX_PERCENT`, `BUDGET_MODE`,
`LOG_LEVEL`, `LOG_PATH` — `internal/config/config.go:292-303`.

---

## 5. Filesystem Path Restrictions

| Restriction                      | Behavior                                                                                | Source                                  |
| -------------------------------- | --------------------------------------------------------------------------------------- | --------------------------------------- |
| Sensitive project-path blocklist | `/`, `/tmp`, `/var`, `/etc`, `/usr`, `$HOME` cannot be the project root                 | `internal/security/security.go:212-238` |
| Sandbox `DeniedPaths`            | Path-prefix blocklist enforced for both command lookup and `ValidatePath`               | `internal/security/sandbox.go:201-216`, `internal/security/sandbox.go:256-286` |
| Sandbox `AllowedPaths`           | When set, paths must be inside one of the allowlisted prefixes                          | `internal/security/sandbox.go:272-283`  |
| Sandbox env scrub                | Only `PATH`, `HOME`, `USER`, `SHELL`, `TERM`, `LANG`, `LC_ALL` plus AI keys are passed  | `internal/security/sandbox.go:218-254`  |
| Network restriction (advisory)   | When `AllowNetwork=false`, sets `NIGHTSHIFT_NO_NETWORK=1` env hint (no real isolation)  | `internal/security/sandbox.go:248-251`  |
| Sandbox temp dir                 | `os.MkdirTemp("", "nightshift-sandbox-*")`; refuses to clean up `/` or system tmp        | `internal/security/sandbox.go:60-74`, `internal/security/sandbox.go:298-316` |
| Audit log dir mode               | `0700` on creation                                                                      | `internal/security/audit.go:65-69`, `internal/security/security.go:60-64` |
| Audit log file mode              | `0600`, append-only `O_CREATE|O_APPEND|O_WRONLY`                                        | `internal/security/audit.go:88-100`     |
| Database dir mode                | `0700` (per the Feb 2026 security audit fix; see `SECURITY_AUDIT.md:50-72`)             | `SECURITY_AUDIT.md:50-72`               |
| First-run sentinel dir mode      | `0755`                                                                                  | `internal/security/security.go:88-98`   |

---

## 6. Audit Logging

Audit logger is initialised by `Manager` and writes JSONL lines per event.

### 6.1 Storage

- Location: `Config.AuditLogPath` (default `~/.local/share/nightshift/audit`).
- File pattern: `audit-YYYY-MM-DD.jsonl`, rotated on day change via
  `RotateIfNeeded()` — `internal/security/audit.go:262-284`.
- Permissions: dir `0700`, file `0600` — `internal/security/audit.go:65-100`.
- Each line is `fsync`'d for durability — `internal/security/audit.go:135-138`.

### 6.2 Event types

| Event                  | Constant                  | Source                                  |
| ---------------------- | ------------------------- | --------------------------------------- |
| Agent start            | `AuditAgentStart`         | `internal/security/audit.go:18`         |
| Agent complete         | `AuditAgentComplete`      | `internal/security/audit.go:19`         |
| Agent error            | `AuditAgentError`         | `internal/security/audit.go:20`         |
| File read              | `AuditFileRead`           | `internal/security/audit.go:21`         |
| File write             | `AuditFileWrite`          | `internal/security/audit.go:22`         |
| File delete            | `AuditFileDelete`         | `internal/security/audit.go:23`         |
| Git commit             | `AuditGitCommit`          | `internal/security/audit.go:24`         |
| Git push               | `AuditGitPush`            | `internal/security/audit.go:25`         |
| Generic git operation  | `AuditGitOperation`       | `internal/security/audit.go:26`         |
| Security check (pass)  | `AuditSecurityCheck`      | `internal/security/audit.go:27`         |
| Security check (deny)  | `AuditSecurityDenied`     | `internal/security/audit.go:28`         |
| Config change          | `AuditConfigChange`       | `internal/security/audit.go:29`         |
| Budget check           | `AuditBudgetCheck`        | `internal/security/audit.go:30`         |

### 6.3 Helper APIs

- `LogOperation(op)` — generic mapping from `Operation` → audit event
  (`internal/security/audit.go:143-154`).
- `LogAgentStart` / `LogAgentComplete` / `LogAgentError`
  (`internal/security/audit.go:175-209`).
- `LogFileModification`, `LogGitOperation`, `LogSecurityCheck`
  (`internal/security/audit.go:212-248`).
- Read-back: `ReadEvents(path)` plus `GetLogFiles()`
  (`internal/security/audit.go:286-327`).

---

## 7. Identified Gaps / Hardening Recommendations

### 7.1 `security.Manager` is constructed nowhere outside tests

The only non-test reference to `internal/security` from the rest of the
codebase is `cmd/nightshift/commands/task.go:16`, and even there it is used
solely to call `security.ValidateProjectPath`. Nothing constructs a
`security.Manager`, calls `CheckPreExecution`, or routes file/git operations
through the gates defined in §1–§2.

Verification:

```text
$ rg -n "security\.NewManager|CheckPreExecution|ValidateWriteAccess|ValidateGitPush" \
      --glob '!*_test.go'
internal/security/security.go: (definitions only — no callers)
```

Implication: `--enable-writes` / `--allow-push` / first-run/read-only modes are
**advertised in error messages but not enforced anywhere**. The orchestrator
runs the agent CLI directly (which then performs whatever filesystem and git
operations the underlying CLI is configured to allow), without nightshift's
own gates being consulted.

Recommended actions:

1. Wire flag plumbing in `cmd/nightshift/commands/root.go` (or each command
   that risks writes) for `--enable-writes`, `--allow-push`,
   `--max-budget-percent`, `--read-only` and feed them into a
   `security.Config`.
2. Construct `security.Manager` once during command init and pass it to the
   orchestrator.
3. Have the orchestrator call `manager.CheckPreExecution` around (a) agent
   invocation, (b) any commit step (currently a no-op at
   `internal/orchestrator/orchestrator.go:663-672`), and (c) push paths once
   they exist.

### 7.2 Audit log is never written by production code paths

Because no one constructs a `Manager`, no audit events are produced at runtime.
Helpers like `LogAgentStart` / `LogGitOperation` exist but have no callers
outside tests. Tie audit calls into the orchestrator alongside §7.1.

### 7.3 `AllowNetworkAgent` and `OpNetworkCall` are defined but unused

`Config.AllowNetworkAgent` (`internal/security/security.go:30`) and
`OpNetworkCall` (`internal/security/security.go:249`) have no enforcement —
`CheckPreExecution` does not branch on network operations, and the sandbox
only sets a `NIGHTSHIFT_NO_NETWORK=1` env hint
(`internal/security/sandbox.go:248-251`). Either remove these surfaces or add a
real gate (and ideally OS-level isolation) before relying on them.

### 7.4 CodexAgent constructor still defaults dangerous-bypass to true

`internal/agents/codex.go:55-66` initialises `bypassPerm: true`. The CLI layer
overrides this safely via `helpers.go:48-69`, but any future consumer that
calls `agents.NewCodexAgent()` directly inherits the unsafe default. Consider
flipping the constructor default to `false` and forcing callers to opt in via
`WithDangerouslyBypassApprovalsAndSandbox(true)`.

### 7.5 SMTP credentials are read directly from env at send time

`internal/reporting/summary.go:314-322` reads `NIGHTSHIFT_SMTP_PASS` from the
process environment with no integration into `CredentialManager`, no masking,
and no audit event when the credential is used. If SMTP becomes a documented
feature, surface it via `ValidateAll()` and audit `email_send` events.

### 7.6 Slack webhook URL is treated as plain config, not a credential

`reporting.slack_webhook` (`internal/config/config.go:142`) is a webhook URL
embedded in YAML. `CredentialManager.CheckConfigForCredentials` would not flag
it (no `token:` / `secret:` substring). At minimum, redact it from any
config-printing command output and prefer an env-only override.

### 7.7 GitHub / td integrations rely entirely on the upstream CLI's auth

`gh` and `td` token storage are out of nightshift's control
(`internal/integrations/github.go:48-50`, `internal/integrations/td.go:45-50`).
Document this explicitly so users know that revoking nightshift's access means
running `gh auth logout` / `td logout`, not editing nightshift config.

### 7.8 Generic viper `AutomaticEnv` accepts any `NIGHTSHIFT_*` key

`internal/config/config.go:293-296` enables `AutomaticEnv()`, which means any
env variable with the `NIGHTSHIFT_` prefix can override any config field. This
is convenient but worth listing in user-facing docs so operators know not to
inject untrusted env vars (e.g. cron environments) without review.

### 7.9 First-run sentinel dir uses `0755` while audit dir uses `0700`

`internal/security/security.go:88-98` creates the sentinel directory as `0755`,
while the audit dir alongside it is `0700`. The sentinel itself is empty, but
align the parent dir mode for consistency.

### 7.10 Project-path blocklist is exact-match only

`ValidateProjectPath` (`internal/security/security.go:212-238`) rejects only
exact equality with the blocklist. A project rooted at `/etc/foo` or
`$HOME/code` is still allowed. This is reasonable today (most users *do* keep
projects under `$HOME`), but an opt-in stricter mode that rejects
`HasPrefix($HOME)` outside an allowlist would harden the
`--dangerously-skip-permissions` story.

---

## Appendix A — Where to look next

- `SECURITY_AUDIT.md` (root) — Feb 2026 audit; several findings here are
  already addressed (default flags flipped, db dir mode tightened).
- `docs/deprecated/user-guide.md:600+` — historical user-facing safety doc that
  references `--enable-writes` (relevant to Gap §7.1).
- `website/docs/configuration.md` — public-docs equivalent.
- `internal/security/security_test.go` — exercises the gates that production
  code does not yet wire up.
