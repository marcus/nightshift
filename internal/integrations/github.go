package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/marcusvorwaller/nightshift/internal/config"
)

// GitHubReader integrates with GitHub issues via gh CLI.
type GitHubReader struct {
	enabled bool
	label   string // Label to filter issues (default: "nightshift")
}

// NewGitHubReader creates a reader based on config.
func NewGitHubReader(cfg *config.Config) *GitHubReader {
	r := &GitHubReader{
		label: "nightshift", // Default label
	}

	// Check task sources for github_issues config
	for _, src := range cfg.Integrations.TaskSources {
		if src.GithubIssues {
			r.enabled = true
			break
		}
	}

	return r
}

func (r *GitHubReader) Name() string {
	return "github"
}

func (r *GitHubReader) Enabled() bool {
	return r.enabled
}

// Read fetches issues from GitHub using gh CLI.
func (r *GitHubReader) Read(ctx context.Context, projectPath string) (*Result, error) {
	// Check if gh is available
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, nil // gh not installed, not an error
	}

	// Check if we're in a git repo with GitHub remote
	if !isGitHubRepo(ctx, projectPath) {
		return nil, nil
	}

	result := &Result{
		Metadata: map[string]any{
			"label": r.label,
		},
	}

	// Fetch issues
	issues, err := r.listIssues(ctx, projectPath)
	if err != nil {
		// GitHub might not be configured or no issues
		return nil, nil
	}

	result.Tasks = issues
	return result, nil
}

// listIssues runs gh issue list and parses the output.
func (r *GitHubReader) listIssues(ctx context.Context, projectPath string) ([]TaskItem, error) {
	args := []string{
		"issue", "list",
		"--label", r.label,
		"--json", "number,title,body,labels,state",
		"--state", "open",
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var ghIssues []ghIssue
	if err := json.Unmarshal(output, &ghIssues); err != nil {
		return nil, err
	}

	var tasks []TaskItem
	for _, issue := range ghIssues {
		labels := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labels[i] = l.Name
		}

		tasks = append(tasks, TaskItem{
			ID:          fmt.Sprintf("gh-%d", issue.Number),
			Title:       issue.Title,
			Description: issue.Body,
			Priority:    extractPriority(labels),
			Labels:      labels,
			Source:      "github",
			Metadata: map[string]string{
				"number": strconv.Itoa(issue.Number),
				"state":  issue.State,
			},
		})
	}

	return tasks, nil
}

// ghIssue represents a GitHub issue from gh CLI JSON output.
type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// extractPriority derives priority from labels.
func extractPriority(labels []string) int {
	for _, label := range labels {
		l := strings.ToLower(label)
		switch {
		case strings.Contains(l, "critical"), strings.Contains(l, "urgent"):
			return 100
		case strings.Contains(l, "high"):
			return 75
		case strings.Contains(l, "medium"):
			return 50
		case strings.Contains(l, "low"):
			return 25
		case strings.HasPrefix(l, "p0"):
			return 100
		case strings.HasPrefix(l, "p1"):
			return 75
		case strings.HasPrefix(l, "p2"):
			return 50
		case strings.HasPrefix(l, "p3"):
			return 25
		}
	}
	return 50 // Default medium priority
}

// Comment adds a comment to an issue.
func (r *GitHubReader) Comment(ctx context.Context, projectPath string, issueNumber int, body string) error {
	cmd := exec.CommandContext(ctx, "gh", "issue", "comment",
		strconv.Itoa(issueNumber),
		"--body", body,
	)
	cmd.Dir = projectPath
	return cmd.Run()
}

// Close closes an issue.
func (r *GitHubReader) Close(ctx context.Context, projectPath string, issueNumber int) error {
	cmd := exec.CommandContext(ctx, "gh", "issue", "close",
		strconv.Itoa(issueNumber),
	)
	cmd.Dir = projectPath
	return cmd.Run()
}

// isGitHubRepo checks if the project is a GitHub repository.
func isGitHubRepo(ctx context.Context, projectPath string) bool {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	url := strings.ToLower(string(output))
	return strings.Contains(url, "github.com")
}
