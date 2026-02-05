package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
		return
	}

	if m.width != 80 {
		t.Errorf("expected width 80, got %d", m.width)
	}
	if m.height != 24 {
		t.Errorf("expected height 24, got %d", m.height)
	}
	if m.activePanel != PanelStatus {
		t.Errorf("expected activePanel PanelStatus, got %d", m.activePanel)
	}
	if m.daemonStatus != StatusIdle {
		t.Errorf("expected daemonStatus StatusIdle, got %d", m.daemonStatus)
	}
	if m.styles == nil {
		t.Error("expected styles to be initialized")
	}
}

func TestSetters(t *testing.T) {
	m := New()

	// Test SetDaemonStatus
	m.SetDaemonStatus(StatusRunning)
	if m.daemonStatus != StatusRunning {
		t.Errorf("expected daemonStatus StatusRunning, got %d", m.daemonStatus)
	}

	// Test SetCurrentTask
	m.SetCurrentTask("test-task")
	if m.currentTask != "test-task" {
		t.Errorf("expected currentTask 'test-task', got %s", m.currentTask)
	}

	// Test SetBudget
	m.SetBudget(50000, 100000)
	if m.budgetUsed != 50000 {
		t.Errorf("expected budgetUsed 50000, got %d", m.budgetUsed)
	}
	if m.budgetTotal != 100000 {
		t.Errorf("expected budgetTotal 100000, got %d", m.budgetTotal)
	}

	// Test SetLastRunTime
	now := time.Now()
	m.SetLastRunTime(now)
	if !m.lastRunTime.Equal(now) {
		t.Error("expected lastRunTime to match")
	}

	// Test SetTokensUsed
	m.SetTokensUsed(25000)
	if m.tokensUsed != 25000 {
		t.Errorf("expected tokensUsed 25000, got %d", m.tokensUsed)
	}
}

func TestTaskManagement(t *testing.T) {
	m := New()

	// Test AddTask
	task := TaskItem{
		ID:     "task-1",
		Name:   "Test Task",
		Status: TaskPending,
	}
	m.AddTask(task)
	if len(m.tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(m.tasks))
	}
	if m.tasks[0].ID != "task-1" {
		t.Errorf("expected task ID 'task-1', got %s", m.tasks[0].ID)
	}

	// Test UpdateTask
	m.UpdateTask("task-1", TaskRunning, 50)
	if m.tasks[0].Status != TaskRunning {
		t.Errorf("expected task status TaskRunning, got %d", m.tasks[0].Status)
	}
	if m.tasks[0].Progress != 50 {
		t.Errorf("expected task progress 50, got %d", m.tasks[0].Progress)
	}

	// Test ClearTasks
	m.ClearTasks()
	if len(m.tasks) != 0 {
		t.Errorf("expected 0 tasks after clear, got %d", len(m.tasks))
	}
}

func TestLogManagement(t *testing.T) {
	m := New()

	// Test AddLog
	m.AddLog("info", "Test message")
	if len(m.logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(m.logs))
	}
	if m.logs[0].Level != "info" {
		t.Errorf("expected log level 'info', got %s", m.logs[0].Level)
	}
	if m.logs[0].Message != "Test message" {
		t.Errorf("expected log message 'Test message', got %s", m.logs[0].Message)
	}

	// Test ClearLogs
	m.ClearLogs()
	if len(m.logs) != 0 {
		t.Errorf("expected 0 logs after clear, got %d", len(m.logs))
	}
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestUpdateWindowSize(t *testing.T) {
	m := New()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := model.(Model)

	if updated.width != 120 {
		t.Errorf("expected width 120, got %d", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("expected height 40, got %d", updated.height)
	}
}

func TestKeyHandlingQuit(t *testing.T) {
	m := New()
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(Model)

	if !updated.quitting {
		t.Error("expected quitting to be true after 'q' key")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestKeyHandlingPanelSwitch(t *testing.T) {
	m := New()

	// Tab should switch panels
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := model.(Model)
	if updated.activePanel != PanelTasks {
		t.Errorf("expected PanelTasks after tab, got %d", updated.activePanel)
	}

	// Another tab
	model, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated = model.(Model)
	if updated.activePanel != PanelLogs {
		t.Errorf("expected PanelLogs after second tab, got %d", updated.activePanel)
	}

	// Another tab should cycle back
	model, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated = model.(Model)
	if updated.activePanel != PanelStatus {
		t.Errorf("expected PanelStatus after third tab, got %d", updated.activePanel)
	}
}

func TestView(t *testing.T) {
	m := New()
	m.SetDaemonStatus(StatusRunning)
	m.SetCurrentTask("lint-fix")
	m.SetBudget(30000, 100000)
	m.AddTask(TaskItem{ID: "1", Name: "Lint Fix", Status: TaskRunning, Progress: 50})
	m.AddLog("info", "Starting task")

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string")
	}

	// Basic content checks
	if !containsAny(view, "Nightshift", "Status") {
		t.Error("View missing status panel content")
	}
	if !containsAny(view, "Tasks") {
		t.Error("View missing task panel content")
	}
	if !containsAny(view, "Logs") {
		t.Error("View missing log panel content")
	}
}

func TestViewWhenQuitting(t *testing.T) {
	m := New()
	m.quitting = true
	view := m.View()
	if view != "" {
		t.Error("View() should return empty string when quitting")
	}
}

func TestStatusStrings(t *testing.T) {
	tests := []struct {
		status   DaemonStatus
		expected string
	}{
		{StatusStopped, "Stopped"},
		{StatusRunning, "Running"},
		{StatusIdle, "Idle"},
		{DaemonStatus(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("DaemonStatus(%d).String() = %s, want %s", tt.status, got, tt.expected)
		}
	}
}

func TestTaskStatusStrings(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskPending, "pending"},
		{TaskRunning, "running"},
		{TaskCompleted, "done"},
		{TaskFailed, "failed"},
		{TaskStatus(99), "?"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("TaskStatus(%d).String() = %s, want %s", tt.status, got, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m"},
		{3 * time.Hour, "3h"},
		{48 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		if got := formatDuration(tt.duration); got != tt.expected {
			t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, got, tt.expected)
		}
	}
}

func TestSpinner(t *testing.T) {
	m := New()
	frames := []string{"|", "/", "-", "\\"}

	for i := 0; i < 8; i++ {
		m.progressTick = i
		got := m.spinner()
		expected := frames[i%4]
		if got != expected {
			t.Errorf("spinner at tick %d = %s, want %s", i, got, expected)
		}
	}
}

func TestProgressBar(t *testing.T) {
	m := New()

	// Test various percentages
	bar0 := m.renderProgressBar(0, 20)
	if !containsAny(bar0, "[", "]") {
		t.Error("Progress bar missing brackets")
	}

	bar50 := m.renderProgressBar(50, 20)
	if !containsAny(bar50, "=", "-") {
		t.Error("Progress bar missing fill characters")
	}

	bar100 := m.renderProgressBar(100, 20)
	if !containsAny(bar100, "=") {
		t.Error("Full progress bar should have fill")
	}
}

func TestHandleNavigation(t *testing.T) {
	m := New()
	m.activePanel = PanelTasks
	m.AddTask(TaskItem{ID: "1", Name: "Task 1"})
	m.AddTask(TaskItem{ID: "2", Name: "Task 2"})
	m.AddTask(TaskItem{ID: "3", Name: "Task 3"})

	// Down navigation
	result := m.handleDown()
	if result.selectedTask != 1 {
		t.Errorf("expected selectedTask 1 after down, got %d", result.selectedTask)
	}

	// Up navigation
	result = result.handleUp()
	if result.selectedTask != 0 {
		t.Errorf("expected selectedTask 0 after up, got %d", result.selectedTask)
	}

	// Home navigation
	result.selectedTask = 2
	result = result.handleHome()
	if result.selectedTask != 0 {
		t.Errorf("expected selectedTask 0 after home, got %d", result.selectedTask)
	}

	// End navigation
	result = result.handleEnd()
	if result.selectedTask != 2 {
		t.Errorf("expected selectedTask 2 after end, got %d", result.selectedTask)
	}
}

// containsAny checks if s contains any of the given substrings.
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(substr) > 0 && len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
