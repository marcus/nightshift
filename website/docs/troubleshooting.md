---
sidebar_position: 10
title: Troubleshooting
---

# Troubleshooting

## Common Issues

**"Something feels off"**
- Run `nightshift doctor` to check config, schedule, and provider health

**"No config file found"**
```bash
nightshift init           # Create project config
nightshift init --global  # Create global config
```

**"Insufficient budget"**
- Check current budget: `nightshift budget`
- Increase `max_percent` in config
- Wait for budget reset (check reset time in output)

**"Calibration confidence is low"**
- Run `nightshift budget snapshot` a few times to collect samples
- Ensure tmux is installed so usage percentages are available
- Keep snapshots running for at least a few days

**"tmux not found"**
- Install tmux or set `budget.billing_mode: api` if you pay per token

**"Week boundary looks wrong"**
- Set `budget.week_start_day` to `monday` or `sunday`

**"Provider not available"**
- Run `nightshift doctor` to check CLI discovery and provider data paths
- Ensure `claude`, `codex`, or `copilot`/`gh` is installed and in PATH
- Remember that `nightshift task run ... --dry-run` previews prompts only; it does not test provider auth

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
