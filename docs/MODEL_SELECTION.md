# Model Selection

Nightshift allows you to configure which AI model each provider uses. This gives you control over cost, quality, and performance tradeoffs.

## Configuration

Add model selection to your `~/.config/nightshift/config.yaml`:

```yaml
providers:
  claude:
    enabled: true
    model: "claude-sonnet-4.5"  # or "opus", "sonnet", "haiku"
    
  codex:
    enabled: true
    model: "gpt-5.2-codex"      # or any supported GPT model
    
  copilot:
    enabled: true
    model: "claude-sonnet-4.6"  # Copilot supports multiple models
```

## Supported Models

### Claude Code

**Model aliases:**
- `sonnet` - Latest Sonnet (recommended for most tasks)
- `opus` - Highest quality, slower
- `haiku` - Fastest, most cost-effective

**Full model names:**
- `claude-sonnet-4-6`
- `claude-sonnet-4-5`
- `claude-opus-4-6`
- `claude-opus-4-6-fast`
- `claude-opus-4-5`
- `claude-sonnet-4`
- `claude-haiku-4-5`

### Codex

**Supported models:**
- `gpt-5.3-codex`
- `gpt-5.2-codex`
- `gpt-5.2`
- `gpt-5.1-codex-max`
- `gpt-5.1-codex`
- `gpt-5.1`
- `gpt-5`
- `gpt-5.1-codex-mini` (budget option)
- `gpt-5-mini` (fastest)
- `gpt-4.1`

### GitHub Copilot

**Supported models:**
- `claude-sonnet-4.6` (recommended)
- `claude-sonnet-4.5`
- `claude-haiku-4.5`
- `claude-opus-4.6`
- `claude-opus-4.6-fast`
- `claude-opus-4.5`
- `claude-sonnet-4`
- `gemini-3-pro-preview`
- `gpt-5.3-codex`
- `gpt-5.2-codex`
- `gpt-5.2`
- `gpt-5.1-codex-max`
- `gpt-5.1-codex`
- `gpt-5.1`
- `gpt-5`
- `gpt-5.1-codex-mini`
- `gpt-5-mini`
- `gpt-4.1`

**Note:** GitHub Copilot supports multiple model providers through its unified API.

## Model Selection Strategy

### Default (No Model Specified)

If you don't specify a model, each provider uses its CLI default:
- **Claude**: Latest stable model
- **Codex**: Latest GPT model
- **Copilot**: GitHub's recommended model

### Recommended Configurations

**Balanced (Quality + Cost):**
```yaml
providers:
  claude:
    model: "sonnet"  # Good balance
  codex:
    model: "gpt-5.2-codex"
  copilot:
    model: "claude-sonnet-4.6"
```

**Budget-Conscious:**
```yaml
providers:
  claude:
    model: "haiku"  # Fastest, cheapest
  codex:
    model: "gpt-5-mini"
  copilot:
    model: "claude-haiku-4.5"
```

**Maximum Quality:**
```yaml
providers:
  claude:
    model: "opus"  # Highest quality
  codex:
    model: "gpt-5.3-codex"
  copilot:
    model: "claude-opus-4.6"
```

## Task-Specific Models

Different task categories may benefit from different models:

**Code Changes (CategoryPR):**
- High-quality models recommended (`opus`, `gpt-5.2-codex`)
- Medium risk requires careful review

**Analysis (CategoryAnalysis):**
- Medium models sufficient (`sonnet`, `gpt-5.1-codex`)
- Focus on speed and cost

**Low-Risk Tasks:**
- Budget models work well (`haiku`, `gpt-5-mini`)
- Lint fixes, documentation updates

## Verification

Check which model is configured:

```bash
# View current config
nightshift config show

# Doctor check shows provider status
nightshift doctor
```

## Model Availability

Models may have different:
- **Rate limits** - Higher-tier models may have stricter limits
- **Latency** - Larger models take longer to respond
- **Cost** - Premium models cost more per token
- **Context windows** - Some models support longer contexts

## Troubleshooting

**"Unknown model" error:**
- Check model name spelling
- Verify model is available in your region
- Try using a model alias instead of full name

**Model not respected:**
- Ensure you've run `nightshift setup` after config changes
- Some CLIs may not support all models
- Check CLI version: `claude --version`, `codex --version`, `copilot --version`

**Rate limiting:**
- Switch to a lower-tier model
- Reduce `budget.max_percent` in config
- Use `--ignore-budget` flag for testing

## Examples

### Per-Provider Configuration

```yaml
providers:
  claude:
    enabled: true
    model: "opus"
    dangerously_skip_permissions: true
    
  codex:
    enabled: true
    model: "gpt-5.2-codex"
    dangerously_bypass_approvals_and_sandbox: true
    
  copilot:
    enabled: true
    model: "claude-sonnet-4.6"
    dangerously_skip_permissions: true
```

### Testing Different Models

```bash
# Test with specific provider
nightshift run --provider claude

# Preview what would run
nightshift preview

# Run specific task
nightshift task --task lint-fix --provider codex
```

## Best Practices

1. **Start with defaults** - Let providers choose optimal models initially
2. **Monitor costs** - Check `nightshift budget` regularly
3. **Match model to task** - Use budget models for low-risk tasks
4. **Test before scheduling** - Run manually before enabling daemon
5. **Keep updated** - New models are released regularly

## Related Documentation

- [Configuration Guide](https://nightshift.haplab.com/docs/configuration)
- [Budget Management](https://nightshift.haplab.com/docs/budget)
- [Provider Setup](../README.md#authentication-subscriptions)
