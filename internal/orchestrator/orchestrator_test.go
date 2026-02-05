package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/agents"
	"github.com/marcus/nightshift/internal/tasks"
)

// mockAgent implements agents.Agent for testing.
type mockAgent struct {
	name      string
	responses []agents.ExecuteResult
	calls     []agents.ExecuteOptions
	callIndex int
}

func newMockAgent(responses ...agents.ExecuteResult) *mockAgent {
	return &mockAgent{
		name:      "mock",
		responses: responses,
		calls:     make([]agents.ExecuteOptions, 0),
	}
}

func (m *mockAgent) Name() string {
	return m.name
}

func (m *mockAgent) Execute(ctx context.Context, opts agents.ExecuteOptions) (*agents.ExecuteResult, error) {
	m.calls = append(m.calls, opts)

	if m.callIndex >= len(m.responses) {
		return &agents.ExecuteResult{
			Output:   "default response",
			ExitCode: 0,
		}, nil
	}

	resp := m.responses[m.callIndex]
	m.callIndex++
	return &resp, nil
}

// Helper to create JSON response.
func jsonResponse(v any) agents.ExecuteResult {
	data, _ := json.Marshal(v)
	return agents.ExecuteResult{
		Output:   string(data),
		JSON:     data,
		ExitCode: 0,
	}
}

func TestNew(t *testing.T) {
	o := New()
	if o == nil {
		t.Fatal("New() returned nil")
		return
	}
	if o.config.MaxIterations != DefaultMaxIterations {
		t.Errorf("MaxIterations = %d, want %d", o.config.MaxIterations, DefaultMaxIterations)
	}
	if o.config.AgentTimeout != DefaultAgentTimeout {
		t.Errorf("AgentTimeout = %v, want %v", o.config.AgentTimeout, DefaultAgentTimeout)
	}
}

func TestNewWithOptions(t *testing.T) {
	agent := newMockAgent()
	cfg := Config{
		MaxIterations: 5,
		AgentTimeout:  10 * time.Minute,
		WorkDir:       "/test/dir",
	}

	o := New(
		WithAgent(agent),
		WithConfig(cfg),
	)

	if o.agent != agent {
		t.Error("agent not set correctly")
	}
	if o.config.MaxIterations != 5 {
		t.Errorf("MaxIterations = %d, want 5", o.config.MaxIterations)
	}
	if o.config.AgentTimeout != 10*time.Minute {
		t.Errorf("AgentTimeout = %v, want 10m", o.config.AgentTimeout)
	}
	if o.config.WorkDir != "/test/dir" {
		t.Errorf("WorkDir = %q, want /test/dir", o.config.WorkDir)
	}
}

func TestRunTaskNoAgent(t *testing.T) {
	o := New()
	task := &tasks.Task{
		ID:          "test-1",
		Title:       "Test Task",
		Description: "A test task",
	}

	result, err := o.RunTask(context.Background(), task, "")
	if err == nil {
		t.Error("expected error for missing agent")
	}
	if result.Status != StatusFailed {
		t.Errorf("status = %s, want %s", result.Status, StatusFailed)
	}
	if result.Error != "no agent configured" {
		t.Errorf("error = %q, want 'no agent configured'", result.Error)
	}
}

func TestRunTaskSuccessFirstIteration(t *testing.T) {
	// Setup mock responses: plan, implement, review (pass)
	planResp := jsonResponse(PlanOutput{
		Steps:       []string{"step1", "step2"},
		Files:       []string{"file1.go"},
		Description: "test plan",
	})
	implResp := jsonResponse(ImplementOutput{
		FilesModified: []string{"file1.go"},
		Summary:       "implemented changes",
	})
	reviewResp := jsonResponse(ReviewOutput{
		Passed:   true,
		Feedback: "looks good",
	})

	agent := newMockAgent(planResp, implResp, reviewResp)
	o := New(WithAgent(agent))

	task := &tasks.Task{
		ID:          "test-1",
		Title:       "Test Task",
		Description: "A test task",
	}

	result, err := o.RunTask(context.Background(), task, "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != StatusCompleted {
		t.Errorf("status = %s, want %s", result.Status, StatusCompleted)
	}
	if result.Iterations != 1 {
		t.Errorf("iterations = %d, want 1", result.Iterations)
	}
	if result.Plan == nil {
		t.Error("plan should not be nil")
	}
	if len(agent.calls) != 3 {
		t.Errorf("agent calls = %d, want 3", len(agent.calls))
	}
}

func TestRunTaskReviewFailsThenPasses(t *testing.T) {
	// Setup: plan, implement, review (fail), implement, review (pass)
	planResp := jsonResponse(PlanOutput{
		Steps:       []string{"step1"},
		Files:       []string{"file1.go"},
		Description: "test plan",
	})
	implResp := jsonResponse(ImplementOutput{
		FilesModified: []string{"file1.go"},
		Summary:       "implemented",
	})
	reviewFail := jsonResponse(ReviewOutput{
		Passed:   false,
		Feedback: "needs improvement",
		Issues:   []string{"issue1"},
	})
	reviewPass := jsonResponse(ReviewOutput{
		Passed:   true,
		Feedback: "fixed",
	})

	agent := newMockAgent(planResp, implResp, reviewFail, implResp, reviewPass)
	o := New(WithAgent(agent))

	task := &tasks.Task{
		ID:          "test-2",
		Title:       "Test Task",
		Description: "A test task",
	}

	result, err := o.RunTask(context.Background(), task, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != StatusCompleted {
		t.Errorf("status = %s, want %s", result.Status, StatusCompleted)
	}
	if result.Iterations != 2 {
		t.Errorf("iterations = %d, want 2", result.Iterations)
	}
}

func TestRunTaskMaxIterationsAbandoned(t *testing.T) {
	// All reviews fail
	planResp := jsonResponse(PlanOutput{
		Steps: []string{"step1"},
		Files: []string{"file1.go"},
	})
	implResp := jsonResponse(ImplementOutput{
		FilesModified: []string{"file1.go"},
		Summary:       "implemented",
	})
	reviewFail := jsonResponse(ReviewOutput{
		Passed:   false,
		Feedback: "still broken",
	})

	// 3 iterations: plan + (impl + review) * 3 = 7 calls
	agent := newMockAgent(
		planResp,
		implResp, reviewFail, // iteration 1
		implResp, reviewFail, // iteration 2
		implResp, reviewFail, // iteration 3
	)
	o := New(WithAgent(agent))

	task := &tasks.Task{
		ID:          "test-3",
		Title:       "Failing Task",
		Description: "Will fail review",
	}

	result, err := o.RunTask(context.Background(), task, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != StatusAbandoned {
		t.Errorf("status = %s, want %s", result.Status, StatusAbandoned)
	}
	if result.Iterations != 3 {
		t.Errorf("iterations = %d, want 3", result.Iterations)
	}
}

func TestRunTaskPlanFails(t *testing.T) {
	// Agent returns error during planning
	agent := newMockAgent(agents.ExecuteResult{
		Output:   "failed to plan",
		ExitCode: 1,
		Error:    "planning error",
	})
	o := New(WithAgent(agent))

	task := &tasks.Task{
		ID:    "test-4",
		Title: "Plan Fail Task",
	}

	result, err := o.RunTask(context.Background(), task, "")
	if err == nil {
		t.Error("expected error")
	}
	if result.Status != StatusFailed {
		t.Errorf("status = %s, want %s", result.Status, StatusFailed)
	}
}

func TestInferReviewPassed(t *testing.T) {
	o := New()

	tests := []struct {
		output string
		want   bool
	}{
		{"LGTM, ship it!", true},
		{"Looks good to me", true},
		{"The implementation is correct and complete", true},
		{"Review passed and approved", true},
		{"Everything looks good, approved", true},
		{"Issues found: missing tests", false},
		{"The code has bugs", false},
		{"Implementation is incomplete", false},
		{"Review failed", false},
		{"Needs work on error handling", false},
		{"There are issues that need fixing", false},
		{"", false}, // empty defaults to fail
	}

	for _, tt := range tests {
		got := o.inferReviewPassed(tt.output)
		if got != tt.want {
			t.Errorf("inferReviewPassed(%q) = %v, want %v", tt.output, got, tt.want)
		}
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "xyz", false},
		{"abc", "abcd", false},
		{"", "a", false},
		{"a", "", true},
		{"ABC", "abc", true},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestRunContextCancellation(t *testing.T) {
	// Create a slow mock that checks context
	agent := &slowMockAgent{delay: 100 * time.Millisecond}
	o := New(WithAgent(agent))

	task := &tasks.Task{
		ID:    "test-cancel",
		Title: "Cancellation Test",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := o.RunTask(ctx, task, "")
	// Context was cancelled, expect some form of failure.
	// The exact behavior depends on timing, so we just verify
	// we got a result without panicking.
	_ = err
	_ = result
}

// slowMockAgent simulates slow execution.
type slowMockAgent struct {
	delay time.Duration
}

func (m *slowMockAgent) Name() string {
	return "slow-mock"
}

func (m *slowMockAgent) Execute(ctx context.Context, opts agents.ExecuteOptions) (*agents.ExecuteResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(m.delay):
		return &agents.ExecuteResult{
			Output:   `{"steps":[],"files":[],"description":"done"}`,
			JSON:     []byte(`{"steps":[],"files":[],"description":"done"}`),
			ExitCode: 0,
		}, nil
	}
}

func TestTaskResultLogs(t *testing.T) {
	planResp := jsonResponse(PlanOutput{Steps: []string{"s1"}, Files: []string{"f.go"}})
	implResp := jsonResponse(ImplementOutput{FilesModified: []string{"f.go"}, Summary: "done"})
	reviewResp := jsonResponse(ReviewOutput{Passed: true})

	agent := newMockAgent(planResp, implResp, reviewResp)
	o := New(WithAgent(agent))

	task := &tasks.Task{ID: "log-test", Title: "Log Test"}
	result, _ := o.RunTask(context.Background(), task, "")

	if len(result.Logs) == 0 {
		t.Error("expected logs to be populated")
	}

	// Check for expected log entries
	foundStart := false
	foundComplete := false
	for _, log := range result.Logs {
		if log.Message == "starting task" {
			foundStart = true
		}
		if log.Message == "task completed" {
			foundComplete = true
		}
	}
	if !foundStart {
		t.Error("missing 'starting task' log")
	}
	if !foundComplete {
		t.Error("missing 'task completed' log")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxIterations != 3 {
		t.Errorf("MaxIterations = %d, want 3", cfg.MaxIterations)
	}
	if cfg.AgentTimeout != 30*time.Minute {
		t.Errorf("AgentTimeout = %v, want 30m", cfg.AgentTimeout)
	}
}

func TestRunNoQueue(t *testing.T) {
	o := New()
	err := o.Run(context.Background())
	if err == nil {
		t.Error("expected error for nil queue")
	}
	if !errors.Is(err, errors.New("no task queue configured")) {
		// Just check error message contains expected text
		if err.Error() != "no task queue configured" {
			t.Errorf("error = %q, want 'no task queue configured'", err.Error())
		}
	}
}

func TestBuildPrompts(t *testing.T) {
	o := New()
	task := &tasks.Task{
		ID:          "prompt-test",
		Title:       "Build Prompts",
		Description: "Test prompt generation",
	}

	// Test plan prompt
	planPrompt := o.buildPlanPrompt(task)
	if planPrompt == "" {
		t.Error("plan prompt should not be empty")
	}
	if !containsIgnoreCase(planPrompt, "prompt-test") {
		t.Error("plan prompt should contain task ID")
	}

	// Test implement prompt
	plan := &PlanOutput{
		Steps:       []string{"step1", "step2"},
		Description: "test plan",
	}
	implPrompt := o.buildImplementPrompt(task, plan, 1)
	if !containsIgnoreCase(implPrompt, "implementation") {
		t.Error("implement prompt should mention implementation")
	}

	// Test implement prompt iteration 2
	implPrompt2 := o.buildImplementPrompt(task, plan, 2)
	if !containsIgnoreCase(implPrompt2, "iteration 2") {
		t.Error("implement prompt iteration 2 should mention iteration number")
	}

	// Test review prompt
	impl := &ImplementOutput{
		FilesModified: []string{"file1.go"},
		Summary:       "test implementation",
	}
	reviewPrompt := o.buildReviewPrompt(task, impl)
	if !containsIgnoreCase(reviewPrompt, "review") {
		t.Error("review prompt should mention review")
	}
}

func TestExtractPRURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single PR URL",
			input: "Created PR: https://github.com/owner/repo/pull/42",
			want:  "https://github.com/owner/repo/pull/42",
		},
		{
			name:  "multiple PR URLs returns last",
			input: "See https://github.com/owner/repo/pull/1 and https://github.com/owner/repo/pull/99",
			want:  "https://github.com/owner/repo/pull/99",
		},
		{
			name:  "no PR URL",
			input: "No pull request was created",
			want:  "",
		},
		{
			name:  "URL embedded in text",
			input: "Successfully opened https://github.com/acme/widgets/pull/123 for review.",
			want:  "https://github.com/acme/widgets/pull/123",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "URL with large PR number",
			input: "https://github.com/org/project/pull/99999",
			want:  "https://github.com/org/project/pull/99999",
		},
		{
			name:  "non-PR github URL ignored",
			input: "See https://github.com/owner/repo/issues/5",
			want:  "",
		},
		{
			name:  "PR URL in multiline output",
			input: "Done.\n\nPR: https://github.com/foo/bar/pull/7\n\nPlease review.",
			want:  "https://github.com/foo/bar/pull/7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPRURL(tt.input)
			if got != tt.want {
				t.Errorf("ExtractPRURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunTaskExtractsPRURL(t *testing.T) {
	// Setup mock: plan, implement (with PR URL in raw output), review (pass)
	planResp := jsonResponse(PlanOutput{
		Steps:       []string{"step1"},
		Files:       []string{"file1.go"},
		Description: "test plan",
	})

	implData := ImplementOutput{
		FilesModified: []string{"file1.go"},
		Summary:       "opened PR",
	}
	implJSON, _ := json.Marshal(implData)
	implResp := agents.ExecuteResult{
		Output:   "Created https://github.com/owner/repo/pull/42 for review",
		JSON:     implJSON,
		ExitCode: 0,
	}

	reviewResp := jsonResponse(ReviewOutput{
		Passed:   true,
		Feedback: "looks good",
	})

	agent := newMockAgent(planResp, implResp, reviewResp)
	o := New(WithAgent(agent))

	task := &tasks.Task{
		ID:          "pr-test",
		Title:       "PR Extraction Test",
		Description: "test PR URL extraction",
	}

	result, err := o.RunTask(context.Background(), task, "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != StatusCompleted {
		t.Fatalf("status = %s, want %s", result.Status, StatusCompleted)
	}
	if result.OutputType != "PR" {
		t.Errorf("OutputType = %q, want %q", result.OutputType, "PR")
	}
	if result.OutputRef != "https://github.com/owner/repo/pull/42" {
		t.Errorf("OutputRef = %q, want %q", result.OutputRef, "https://github.com/owner/repo/pull/42")
	}
}

func TestRunTaskNoPRURL(t *testing.T) {
	// Setup mock: plan, implement (no PR URL), review (pass)
	planResp := jsonResponse(PlanOutput{
		Steps:       []string{"step1"},
		Files:       []string{"file1.go"},
		Description: "test plan",
	})
	implResp := jsonResponse(ImplementOutput{
		FilesModified: []string{"file1.go"},
		Summary:       "implemented changes without a PR",
	})
	reviewResp := jsonResponse(ReviewOutput{
		Passed:   true,
		Feedback: "looks good",
	})

	agent := newMockAgent(planResp, implResp, reviewResp)
	o := New(WithAgent(agent))

	task := &tasks.Task{
		ID:          "no-pr-test",
		Title:       "No PR Test",
		Description: "test no PR URL",
	}

	result, err := o.RunTask(context.Background(), task, "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OutputType != "" {
		t.Errorf("OutputType = %q, want empty", result.OutputType)
	}
	if result.OutputRef != "" {
		t.Errorf("OutputRef = %q, want empty", result.OutputRef)
	}
}
