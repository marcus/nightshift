# AI Coding Agent Integration via tmux

This document describes how to programmatically interact with Claude Code and OpenAI Codex CLIs using tmux for session management, command input, and output capture.

## Why tmux?

Claude Code is designed as an interactive TUI application. It doesn't expose a simple `--usage` flag or machine-readable output for most operations. However, we can work around this by:

1. Running Claude Code inside a tmux session
2. Sending keystrokes via `tmux send-keys`
3. Capturing output via `tmux capture-pane`

This approach is reliable because tmux gives us:
- **Persistent sessions** that survive disconnects
- **Programmatic input** via `send-keys`
- **Screen scraping** via `capture-pane` (no OCR needed—it's just text)

## Basic Pattern

```bash
# 1. Create a detached tmux session
tmux new-session -d -s claude-session

# 2. Start Claude Code in that session
tmux send-keys -t claude-session 'claude' Enter

# 3. Wait for startup
sleep 3

# 4. Capture current screen state
tmux capture-pane -t claude-session -p
```

## Handling Interactive Prompts

Claude Code often shows trust prompts or confirmation dialogs. Handle these by:

```bash
# Send Enter to confirm (e.g., "Yes, proceed")
tmux send-keys -t claude-session Enter

# Send Escape to cancel
tmux send-keys -t claude-session Escape

# Navigate menus with arrow keys
tmux send-keys -t claude-session Down
tmux send-keys -t claude-session Up
```

## Example: Fetching Usage Stats

```bash
#!/bin/bash
SESSION="claude-usage-$$"

# Start session and Claude Code
tmux new-session -d -s "$SESSION"
tmux send-keys -t "$SESSION" 'claude' Enter
sleep 3

# Handle trust prompt (press Enter to confirm)
OUTPUT=$(tmux capture-pane -t "$SESSION" -p)
if echo "$OUTPUT" | grep -q "Do you trust"; then
    tmux send-keys -t "$SESSION" Enter
    sleep 3
fi

# Run /usage command
tmux send-keys -t "$SESSION" '/usage' Enter
sleep 2

# Capture the usage output
USAGE=$(tmux capture-pane -t "$SESSION" -p)

# Extract percentages (example parsing)
WEEKLY_ALL=$(echo "$USAGE" | grep -A1 "Current week (all models)" | grep -oE '[0-9]+%' | head -1)
WEEKLY_SONNET=$(echo "$USAGE" | grep -A1 "Current week (Sonnet only)" | grep -oE '[0-9]+%' | head -1)

echo "Weekly (all models): $WEEKLY_ALL"
echo "Weekly (Sonnet only): $WEEKLY_SONNET"

# Cleanup
tmux send-keys -t "$SESSION" Escape
sleep 1
tmux send-keys -t "$SESSION" '/exit' Enter
sleep 1
tmux kill-session -t "$SESSION" 2>/dev/null
```

## Example: Running a Task

```bash
#!/bin/bash
SESSION="claude-task-$$"
TASK="Review the code in src/ and suggest improvements"

tmux new-session -d -s "$SESSION"
tmux send-keys -t "$SESSION" 'claude' Enter
sleep 3

# Handle trust prompt
tmux send-keys -t "$SESSION" Enter
sleep 3

# Send the task
tmux send-keys -t "$SESSION" "$TASK" Enter

# Wait for completion (poll for idle state)
while true; do
    OUTPUT=$(tmux capture-pane -t "$SESSION" -p)
    # Check if prompt is waiting for input (adjust pattern as needed)
    if echo "$OUTPUT" | grep -qE '❯.*$'; then
        break
    fi
    sleep 5
done

# Capture final output
RESULT=$(tmux capture-pane -t "$SESSION" -p -S -1000)  # -S -1000 captures scrollback

# Cleanup
tmux send-keys -t "$SESSION" '/exit' Enter
sleep 1
tmux kill-session -t "$SESSION" 2>/dev/null

echo "$RESULT"
```

## Key Commands Reference

| Action | Command |
|--------|---------|
| Create detached session | `tmux new-session -d -s NAME` |
| Send text/keys | `tmux send-keys -t NAME 'text' Enter` |
| Capture visible pane | `tmux capture-pane -t NAME -p` |
| Capture with scrollback | `tmux capture-pane -t NAME -p -S -1000` |
| Kill session | `tmux kill-session -t NAME` |
| List sessions | `tmux list-sessions` |
| Send Escape | `tmux send-keys -t NAME Escape` |
| Send Ctrl+C | `tmux send-keys -t NAME C-c` |

## Parsing Output

Claude Code uses Unicode box-drawing characters and ANSI formatting. When parsing:

1. **Strip ANSI codes** if needed: `sed 's/\x1b\[[0-9;]*m//g'`
2. **Look for specific patterns** like percentages (`[0-9]+%`) or status indicators
3. **Use the progress bar** in the status line: `● ░░░░░░░░░░ 100%` indicates usage

## Working Directory

Claude Code uses the working directory of the tmux session. To work in a specific project:

```bash
tmux new-session -d -s "$SESSION" -c "/path/to/project"
```

Or change directory before starting Claude:

```bash
tmux send-keys -t "$SESSION" 'cd /path/to/project && claude' Enter
```

## Gotchas

1. **Timing**: Claude Code startup time varies. Use sufficient `sleep` or poll for ready state.
2. **Trust prompts**: First run in a directory shows trust confirmation. Handle it or pre-trust directories.
3. **Screen size**: tmux pane size affects output wrapping. Set size if needed: `tmux resize-pane -t NAME -x 120 -y 40`
4. **Session naming**: Use unique names (e.g., with `$$` PID) to avoid conflicts.
5. **Cleanup**: Always kill sessions to avoid orphaned processes.

## Integration with Nightshift

For Nightshift, this pattern enables:

1. **Budget checking**: Query `/usage` before running tasks to verify available budget
2. **Task execution**: Send prompts and capture results
3. **Progress monitoring**: Poll `capture-pane` to detect completion
4. **Graceful shutdown**: Send `/exit` or `Escape` to cleanly terminate

The Agent Spawner component should wrap this pattern in a robust Go implementation with proper error handling, timeouts, and retry logic.

## Future Improvements

- Claude Code may eventually expose `--usage` or `--json` flags for machine-readable output
- A local SQLite/JSON cache of usage stats could reduce API calls
- MCP server mode might provide a cleaner programmatic interface

---

## Codex CLI

OpenAI Codex uses similar patterns but with different commands.

### Fetching Usage Stats (Codex)

Codex uses `/status` instead of `/usage`:

```bash
#!/bin/bash
SESSION="codex-status-$$"

tmux new-session -d -s "$SESSION"
tmux send-keys -t "$SESSION" 'codex' Enter
sleep 3

# Handle update prompt if it appears (select "Skip")
OUTPUT=$(tmux capture-pane -t "$SESSION" -p)
if echo "$OUTPUT" | grep -q "Update available"; then
    tmux send-keys -t "$SESSION" Down Enter  # Select "Skip"
    sleep 2
fi

# Handle folder trust prompt
OUTPUT=$(tmux capture-pane -t "$SESSION" -p)
if echo "$OUTPUT" | grep -q "allow Codex to work"; then
    tmux send-keys -t "$SESSION" Enter  # Accept default
    sleep 2
fi

# Run /status
tmux send-keys -t "$SESSION" '/status' Enter
sleep 2

# Capture output
STATUS=$(tmux capture-pane -t "$SESSION" -p -S -30)

# Extract usage percentages
FIVE_HOUR=$(echo "$STATUS" | grep "5h limit" | grep -oE '[0-9]+%' | head -1)
WEEKLY=$(echo "$STATUS" | grep "Weekly limit" | grep -oE '[0-9]+%' | head -1)

echo "Codex 5h limit: $FIVE_HOUR left"
echo "Codex weekly: $WEEKLY left"

# Cleanup
tmux send-keys -t "$SESSION" '/exit' Enter
sleep 1
tmux kill-session -t "$SESSION" 2>/dev/null
```

### Key Differences from Claude Code

| Feature | Claude Code | Codex |
|---------|-------------|-------|
| Usage command | `/usage` | `/status` |
| Exit command | `/exit` | `/exit` |
| Trust prompt | "Do you trust the files" | "allow Codex to work in this folder" |
| Update prompt | None (auto-updates) | Interactive prompt with Skip option |
| Local usage cache | `~/.claude/stats-cache.json` | None (server-side only) |
| Config location | `~/.claude/` | `~/.codex/` |

### Codex-Specific Prompts

Codex shows additional prompts you may need to handle:

1. **Update prompt**: Appears when new version available
   - Options: "Update now", "Skip", "Skip until next version"
   - Navigate with arrow keys, confirm with Enter

2. **Folder permission prompt**: Asks about edit approval mode
   - Options: "Yes, allow without asking" or "No, ask me"
   - Default (first option) is usually fine for automation

---

*Last updated: 2026-02-04*
