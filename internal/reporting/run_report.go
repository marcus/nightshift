package reporting

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultRunReportPath returns the default path for a run report file.
func DefaultRunReportPath(ts time.Time) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "reports",
		fmt.Sprintf("run-%s.md", ts.Format("2006-01-02-150405")))
}

// RenderRunReport renders a markdown report for a single run.
func RenderRunReport(results *RunResults, logPath string) (string, error) {
	if results == nil {
		return "", fmt.Errorf("results cannot be nil")
	}

	var completed, failed, skipped []TaskResult
	for _, task := range results.Tasks {
		switch task.Status {
		case "completed":
			completed = append(completed, task)
		case "failed":
			failed = append(failed, task)
		case "skipped":
			skipped = append(skipped, task)
		}
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# Nightshift Run - %s\n\n", results.StartTime.Format("2006-01-02 15:04"))

	buf.WriteString("## Summary\n")
	duration := results.EndTime.Sub(results.StartTime)
	fmt.Fprintf(&buf, "- Duration: %s\n", formatDuration(duration))
	if results.StartBudget > 0 {
		fmt.Fprintf(&buf, "- Budget: %s start, %s used, %s remaining\n",
			formatTokens(results.StartBudget),
			formatTokens(results.UsedBudget),
			formatTokens(results.RemainingBudget),
		)
	}
	fmt.Fprintf(&buf, "- Tasks: %d completed, %d failed, %d skipped\n",
		len(completed), len(failed), len(skipped))
	if logPath != "" {
		fmt.Fprintf(&buf, "- Logs: %s\n", logPath)
	}
	buf.WriteString("\n")

	writeTaskSection(&buf, "Tasks Completed", completed, "")
	writeTaskSection(&buf, "Tasks Failed", failed, "")
	writeTaskSection(&buf, "Tasks Skipped", skipped, "Skip reason: ")

	return buf.String(), nil
}

// SaveRunReport writes a run report to disk.
func SaveRunReport(results *RunResults, path string, logPath string) error {
	content, err := RenderRunReport(results, logPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating report dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	return nil
}

func writeTaskSection(buf *bytes.Buffer, title string, tasks []TaskResult, reasonPrefix string) {
	if len(tasks) == 0 {
		return
	}
	buf.WriteString("## " + title + "\n")
	for _, task := range tasks {
		line := fmt.Sprintf("- %s: %s (%s)", task.Project, task.Title, task.TaskType)
		if task.TokensUsed > 0 {
			line += fmt.Sprintf(" — %s tokens", formatTokens(task.TokensUsed))
		}
		if task.Duration > 0 {
			line += fmt.Sprintf(" — %s", formatDuration(task.Duration))
		}
		if task.OutputRef != "" {
			line += fmt.Sprintf(" — output: %s", task.OutputRef)
		}
		if reasonPrefix != "" && task.SkipReason != "" {
			line += fmt.Sprintf(" — %s%s", reasonPrefix, task.SkipReason)
		}
		buf.WriteString(line + "\n")
	}
	buf.WriteString("\n")
}
