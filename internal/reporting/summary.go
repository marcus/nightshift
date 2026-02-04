// Package reporting generates morning summary reports for nightshift runs.
// Reports are generated as markdown and can be saved to disk or sent via notifications.
package reporting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/marcusvorwaller/nightshift/internal/config"
	"github.com/marcusvorwaller/nightshift/internal/logging"
)

// TaskResult represents a completed or skipped task in the run.
type TaskResult struct {
	Project     string `json:"project"`
	TaskType    string `json:"task_type"`
	Title       string `json:"title"`
	Status      string `json:"status"` // completed, failed, skipped
	OutputType  string `json:"output_type,omitempty"` // PR, Report, Analysis, etc.
	OutputRef   string `json:"output_ref,omitempty"`  // PR number, report path, etc.
	TokensUsed  int    `json:"tokens_used"`
	SkipReason  string `json:"skip_reason,omitempty"` // e.g., "insufficient budget"
	Duration    time.Duration `json:"duration,omitempty"`
}

// RunResults holds all results from a nightshift run.
type RunResults struct {
	Date          time.Time     `json:"date"`
	StartBudget   int           `json:"start_budget"`
	UsedBudget    int           `json:"used_budget"`
	RemainingBudget int         `json:"remaining_budget"`
	Tasks         []TaskResult  `json:"tasks"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
}

// Summary represents a generated morning summary.
type Summary struct {
	Date            time.Time
	Content         string
	ProjectCounts   map[string]int
	CompletedTasks  []TaskResult
	SkippedTasks    []TaskResult
	FailedTasks     []TaskResult
	BudgetStart     int
	BudgetUsed      int
	BudgetRemaining int
}

// Generator creates morning summary reports.
type Generator struct {
	cfg    *config.Config
	logger *logging.Logger
}

// NewGenerator creates a summary generator with the given configuration.
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		cfg:    cfg,
		logger: logging.Component("reporting"),
	}
}

// Generate creates a summary from run results.
func (g *Generator) Generate(results *RunResults) (*Summary, error) {
	if results == nil {
		return nil, fmt.Errorf("results cannot be nil")
	}

	summary := &Summary{
		Date:            results.Date,
		BudgetStart:     results.StartBudget,
		BudgetUsed:      results.UsedBudget,
		BudgetRemaining: results.RemainingBudget,
		ProjectCounts:   make(map[string]int),
		CompletedTasks:  make([]TaskResult, 0),
		SkippedTasks:    make([]TaskResult, 0),
		FailedTasks:     make([]TaskResult, 0),
	}

	// Categorize tasks and count by project
	for _, task := range results.Tasks {
		switch task.Status {
		case "completed":
			summary.CompletedTasks = append(summary.CompletedTasks, task)
			summary.ProjectCounts[task.Project]++
		case "skipped":
			summary.SkippedTasks = append(summary.SkippedTasks, task)
		case "failed":
			summary.FailedTasks = append(summary.FailedTasks, task)
		}
	}

	// Generate markdown content
	summary.Content = g.renderMarkdown(summary, results)

	return summary, nil
}

// renderMarkdown generates the markdown summary content.
func (g *Generator) renderMarkdown(summary *Summary, results *RunResults) string {
	var buf bytes.Buffer

	// Header
	buf.WriteString(fmt.Sprintf("# Nightshift Summary - %s\n\n", summary.Date.Format("2006-01-02")))

	// Budget section
	buf.WriteString("## Budget\n")
	usedPercent := 0
	if summary.BudgetStart > 0 {
		usedPercent = (summary.BudgetUsed * 100) / summary.BudgetStart
	}
	buf.WriteString(fmt.Sprintf("- Started with: %s tokens\n", formatTokens(summary.BudgetStart)))
	buf.WriteString(fmt.Sprintf("- Used: %s tokens (%d%%)\n", formatTokens(summary.BudgetUsed), usedPercent))
	buf.WriteString(fmt.Sprintf("- Remaining: %s tokens\n\n", formatTokens(summary.BudgetRemaining)))

	// Projects processed section
	if len(summary.ProjectCounts) > 0 {
		buf.WriteString("## Projects Processed\n")

		// Sort projects by task count (descending)
		type projectCount struct {
			name  string
			count int
		}
		projects := make([]projectCount, 0, len(summary.ProjectCounts))
		for name, count := range summary.ProjectCounts {
			projects = append(projects, projectCount{name, count})
		}
		sort.Slice(projects, func(i, j int) bool {
			return projects[i].count > projects[j].count
		})

		for i, p := range projects {
			taskWord := "task"
			if p.count != 1 {
				taskWord = "tasks"
			}
			buf.WriteString(fmt.Sprintf("%d. **%s** (%d %s)\n", i+1, filepath.Base(p.name), p.count, taskWord))
		}
		buf.WriteString("\n")
	}

	// Tasks completed section
	if len(summary.CompletedTasks) > 0 {
		buf.WriteString("## Tasks Completed\n")
		for _, task := range summary.CompletedTasks {
			buf.WriteString(g.formatTaskLine(task))
		}
		buf.WriteString("\n")
	}

	// Failed tasks section
	if len(summary.FailedTasks) > 0 {
		buf.WriteString("## Tasks Failed\n")
		for _, task := range summary.FailedTasks {
			buf.WriteString(fmt.Sprintf("- **%s**: %s\n", task.Title, task.SkipReason))
		}
		buf.WriteString("\n")
	}

	// Tasks skipped section
	if len(summary.SkippedTasks) > 0 {
		buf.WriteString("## Tasks Skipped (insufficient budget)\n")
		for _, task := range summary.SkippedTasks {
			reason := task.SkipReason
			if reason == "" {
				reason = "insufficient budget"
			}
			buf.WriteString(fmt.Sprintf("- %s (%s)\n", task.Title, reason))
		}
		buf.WriteString("\n")
	}

	// What's next section
	whatsNext := g.generateWhatsNext(summary)
	if len(whatsNext) > 0 {
		buf.WriteString("## What's Next?\n")
		for _, item := range whatsNext {
			buf.WriteString(fmt.Sprintf("- %s\n", item))
		}
		buf.WriteString("\n")
	}

	// Run duration
	if !results.StartTime.IsZero() && !results.EndTime.IsZero() {
		duration := results.EndTime.Sub(results.StartTime)
		buf.WriteString(fmt.Sprintf("---\n*Run duration: %s*\n", formatDuration(duration)))
	}

	return buf.String()
}

// formatTaskLine formats a single completed task as a markdown line.
func (g *Generator) formatTaskLine(task TaskResult) string {
	prefix := ""
	if task.OutputType != "" && task.OutputRef != "" {
		prefix = fmt.Sprintf("[%s %s] ", task.OutputType, task.OutputRef)
	} else if task.OutputType != "" {
		prefix = fmt.Sprintf("[%s] ", task.OutputType)
	}

	projectName := filepath.Base(task.Project)
	if task.Project == "" {
		return fmt.Sprintf("- %s%s\n", prefix, task.Title)
	}
	return fmt.Sprintf("- %s%s in %s\n", prefix, task.Title, projectName)
}

// generateWhatsNext creates action items based on completed tasks.
func (g *Generator) generateWhatsNext(summary *Summary) []string {
	var items []string

	for _, task := range summary.CompletedTasks {
		switch task.OutputType {
		case "PR":
			items = append(items, fmt.Sprintf("Review %s in %s", task.OutputRef, filepath.Base(task.Project)))
		case "Report":
			items = append(items, fmt.Sprintf("Review %s report (see %s)", task.TaskType, task.OutputRef))
		case "Analysis":
			items = append(items, fmt.Sprintf("Consider %s findings (see report)", task.Title))
		}
	}

	// Add suggestions for skipped high-priority tasks
	for _, task := range summary.SkippedTasks {
		if strings.Contains(task.SkipReason, "budget") {
			items = append(items, fmt.Sprintf("Consider running %s with increased budget", task.Title))
			break // Only add one budget suggestion
		}
	}

	return items
}

// Save writes the summary to a file.
func (g *Generator) Save(summary *Summary, path string) error {
	if summary == nil {
		return fmt.Errorf("summary cannot be nil")
	}

	// Expand path
	path = expandPath(path)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating summary directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, []byte(summary.Content), 0644); err != nil {
		return fmt.Errorf("writing summary file: %w", err)
	}

	g.logger.Infof("summary saved to %s", path)
	return nil
}

// DefaultSummaryPath returns the default path for a summary file.
func DefaultSummaryPath(date time.Time) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "summaries",
		fmt.Sprintf("summary-%s.md", date.Format("2006-01-02")))
}

// SendNotifications sends the summary via configured notification channels.
func (g *Generator) SendNotifications(summary *Summary) error {
	if summary == nil {
		return fmt.Errorf("summary cannot be nil")
	}

	if g.cfg == nil || !g.cfg.Reporting.MorningSummary {
		return nil // Notifications disabled
	}

	var errs []error

	// Send email if configured
	if g.cfg.Reporting.Email != nil && *g.cfg.Reporting.Email != "" {
		if err := g.sendEmail(summary, *g.cfg.Reporting.Email); err != nil {
			g.logger.Errorf("email notification failed: %v", err)
			errs = append(errs, fmt.Errorf("email: %w", err))
		}
	}

	// Send Slack if configured
	if g.cfg.Reporting.SlackWebhook != nil && *g.cfg.Reporting.SlackWebhook != "" {
		if err := g.sendSlack(summary, *g.cfg.Reporting.SlackWebhook); err != nil {
			g.logger.Errorf("slack notification failed: %v", err)
			errs = append(errs, fmt.Errorf("slack: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}

	return nil
}

// sendEmail sends the summary via email.
func (g *Generator) sendEmail(summary *Summary, to string) error {
	// Get SMTP settings from environment
	smtpHost := os.Getenv("NIGHTSHIFT_SMTP_HOST")
	smtpPort := os.Getenv("NIGHTSHIFT_SMTP_PORT")
	smtpUser := os.Getenv("NIGHTSHIFT_SMTP_USER")
	smtpPass := os.Getenv("NIGHTSHIFT_SMTP_PASS")
	smtpFrom := os.Getenv("NIGHTSHIFT_SMTP_FROM")

	if smtpHost == "" {
		return fmt.Errorf("NIGHTSHIFT_SMTP_HOST not set")
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if smtpFrom == "" {
		smtpFrom = "nightshift@localhost"
	}

	subject := fmt.Sprintf("Nightshift Summary - %s", summary.Date.Format("2006-01-02"))

	// Build email message
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		smtpFrom, to, subject, summary.Content)

	var auth smtp.Auth
	if smtpUser != "" && smtpPass != "" {
		auth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	}

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	if err := smtp.SendMail(addr, auth, smtpFrom, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	g.logger.Infof("email sent to %s", to)
	return nil
}

// sendSlack sends the summary to a Slack webhook.
func (g *Generator) sendSlack(summary *Summary, webhookURL string) error {
	// Build Slack message payload
	payload := map[string]any{
		"text": fmt.Sprintf("*Nightshift Summary - %s*", summary.Date.Format("2006-01-02")),
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": fmt.Sprintf("Nightshift Summary - %s", summary.Date.Format("2006-01-02")),
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": g.formatSlackSummary(summary),
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("posting to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	g.logger.Info("slack notification sent")
	return nil
}

// formatSlackSummary creates a Slack-friendly summary format.
func (g *Generator) formatSlackSummary(summary *Summary) string {
	var buf bytes.Buffer

	// Budget
	usedPercent := 0
	if summary.BudgetStart > 0 {
		usedPercent = (summary.BudgetUsed * 100) / summary.BudgetStart
	}
	buf.WriteString(fmt.Sprintf("*Budget:* %s used (%d%%) of %s\n\n",
		formatTokens(summary.BudgetUsed), usedPercent, formatTokens(summary.BudgetStart)))

	// Tasks completed
	if len(summary.CompletedTasks) > 0 {
		buf.WriteString(fmt.Sprintf("*Tasks Completed:* %d\n", len(summary.CompletedTasks)))
		for _, task := range summary.CompletedTasks {
			buf.WriteString(fmt.Sprintf("  - %s\n", task.Title))
		}
		buf.WriteString("\n")
	}

	// Projects
	if len(summary.ProjectCounts) > 0 {
		buf.WriteString(fmt.Sprintf("*Projects:* %d processed\n", len(summary.ProjectCounts)))
	}

	// Skipped
	if len(summary.SkippedTasks) > 0 {
		buf.WriteString(fmt.Sprintf("\n_Skipped %d tasks (budget constraints)_", len(summary.SkippedTasks)))
	}

	return buf.String()
}

// Helper functions

// formatTokens formats a token count with commas for readability.
func formatTokens(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	return fmt.Sprintf("%d,%03d", tokens/1000, tokens%1000)
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
