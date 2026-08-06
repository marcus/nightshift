---
sidebar_position: 10
title: Troubleshooting
---

# Troubleshooting

## Common Issues

**"Something feels off"**
- Run `nightshift doctor` to check config, schedule, and provider health
- Run `nightshift config validate` after editing YAML by hand

**"No config file found"**
```bash
nightshift init           # Create project config in the current directory
nightshift init --global  # Create the global config at ~/.config/nightshift/config.yaml
```
- If you want the guided version, run `nightshift setup` instead. It creates the global config, checks providers, and can install the daemon service.

**"Insufficient budget"**
- Check current budget: `nightshift budget`
- Increase `max_percent` in config
- Wait for the next configured reset window (check reset time in the output)

**"Calibration confidence is low"**
- Run `nightshift budget snapshot` a few times to collect samples
- Ensure tmux is installed so usage percentages are available
- Keep snapshots running for at least a few days

**"tmux not found"**
- Install tmux, or set `budget.billing_mode: api` if you pay per token and want to skip local usage calibration

**"Week boundary looks wrong"**
- Set `budget.week_start_day` to `monday` or `sunday` and verify your `budget.mode`

**"Provider not available"**
- Ensure the provider CLI is installed and in PATH:
  - Claude: `claude`
  - Codex: `codex`
  - Copilot: `gh` with the Copilot extension, or the standalone `copilot` binary
- Check `providers.preference` and each provider's `enabled` flag in config

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
