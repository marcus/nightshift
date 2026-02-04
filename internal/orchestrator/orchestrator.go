// Package orchestrator coordinates AI agents working on tasks.
// Implements the plan-implement-review loop for task execution.
package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/marcusvorwaller/nightshift/internal/agents"
	"github.com/marcusvorwaller/nightshift/internal/budget"
	"github.com/marcusvorwaller/nightshift/internal/logging"
	"github.com/marcusvorwaller/nightshift/internal/tasks"
)

// Constants for orchestration.
const (
	DefaultMaxIterations = 3
	DefaultAgentTimeout  = 30 * time.Minute
)

// TaskStatus represents the outcome of task execution.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusPlanning  TaskStatus = "planning"
	StatusExecuting TaskStatus = "executing"
	StatusReviewing TaskStatus = "reviewing"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusAbandoned TaskStatus = "abandoned"
)

// TaskResult holds the outcome of orchestrating a task.
type TaskResult struct {
	TaskID     string        `json:"task_id"`
	Status     TaskStatus    `json:"status"`
	Iterations int           `json:"iterations"`
	Plan       *PlanOutput   `json:"plan,omitempty"`
	Output     string        `json:"output,omitempty"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
	Logs       []LogEntry    `json:"logs"`
}

// PlanOutput represents structured plan from the plan agent.
type PlanOutput struct {
	Steps       []string `json:"steps"`
	Files       []string `json:"files"`
	Description string   `json:"description"`
	Raw         string   `json:"raw,omitempty"`
}

// ImplementOutput represents structured output from implement agent.
type ImplementOutput struct {
	FilesModified []string `json:"files_modified"`
	Summary       string   `json:"summary"`
	Raw           string   `json:"raw,omitempty"`
}

// ReviewOutput represents structured output from review agent.
type ReviewOutput struct {
	Passed   bool     `json:"passed"`
	Feedback string   `json:"feedback"`
	Issues   []string `json:"issues,omitempty"`
	Raw      string   `json:"raw,omitempty"`
}

// LogEntry captures a timestamped log message.
type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// Config holds orchestrator configuration.
type Config struct {
	MaxIterations int           // Max review iterations (default: 3)
	AgentTimeout  time.Duration // Per-agent timeout (default: 30min)
	WorkDir       string        // Working directory for agents
}

// DefaultConfig returns default orchestrator config.
func DefaultConfig() Config {
	return Config{
		MaxIterations: DefaultMaxIterations,
		AgentTimeout:  DefaultAgentTimeout,
	}
}

// Orchestrator manages agent execution using plan-implement-review loop.
type Orchestrator struct {
	agent  agents.Agent
	budget *budget.Tracker
	queue  *tasks.Queue
	config Config
	logger *logging.Logger
}

// Option configures an Orchestrator.
type Option func(*Orchestrator)

// WithAgent sets the agent for task execution.
func WithAgent(a agents.Agent) Option {
	return func(o *Orchestrator) {
		o.agent = a
	}
}

// WithBudget sets the budget tracker.
func WithBudget(b *budget.Tracker) Option {
	return func(o *Orchestrator) {
		o.budget = b
	}
}

// WithQueue sets the task queue.
func WithQueue(q *tasks.Queue) Option {
	return func(o *Orchestrator) {
		o.queue = q
	}
}

// WithConfig sets orchestrator configuration.
func WithConfig(c Config) Option {
	return func(o *Orchestrator) {
		o.config = c
	}
}

// WithLogger sets the logger.
func WithLogger(l *logging.Logger) Option {
	return func(o *Orchestrator) {
		o.logger = l
	}
}

// New creates an orchestrator with the given options.
func New(opts ...Option) *Orchestrator {
	o := &Orchestrator{
		config: DefaultConfig(),
		logger: logging.Component("orchestrator"),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// RunTask executes a single task through the plan-implement-review loop.
func (o *Orchestrator) RunTask(ctx context.Context, task *tasks.Task, workDir string) (*TaskResult, error) {
	start := time.Now()
	result := &TaskResult{
		TaskID: task.ID,
		Status: StatusPending,
		Logs:   make([]LogEntry, 0),
	}

	o.log(result, "info", "starting task", map[string]any{"task_id": task.ID, "title": task.Title})

	if o.agent == nil {
		result.Status = StatusFailed
		result.Error = "no agent configured"
		result.Duration = time.Since(start)
		return result, errors.New("no agent configured")
	}

	// Override workDir from config if provided
	if workDir == "" && o.config.WorkDir != "" {
		workDir = o.config.WorkDir
	}

	// Step 1: Plan
	result.Status = StatusPlanning
	o.log(result, "info", "planning", nil)

	plan, err := o.plan(ctx, task, workDir)
	if err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("planning failed: %v", err)
		result.Duration = time.Since(start)
		o.log(result, "error", "plan failed", map[string]any{"error": err.Error()})
		return result, err
	}
	result.Plan = plan
	o.log(result, "info", "plan created", map[string]any{"steps": len(plan.Steps)})

	// Step 2-4: Implement -> Review loop
	for iteration := 1; iteration <= o.config.MaxIterations; iteration++ {
		result.Iterations = iteration
		o.log(result, "info", "iteration start", map[string]any{"iteration": iteration})

		// Implement
		result.Status = StatusExecuting
		impl, err := o.implement(ctx, task, plan, workDir, iteration)
		if err != nil {
			result.Status = StatusFailed
			result.Error = fmt.Sprintf("implement failed (iteration %d): %v", iteration, err)
			result.Duration = time.Since(start)
			o.log(result, "error", "implement failed", map[string]any{"iteration": iteration, "error": err.Error()})
			return result, err
		}
		result.Output = impl.Summary
		o.log(result, "info", "implementation complete", map[string]any{"files_modified": len(impl.FilesModified)})

		// Review
		result.Status = StatusReviewing
		review, err := o.review(ctx, task, impl, workDir)
		if err != nil {
			result.Status = StatusFailed
			result.Error = fmt.Sprintf("review failed (iteration %d): %v", iteration, err)
			result.Duration = time.Since(start)
			o.log(result, "error", "review failed", map[string]any{"iteration": iteration, "error": err.Error()})
			return result, err
		}

		if review.Passed {
			// Success - commit and return
			o.log(result, "info", "review passed", map[string]any{"iteration": iteration})
			if err := o.commit(ctx, task, impl, workDir); err != nil {
				o.log(result, "warn", "commit failed", map[string]any{"error": err.Error()})
				// Don't fail the task if commit fails, just log it
			}
			result.Status = StatusCompleted
			result.Duration = time.Since(start)
			o.log(result, "info", "task completed", map[string]any{"duration": result.Duration.String()})
			return result, nil
		}

		// Review failed
		o.log(result, "warn", "review failed", map[string]any{
			"iteration": iteration,
			"feedback":  review.Feedback,
			"issues":    review.Issues,
		})

		// If max iterations reached, abandon
		if iteration >= o.config.MaxIterations {
			result.Status = StatusAbandoned
			result.Error = fmt.Sprintf("max iterations (%d) reached: %s", o.config.MaxIterations, review.Feedback)
			result.Duration = time.Since(start)
			o.log(result, "error", "task abandoned", map[string]any{"reason": "max iterations"})
			return result, nil
		}

		// Update plan with review feedback for next iteration
		plan.Description = fmt.Sprintf("%s\n\nReview feedback (iteration %d): %s", plan.Description, iteration, review.Feedback)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// plan spawns the plan agent to create an execution plan.
func (o *Orchestrator) plan(ctx context.Context, task *tasks.Task, workDir string) (*PlanOutput, error) {
	prompt := o.buildPlanPrompt(task)

	ctx, cancel := context.WithTimeout(ctx, o.config.AgentTimeout)
	defer cancel()

	execResult, err := o.agent.Execute(ctx, agents.ExecuteOptions{
		Prompt:  prompt,
		WorkDir: workDir,
		Timeout: o.config.AgentTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("agent execution: %w", err)
	}

	if !execResult.IsSuccess() {
		return nil, fmt.Errorf("agent returned error: %s", execResult.Error)
	}

	plan := &PlanOutput{Raw: execResult.Output}

	// Try to parse structured JSON output
	if len(execResult.JSON) > 0 {
		if err := json.Unmarshal(execResult.JSON, plan); err != nil {
			// JSON parse failed, use raw output
			plan.Description = execResult.Output
		}
	} else {
		plan.Description = execResult.Output
	}

	return plan, nil
}

// implement spawns the implement agent to execute the plan.
func (o *Orchestrator) implement(ctx context.Context, task *tasks.Task, plan *PlanOutput, workDir string, iteration int) (*ImplementOutput, error) {
	prompt := o.buildImplementPrompt(task, plan, iteration)

	ctx, cancel := context.WithTimeout(ctx, o.config.AgentTimeout)
	defer cancel()

	execResult, err := o.agent.Execute(ctx, agents.ExecuteOptions{
		Prompt:  prompt,
		WorkDir: workDir,
		Files:   plan.Files,
		Timeout: o.config.AgentTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("agent execution: %w", err)
	}

	if !execResult.IsSuccess() {
		return nil, fmt.Errorf("agent returned error: %s", execResult.Error)
	}

	impl := &ImplementOutput{Raw: execResult.Output}

	// Try to parse structured JSON output
	if len(execResult.JSON) > 0 {
		if err := json.Unmarshal(execResult.JSON, impl); err != nil {
			impl.Summary = execResult.Output
		}
	} else {
		impl.Summary = execResult.Output
	}

	return impl, nil
}

// review spawns the review agent to check the implementation.
func (o *Orchestrator) review(ctx context.Context, task *tasks.Task, impl *ImplementOutput, workDir string) (*ReviewOutput, error) {
	prompt := o.buildReviewPrompt(task, impl)

	ctx, cancel := context.WithTimeout(ctx, o.config.AgentTimeout)
	defer cancel()

	execResult, err := o.agent.Execute(ctx, agents.ExecuteOptions{
		Prompt:  prompt,
		WorkDir: workDir,
		Files:   impl.FilesModified,
		Timeout: o.config.AgentTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("agent execution: %w", err)
	}

	if !execResult.IsSuccess() {
		return nil, fmt.Errorf("agent returned error: %s", execResult.Error)
	}

	review := &ReviewOutput{Raw: execResult.Output}

	// Try to parse structured JSON output
	if len(execResult.JSON) > 0 {
		if err := json.Unmarshal(execResult.JSON, review); err != nil {
			// Parse failed, try to detect pass/fail from text
			review.Passed = o.inferReviewPassed(execResult.Output)
			review.Feedback = execResult.Output
		}
	} else {
		review.Passed = o.inferReviewPassed(execResult.Output)
		review.Feedback = execResult.Output
	}

	return review, nil
}

// commit finalizes successful task completion.
func (o *Orchestrator) commit(ctx context.Context, task *tasks.Task, impl *ImplementOutput, workDir string) error {
	// For now, commit is a no-op. In full implementation:
	// - Create git commit with changes
	// - Update task state
	// - Send notifications
	o.logger.Infof("commit: task=%s files=%d", task.ID, len(impl.FilesModified))
	return nil
}

// Run processes all tasks in queue until empty or budget exhausted.
func (o *Orchestrator) Run(ctx context.Context) error {
	if o.queue == nil {
		return errors.New("no task queue configured")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		task := o.queue.Next()
		if task == nil {
			o.logger.Info("queue empty, stopping")
			return nil
		}

		// Check budget before running
		if o.budget != nil {
			// TODO: Implement budget check based on task cost estimate
		}

		result, err := o.RunTask(ctx, task, o.config.WorkDir)
		if err != nil {
			o.logger.Errorf("task %s failed: %v", task.ID, err)
			continue
		}

		o.logger.Infof("task %s: status=%s iterations=%d duration=%s",
			result.TaskID, result.Status, result.Iterations, result.Duration)
	}
}

// Prompt builders

func (o *Orchestrator) buildPlanPrompt(task *tasks.Task) string {
	return fmt.Sprintf(`You are a planning agent. Create a detailed execution plan for this task.

## Task
ID: %s
Title: %s
Description: %s

## Instructions
1. Analyze the task requirements
2. Identify files that need to be modified
3. Create step-by-step implementation plan
4. Output your plan as JSON:

{
  "steps": ["step1", "step2", ...],
  "files": ["file1.go", "file2.go", ...],
  "description": "overall approach"
}
`, task.ID, task.Title, task.Description)
}

func (o *Orchestrator) buildImplementPrompt(task *tasks.Task, plan *PlanOutput, iteration int) string {
	iterationNote := ""
	if iteration > 1 {
		iterationNote = fmt.Sprintf("\n\n## Note\nThis is iteration %d. Previous attempts did not pass review. Pay attention to the feedback in the plan description.", iteration)
	}

	return fmt.Sprintf(`You are an implementation agent. Execute the plan for this task.

## Task
ID: %s
Title: %s
Description: %s

## Plan
%s

## Steps
%v
%s
## Instructions
1. Implement the plan step by step
2. Make all necessary code changes
3. Ensure tests pass
4. Output a summary as JSON:

{
  "files_modified": ["file1.go", ...],
  "summary": "what was done"
}
`, task.ID, task.Title, task.Description, plan.Description, plan.Steps, iterationNote)
}

func (o *Orchestrator) buildReviewPrompt(task *tasks.Task, impl *ImplementOutput) string {
	return fmt.Sprintf(`You are a code review agent. Review this implementation.

## Task
ID: %s
Title: %s
Description: %s

## Implementation Summary
%s

## Files Modified
%v

## Instructions
1. Check if implementation meets task requirements
2. Verify code quality and correctness
3. Check for bugs or issues
4. Output your review as JSON:

{
  "passed": true/false,
  "feedback": "detailed feedback",
  "issues": ["issue1", "issue2", ...]
}

Set "passed" to true ONLY if the implementation is correct and complete.
`, task.ID, task.Title, task.Description, impl.Summary, impl.FilesModified)
}

// inferReviewPassed attempts to detect pass/fail from unstructured text.
func (o *Orchestrator) inferReviewPassed(output string) bool {
	// Simple heuristic: look for positive indicators
	// This is a fallback when JSON parsing fails
	passIndicators := []string{
		"passed", "approved", "looks good", "lgtm", "ship it",
		"no issues", "complete", "correct", "successful",
	}
	failIndicators := []string{
		"failed", "rejected", "issues found", "needs work",
		"incorrect", "bug", "error", "missing", "incomplete",
	}

	output = string([]byte(output)) // normalize

	passScore := 0
	failScore := 0

	for _, ind := range passIndicators {
		if containsIgnoreCase(output, ind) {
			passScore++
		}
	}
	for _, ind := range failIndicators {
		if containsIgnoreCase(output, ind) {
			failScore++
		}
	}

	return passScore > failScore
}

func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	// Convert to lowercase for comparison
	sLower := toLowerASCII(s)
	substrLower := toLowerASCII(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func toLowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// log adds a log entry to the result and logs via logger.
func (o *Orchestrator) log(result *TaskResult, level, msg string, fields map[string]any) {
	entry := LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Fields:  fields,
	}
	result.Logs = append(result.Logs, entry)

	// Also log via structured logger
	switch level {
	case "debug":
		o.logger.DebugCtx(msg, fields)
	case "info":
		o.logger.InfoCtx(msg, fields)
	case "warn":
		o.logger.WarnCtx(msg, fields)
	case "error":
		o.logger.ErrorCtx(msg, fields)
	}
}
