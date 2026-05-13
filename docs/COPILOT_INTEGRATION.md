# GitHub Copilot Integration

Nightshift can execute tasks with GitHub Copilot through the standalone
`copilot` command or through the GitHub CLI Copilot extension.

## Requirements

- A GitHub account with a Copilot subscription.
- Either the standalone `copilot` command or `gh` with the
  `github/gh-copilot` extension.
- An authenticated CLI session for the command Nightshift will use.

Install with the GitHub CLI:

```bash
gh extension install github/gh-copilot
gh auth login
```

Or install the standalone command:

```bash
npm install -g @github/copilot
# or
curl -fsSL https://gh.io/copilot-install | bash
```

For automatic provider selection, Nightshift prefers `copilot` when it is
available in `PATH`; otherwise it falls back to `gh copilot`.

## Configuration

Copilot uses the same provider configuration shape as Claude Code and Codex:

```yaml
providers:
  preference:
    - claude
    - codex
    - copilot
  copilot:
    enabled: true
    data_path: "~/.copilot"
    dangerously_skip_permissions: false
```

`providers.preference` controls provider selection order. Copilot is enabled by
default, but it is only selected when a usable CLI is present and budget checks
allow it.

`providers.copilot.data_path` is used for Nightshift's local Copilot usage
tracking file. The default is `~/.copilot`.

## Execution Model

Nightshift invokes Copilot in non-interactive prompt mode:

```bash
copilot -p "<prompt>" --no-ask-user --silent
```

When using `gh`, Nightshift passes the same prompt flags through the extension:

```bash
gh copilot -- -p "<prompt>" --no-ask-user --silent
```

`--no-ask-user` prevents scheduled runs from blocking on Copilot's ask-user
tool. `--silent` keeps output focused on the agent response.

If `providers.copilot.dangerously_skip_permissions` is `true`, Nightshift also
passes:

```bash
--allow-all-tools --allow-all-urls
```

Only enable that flag for trusted repositories and trusted execution
environments. It allows Copilot to use tools and URLs without interactive
approval.

## Budget Tracking

GitHub Copilot does not expose authoritative usage data to Nightshift through a
local file or public API. Nightshift therefore has only local, request-count
based support for Copilot budget checks:

- the Copilot provider can read `nightshift-usage.json` under
  `providers.copilot.data_path`;
- the local counter resets on the first day of each month at 00:00 UTC;
- each request in that local counter is treated as one premium request;
- daily mode estimates daily usage from the monthly counter divided by elapsed
  days in the current month;
- weekly mode compares the monthly counter directly with the supplied monthly
  request limit.

Important limitation: current Copilot task execution does not automatically
increment this counter. The `IncrementRequestCount` helper exists on the
provider, but `nightshift run` and `nightshift task run` execute Copilot through
the agent wrapper and do not call that helper after successful runs. External
Copilot usage is also invisible to Nightshift.

As a result, Copilot budget status is an estimate based on whatever local
counter data already exists. It should not be treated as an authoritative
GitHub Copilot quota reading.

The run budget manager uses `budget.per_provider.copilot`, or
`budget.weekly_tokens` if no per-provider value is set, as a configured
allowance input. Because the shared budget model is token-oriented, Copilot
handling is approximate. Use a conservative Copilot value if you rely on request
limits:

```yaml
budget:
  per_provider:
    copilot: 100
```

## Troubleshooting

Run:

```bash
nightshift doctor
```

If Copilot is skipped during provider selection:

- confirm `providers.copilot.enabled: true`;
- confirm `providers.preference` includes `copilot` if you customized it;
- run `which copilot` or `which gh`;
- when using `gh`, run `gh extension list` and confirm `github/gh-copilot` is
  installed;
- run the Copilot CLI directly to confirm authentication.

If a scheduled run blocks or fails on permissions, either keep Copilot later in
`providers.preference` and use another provider for unattended runs, or enable
`providers.copilot.dangerously_skip_permissions` only for trusted projects.

## Related Docs

- [Technical specification](SPEC.md)
- [Run lifecycle](guides/run-lifecycle.md)
- [Configuration docs](../website/docs/configuration.md)
- [Integrations docs](../website/docs/integrations.md)
