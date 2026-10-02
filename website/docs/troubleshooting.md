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
nightshift config validate
```

**"No schedule configured"**
- Set either `schedule.cron` or `schedule.interval` in config
- Use `nightshift setup` or `nightshift init` if you want the bootstrap flow
- Re-run `nightshift config validate` after editing the schedule

**"Insufficient budget"**
- Check current budget: `nightshift budget`
- Increase `budget.max_percent` in config
- Wait for budget reset (check the reset time in the output)

**"Calibration confidence is low"**
- Run `nightshift budget snapshot` a few times to collect samples
- Ensure `tmux` is installed so usage percentages are available
- Keep snapshots running for at least a few days

**"tmux not found"**
- Install `tmux` or set `budget.billing_mode: api` if you pay per token

**"Week boundary looks wrong"**
- Set `budget.week_start_day` to `monday` or `sunday`

**"Provider not available"**
- Ensure Claude Code, Codex, or Copilot is installed and in `PATH`
- For Copilot, install either `gh` or the standalone `copilot` binary, then run `gh auth login` if you use `gh copilot`
- Check API key or subscription login state for the provider you are using

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
