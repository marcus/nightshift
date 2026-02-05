// Package ui provides a terminal UI for monitoring nightshift runs.
// Uses Bubbletea for interactive display of progress and logs.
package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Panel represents which panel is currently focused.
type Panel int

const (
	PanelStatus Panel = iota
	PanelTasks
	PanelLogs
)

// DaemonStatus represents the daemon's current state.
type DaemonStatus int

const (
	StatusStopped DaemonStatus = iota
	StatusRunning
	StatusIdle
)

func (s DaemonStatus) String() string {
	switch s {
	case StatusStopped:
		return "Stopped"
	case StatusRunning:
		return "Running"
	case StatusIdle:
		return "Idle"
	default:
		return "Unknown"
	}
}

// TaskStatus represents a task's current state.
type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)

func (s TaskStatus) String() string {
	switch s {
	case TaskPending:
		return "pending"
	case TaskRunning:
		return "running"
	case TaskCompleted:
		return "done"
	case TaskFailed:
		return "failed"
	default:
		return "?"
	}
}

// TaskItem represents a task in the task list.
type TaskItem struct {
	ID       string
	Name     string
	Status   TaskStatus
	Progress int // 0-100
}

// LogEntry represents a log line.
type LogEntry struct {
	Time    time.Time
	Level   string
	Message string
}

// Model holds the TUI state.
type Model struct {
	// Display state
	width       int
	height      int
	activePanel Panel
	quitting    bool

	// Status panel
	daemonStatus  DaemonStatus
	currentTask   string
	budgetUsed    int64
	budgetTotal   int64
	lastRunTime   time.Time
	tokensUsed    int64

	// Task list
	tasks        []TaskItem
	taskScroll   int
	selectedTask int

	// Logs
	logs       []LogEntry
	logScroll  int

	// Progress
	progressTick int

	// Styles
	styles *Styles
}

// Styles holds lipgloss styles for the UI.
type Styles struct {
	// Panel borders
	ActiveBorder   lipgloss.Style
	InactiveBorder lipgloss.Style

	// Text styles
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Highlight   lipgloss.Style
	Muted       lipgloss.Style

	// Status indicators
	StatusOK      lipgloss.Style
	StatusWarn    lipgloss.Style
	StatusError   lipgloss.Style
	StatusRunning lipgloss.Style

	// Task list
	TaskSelected lipgloss.Style
	TaskNormal   lipgloss.Style

	// Log levels
	LogDebug lipgloss.Style
	LogInfo  lipgloss.Style
	LogWarn  lipgloss.Style
	LogError lipgloss.Style

	// Help bar
	HelpKey  lipgloss.Style
	HelpText lipgloss.Style
}

// newStyles creates the default style set.
func newStyles() *Styles {
	subtle := lipgloss.AdaptiveColor{Light: "#666", Dark: "#888"}
	highlight := lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	green := lipgloss.AdaptiveColor{Light: "#22863a", Dark: "#3fb950"}
	yellow := lipgloss.AdaptiveColor{Light: "#b08800", Dark: "#d29922"}
	red := lipgloss.AdaptiveColor{Light: "#cb2431", Dark: "#f85149"}
	blue := lipgloss.AdaptiveColor{Light: "#0366d6", Dark: "#58a6ff"}

	return &Styles{
		ActiveBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight),

		InactiveBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#333", Dark: "#ccc"}),

		Label: lipgloss.NewStyle().
			Foreground(subtle),

		Value: lipgloss.NewStyle().
			Bold(true),

		Highlight: lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true),

		Muted: lipgloss.NewStyle().
			Foreground(subtle),

		StatusOK: lipgloss.NewStyle().
			Foreground(green).
			Bold(true),

		StatusWarn: lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true),

		StatusError: lipgloss.NewStyle().
			Foreground(red).
			Bold(true),

		StatusRunning: lipgloss.NewStyle().
			Foreground(blue).
			Bold(true),

		TaskSelected: lipgloss.NewStyle().
			Background(highlight).
			Foreground(lipgloss.Color("#fff")).
			Bold(true),

		TaskNormal: lipgloss.NewStyle(),

		LogDebug: lipgloss.NewStyle().Foreground(subtle),
		LogInfo:  lipgloss.NewStyle().Foreground(blue),
		LogWarn:  lipgloss.NewStyle().Foreground(yellow),
		LogError: lipgloss.NewStyle().Foreground(red),

		HelpKey: lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true),

		HelpText: lipgloss.NewStyle().
			Foreground(subtle),
	}
}

// tickMsg is sent periodically to update the UI.
type tickMsg time.Time

// New creates a new TUI model.
func New() *Model {
	return &Model{
		width:        80,
		height:       24,
		activePanel:  PanelStatus,
		daemonStatus: StatusIdle,
		budgetTotal:  100000,
		tasks:        make([]TaskItem, 0),
		logs:         make([]LogEntry, 0),
		styles:       newStyles(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

// tickCmd returns a command that ticks every second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.progressTick++
		return m, tickCmd()
	}

	return m, nil
}

// handleKey processes keyboard input.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "tab", "right", "l":
		m.activePanel = (m.activePanel + 1) % 3
		return m, nil

	case "shift+tab", "left", "h":
		m.activePanel = (m.activePanel + 2) % 3
		return m, nil

	case "up", "k":
		return m.handleUp(), nil

	case "down", "j":
		return m.handleDown(), nil

	case "home", "g":
		return m.handleHome(), nil

	case "end", "G":
		return m.handleEnd(), nil
	}

	return m, nil
}

// handleUp handles up arrow / k key.
func (m Model) handleUp() Model {
	switch m.activePanel {
	case PanelTasks:
		if m.selectedTask > 0 {
			m.selectedTask--
		}
	case PanelLogs:
		if m.logScroll > 0 {
			m.logScroll--
		}
	}
	return m
}

// handleDown handles down arrow / j key.
func (m Model) handleDown() Model {
	switch m.activePanel {
	case PanelTasks:
		if m.selectedTask < len(m.tasks)-1 {
			m.selectedTask++
		}
	case PanelLogs:
		maxScroll := len(m.logs) - 1
		if m.logScroll < maxScroll {
			m.logScroll++
		}
	}
	return m
}

// handleHome handles home / g key.
func (m Model) handleHome() Model {
	switch m.activePanel {
	case PanelTasks:
		m.selectedTask = 0
	case PanelLogs:
		m.logScroll = 0
	}
	return m
}

// handleEnd handles end / G key.
func (m Model) handleEnd() Model {
	switch m.activePanel {
	case PanelTasks:
		if len(m.tasks) > 0 {
			m.selectedTask = len(m.tasks) - 1
		}
	case PanelLogs:
		if len(m.logs) > 0 {
			m.logScroll = len(m.logs) - 1
		}
	}
	return m
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Calculate panel dimensions
	topHeight := m.height / 2
	bottomHeight := m.height - topHeight - 3 // -3 for help bar and padding
	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	// Build panels
	statusPanel := m.renderStatusPanel(leftWidth-2, topHeight-2)
	taskPanel := m.renderTaskPanel(rightWidth-2, topHeight-2)
	logPanel := m.renderLogPanel(m.width-2, bottomHeight-2)

	// Apply borders
	statusBorder := m.getBorder(PanelStatus).Width(leftWidth - 2).Height(topHeight - 2)
	taskBorder := m.getBorder(PanelTasks).Width(rightWidth - 2).Height(topHeight - 2)
	logBorder := m.getBorder(PanelLogs).Width(m.width - 2).Height(bottomHeight - 2)

	// Layout
	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		statusBorder.Render(statusPanel),
		taskBorder.Render(taskPanel),
	)

	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		topRow,
		logBorder.Render(logPanel),
		helpBar,
	)
}

// getBorder returns the appropriate border style for a panel.
func (m Model) getBorder(panel Panel) lipgloss.Style {
	if m.activePanel == panel {
		return m.styles.ActiveBorder
	}
	return m.styles.InactiveBorder
}

// renderStatusPanel renders the status panel content.
func (m Model) renderStatusPanel(width, height int) string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render("Nightshift Status"))
	b.WriteString("\n\n")

	// Daemon status
	statusStyle := m.styles.StatusOK
	statusText := "Running"
	switch m.daemonStatus {
	case StatusStopped:
		statusStyle = m.styles.StatusError
		statusText = "Stopped"
	case StatusIdle:
		statusStyle = m.styles.StatusWarn
		statusText = "Idle"
	case StatusRunning:
		statusStyle = m.styles.StatusRunning
		statusText = "Running"
	}

	b.WriteString(m.styles.Label.Render("Daemon: "))
	b.WriteString(statusStyle.Render(statusText))
	b.WriteString("\n\n")

	// Current task
	b.WriteString(m.styles.Label.Render("Task: "))
	if m.currentTask != "" {
		b.WriteString(m.styles.Value.Render(m.currentTask))
	} else {
		b.WriteString(m.styles.Muted.Render("None"))
	}
	b.WriteString("\n\n")

	// Budget
	b.WriteString(m.styles.Label.Render("Budget: "))
	budgetPct := float64(0)
	if m.budgetTotal > 0 {
		budgetPct = float64(m.budgetUsed) / float64(m.budgetTotal) * 100
	}
	remaining := m.budgetTotal - m.budgetUsed
	budgetStr := fmt.Sprintf("%dk / %dk (%.0f%% used, %dk remaining)",
		m.budgetUsed/1000, m.budgetTotal/1000, budgetPct, remaining/1000)
	b.WriteString(m.styles.Value.Render(budgetStr))
	b.WriteString("\n\n")

	// Progress bar for budget
	b.WriteString(m.renderProgressBar(int(budgetPct), width-4))
	b.WriteString("\n\n")

	// Last run
	b.WriteString(m.styles.Label.Render("Last Run: "))
	if !m.lastRunTime.IsZero() {
		ago := time.Since(m.lastRunTime)
		b.WriteString(m.styles.Value.Render(formatDuration(ago) + " ago"))
	} else {
		b.WriteString(m.styles.Muted.Render("Never"))
	}
	b.WriteString("\n")

	// Tokens used today
	b.WriteString(m.styles.Label.Render("Tokens Today: "))
	b.WriteString(m.styles.Value.Render(fmt.Sprintf("%dk", m.tokensUsed/1000)))

	return b.String()
}

// renderProgressBar renders a progress bar.
func (m Model) renderProgressBar(pct, width int) string {
	if width < 10 {
		width = 10
	}

	filled := width * pct / 100
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("=", filled) + strings.Repeat("-", width-filled)

	// Color based on percentage
	style := m.styles.StatusOK
	if pct > 80 {
		style = m.styles.StatusError
	} else if pct > 50 {
		style = m.styles.StatusWarn
	}

	return "[" + style.Render(bar) + "]"
}

// renderTaskPanel renders the task list panel.
func (m Model) renderTaskPanel(width, height int) string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render("Tasks"))
	b.WriteString("\n\n")

	if len(m.tasks) == 0 {
		b.WriteString(m.styles.Muted.Render("No tasks queued"))
		return b.String()
	}

	// Calculate visible tasks
	visibleTasks := height - 4 // Account for title and padding
	if visibleTasks < 1 {
		visibleTasks = 1
	}

	// Adjust scroll if selected task is out of view
	if m.selectedTask < m.taskScroll {
		m.taskScroll = m.selectedTask
	} else if m.selectedTask >= m.taskScroll+visibleTasks {
		m.taskScroll = m.selectedTask - visibleTasks + 1
	}

	// Render visible tasks
	for i := m.taskScroll; i < len(m.tasks) && i < m.taskScroll+visibleTasks; i++ {
		task := m.tasks[i]

		// Status indicator
		var statusIcon string
		var statusStyle lipgloss.Style
		switch task.Status {
		case TaskPending:
			statusIcon = "o"
			statusStyle = m.styles.Muted
		case TaskRunning:
			statusIcon = m.spinner()
			statusStyle = m.styles.StatusRunning
		case TaskCompleted:
			statusIcon = "*"
			statusStyle = m.styles.StatusOK
		case TaskFailed:
			statusIcon = "x"
			statusStyle = m.styles.StatusError
		}

		// Build task line
		line := fmt.Sprintf(" %s %s", statusStyle.Render(statusIcon), task.Name)

		// Highlight selected task
		if i == m.selectedTask && m.activePanel == PanelTasks {
			line = m.styles.TaskSelected.Render(line)
		}

		// Add progress if running
		if task.Status == TaskRunning && task.Progress > 0 {
			line += m.styles.Muted.Render(fmt.Sprintf(" (%d%%)", task.Progress))
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.tasks) > visibleTasks {
		scrollInfo := fmt.Sprintf(" [%d/%d]", m.taskScroll+1, len(m.tasks))
		b.WriteString(m.styles.Muted.Render(scrollInfo))
	}

	return b.String()
}

// spinner returns a spinner character based on the current tick.
func (m Model) spinner() string {
	frames := []string{"|", "/", "-", "\\"}
	return frames[m.progressTick%len(frames)]
}

// renderLogPanel renders the log viewer panel.
func (m Model) renderLogPanel(width, height int) string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render("Logs"))
	b.WriteString("\n\n")

	if len(m.logs) == 0 {
		b.WriteString(m.styles.Muted.Render("No logs yet"))
		return b.String()
	}

	// Calculate visible logs
	visibleLogs := height - 4
	if visibleLogs < 1 {
		visibleLogs = 1
	}

	// Render visible logs
	start := m.logScroll
	if start+visibleLogs > len(m.logs) {
		start = len(m.logs) - visibleLogs
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < len(m.logs) && i < start+visibleLogs; i++ {
		entry := m.logs[i]

		// Time
		timeStr := entry.Time.Format("15:04:05")

		// Level with color
		var levelStyle lipgloss.Style
		switch entry.Level {
		case "debug":
			levelStyle = m.styles.LogDebug
		case "info":
			levelStyle = m.styles.LogInfo
		case "warn":
			levelStyle = m.styles.LogWarn
		case "error":
			levelStyle = m.styles.LogError
		default:
			levelStyle = m.styles.Muted
		}

		// Truncate message if needed
		maxMsgLen := width - 20
		msg := entry.Message
		if len(msg) > maxMsgLen && maxMsgLen > 3 {
			msg = msg[:maxMsgLen-3] + "..."
		}

		line := fmt.Sprintf("%s %s %s",
			m.styles.Muted.Render(timeStr),
			levelStyle.Render(fmt.Sprintf("[%-5s]", entry.Level)),
			msg,
		)

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.logs) > visibleLogs {
		scrollInfo := fmt.Sprintf(" [%d/%d]", m.logScroll+1, len(m.logs))
		b.WriteString(m.styles.Muted.Render(scrollInfo))
	}

	return b.String()
}

// renderHelpBar renders the help bar at the bottom.
func (m Model) renderHelpBar() string {
	helpItems := []struct {
		key  string
		desc string
	}{
		{"tab", "switch panel"},
		{"j/k", "up/down"},
		{"q", "quit"},
	}

	var parts []string
	for _, item := range helpItems {
		parts = append(parts, fmt.Sprintf("%s %s",
			m.styles.HelpKey.Render(item.key),
			m.styles.HelpText.Render(item.desc),
		))
	}

	return "  " + strings.Join(parts, "  |  ")
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// SetDaemonStatus updates the daemon status.
func (m *Model) SetDaemonStatus(status DaemonStatus) {
	m.daemonStatus = status
}

// SetCurrentTask updates the current task name.
func (m *Model) SetCurrentTask(task string) {
	m.currentTask = task
}

// SetBudget updates the budget values.
func (m *Model) SetBudget(used, total int64) {
	m.budgetUsed = used
	m.budgetTotal = total
}

// SetLastRunTime updates the last run time.
func (m *Model) SetLastRunTime(t time.Time) {
	m.lastRunTime = t
}

// SetTokensUsed updates the tokens used today.
func (m *Model) SetTokensUsed(tokens int64) {
	m.tokensUsed = tokens
}

// AddTask adds a task to the task list.
func (m *Model) AddTask(task TaskItem) {
	m.tasks = append(m.tasks, task)
}

// UpdateTask updates an existing task by ID.
func (m *Model) UpdateTask(id string, status TaskStatus, progress int) {
	for i := range m.tasks {
		if m.tasks[i].ID == id {
			m.tasks[i].Status = status
			m.tasks[i].Progress = progress
			break
		}
	}
}

// ClearTasks removes all tasks.
func (m *Model) ClearTasks() {
	m.tasks = make([]TaskItem, 0)
	m.selectedTask = 0
	m.taskScroll = 0
}

// AddLog adds a log entry.
func (m *Model) AddLog(level, message string) {
	m.logs = append(m.logs, LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: message,
	})
	// Auto-scroll to bottom if not actively scrolling
	if m.logScroll == len(m.logs)-2 || len(m.logs) == 1 {
		m.logScroll = len(m.logs) - 1
	}
}

// ClearLogs removes all logs.
func (m *Model) ClearLogs() {
	m.logs = make([]LogEntry, 0)
	m.logScroll = 0
}

// Run starts the TUI.
func (m *Model) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunWithProgram starts the TUI and returns the program for external control.
func (m *Model) RunWithProgram() (*tea.Program, error) {
	p := tea.NewProgram(m, tea.WithAltScreen())
	go func() {
		_, _ = p.Run()
	}()
	return p, nil
}
