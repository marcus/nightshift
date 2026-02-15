# Security & Safety

Security is a core pillar of Nightshift, designed to protect the user's codebase and manage the risks associated with autonomous AI operations.

## Safety Modes
Nightshift operates in two primary modes to ensure the user is always in control:

- **ReadOnly Mode**: The default mode for new installations or when unconfigured. It allows Nightshift to analyze projects and generate reports but prevents any file modifications, branch creation, or remote pushes.
- **Normal Mode**: Enables full maintenance capabilities. This mode must be explicitly enabled and is typically used once the user has verified Nightshift's behavior in their environment.

## Credential Management
Tokens and API keys for AI providers (e.g., Anthropic, OpenAI) are handled with high security:
- **Environment Isolation**: Prefer loading secrets from environment variables or secure local stores.
- **Validation**: Every provider connection is validated for connectivity and remaining budget before tasks are started.
- **No Persistence**: Nightshift does *not* store raw API keys in its SQLite database.

## Execution Sandboxing
To prevent accidental side effects during maintenance:
- **Branch-Based Isolation**: All AI-driven changes are performed on dedicated feature branches (e.g., `nightshift/fix-bus-factor`).
- **Local-Only by Default**: Nightshift will not push branches to remote repositories unless the `AllowGitPush` security flag is explicitly set.
- **Pull Request Workflow**: Changes are submitted as Pull Requests using the GitHub CLI (`gh`), requiring human review and approval before merging into the primary branch.

## Audit Logging
Every significant action taken by Nightshift is recorded in an audit log (located in `~/.local/share/nightshift/audit/`):
- **Task Assignments**: Who assigned what task to which project.
- **AI Interactions**: Logs of prompts and responses (redacted where necessary).
- **Filesystem Changes**: Records of which files were modified.
- **Budget Consumption**: Real-time tracking of token usage against daily/weekly limits.

## Budget Safeguards
Integrated budget management prevents runaway costs:
- **Hard Limits**: Daily and weekly token/dollar limits.
- **Auto-Calibration**: Providers can be automatically disabled if they exceed their allocated share of the total budget.
- **Cost Estimation**: Where possible, Nightshift estimates task costs before execution.
