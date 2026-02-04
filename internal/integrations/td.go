package integrations

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

// TDReader integrates with the td task management CLI.
type TDReader struct {
	enabled    bool
	teachAgent bool
}

// NewTDReader creates a reader based on config.
func NewTDReader(cfg *config.Config) *TDReader {
	r := &TDReader{}

	// Check task sources for td config
	for _, src := range cfg.Integrations.TaskSources {
		if src.TD != nil && src.TD.Enabled {
			r.enabled = true
			r.teachAgent = src.TD.TeachAgent
			break
		}
	}

	return r
}

func (r *TDReader) Name() string {
	return "td"
}

func (r *TDReader) Enabled() bool {
	return r.enabled
}

// Read fetches tasks from td CLI.
func (r *TDReader) Read(ctx context.Context, projectPath string) (*Result, error) {
	// Check if td is available
	if _, err := exec.LookPath("td"); err != nil {
		return nil, nil // td not installed, not an error
	}

	result := &Result{
		Metadata: map[string]any{
			"teach_agent": r.teachAgent,
		},
	}

	// Fetch tasks using td list --format json
	tasks, err := r.listTasks(ctx, projectPath)
	if err != nil {
		// td might not be configured for this project
		return nil, nil
	}

	result.Tasks = tasks

	// Add agent teaching context if configured
	if r.teachAgent {
		result.Context = tdUsageContext
	}

	return result, nil
}

// listTasks runs `td list --format json` and parses the output.
func (r *TDReader) listTasks(ctx context.Context, projectPath string) ([]TaskItem, error) {
	cmd := exec.CommandContext(ctx, "td", "list", "--format", "json")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse td JSON output
	var tdTasks []tdTask
	if err := json.Unmarshal(output, &tdTasks); err != nil {
		// Try parsing as object with tasks array
		var wrapper struct {
			Tasks []tdTask `json:"tasks"`
		}
		if err := json.Unmarshal(output, &wrapper); err != nil {
			return nil, err
		}
		tdTasks = wrapper.Tasks
	}

	// Convert to TaskItems
	var tasks []TaskItem
	for _, t := range tdTasks {
		tasks = append(tasks, TaskItem{
			ID:          t.ID,
			Title:       t.Subject,
			Description: t.Description,
			Priority:    parsePriority(t.Priority),
			Labels:      t.Labels,
			Source:      "td",
			Metadata: map[string]string{
				"status": t.Status,
				"owner":  t.Owner,
			},
		})
	}

	return tasks, nil
}

// tdTask represents a task from td's JSON output.
type tdTask struct {
	ID          string   `json:"id"`
	Subject     string   `json:"subject"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Owner       string   `json:"owner"`
	Labels      []string `json:"labels"`
}

// parsePriority converts td priority string to int.
func parsePriority(p string) int {
	switch strings.ToLower(p) {
	case "critical", "urgent":
		return 100
	case "high":
		return 75
	case "medium", "normal":
		return 50
	case "low":
		return 25
	default:
		// Try parsing as number
		if n, err := strconv.Atoi(p); err == nil {
			return n
		}
		return 50
	}
}

// Assign marks a task as assigned in td.
func (r *TDReader) Assign(ctx context.Context, projectPath, taskID string) error {
	cmd := exec.CommandContext(ctx, "td", "assign", taskID)
	cmd.Dir = projectPath
	return cmd.Run()
}

// Complete marks a task as done in td.
func (r *TDReader) Complete(ctx context.Context, projectPath, taskID string) error {
	cmd := exec.CommandContext(ctx, "td", "complete", taskID)
	cmd.Dir = projectPath
	return cmd.Run()
}

// tdUsageContext provides instructions for agents to use td.
const tdUsageContext = `## td Task Management

This project uses td for task management. Available commands:

- View tasks: td list
- Get task details: td get TASK_ID
- Mark assigned: td assign TASK_ID
- Mark complete: td complete TASK_ID
- Add comment: td comment TASK_ID "message"

When working on a task:
1. First run td assign TASK_ID to claim it
2. Work on the task
3. Run td complete TASK_ID when done
`
