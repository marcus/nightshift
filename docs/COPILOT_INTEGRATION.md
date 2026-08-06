# GitHub Copilot Integration

Nightshift can run tasks with GitHub Copilot through either the GitHub CLI
Copilot extension or the standalone `copilot` binary.

## Requirements

- A GitHub account with a Copilot subscription.
- Either `gh` with the `github/gh-copilot` extension, or the standalone
  `copilot` command.
- A working login for the CLI you choose.

Install with the GitHub CLI:

```bash
gh extension install github/gh-copilot
gh auth login
```

Or install the standalone CLI:

```bash
npm install -g @github/copilot
# or
curl -fsSL https://gh.io/copilot-install | bash
```

Nightshift prefers the standalone `copilot` binary when it is available. If it
is not in `PATH`, it falls back to `gh copilot`.

## Configuration

Copilot uses the same provider configuration model as Claude Code and Codex:

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
default in config loading, but it is only selected when its CLI is available and
its budget allowance is positive.

`providers.copilot.data_path` is where Nightshift reads and writes its local
Copilot budget tracking file. The default is `~/.copilot`.

## Execution Model

Nightshift invokes Copilot in non-interactive prompt mode:

```bash
copilot -p "<prompt>" --no-ask-user --silent
```

When using `gh`, the same flags are passed through to the Copilot extension:

```bash
gh copilot -- -p "<prompt>" --no-ask-user --silent
```

The `--no-ask-user` flag disables Copilot's ask-user tool so scheduled runs do
not block waiting for input. The `--silent` flag keeps output focused on the
agent response.

If `providers.copilot.dangerously_skip_permissions: true`, Nightshift also adds:

```bash
--allow-all-tools --allow-all-urls
```

Leave this disabled unless the target repository and execution environment are
trusted. It allows Copilot to use tools and URLs without interactive approval.

## Budget Tracking

GitHub Copilot does not expose an authoritative local usage file or API endpoint
for Nightshift to query. Nightshift therefore models Copilot as a monthly,
request-based provider:

- each counted request is treated as one premium request;
- the local counter resets on the first day of each month at 00:00 UTC;
- the counter is stored at `~/.copilot/nightshift-usage.json`, or under the
  configured `providers.copilot.data_path`;
- `budget.weekly_tokens` or `budget.per_provider.copilot` is converted to an
  approximate monthly request limit by multiplying the configured value by four.

Current limitation: the provider includes `IncrementRequestCount`, but the
`nightshift run` and `nightshift task run` execution paths do not call it after
successful Copilot tasks. Budget displays and provider selection can read an
existing local counter, but Nightshift does not yet automatically count Copilot
executions it launches. External Copilot usage is also not visible to
Nightshift.

Because of those limits, Copilot budget enforcement is an estimate. Use a
conservative `budget.per_provider.copilot` value if you rely on Copilot monthly
request limits:

```yaml
budget:
  per_provider:
    copilot: 100
```

## Troubleshooting

Check that Nightshift can see the CLI:

```bash
nightshift doctor
```

If Copilot is skipped during provider selection:

- confirm `providers.copilot.enabled: true`;
- confirm `providers.preference` includes `copilot` if you customized the list;
- run `which copilot` or `which gh`;
- for `gh`, run `gh extension list` and confirm `github/gh-copilot` is present;
- run the Copilot CLI directly to confirm authentication.

If a scheduled run hangs or fails on permissions, either keep Copilot later in
`providers.preference` and use Claude/Codex for unattended runs, or explicitly
enable `providers.copilot.dangerously_skip_permissions` for trusted projects.

## Related Docs

- [Technical specification](SPEC.md)
- [Run lifecycle](guides/run-lifecycle.md)
- [Configuration docs](../website/docs/configuration.md)
- [Integrations docs](../website/docs/integrations.md)
