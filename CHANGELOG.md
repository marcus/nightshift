# Changelog

All notable changes to nightshift are documented in this file.

## [Unreleased]

### Fixes
- **Run project limits** — apply `--max-projects` after filtering projects already processed today, so skipped projects no longer consume the active project limit (#45)
- **Mobile docs navigation** — keep the mobile sidebar visible when the docs site shows the secondary panel (#46)

### Other
- **Development hooks** — add `make install-hooks` and a pre-commit hook for gofmt, go vet, and go build checks (#44)

## [v0.3.4] - 2026-02-28

### Features
- **Configurable agent timeouts** — add `--timeout` to `nightshift run` and `nightshift daemon`, with daemon re-exec forwarding the flag (#27)
- **Expanded PII scanner guidance** — add detailed instructions for detecting hardcoded PII, leaked env files, unsafe storage, and related exposure patterns in the built-in task (#34)

### Fixes
- **Timeout handling and diagnostics** — preserve partial output on timeout, terminate full process groups, and surface partial logs from failed plan/implement/review steps (#33)
- **Copilot CLI integration** — improve binary resolution, permission gating, and CLI flag handling for Copilot runs (#39)
- **Provider config YAML serialization** — write provider settings with the correct YAML keys during setup (#43)
- **Configured run limits and budget fallback** — honor `schedule.max_projects` and `schedule.max_tasks`, improve budget calibration at day and week boundaries, and preserve Codex fallback permissions in headless runs (#42)

### Other
- **Task reference docs** — add a comprehensive reference page for all 59 built-in tasks and refresh related task documentation (#30)
- **Docs cleanup** — remove auto-generated implementation docs from the repository (#40)
- **Low-risk cleanup** — resolve Copilot helper lint warnings and replace `WriteString(fmt.Sprintf(...))` patterns with `fmt.Fprintf` in reporting and setup code (#38, #41)

## [v0.3.3] - 2026-02-19

### Features
- **Branch selection support** — select which branch to run tasks against (#12, thanks @andrew-t-james-wc)

### Fixes
- **gofmt formatting** — fix gofmt formatting across multiple files (#17, thanks @cedricfarinazzo)

## [v0.3.2] - 2026-02-17

### Bug Fixes
- **Block task run in sensitive directories** — refuse to run when project path is `$HOME`, `/`, `/tmp`, `/var`, `/etc`, or `/usr` to prevent accidental credential exposure (#14, thanks @davemac)
- **Fix codex exec for non-interactive runs** — switch from removed `--quiet` flag to `exec` subcommand for Codex 0.98.0 compatibility (#11, thanks @brandon93s)

### Other
- Bus-factor analyzer for code ownership concentration
- Security audit improvements and linter fixes
- Extended test coverage for snapshots, budget, setup, and backward compatibility

## [v0.3.1] - 2026-02-08

### Security

#### Breaking Changes (Opt-In Required for Old Behavior)
- **Default behavior change:** `dangerously_skip_permissions` and `dangerously_bypass_approvals_and_sandbox` now default to `false` (secure)
  - In v0.3.0, these defaulted to `true`, which skipped security prompts
  - Users upgrading from v0.3.0 **who run unattended** (daemon, cron, CI) must explicitly set these flags to `true` in config, or use `--yes` flag
  - Users running **interactively** will now see security prompts (recommended)
  - See [Migration Guide](docs/MIGRATION-v0.3.0-to-v0.3.1.md) for details
- **Database directory permissions:** changed from `0755` to `0700`
  - Existing databases continue to work (no action required)
  - New databases now restrict access to owner only (security improvement)

#### Non-Breaking Improvements
- Shell path escaping improved in setup wizard
- Better security defaults for new installations

### Backward Compatibility
- All v0.3.0 configuration files load correctly in v0.3.1
- Configuration defaults (except dangerous flags) remain unchanged
- Existing databases work without migration
- Environment variable overrides still work
- CLI interface stable for scripts and automation
- Full backward compatibility testing added

### Improvements
- Homebrew formula now builds from source (avoids macOS Gatekeeper warnings)

## [v0.3.0] - 2026-02-01

### Features
- Initial public release
- Daemon mode with launchd/systemd integration
- Support for Claude Code and Codex CLI agents
- Budget-aware task selection
- Project and task configuration via YAML
- Doctor command for setup validation
- Comprehensive logging and reporting
