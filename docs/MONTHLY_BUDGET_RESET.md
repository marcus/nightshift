# Monthly Budget Reset - Quick Reference

## When Does It Reset?

**1st of each month at 00:00:00 UTC**

This matches GitHub Copilot's documented premium request reset behavior.

## How Does It Work?

The reset is **automatic** and triggered on-demand:

1. Every time you query usage (`GetRequestCount()`)
2. The system checks if the month has changed
3. If yes, counter resets to 0 automatically
4. Data is persisted to `~/.copilot/nightshift-usage.json`

## Reset Detection

```go
currentMonth := time.Now().UTC().Format("2006-01")  // "2026-02"

if data.Month != currentMonth {
    // New month detected - reset!
    data.RequestCount = 0
    data.LastReset = firstOfMonth(now)
    data.Month = currentMonth
}
```

## Example Timeline

```
2026-01-31 23:59:59 UTC - 85 requests used (January)
2026-02-01 00:00:00 UTC - RESET POINT
2026-02-01 00:00:01 UTC - 0 requests used (February)
```

## Edge Cases

### Month Boundaries
✅ December → January (year rollover)
```
2026-12-31 → 2027-01-01
```

### Leap Years
✅ February length handled automatically
```
2024-02-29 (leap year) → 2024-03-01
2025-02-28 (non-leap) → 2025-03-01
```

### Different Month Lengths
✅ All month lengths supported (28-31 days)

### Timezone Safety
✅ Always uses UTC internally
```go
time.Now().UTC()  // Never local time
```

### Concurrent Access
✅ File-based persistence prevents data loss across restarts

### Missed Resets
✅ No problem - detected on next request
```
// If nightshift wasn't running on Feb 1st:
Jan 31: 85 requests
(nightshift stopped)
Feb 5: Start nightshift - auto-detects month change → resets to 0
```

## Testing

### Test Monthly Reset

```go
// Simulate January data
testData := &CopilotUsageData{
    RequestCount: 99,
    Month:        "2026-01",  // Old month
}
provider.SaveUsageData(testData)

// Get count in February (auto-resets)
count, _ := provider.GetRequestCount()  
// count == 0 (reset happened automatically)
```

### Test Next Reset Time

```go
resetTime := provider.GetMonthlyResetTime()
// Always returns: next month's 1st at 00:00:00 UTC

// Example: If now is 2026-02-15
// Returns: 2026-03-01 00:00:00 UTC
```

## Debugging

Check current usage data:
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

## Implementation Files

- `internal/providers/copilot.go` - Reset logic
- `internal/providers/copilot_test.go` - Reset tests
- `internal/budget/budget.go` - Budget manager integration
- `docs/COPILOT_INTEGRATION.md` - Full documentation

## Why UTC?

1. **Consistency**: Same reset time worldwide
2. **No DST Issues**: UTC never changes
3. **GitHub Standard**: GitHub uses UTC for rate limits
4. **Testability**: Deterministic behavior
5. **Cross-Machine**: Works across different timezones

## Related Functions

```go
// Get next reset time
resetTime := provider.GetMonthlyResetTime()

// Get reset time for specific mode
resetTime, err := provider.GetResetTime("daily")   // Returns monthly reset
resetTime, err := provider.GetResetTime("weekly")  // Returns monthly reset

// First of current month
first := firstOfMonth(time.Now().UTC())

// Days in current month
days := daysInCurrentMonth(time.Now().UTC())
```

## FAQ

**Q: What if I'm in a different timezone?**  
A: Doesn't matter! Reset happens at UTC midnight. Your local time may be different but the reset is consistent globally.

**Q: What if nightshift isn't running at midnight?**  
A: No problem. The reset is detected on the next request, not at an exact time.

**Q: Can I manually trigger a reset?**  
A: Yes, but not recommended. You can delete the usage file or edit the month field.

**Q: What happens if my system clock is wrong?**  
A: The reset will happen based on your system's UTC time. Fix your system clock for accurate tracking.

**Q: Does this sync with GitHub's actual resets?**  
A: The timing matches GitHub's documented behavior (1st of month UTC), but the count is local only.
