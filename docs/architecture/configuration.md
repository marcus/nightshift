# Configuration Overview

Nightshift is highly configurable to suit different workflows, budgets, and project structures. Configuration can be managed globally or per-project using YAML files.

## Configuration Files

- **Global Configuration**: Typically located at `~/.config/nightshift/config.yaml`.
- **Per-Project Configuration**: A `nightshift.yaml` or `.nightshift.yaml` file in the root of a repository.

## Configuration Structure

The configuration is divided into several logical sections:

### 1. Scheduling (`schedule`)
Controls when Nightshift runs and its execution constraints.

- **`cron`**: Standard cron expression (e.g., `"0 2 * * *"` for 2 AM daily).
- **`interval`**: Alternative to cron; a duration string (e.g., `"1h"`, `"8h"`).
- **`window`**: Optional time window constraint to ensure runs only happen during "nightshift" hours.
    - **`start`**: Start time (e.g., `"22:00"`).
    - **`end`**: End time (e.g., `"06:00"`).
    - **`timezone`**: Timezone for the window (e.g., `"America/Denver"`).

### 2. Budget Management (`budget`)
Defines how token resources are allocated and protected.

- **`mode`**: `daily` or `weekly`.
- **`max_percent`**: Maximum percentage of the budget to use in a single run (default: `75`).
- **`reserve_percent`**: Percentage of the budget to always keep in reserve for daytime use (default: `5`).
- **`aggressive_end_of_week`**: If true, increases budget usage in the last 2 days of the cycle (weekly mode only).
- **`weekly_tokens`**: Fallback weekly budget if auto-calibration is disabled.
- **`billing_mode`**: `subscription` or `api`.
- **`calibrate_enabled`**: Enable automatic budget calibration based on provider usage (default: `true`).
- **`snapshot_interval`**: How often to take usage snapshots (e.g., `"30m"`).
- **`snapshot_retention_days`**: How many days to keep snapshots.
- **`week_start_day`**: `monday` or `sunday`.
- **`db_path`**: Override the default SQLite database path.

### 3. AI Providers (`providers`)
Configures the LLM agents used for task execution.

- **`preference`**: Ordered list of providers to use (e.g., `["claude", "codex"]`).
- **`claude` / `codex`**:
    - **`enabled`**: Whether the provider is active.
    - **`data_path`**: Path to the provider's data directory.
    - **`dangerously_skip_permissions`**: Skip interactive permission prompts.
    - **`dangerously_bypass_approvals_and_sandbox`**: Bypass human-in-the-loop approvals and sandboxing (use with extreme caution).

### 4. Project Management (`projects`)
Defines the repositories Nightshift should monitor.

- **`path`**: Absolute path to a local repository.
- **`pattern`**: Glob pattern for discovering multiple repositories (e.g., `"~/code/work/*"`).
- **`exclude`**: List of paths to exclude from discovery.
- **`priority`**: Higher values indicate higher priority for budget allocation.
- **`tasks`**: List of task types to enable specifically for this project.
- **`config`**: Path to a specific configuration file for the project.

### 5. Task Selection (`tasks`)
Configures which maintenance tasks are performed.

- **`enabled`**: List of task types to enable globally.
- **`disabled`**: List of task types to explicitly disable.
- **`priorities`**: Map of task types to priority levels.
- **`intervals`**: Map of task types to minimum cooldown durations (e.g., `{"lint-fix": "24h"}`).
- **`custom`**: Define user-specific tasks.
    - **`type`**: Unique identifier for the task.
    - **`name`**: Human-readable name.
    - **`description`**: The prompt provided to the agent.
    - **`category`**: `pr`, `analysis`, `options`, `safe`, `map`, or `emergency`.
    - **`cost_tier`**: `low`, `medium`, `high`, or `very-high`.
    - **`risk_level`**: `low`, `medium`, or `high`.
    - **`interval`**: Minimum cooldown for this custom task.

### 6. Integrations (`integrations`)
External tools and data sources.

- **`claude_md`**: If true, reads `claude.md` for context.
- **`agents_md`**: If true, reads `agents.md` for context.
- **`task_sources`**:
    - **`github_issues`**: Fetch tasks from GitHub issues labeled `nightshift`.
    - **`file`**: Load tasks from a specific local file.
    - **`td`**: Integrate with the `td` task management tool.

### 7. Logging & Reporting (`logging`, `reporting`)
- **`logging.level`**: `debug`, `info`, `warn`, `error`.
- **`logging.format`**: `text` or `json`.
- **`reporting.morning_summary`**: If true, generates a summary of the night's activities.
- **`reporting.email`**: Optional email for notifications.
- **`reporting.slack_webhook`**: Optional Slack webhook for notifications.
