# Adding Tasks to Nightshift

## Built-in Tasks

Nightshift ships with 50+ built-in task types organized into six categories. To add a new built-in task, modify the task registry in `internal/tasks/tasks.go`.

### Step 1: Define the Task Type Constant

Add a `TaskType` constant in the appropriate category block:

```go
// Category 1: "It's done - here's the PR"
const (
    // ... existing constants ...
    TaskMyNewTask TaskType = "my-new-task"
)
```

### Step 2: Add the Registry Entry

Add a `TaskDefinition` to the `registry` map:

```go
TaskMyNewTask: {
    Type:            TaskMyNewTask,
    Category:        CategoryPR,       // See categories below
    Name:            "My New Task",
    Description:     "Short summary shown in task lists",
    AgentInstructions: `Optional richer instructions for the agent.
Use this when the CLI summary should stay concise but the prompt
needs more concrete scope or guardrails.`,
    CostTier:        CostMedium,       // Estimated token usage
    RiskLevel:       RiskLow,          // Risk of unintended changes
    DefaultInterval: 72 * time.Hour,   // Minimum time between runs per project
},
```

Use `Description` for the concise user-facing summary shown in task lists, previews, and JSON metadata. Add `AgentInstructions` when the agent needs richer directions; if it is omitted, Nightshift falls back to `Description`.

`commit-normalize` is the reference example for this split: it stays a short "Standardize commit message format" entry in the catalog, while the agent prompt makes the scope explicit:

- treat it as a PR task for future commit-message consistency
- inspect existing repo conventions before choosing a format
- avoid rewriting published history
- preserve Nightshift's required commit trailers
- prefer small, explainable enforcement or documentation changes

### Step 3: Update the Completeness Test

Add your constant to `TestRegistryCompleteness` in `internal/tasks/tasks_test.go`:

```go
taskTypes := []TaskType{
    // ... existing types ...
    TaskMyNewTask,
}
```

### Categories

| Category | Constant | Output Type |
|----------|----------|-------------|
| PR | `CategoryPR` | Review-ready PRs |
| Analysis | `CategoryAnalysis` | Reports and findings |
| Options | `CategoryOptions` | Decisions for the user |
| Safe | `CategorySafe` | Simulations with no side effects |
| Map | `CategoryMap` | Context and topology |
| Emergency | `CategoryEmergency` | Incident response artifacts |

### Cost Tiers

| Tier | Token Range | Use For |
|------|-------------|---------|
| `CostLow` | 10-50K | Simple lint, formatting |
| `CostMedium` | 50-150K | Analysis, small PRs |
| `CostHigh` | 150-500K | Multi-file changes, reviews |
| `CostVeryHigh` | 500K+ | Large refactors, simulations |

### DisabledByDefault

Set `DisabledByDefault: true` for tasks that require specific tooling or opt-in workflows (e.g., `td-review` requires the td CLI). These tasks won't run unless explicitly added to `tasks.enabled` in config.

---

## Custom Tasks

Define custom tasks in your nightshift config to run your own prompts on schedule.

### Config Format

```yaml
tasks:
  custom:
    - type: pr-review
      name: "PR Review Session"
      description: |
        Review all open PRs. Fix obvious issues immediately.
        Create tasks for bigger problems that need follow-up.
      category: pr
      cost_tier: high
      risk_level: medium
      interval: "72h"

    - type: security-scan
      name: "Custom Security Scan"
      description: |
        Scan for hardcoded credentials and secrets.
        Check for SQL injection and XSS vulnerabilities.
      category: analysis
      cost_tier: medium
      risk_level: low
      interval: "48h"
```

### Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | Yes | — | Kebab-case slug (`[a-z0-9-]+`). Must not collide with built-in task names. |
| `name` | Yes | — | Human-readable name shown in CLI output. |
| `description` | Yes | — | The prompt sent to the AI agent. Write it as clear instructions. |
| `category` | No | `analysis` | One of: `pr`, `analysis`, `options`, `safe`, `map`, `emergency`. |
| `cost_tier` | No | `medium` | One of: `low`, `medium`, `high`, `very-high`. Controls budget filtering. |
| `risk_level` | No | `low` | One of: `low`, `medium`, `high`. |
| `interval` | No | Category default | Go duration string (e.g. `48h`, `168h`). Minimum time between runs per project. |

### How It Works

Custom tasks register into the same task registry as built-in tasks. They participate in the same scoring, cooldown, budget filtering, and plan-implement-review orchestration cycle. For custom tasks, the `description` field still becomes the agent's task prompt.

Custom tasks appear in `nightshift task list` with a `[custom]` label and in JSON output with a `"custom": true` field.

### Enabling Custom Tasks

Custom tasks are automatically enabled when defined. To disable one, add its type to `tasks.disabled`:

```yaml
tasks:
  disabled:
    - security-scan
  custom:
    - type: security-scan
      # ...
```

### Tips

- For built-in tasks, keep `Description` short and add `AgentInstructions` only when the agent needs extra guardrails
- For custom tasks, write `description` as direct instructions to the agent — it becomes the task prompt
- Use `nightshift preview --explain` to verify custom tasks appear in the run plan
- Run `nightshift run --task my-custom-type --dry-run` to test a specific custom task
- Custom task types must not match any built-in task name
