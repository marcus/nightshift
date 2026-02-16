# GitHub Copilot CLI Integration Documentation

## Overview

This document explains how GitHub Copilot CLI support was implemented in Nightshift, including the monthly budget tracking system.

## Architecture

### Provider Layer (`internal/providers/copilot.go`)

The Copilot provider implements GitHub Copilot CLI integration with local request tracking:

#### Key Features

1. **CLI Execution**: Wraps `gh copilot` commands
2. **Local Request Tracking**: Counts requests in `~/.copilot/nightshift-usage.json`
3. **Monthly Reset**: Automatically resets counters on the 1st of each month at 00:00:00 UTC

#### Usage Tracking Approach

**Important Limitation**: GitHub Copilot CLI does not expose usage metrics via API or local files like Codex or Claude do.

**Our Solution**:
- Track requests locally by counting each execution
- Each successful request = 1 premium request (conservative estimate)
- Store data in JSON format with month identifier
- Auto-reset when month changes

**What This Means**:
- ✅ Tracks usage made through Nightshift
- ❌ Cannot track external Copilot usage (IDE, web, etc.)
- ❌ No way to query remaining quota from GitHub servers
- ❌ No authoritative usage data from GitHub API

#### Request Counting

```go
// After each successful Copilot request
provider.IncrementRequestCount()

// Get current month's count
count, err := provider.GetRequestCount()

// Usage automatically resets on month boundary
```

### Agent Layer (`internal/agents/copilot.go`)

The Copilot agent implements the Agent interface using GitHub CLI:

#### Command Structure

According to [GitHub CLI documentation](https://docs.github.com/en/copilot/reference/cli-command-reference):

```bash
gh copilot suggest -t <type> <prompt>
```

**Types**:
- `shell` - For shell commands (used by Nightshift)
- `gh` - For GitHub CLI commands
- `git` - For Git commands

#### Implementation Notes

1. **Binary Path**: Uses `gh` (GitHub CLI) not a separate copilot binary
2. **Extension Required**: Requires `gh extension install github/gh-copilot`
3. **Non-Interactive Mode**: Passes prompts as command arguments
4. **File Context**: Supports including file contents via stdin

### Budget System Integration

#### CopilotUsageProvider Interface

Added to `internal/budget/budget.go`:

```go
type CopilotUsageProvider interface {
    UsageProvider
    GetUsedPercent(mode string, monthlyLimit int64) (float64, error)
    GetResetTime(mode string) (time.Time, error)
}
```

#### Budget Manager Changes

Updated `NewManager` and `NewManagerFromProviders` to accept Copilot provider:

```go
func NewManager(
    cfg *config.Config, 
    claude ClaudeUsageProvider, 
    codex CodexUsageProvider, 
    copilot CopilotUsageProvider,  // NEW
    opts ...Option
) *Manager
```

#### Monthly Budget Calculation

**Daily Mode**:
```
daily_allocation = monthly_limit / days_in_month
today_estimate = total_requests / days_elapsed
used_percent = (today_estimate / daily_allocation) * 100
```

**Weekly Mode**:
```
used_percent = (total_requests / monthly_limit) * 100
```

**Note**: Copilot resets monthly, not weekly. Weekly mode is an approximation.

## Monthly Budget Reset

### Reset Behavior

- **Trigger**: First day of month at 00:00:00 UTC
- **Automatic**: No manual intervention needed
- **Detection**: Checked on every request count operation
- **Safe**: Works across restarts and timezone changes

### Reset Logic

```go
func (c *Copilot) GetRequestCount() (int64, error) {
    data, err := c.LoadUsageData()
    // ...
    
    now := time.Now().UTC()
    currentMonth := now.Format("2006-01")
    
    // Auto-reset if month changed
    if data.Month != currentMonth {
        data.RequestCount = 0
        data.LastReset = firstOfMonth(now)
        data.Month = currentMonth
        c.SaveUsageData(data)
    }
    
    return data.RequestCount, nil
}
```

### Reset Timestamp Calculation

```go
func (c *Copilot) GetMonthlyResetTime() time.Time {
    now := time.Now().UTC()
    // Next reset is 1st of next month at 00:00:00 UTC
    year := now.Year()
    month := now.Month() + 1
    if month > 12 {
        month = 1
        year++
    }
    return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
}
```

### Edge Cases Handled

1. **Month Boundaries**: December → January year rollover
2. **Leap Years**: February day count varies
3. **Different Month Lengths**: 28-31 days per month
4. **Timezone Safety**: All timestamps in UTC
5. **Concurrent Access**: File-based persistence prevents data loss

## Configuration

### Provider Path

Copilot data is stored at `~/.copilot` by default:

```go
// Default path
copilot := providers.NewCopilot()

// Custom path
copilot := providers.NewCopilotWithPath("/custom/path")
```

### Budget Configuration

Add to your nightshift config:

```yaml
providers:
  copilot:
    enabled: true
    # Monthly request limit (adjust to your plan)
    budget: 2000  # requests per month
```

**Note**: The budget value should match your GitHub Copilot plan's monthly request limit.

## Testing

### Provider Tests (`copilot_test.go`)

- Request counting and increment
- Monthly reset logic
- Usage percent calculation (daily/weekly)
- Data persistence and loading
- Helper function validation

### Agent Tests (`copilot_test.go`)

- Command execution
- Timeout handling
- Error handling
- JSON output extraction
- File context inclusion

### Test Coverage

Run tests:
```bash
go test ./internal/providers -run Copilot
go test ./internal/agents -run Copilot
```

## Limitations and Assumptions

### Limitations

1. **No External Tracking**: Cannot see usage from other Copilot interfaces
2. **No API Access**: GitHub doesn't expose Copilot usage via API
3. **Request-Based**: Assumes 1 prompt = 1 premium request (may be inaccurate)
4. **Local Only**: Usage data stored locally, not synced across machines
5. **Interactive CLI**: GitHub Copilot CLI is designed for interactive use

### Assumptions

1. Each prompt execution counts as 1 premium request
2. Monthly limits reset on the 1st at 00:00:00 UTC (per GitHub's documented behavior)
3. The `gh copilot suggest` command is stable and won't change significantly
4. Users have the GitHub CLI installed and authenticated
5. The `gh-copilot` extension is installed

### Future Improvements

If GitHub adds official APIs in the future:

1. Replace local tracking with API calls
2. Get authoritative usage data
3. Query remaining quota
4. Track external usage
5. Support more accurate request counting

## Installation Requirements

Before using Copilot in Nightshift:

1. Install GitHub CLI:
   ```bash
   # macOS
   brew install gh
   
   # Linux
   sudo apt install gh  # or equivalent
   ```

2. Authenticate:
   ```bash
   gh auth login
   ```

3. Install Copilot extension:
   ```bash
   gh extension install github/gh-copilot
   ```

4. Verify installation:
   ```bash
   gh copilot --version
   ```

## Usage Example

```go
// Initialize provider
copilot := providers.NewCopilot()

// Initialize agent
agent := agents.NewCopilotAgent()

// Execute a task
result, err := agent.Execute(ctx, agents.ExecuteOptions{
    Prompt: "How do I list files in Go?",
    WorkDir: "/project",
})

// Track the request
if result.IsSuccess() {
    copilot.IncrementRequestCount()
}

// Check usage
count, _ := copilot.GetRequestCount()
fmt.Printf("Used %d requests this month\n", count)
```

## References

- [GitHub Copilot CLI Installation Guide](https://docs.github.com/en/copilot/how-tos/copilot-cli/install-copilot-cli)
- [GitHub Copilot CLI Command Reference](https://docs.github.com/en/copilot/reference/cli-command-reference)
- GitHub Copilot Premium Request Limits (see your plan details)

## Security Considerations

1. **Local Data Storage**: Usage data stored in plain text JSON (not sensitive)
2. **File Permissions**: Default 0644 for usage file (user-readable)
3. **No Credentials Stored**: Relies on GitHub CLI's authentication
4. **CLI Execution**: Spawns `gh` subprocess (inherits user permissions)

## Maintenance

### Updating Usage Data Format

If the usage data format needs to change:

1. Update `CopilotUsageData` struct
2. Add migration logic in `LoadUsageData`
3. Update tests
4. Document the change

### Monitoring

Check usage data manually:
```bash
cat ~/.copilot/nightshift-usage.json
```

Example output:
```json
{
  "request_count": 42,
  "last_reset": "2026-02-01T00:00:00Z",
  "month": "2026-02"
}
```
