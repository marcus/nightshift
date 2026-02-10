# Migration Guide: v0.3.0 to v0.3.1

## Overview

v0.3.1 includes important security fixes that may require action if you're upgrading from v0.3.0. This guide explains what changed and what you need to do.

## Breaking Changes

### 1. Security Prompts Now Enabled by Default

**What changed:** In v0.3.0, the two dangerous flags defaulted to `true`:
- `dangerously_skip_permissions` (allowed interactive prompts to be skipped)
- `dangerously_bypass_approvals_and_sandbox` (allowed security checks to be bypassed)

In v0.3.1, both flags now default to `false` for security. This means:
- **Interactive permission prompts will appear** even if you didn't explicitly configure them
- **Approval prompts will appear** before executing sensitive operations
- **Sandboxing will be enforced** by default

**Why:** This makes security the default behavior, reducing the risk of unintended privilege escalation.

**What to do:**
- If you want the **old behavior** (skip prompts, bypass approvals), explicitly set these flags to `true` in your config:

```yaml
providers:
  claude:
    dangerously_skip_permissions: true
  codex:
    dangerously_bypass_approvals_and_sandbox: true
```

- If you're running Nightshift **unattended** (daemon, cron, CI), you may need to set these flags to `true` so the process doesn't hang waiting for input.
- If you're running Nightshift **interactively**, the default (`false`) is recommended—you'll see prompts, which is the safe default.

### 2. Database Directory Permissions Changed (0755 → 0700)

**What changed:** In v0.3.0, the database directory was created with `0755` (world-readable).

In v0.3.1, it's created with `0700` (owner-only access) for security.

**Why:** Database files contain sensitive runtime state and should not be readable by other users.

**What to do:**
- **No action required.** Existing databases will continue to work in v0.3.1.
- **Optional:** If you want to adopt the stricter permissions on an existing database:
  ```bash
  chmod 0700 ~/.local/share/nightshift
  ```

### 3. Shell Path Escaping Improved

**What changed:** v0.3.1 properly escapes directory paths when adding them to shell config files (`.bashrc`, `.zshrc`, etc.).

**Why:** This prevents shell injection if a path contains special characters or spaces.

**What to do:**
- **No action required.** This change is internal and backward-compatible.
- If you manually added paths to your shell config and used simple paths (no spaces, no special characters), it will continue to work.

## Configuration Compatibility

### Old Config Files Still Load

If you have a `~/.config/nightshift/config.yaml` from v0.3.0:
- **It will still load correctly** in v0.3.1
- Default values haven't changed except for the dangerous flags
- All existing settings are preserved

Example old config:
```yaml
budget:
  mode: daily
  max_percent: 75
logging:
  level: info
```

This loads fine in v0.3.1. The dangerous flags will default to `false` (safe).

### New Dangerous Flag Handling

When v0.3.1 loads an old config that doesn't mention the dangerous flags:
- `dangerously_skip_permissions` defaults to `false` (now requires explicit opt-in)
- `dangerously_bypass_approvals_and_sandbox` defaults to `false` (now requires explicit opt-in)

When v0.3.1 loads an old config that **does** mention these flags:
- Your explicit setting is preserved (e.g., if you set them to `true`, they stay `true`)

## Database Migration

### Schema Changes

v0.3.1 adds no new migrations beyond v0.3.0. All existing databases continue to work.

Current schema includes:
- Migration 1: initial schema (projects, task_history, assigned_tasks, run_history, snapshots)
- Migration 2: added session_reset_time and weekly_reset_time to snapshots
- Migration 3: added provider column to run_history

**No action needed.** Migrations are applied automatically on first run.

### Existing Databases

If you're upgrading from v0.3.0:
1. Stop any running Nightshift processes
2. Upgrade the binary
3. Run any command (e.g., `nightshift status`) to trigger database initialization
4. The database will be opened, permissions applied, and you're done

## Testing Your Upgrade

After upgrading, verify backward compatibility:

```bash
# Test that config still loads
nightshift status

# Test that database still works
nightshift run --dry-run

# If running unattended (daemon/cron), test with explicit flags:
nightshift run --yes --ignore-budget
```

## Unattended Execution (Daemon/Cron)

If you run Nightshift unattended (via daemon, cron, or CI):

**Before:** Dangerous flags defaulted to `true`, so no prompts appeared.

**After:** Dangerous flags default to `false`, which means interactive prompts **will** appear and cause hangs.

**Fix:** Explicitly opt-in to skipping prompts:

```yaml
providers:
  claude:
    dangerously_skip_permissions: true
  codex:
    dangerously_bypass_approvals_and_sandbox: true
```

Or use CLI flags:
```bash
nightshift run --yes
```

## Rollback

If you need to rollback to v0.3.0:
1. Revert to the v0.3.0 binary
2. Your config and database are compatible—no changes needed
3. Dangerous flags will be interpreted with v0.3.0 defaults (true)

## Summary of Changes

| Aspect | v0.3.0 | v0.3.1 | Impact |
|--------|--------|--------|--------|
| `dangerously_skip_permissions` default | `true` | `false` | Permission prompts now appear by default |
| `dangerously_bypass_approvals_and_sandbox` default | `true` | `false` | Approval prompts now appear by default |
| DB directory permissions | `0755` (world-readable) | `0700` (owner-only) | Stricter security (old DBs still work) |
| Shell path escaping | Basic | Proper escaping | No functional change for valid paths |

## Questions?

If your config or database doesn't work after upgrading, double-check:
1. Are you running an unattended process? If so, set dangerous flags to `true` or use `--yes`.
2. Did you manually edit shell config files? Verify the PATH entries are still valid.
3. Run `nightshift status` to check for any errors in loading config or database.
