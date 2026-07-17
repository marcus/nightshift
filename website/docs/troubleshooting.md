---
sidebar_position: 10
title: Troubleshooting
---

# Troubleshooting

## Common Issues

**"Something feels off"**
- Run `nightshift doctor` to check config, schedule, provider, and budget health

**"No config file found"**
```bash
nightshift setup          # Guided bootstrap with provider and daemon checks
nightshift init           # Create nightshift.yaml in the current directory
nightshift init --global  # Create ~/.config/nightshift/config.yaml
nightshift config         # Show source paths and merged defaults
nightshift config validate
```

Nightshift can load built-in defaults without either file, so `config` or a manual `run` may still work. A project config is discovered only in the current directory, or in the directory passed with a command's `--project` flag.

**"No schedule configured"**
- Set either `schedule.cron` or `schedule.interval` in config
- Use `nightshift setup` or `nightshift init` if you want the bootstrap flow
- Re-run `nightshift config validate` and `nightshift preview` after editing the schedule

`config validate` checks that cron and interval are not both set, but it currently permits both to be absent. `preview` and `daemon start` perform the missing-schedule check. A manual `nightshift run` intentionally does not require a schedule.

**"Insufficient budget"**
- Check current budget: `nightshift budget`
- Inspect the budget source, reserve, and daytime forecast in the output
- Increase `budget.max_percent`, reduce `reserve_percent`, or correct `weekly_tokens` / `per_provider` if those values are intentional
- Wait for budget reset (check the reset time in the output)
- For a deliberate one-off override, use `nightshift run --ignore-budget`; the daemon has no equivalent override
- A direct `nightshift task run TASK --provider PROVIDER` is not budget-gated

Setting `max_percent: 0` does not disable spending; calculation treats zero as the default 75. Disable providers or remove eligible work if you need a hard stop.

**"Calibration confidence is low"**
- Run `nightshift budget snapshot` several times across the week
- Ensure `tmux` is installed so usage percentages are available
- Confirm the provider CLI is authenticated and its `/usage` or `/status` display is available
- Use `nightshift budget history` to confirm snapshots have both local usage and a scraped percentage

Only samples with local usage greater than zero and percentages from 10% through 95% are calibrated. One or two usable samples always produce low confidence. If the current week is empty, Nightshift can reuse the previous week's samples.

**"tmux not found"**
- Install `tmux`, then retry `nightshift budget snapshot`
- Use `--local-only` when you want history without calibration
- Set `budget.billing_mode: api` and explicit token limits when you pay per token

Missing tmux does not prevent task execution. It prevents Claude/Codex percentage scraping, so subscription calibration falls back to config until usable scraped snapshots exist.

**"Week boundary looks wrong"**
- Set `budget.week_start_day` to `monday` or `sunday`

That setting changes snapshot/calibration grouping. Weekly allowance reset timing remains provider-specific: Claude assumes Sunday, Codex uses its reported secondary reset when available, and Copilot uses the next monthly reset.

**"Provider not available"**
- Run `nightshift doctor` and `command -v claude codex copilot gh`
- Verify the provider's `enabled` key and its position in `providers.preference`
- Authenticate the selected CLI and verify it directly before retrying Nightshift
- `run` and `daemon` augment `PATH` with common user locations such as `~/.local/bin`, `~/go/bin`, `~/.cargo/bin`, `~/.npm-global/bin`, `/usr/local/bin`, and `/opt/homebrew/bin`; custom install directories still need to be exported
- `task run` performs direct PATH discovery and does not add those common directories first

**"Copilot is installed but Nightshift cannot find it"**

```bash
command -v copilot
copilot login
copilot --version
```

Use the standalone `copilot` executable when possible. Nightshift falls back to `gh`, but this release's availability probe requires `gh extension list` to contain `gh-copilot`. GitHub retired that legacy extension, so a current `gh copilot` wrapper may still be rejected by the probe. If standalone authentication fails, run `copilot login` (or `/login` inside Copilot), confirm the account has Copilot CLI access, and check organization policy.

**"Installed service runs differently from preview"**
- launchd, systemd, and cron installations execute `nightshift run` directly, not the persistent daemon
- Service generators only translate a subset of schedule settings and do not enforce `schedule.window`
- Use `nightshift daemon start --foreground` to debug full scheduler/window behavior
- See [Scheduling](/docs/scheduling) for the exact service conversions

## Debug Mode

Enable verbose logging:

```bash
nightshift run --verbose
```

Or set log level in config:

```yaml
logging:
  level: debug    # debug | info | warn | error
```

## Getting Help

```bash
nightshift --help
nightshift <command> --help
```

Report issues: https://github.com/marcus/nightshift/issues
