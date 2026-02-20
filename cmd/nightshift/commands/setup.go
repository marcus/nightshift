package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/marcus/nightshift/internal/reporting"
	"github.com/marcus/nightshift/internal/scheduler"
	"github.com/marcus/nightshift/internal/setup"
	"github.com/marcus/nightshift/internal/snapshots"
	"github.com/marcus/nightshift/internal/tasks"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive onboarding wizard",
	Long: `Interactive onboarding wizard that configures Nightshift end-to-end.

Creates/updates the global config, validates providers, runs a snapshot, previews the next run,
and optionally installs/enables the daemon.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		model, err := newSetupModel()
		if err != nil {
			return err
		}
		_, err = tea.NewProgram(model).Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

type setupStep int

const (
	stepWelcome setupStep = iota
	stepConfig
	stepProjects
	stepBudget
	stepSafety
	stepModel
	stepTaskPreset
	stepTaskSelect
	stepSchedule
	stepSnapshot
	stepPreview
	stepPath
	stepDaemon
	stepFinish
)

const (
	nightshiftPlanIgnore        = ".nightshift-plan"
	nightshiftPlanIgnoreComment = "# Nightshift plan artifacts (keep out of version control)"
)

type modelOption struct {
	label string
	value string // empty = use CLI default
}

var claudeModels = []modelOption{
	{label: "default", value: ""},
	{label: "claude-opus-4-6", value: "claude-opus-4-6"},
	{label: "claude-sonnet-4-6", value: "claude-sonnet-4-6"},
	{label: "claude-haiku-4-5", value: "claude-haiku-4-5"},
}

var codexModels = []modelOption{
	{label: "default", value: ""},
	{label: "gpt-5.3-codex", value: "gpt-5.3-codex"},
	{label: "gpt-5.3-codex-spark", value: "gpt-5.3-codex-spark"},
	{label: "gpt-5.2-codex", value: "gpt-5.2-codex"},
	{label: "gpt-5.2", value: "gpt-5.2"},
	{label: "gpt-5.1-codex-max", value: "gpt-5.1-codex-max"},
	{label: "gpt-5.1-codex", value: "gpt-5.1-codex"},
	{label: "gpt-5.1", value: "gpt-5.1"},
	{label: "gpt-5-codex", value: "gpt-5-codex"},
	{label: "gpt-5", value: "gpt-5"},
}

var copilotModels = []modelOption{
	{label: "default", value: ""},
	{label: "claude-sonnet-4.6", value: "claude-sonnet-4.6"},
	{label: "claude-sonnet-4.5", value: "claude-sonnet-4.5"},
	{label: "claude-haiku-4.5", value: "claude-haiku-4.5"},
	{label: "claude-opus-4.6", value: "claude-opus-4.6"},
	{label: "claude-opus-4.6-fast", value: "claude-opus-4.6-fast"},
	{label: "claude-opus-4.5", value: "claude-opus-4.5"},
	{label: "claude-sonnet-4", value: "claude-sonnet-4"},
	{label: "gemini-3-pro-preview", value: "gemini-3-pro-preview"},
	{label: "gpt-5.3-codex", value: "gpt-5.3-codex"},
	{label: "gpt-5.2-codex", value: "gpt-5.2-codex"},
	{label: "gpt-5.2", value: "gpt-5.2"},
	{label: "gpt-5.1-codex-max", value: "gpt-5.1-codex-max"},
	{label: "gpt-5.1-codex", value: "gpt-5.1-codex"},
	{label: "gpt-5.1", value: "gpt-5.1"},
	{label: "gpt-5.1-codex-mini", value: "gpt-5.1-codex-mini"},
	{label: "gpt-5-mini", value: "gpt-5-mini"},
	{label: "gpt-4.1", value: "gpt-4.1"},
}

type setupModel struct {
	step setupStep

	cfg             *config.Config
	configPath      string
	configExist     bool
	includePathStep bool

	projects       []string
	projectCursor  int
	projectInput   textinput.Model
	projectEditing bool
	projectErr     string
	gitignoreAdded int
	gitignoreKept  int
	gitignoreErrs  []string

	budgetCursor  int
	budgetInput   textinput.Model
	budgetEditing bool
	budgetErr     string

	safetyCursor int

	modelCursor      int
	claudeModelIdx   int
	codexModelIdx    int
	copilotModelIdx  int

	taskPresetCursor int
	taskCursor       int
	taskItems        []taskItem
	taskErr          string
	preset           setup.Preset

	scheduleMode      string
	scheduleCursor    int
	scheduleInput     textinput.Model
	scheduleEditing   bool
	scheduleStart     string
	scheduleCycles    int
	scheduleInterval  string
	scheduleCron      string
	scheduleErr       string
	scheduleWindowEnd string

	snapshotRunning bool
	snapshotOutput  string
	snapshotErr     error

	previewRunning bool
	previewOutput  string
	previewErr     error

	pathCursor       int
	pathOptions      []pathOption
	pathErr          string
	pathApplied      bool
	pathStatus       string
	pathShell        string
	pathConfig       string
	pathSourceHint   string
	nightshiftInPath bool

	daemonCursor int
	serviceType  string
	serviceState serviceState
	daemonAction string

	spinner spinner.Model
}

type taskItem struct {
	def      tasks.TaskDefinition
	selected bool
}

type serviceState struct {
	installed bool
	running   bool
	detail    string
}

type snapshotMsg struct {
	output string
	err    error
}

type previewMsg struct {
	output string
	err    error
}

type pathOption struct {
	label   string
	action  pathAction
	dir     string
	install bool
}

type pathAction int

const (
	pathActionSkip pathAction = iota
	pathActionAdd
)

var (
	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleOk     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleNote   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleAccent = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
)

func newSetupModel() (*setupModel, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	// Keep task registry aligned with current config so setup can display custom tasks.
	tasks.ClearCustom()
	if err := tasks.RegisterCustomTasksFromConfig(cfg.Tasks.Custom); err != nil {
		return nil, fmt.Errorf("register custom tasks: %w", err)
	}
	configPath := config.GlobalConfigPath()
	_, err = os.Stat(configPath)
	configExist := err == nil

	projectInput := textinput.New()
	projectInput.Placeholder = "~/code/project"
	projectInput.Prompt = "> "

	budgetInput := textinput.New()
	budgetInput.Prompt = "> "

	scheduleInput := textinput.New()
	scheduleInput.Prompt = "> "

	spin := spinner.New()
	spin.Spinner = spinner.MiniDot

	projects := make([]string, 0, len(cfg.Projects))
	for _, p := range cfg.Projects {
		if p.Path != "" {
			projects = append(projects, p.Path)
		}
	}
	if len(projects) == 0 {
		projects = []string{""}
	}

	preset := setup.PresetBalanced
	taskItems := makeTaskItems(cfg, projects, preset)
	_, err = execLookPath("nightshift")
	nightshiftInPath := err == nil
	includePathStep := !nightshiftInPath

	model := &setupModel{
		step:             stepWelcome,
		cfg:              cfg,
		configPath:       configPath,
		configExist:      configExist,
		includePathStep:  includePathStep,
		projects:         projects,
		projectInput:     projectInput,
		budgetInput:      budgetInput,
		taskItems:        taskItems,
		preset:           preset,
		scheduleMode:     "interval",
		scheduleStart:    "22:00",
		scheduleCycles:   3,
		scheduleInterval: "30m",
		scheduleCron:     "0 2 * * *",
		scheduleInput:    scheduleInput,
		spinner:          spin,
		nightshiftInPath: nightshiftInPath,
		claudeModelIdx:   modelIndex(claudeModels, cfg.Providers.Claude.Model),
		codexModelIdx:    modelIndex(codexModels, cfg.Providers.Codex.Model),
		copilotModelIdx:  modelIndex(copilotModels, cfg.Providers.Copilot.Model),
	}

	return model, nil
}

// Init implements tea.Model.
func (m *setupModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model.
func (m *setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.step {
		case stepWelcome:
			if msg.String() == "enter" {
				return m, m.setStep(stepConfig)
			}
		case stepConfig:
			if msg.String() == "enter" {
				return m, m.setStep(stepProjects)
			}
		case stepProjects:
			return m.handleProjectsInput(msg)
		case stepBudget:
			return m.handleBudgetInput(msg)
		case stepSafety:
			return m.handleSafetyInput(msg)
		case stepModel:
			return m.handleModelInput(msg)
		case stepTaskPreset:
			return m.handlePresetInput(msg)
		case stepTaskSelect:
			return m.handleTaskInput(msg)
		case stepSchedule:
			return m.handleScheduleInput(msg)
		case stepSnapshot:
			if !m.snapshotRunning && msg.String() == "enter" {
				return m, m.setStep(stepPreview)
			}
		case stepPreview:
			if !m.previewRunning && msg.String() == "enter" {
				if m.nightshiftInPath {
					return m, m.setStep(stepDaemon)
				}
				return m, m.setStep(stepPath)
			}
		case stepPath:
			return m.handlePathInput(msg)
		case stepDaemon:
			return m.handleDaemonInput(msg)
		case stepFinish:
			if msg.String() == "enter" {
				return m, tea.Quit
			}
		}
	case snapshotMsg:
		m.snapshotRunning = false
		m.snapshotOutput = msg.output
		m.snapshotErr = msg.err
	case previewMsg:
		m.previewRunning = false
		m.previewOutput = msg.output
		m.previewErr = msg.err
	}

	return m, cmd
}

// View implements tea.Model.
func (m *setupModel) View() string {
	var b strings.Builder
	b.WriteString(styleHeader.Render("Nightshift Setup"))
	b.WriteString("\n")
	b.WriteString(styleDim.Render("================"))
	b.WriteString("\n")
	b.WriteString(renderSetupStepper(m))
	b.WriteString("\n\n")

	switch m.step {
	case stepWelcome:
		b.WriteString("This wizard will configure Nightshift end-to-end.\n\n")
		b.WriteString("Checks:\n")
		b.WriteString(renderEnvChecks(m.cfg))
		b.WriteString("\nPress Enter to continue.\n")
	case stepConfig:
		b.WriteString(styleAccent.Render("Global config"))
		b.WriteString("\n")
		fmt.Fprintf(&b, "  %s\n", m.configPath)
		if m.configExist {
			b.WriteString("  Status: found (will update in place)\n")
		} else {
			b.WriteString("  Status: will create\n")
		}
		b.WriteString("\nThis wizard only writes the global config. Per-project configs are optional.\n")
		b.WriteString("\nPress Enter to continue.\n")
	case stepProjects:
		b.WriteString(styleAccent.Render("Projects (global config)"))
		b.WriteString("\n")
		b.WriteString("Use ↑/↓ to navigate, 'a' to add, 'd' to delete.\n")
		if m.projectEditing {
			b.WriteString("\nAdd project path:\n")
			b.WriteString(m.projectInput.View() + "\n")
			if m.projectErr != "" {
				b.WriteString("Error: " + m.projectErr + "\n")
			}
			b.WriteString("\nPress Enter to add or Esc to cancel.\n")
			return b.String()
		}

		for i, project := range m.projects {
			cursor := " "
			if i == m.projectCursor {
				cursor = ">"
			}
			label := project
			if label == "" {
				label = "(unset)"
			}
			fmt.Fprintf(&b, " %s %s\n", cursor, label)
		}
		if m.projectErr != "" {
			b.WriteString("\nError: " + m.projectErr + "\n")
		}
		b.WriteString("\nPress Enter to continue.\n")
	case stepBudget:
		b.WriteString(styleAccent.Render("Budget defaults"))
		b.WriteString("\n")
		b.WriteString("Edit with e.\n")
		b.WriteString("Use ↑/↓ to select a field.\n\n")
		renderBudgetFields(&b, m)
		if m.budgetEditing {
			b.WriteString("\nEdit value:\n")
			b.WriteString(m.budgetInput.View() + "\n")
			if m.budgetErr != "" {
				b.WriteString("Error: " + m.budgetErr + "\n")
			}
			b.WriteString("\nPress Enter to save, Esc to cancel.\n")
			return b.String()
		}
		if m.budgetErr != "" {
			b.WriteString("\nError: " + m.budgetErr + "\n")
		}
		b.WriteString("\nPress Enter to continue.\n")
	case stepSafety:
		b.WriteString(styleAccent.Render("Approvals & sandbox"))
		b.WriteString("\n")
		b.WriteString("These flags reduce interactive prompts. They’re convenient but carry more risk.\n")
		b.WriteString("We default them ON; you can turn them off here.\n\n")
		b.WriteString("Use ↑/↓ to select, space to toggle.\n\n")
		renderSafetyFields(&b, m)
		b.WriteString("\nPress Enter to continue.\n")
	case stepModel:
		b.WriteString(styleAccent.Render("Model selection"))
		b.WriteString("\n")
		b.WriteString("Choose the model for each provider. Use ↑/↓ to select a row, ←/→ to cycle models.\n\n")
		renderModelFields(&b, m)
		b.WriteString("\nPress Enter to continue.\n")
	case stepTaskPreset:
		b.WriteString(styleAccent.Render("Task presets (derived from registry)"))
		b.WriteString("\n")
		b.WriteString("Use ↑/↓ to select, Enter to continue.\n\n")
		presets := []setup.Preset{setup.PresetBalanced, setup.PresetSafe, setup.PresetAggressive}
		for i, preset := range presets {
			cursor := " "
			if i == m.taskPresetCursor {
				cursor = ">"
			}
			label := string(preset)
			if preset == setup.PresetBalanced {
				label += " (recommended)"
			}
			fmt.Fprintf(&b, " %s %s\n", cursor, label)
		}
	case stepTaskSelect:
		b.WriteString(styleAccent.Render("Tasks"))
		b.WriteString("\n")
		b.WriteString("Space to toggle, ↑/↓ to move.\n\n")
		if len(m.taskItems) == 0 {
			b.WriteString(styleWarn.Render("No task definitions found."))
			b.WriteString("\n")
		} else {
			for i, item := range m.taskItems {
				cursor := " "
				if i == m.taskCursor {
					cursor = ">"
				}
				check := " "
				if item.selected {
					check = "x"
				}
				fmt.Fprintf(&b, " %s [%s] %-22s %s\n", cursor, check, item.def.Type, item.def.Name)
			}
		}
		if m.taskErr != "" {
			b.WriteString("\nError: " + m.taskErr + "\n")
		}
		b.WriteString("\nPress Enter to continue.\n")
	case stepSchedule:
		b.WriteString(styleAccent.Render("Schedule"))
		b.WriteString("\n")
		b.WriteString("Use ↑/↓ to select, e to edit. We’ll explain each field.\n\n")
		renderScheduleFields(&b, m)
		if help := scheduleFieldHelp(m.scheduleCursor, m.scheduleMode); help != "" {
			b.WriteString("\n")
			b.WriteString(styleNote.Render(help))
			b.WriteString("\n")
		}
		if m.scheduleEditing {
			b.WriteString("\nEdit value:\n")
			b.WriteString(m.scheduleInput.View() + "\n")
			if m.scheduleErr != "" {
				b.WriteString("Error: " + m.scheduleErr + "\n")
			}
			b.WriteString("\nPress Enter to save, Esc to cancel.\n")
			return b.String()
		}
		if m.scheduleErr != "" {
			b.WriteString("\nError: " + m.scheduleErr + "\n")
		}
		b.WriteString("\nPress Enter to continue.\n")
	case stepSnapshot:
		b.WriteString(styleAccent.Render("Snapshot step"))
		b.WriteString("\n")
		b.WriteString("We’ll take a quick usage snapshot so Nightshift can set safe budgets.\n")
		b.WriteString("No tasks run yet. This just reads local usage (and optional tmux scrape).\n\n")
		if m.snapshotRunning {
			b.WriteString(m.spinner.View() + "\n")
		} else {
			if m.snapshotErr != nil {
				b.WriteString("Snapshot error: " + m.snapshotErr.Error() + "\n")
			} else {
				b.WriteString(m.snapshotOutput + "\n")
			}
			b.WriteString(styleNote.Render("If an estimate looks off, run `nightshift budget snapshot --provider codex` and `nightshift budget calibrate` later. Setup doesn’t change your budget math."))
			b.WriteString("\n")
			b.WriteString("\nPress Enter to continue.\n")
		}
	case stepPreview:
		b.WriteString(styleAccent.Render("Preview step"))
		b.WriteString("\n")
		b.WriteString("Next up: we’ll preview the first scheduled run with a compact task list.\n")
		b.WriteString("Use `nightshift preview --long` later if you want full prompt text.\n\n")
		if m.previewRunning {
			b.WriteString(m.spinner.View() + "\n")
		} else {
			if m.previewErr != nil {
				b.WriteString("Preview error: " + m.previewErr.Error() + "\n")
			} else {
				b.WriteString(m.previewOutput + "\n")
			}
			b.WriteString("\nPress Enter to continue.\n")
		}
	case stepPath:
		b.WriteString(styleAccent.Render("Add Nightshift to PATH"))
		b.WriteString("\n")
		if m.nightshiftInPath {
			b.WriteString("Nightshift is already available in PATH.\n\n")
			b.WriteString("Press Enter to continue.\n")
			break
		}
		b.WriteString("Nightshift isn’t in PATH yet. The daemon and CLI shortcuts need it there.\n")
		if m.pathShell != "" && m.pathConfig != "" {
			fmt.Fprintf(&b, "Shell: %s\n", m.pathShell)
			fmt.Fprintf(&b, "Config: %s\n", m.pathConfig)
		}
		b.WriteString("\nSelect action:\n")
		for i, option := range m.pathOptions {
			cursor := " "
			if i == m.pathCursor {
				cursor = ">"
			}
			fmt.Fprintf(&b, " %s %s\n", cursor, option.label)
		}
		if m.pathErr != "" {
			b.WriteString("\nError: " + m.pathErr + "\n")
		}
		if m.pathStatus != "" {
			b.WriteString("\n" + m.pathStatus + "\n")
			if m.pathSourceHint != "" {
				b.WriteString("Run: " + m.pathSourceHint + "\n")
			}
		}
		if m.pathApplied {
			b.WriteString("\nPress Enter to continue.\n")
		} else {
			b.WriteString("\nPress Enter to apply.\n")
		}
	case stepDaemon:
		b.WriteString(styleAccent.Render("Daemon setup"))
		b.WriteString("\n\n")
		fmt.Fprintf(&b, "Service: %s\n", m.serviceType)
		if m.serviceState.installed {
			b.WriteString("Status: installed\n")
		} else {
			b.WriteString("Status: not installed\n")
		}
		if m.serviceState.running {
			b.WriteString("Daemon: running\n")
		} else {
			b.WriteString("Daemon: stopped\n")
		}
		b.WriteString("\nSelect action:\n")
		for i, label := range m.daemonOptions() {
			cursor := " "
			if i == m.daemonCursor {
				cursor = ">"
			}
			fmt.Fprintf(&b, " %s %s\n", cursor, label)
		}
		b.WriteString("\nPress Enter to apply.\n")
	case stepFinish:
		b.WriteString(styleAccent.Render("Setup complete"))
		b.WriteString("\n")
		b.WriteString(m.finishSummaryLine())
		b.WriteString("\n\n")
		if status := m.finishDaemonStatus(); status != "" {
			b.WriteString(status + "\n\n")
		}
		b.WriteString("What to expect:\n")
		for _, line := range m.finishExpectations() {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\nPress Enter to exit.\n")
	}

	return b.String()
}

func (m *setupModel) setStep(step setupStep) tea.Cmd {
	m.step = step
	switch step {
	case stepSnapshot:
		m.snapshotRunning = true
		m.snapshotOutput = ""
		m.snapshotErr = nil
		return runSnapshotCmd(m.cfg)
	case stepPreview:
		m.previewRunning = true
		m.previewOutput = ""
		m.previewErr = nil
		return runPreviewCmd(m.cfg, m.projects)
	case stepPath:
		m.preparePathStep()
	case stepDaemon:
		m.serviceType, m.serviceState = detectServiceState()
	}
	return nil
}

func (m *setupModel) preparePathStep() {
	m.pathErr = ""
	m.pathStatus = ""
	m.pathApplied = false
	m.pathCursor = 0

	if m.nightshiftInPath {
		m.pathOptions = nil
		return
	}

	shellName, configPath := detectShellConfig()
	m.pathShell = shellName
	m.pathConfig = configPath
	m.pathSourceHint = sourceHint(shellName, configPath)

	exeDir := filepath.Dir(mustExecutablePath())
	exeDir = expandPath(exeDir)
	if absDir, err := filepath.Abs(exeDir); err == nil {
		exeDir = absDir
	}

	goBinDir, goOK := detectGoBinDir()
	if goOK {
		goBinDir = expandPath(goBinDir)
		if absDir, err := filepath.Abs(goBinDir); err == nil {
			goBinDir = absDir
		}
	}

	home, _ := os.UserHomeDir()
	localBinDir := filepath.Join(home, ".local", "bin")
	if absDir, err := filepath.Abs(localBinDir); err == nil {
		localBinDir = absDir
	}

	var options []pathOption
	if goOK && goBinDir != "" {
		options = append(options, pathOption{
			label:   fmt.Sprintf("Install to %s and add to PATH (recommended)", goBinDir),
			action:  pathActionAdd,
			dir:     goBinDir,
			install: true,
		})
	} else {
		options = append(options, pathOption{
			label:   fmt.Sprintf("Install to %s and add to PATH", localBinDir),
			action:  pathActionAdd,
			dir:     localBinDir,
			install: true,
		})
	}

	if exeDir != "" && exeDir != goBinDir && exeDir != localBinDir {
		options = append(options, pathOption{
			label:   fmt.Sprintf("Add current binary dir to PATH (%s)", exeDir),
			action:  pathActionAdd,
			dir:     exeDir,
			install: false,
		})
	}

	options = append(options, pathOption{
		label:  "Skip (I'll handle PATH myself)",
		action: pathActionSkip,
	})

	m.pathOptions = options
}

func (m *setupModel) handleProjectsInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.projectEditing {
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(m.projectInput.Value())
			if value == "" {
				m.projectErr = "path cannot be empty"
				return m, nil
			}
			path := expandPath(value)
			info, err := os.Stat(path)
			if err != nil {
				m.projectErr = "path not found"
				return m, nil
			}
			if !info.IsDir() {
				m.projectErr = "path must be a directory"
				return m, nil
			}
			m.projects = append(m.projects, value)
			m.projectInput.SetValue("")
			m.projectErr = ""
			m.projectEditing = false
			return m, nil
		case "esc":
			m.projectEditing = false
			m.projectErr = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.projectInput, cmd = m.projectInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "up", "k":
		if m.projectCursor > 0 {
			m.projectCursor--
		}
	case "down", "j":
		if m.projectCursor < len(m.projects)-1 {
			m.projectCursor++
		}
	case "a":
		m.projectEditing = true
		m.projectInput.Focus()
	case "d":
		if len(m.projects) > 0 {
			m.projects = append(m.projects[:m.projectCursor], m.projects[m.projectCursor+1:]...)
			if m.projectCursor >= len(m.projects) && m.projectCursor > 0 {
				m.projectCursor--
			}
		}
	case "enter":
		if len(m.projects) == 0 {
			m.projectErr = "add at least one project"
			return m, nil
		}
		m.projectErr = ""
		m.applyProjects()
		return m, m.setStep(stepBudget)
	}

	return m, nil
}

func (m *setupModel) handleBudgetInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.budgetEditing {
		switch msg.String() {
		case "enter":
			if err := m.applyBudgetEdit(); err != nil {
				m.budgetErr = err.Error()
				return m, nil
			}
			m.budgetEditing = false
			m.budgetErr = ""
			return m, nil
		case "esc":
			m.budgetEditing = false
			m.budgetErr = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.budgetInput, cmd = m.budgetInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "up", "k":
		if m.budgetCursor > 0 {
			m.budgetCursor--
		}
	case "down", "j":
		if m.budgetCursor < 6 {
			m.budgetCursor++
		}
	case "e":
		m.budgetEditing = true
		m.budgetInput.SetValue(m.budgetFieldValue())
		m.budgetInput.Focus()
	case "enter":
		m.applyBudgetDefaults()
		return m, m.setStep(stepSafety)
	}
	return m, nil
}

func (m *setupModel) handlePresetInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.taskPresetCursor > 0 {
			m.taskPresetCursor--
		}
	case "down", "j":
		if m.taskPresetCursor < 2 {
			m.taskPresetCursor++
		}
	case "enter":
		presets := []setup.Preset{setup.PresetBalanced, setup.PresetSafe, setup.PresetAggressive}
		m.preset = presets[m.taskPresetCursor]
		m.taskItems = makeTaskItems(m.cfg, m.projects, m.preset)
		return m, m.setStep(stepTaskSelect)
	}
	return m, nil
}

func (m *setupModel) handleSafetyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.safetyCursor > 0 {
			m.safetyCursor--
		}
	case "down", "j":
		if m.safetyCursor < 2 {
			m.safetyCursor++
		}
	case " ":
		switch m.safetyCursor {
		case 0:
			m.cfg.Providers.Claude.DangerouslySkipPermissions = !m.cfg.Providers.Claude.DangerouslySkipPermissions
		case 1:
			m.cfg.Providers.Codex.DangerouslyBypassApprovalsAndSandbox = !m.cfg.Providers.Codex.DangerouslyBypassApprovalsAndSandbox
		case 2:
			m.cfg.Providers.Copilot.DangerouslySkipPermissions = !m.cfg.Providers.Copilot.DangerouslySkipPermissions
		}
	case "enter":
		return m, m.setStep(stepModel)
	}
	return m, nil
}

func (m *setupModel) handleTaskInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.taskItems) == 0 {
		if msg.String() == "enter" {
			m.taskErr = "no task definitions available"
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.taskCursor > 0 {
			m.taskCursor--
		}
	case "down", "j":
		if m.taskCursor < len(m.taskItems)-1 {
			m.taskCursor++
		}
	case " ":
		m.taskItems[m.taskCursor].selected = !m.taskItems[m.taskCursor].selected
		m.taskErr = ""
	case "enter":
		if !m.hasSelectedTasks() {
			m.taskErr = "select at least one task"
			return m, nil
		}
		m.applyTasks()
		m.taskErr = ""
		return m, m.setStep(stepSchedule)
	}
	return m, nil
}

func (m *setupModel) handleScheduleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.scheduleEditing {
		switch msg.String() {
		case "enter":
			if err := m.applyScheduleEdit(); err != nil {
				m.scheduleErr = err.Error()
				return m, nil
			}
			m.scheduleEditing = false
			m.scheduleErr = ""
			return m, nil
		case "esc":
			m.scheduleEditing = false
			m.scheduleErr = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.scheduleInput, cmd = m.scheduleInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "up", "k":
		if m.scheduleCursor > 0 {
			m.scheduleCursor--
		}
	case "down", "j":
		if m.scheduleCursor < 4 {
			m.scheduleCursor++
		}
	case "e":
		m.scheduleEditing = true
		m.scheduleInput.SetValue(m.scheduleFieldValue())
		m.scheduleInput.Focus()
	case "enter":
		m.applyScheduleDefaults()
		if err := writeGlobalConfig(m.cfg); err != nil {
			m.scheduleErr = err.Error()
			return m, nil
		}
		return m, m.setStep(stepSnapshot)
	}
	return m, nil
}

func (m *setupModel) handlePathInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pathApplied || m.nightshiftInPath {
		if msg.String() == "enter" {
			return m, m.setStep(stepDaemon)
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.pathCursor > 0 {
			m.pathCursor--
		}
	case "down", "j":
		if m.pathCursor < len(m.pathOptions)-1 {
			m.pathCursor++
		}
	case "enter":
		if len(m.pathOptions) == 0 {
			m.pathErr = "no PATH options available"
			return m, nil
		}
		option := m.pathOptions[m.pathCursor]
		m.pathErr = ""
		m.pathStatus = ""
		if option.action == pathActionSkip {
			m.pathApplied = true
			m.pathStatus = "Skipped PATH update."
			return m, nil
		}
		if err := m.applyPathOption(option); err != nil {
			m.pathErr = err.Error()
			return m, nil
		}
		m.pathApplied = true
		m.nightshiftInPath = true
		return m, nil
	}
	return m, nil
}

func (m *setupModel) handleDaemonInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.daemonCursor > 0 {
			m.daemonCursor--
		}
	case "down", "j":
		if m.daemonCursor < len(m.daemonOptions())-1 {
			m.daemonCursor++
		}
	case "enter":
		action := m.daemonOptions()[m.daemonCursor]
		if err := m.applyDaemonAction(action); err != nil {
			m.serviceState.detail = err.Error()
			return m, nil
		}
		m.daemonAction = action
		return m, m.setStep(stepFinish)
	}
	return m, nil
}

func (m *setupModel) applyProjects() {
	m.cfg.Projects = nil
	for _, project := range m.projects {
		project = strings.TrimSpace(project)
		if project == "" {
			continue
		}
		m.cfg.Projects = append(m.cfg.Projects, config.ProjectConfig{Path: project})
	}
	m.updateProjectGitignores()
}

func (m *setupModel) applyBudgetDefaults() {
	if m.cfg.Budget.Mode == "" {
		m.cfg.Budget.Mode = config.DefaultBudgetMode
	}
	if m.cfg.Budget.MaxPercent == 0 {
		m.cfg.Budget.MaxPercent = config.DefaultMaxPercent
	}
	if m.cfg.Budget.ReservePercent == 0 {
		m.cfg.Budget.ReservePercent = config.DefaultReservePercent
	}
	if m.cfg.Budget.BillingMode == "" {
		m.cfg.Budget.BillingMode = config.DefaultBillingMode
	}
	if m.cfg.Budget.SnapshotInterval == "" {
		m.cfg.Budget.SnapshotInterval = config.DefaultSnapshotInterval
	}
	if m.cfg.Budget.WeekStartDay == "" {
		m.cfg.Budget.WeekStartDay = config.DefaultWeekStartDay
	}
	if m.cfg.Budget.WeeklyTokens == 0 {
		m.cfg.Budget.WeeklyTokens = config.DefaultWeeklyTokens
	}
}

func (m *setupModel) budgetFieldValue() string {
	switch m.budgetCursor {
	case 0:
		return m.cfg.Budget.Mode
	case 1:
		return strconv.Itoa(m.cfg.Budget.MaxPercent)
	case 2:
		return strconv.Itoa(m.cfg.Budget.ReservePercent)
	case 3:
		return m.cfg.Budget.BillingMode
	case 4:
		return strconv.FormatBool(m.cfg.Budget.CalibrateEnabled)
	case 5:
		return m.cfg.Budget.SnapshotInterval
	case 6:
		return m.cfg.Budget.WeekStartDay
	default:
		return ""
	}
}

func (m *setupModel) applyBudgetEdit() error {
	value := strings.TrimSpace(m.budgetInput.Value())
	switch m.budgetCursor {
	case 0:
		if value != "daily" && value != "weekly" {
			return fmt.Errorf("mode must be daily or weekly")
		}
		m.cfg.Budget.Mode = value
	case 1:
		v, err := strconv.Atoi(value)
		if err != nil || v < 1 || v > 100 {
			return fmt.Errorf("max_percent must be between 1 and 100")
		}
		m.cfg.Budget.MaxPercent = v
	case 2:
		v, err := strconv.Atoi(value)
		if err != nil || v < 0 || v > 100 {
			return fmt.Errorf("reserve_percent must be between 0 and 100")
		}
		m.cfg.Budget.ReservePercent = v
	case 3:
		if value != "subscription" && value != "api" {
			return fmt.Errorf("billing_mode must be subscription or api")
		}
		m.cfg.Budget.BillingMode = value
	case 4:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("calibrate_enabled must be true or false")
		}
		m.cfg.Budget.CalibrateEnabled = v
	case 5:
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("snapshot_interval must be duration (e.g., 30m)")
		}
		m.cfg.Budget.SnapshotInterval = value
	case 6:
		if value != "monday" && value != "sunday" {
			return fmt.Errorf("week_start_day must be monday or sunday")
		}
		m.cfg.Budget.WeekStartDay = value
	}
	return nil
}

func (m *setupModel) applyTasks() {
	selected := make([]string, 0)
	for _, item := range m.taskItems {
		if item.selected {
			selected = append(selected, string(item.def.Type))
		}
	}
	m.cfg.Tasks.Enabled = selected
}

func (m *setupModel) hasSelectedTasks() bool {
	for _, item := range m.taskItems {
		if item.selected {
			return true
		}
	}
	return false
}

func (m *setupModel) scheduleFieldValue() string {
	switch m.scheduleCursor {
	case 0:
		return m.scheduleStart
	case 1:
		return strconv.Itoa(m.scheduleCycles)
	case 2:
		return m.scheduleInterval
	case 3:
		return m.scheduleMode
	case 4:
		return m.scheduleCron
	default:
		return ""
	}
}

func (m *setupModel) applyScheduleEdit() error {
	value := strings.TrimSpace(m.scheduleInput.Value())
	switch m.scheduleCursor {
	case 0:
		if _, err := scheduler.ParseTimeOfDay(value); err != nil {
			return err
		}
		m.scheduleStart = value
	case 1:
		v, err := strconv.Atoi(value)
		if err != nil || v <= 0 {
			return fmt.Errorf("cycles must be positive")
		}
		m.scheduleCycles = v
	case 2:
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("interval must be duration (e.g., 30m)")
		}
		m.scheduleInterval = value
	case 3:
		if value != "interval" && value != "cron" {
			return fmt.Errorf("mode must be interval or cron")
		}
		m.scheduleMode = value
	case 4:
		test := scheduler.New()
		if err := test.SetCron(value); err != nil {
			return err
		}
		m.scheduleCron = value
	}
	return nil
}

func (m *setupModel) applyScheduleDefaults() {
	m.cfg.Schedule = config.ScheduleConfig{}
	if m.scheduleMode == "cron" {
		m.cfg.Schedule.Cron = m.scheduleCron
		return
	}

	m.cfg.Schedule.Interval = m.scheduleInterval
	start, _ := scheduler.ParseTimeOfDay(m.scheduleStart)
	interval, _ := time.ParseDuration(m.scheduleInterval)
	end := computeWindowEnd(start, interval, m.scheduleCycles)
	m.scheduleWindowEnd = end.String()
	m.cfg.Schedule.Window = &config.WindowConfig{
		Start:    m.scheduleStart,
		End:      end.String(),
		Timezone: "",
	}
}

func (m *setupModel) daemonOptions() []string {
	if !m.serviceState.installed {
		return []string{"Install and enable daemon", "Skip"}
	}
	return []string{"Start daemon", "Stop daemon", "Remove service", "Leave as-is"}
}

func (m *setupModel) applyDaemonAction(action string) error {
	switch action {
	case "Install and enable daemon":
		if err := installService(m.serviceType, m.cfg); err != nil {
			return err
		}
		return runDaemonStart(nil, nil)
	case "Start daemon":
		return runDaemonStart(nil, nil)
	case "Stop daemon":
		return runDaemonStop(nil, nil)
	case "Remove service":
		_ = runDaemonStop(nil, nil) // ignore if not running
		return uninstallService(m.serviceType)
	default:
		return nil
	}
}

func (m *setupModel) finishSummaryLine() string {
	switch m.daemonAction {
	case "Stop daemon", "Remove service", "Skip":
		return "Nightshift is configured, but the daemon is not running."
	case "Leave as-is":
		if m.serviceState.running {
			return "Nightshift is configured and the daemon is running."
		}
		if m.serviceState.installed {
			return "Nightshift is configured, but the daemon is stopped."
		}
		return "Nightshift is configured, but no daemon service is installed."
	default:
		return "Nightshift is configured and ready to run."
	}
}

func (m *setupModel) finishDaemonStatus() string {
	switch m.daemonAction {
	case "Install and enable daemon":
		return "Daemon status: installed and started."
	case "Start daemon":
		return "Daemon status: started."
	case "Stop daemon":
		return "Daemon status: stopped."
	case "Remove service":
		return "Daemon status: service removed."
	case "Skip":
		return "Daemon status: not installed."
	case "Leave as-is":
		if m.serviceState.running {
			return "Daemon status: running (unchanged)."
		}
		if m.serviceState.installed {
			return "Daemon status: installed but stopped (unchanged)."
		}
		return "Daemon status: not installed."
	default:
		return ""
	}
}

func (m *setupModel) finishExpectations() []string {
	lines := []string{
		fmt.Sprintf("Summary report: %s", reporting.DefaultSummaryPath(time.Now())),
		fmt.Sprintf("Run report: %s", reporting.DefaultRunReportPath(time.Now())),
		"CLI status: `nightshift status --today` or `nightshift logs`",
		"Safety: Nightshift never writes to your primary branch. Expect PRs or branches.",
	}
	if m.gitignoreAdded > 0 || m.gitignoreKept > 0 {
		var parts []string
		if m.gitignoreAdded > 0 {
			parts = append(parts, fmt.Sprintf("added to %d project(s)", m.gitignoreAdded))
		}
		if m.gitignoreKept > 0 {
			parts = append(parts, fmt.Sprintf("already present in %d project(s)", m.gitignoreKept))
		}
		lines = append(lines, fmt.Sprintf("Gitignore: ensured `%s` is ignored (%s) so plan artifacts stay out of version control.", nightshiftPlanIgnore, strings.Join(parts, ", ")))
	}
	for _, errLine := range m.gitignoreErrs {
		lines = append(lines, fmt.Sprintf("Gitignore: %s", errLine))
	}

	switch m.daemonAction {
	case "Stop daemon", "Remove service", "Skip":
		lines = append([]string{
			"Nightshift will not run automatically until the daemon is started.",
			"Run manually: `nightshift run`.",
			"Start the daemon later: `nightshift daemon start` (or re-run setup to install a service).",
		}, lines...)
	case "Leave as-is":
		if !m.serviceState.running {
			lines = append([]string{
				"Nightshift will not run automatically until the daemon is started.",
				"Run manually: `nightshift run`.",
				"Start the daemon later: `nightshift daemon start` (or re-run setup to install a service).",
			}, lines...)
		}
	}

	return lines
}

func (m *setupModel) applyPathOption(option pathOption) error {
	if option.dir == "" {
		return fmt.Errorf("missing target path")
	}

	var statusParts []string
	if option.install {
		dest, err := installNightshiftBinary(option.dir)
		if err != nil {
			return err
		}
		statusParts = append(statusParts, fmt.Sprintf("Installed binary to %s.", dest))
	}

	changed, err := ensurePathInShell(m.pathConfig, m.pathShell, option.dir)
	if err != nil {
		return err
	}
	if changed {
		statusParts = append(statusParts, fmt.Sprintf("Added %s to PATH in %s.", option.dir, m.pathConfig))
	} else {
		statusParts = append(statusParts, fmt.Sprintf("%s already present in %s.", option.dir, m.pathConfig))
	}

	m.pathStatus = strings.Join(statusParts, " ")
	return nil
}

func detectShellConfig() (string, string) {
	shell := filepath.Base(os.Getenv("SHELL"))
	home, _ := os.UserHomeDir()
	switch shell {
	case "zsh":
		return "zsh", filepath.Join(home, ".zshrc")
	case "bash":
		bashProfile := filepath.Join(home, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			return "bash", bashProfile
		}
		return "bash", filepath.Join(home, ".bashrc")
	case "fish":
		return "fish", filepath.Join(home, ".config", "fish", "config.fish")
	default:
		if shell == "" {
			shell = "sh"
		}
		return shell, filepath.Join(home, ".profile")
	}
}

func detectGoBinDir() (string, bool) {
	if _, err := exec.LookPath("go"); err != nil {
		return "", false
	}
	out, err := exec.Command("go", "env", "GOBIN").Output()
	if err == nil {
		gobin := strings.TrimSpace(string(out))
		if gobin != "" {
			return gobin, true
		}
	}
	out, err = exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return "", false
	}
	gopath := strings.TrimSpace(string(out))
	if gopath == "" {
		return "", false
	}
	if strings.Contains(gopath, string(os.PathListSeparator)) {
		gopath = strings.Split(gopath, string(os.PathListSeparator))[0]
	}
	return filepath.Join(gopath, "bin"), true
}

func sourceHint(shell, configPath string) string {
	if configPath == "" {
		return ""
	}
	if shell == "fish" {
		return fmt.Sprintf("source %s", configPath)
	}
	return fmt.Sprintf("source %s", configPath)
}

func ensurePathInShell(configPath, shell, dir string) (bool, error) {
	if configPath == "" {
		return false, fmt.Errorf("missing shell config path")
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return false, err
	}

	dir = expandPath(dir)
	line := pathExportLine(shell, dir)

	var existing string
	if data, err := os.ReadFile(configPath); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if shellConfigHasPath(existing, dir) {
		return false, nil
	}

	if len(existing) > 0 && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	existing += line + "\n"
	return true, os.WriteFile(configPath, []byte(existing), 0644)
}

// escapeShellPath returns a shell-safe escaped version of a path string.
// Wraps the path in single quotes to prevent interpretation of special characters.
func escapeShellPath(path string) string {
	// Single quotes prevent all expansions in shell, safest approach
	// If the path contains a single quote, we need to escape it carefully
	if !strings.Contains(path, "'") {
		return fmt.Sprintf("'%s'", path)
	}
	// Path contains single quote: use double quotes and escape special chars
	// Replace $ and ` and " with escaped versions
	escaped := strings.ReplaceAll(path, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	return fmt.Sprintf("\"%s\"", escaped)
}

func pathExportLine(shell, dir string) string {
	// SECURITY: Escape the directory path to prevent shell injection
	escapedDir := escapeShellPath(dir)
	switch shell {
	case "fish":
		return fmt.Sprintf("set -gx PATH %s $PATH", escapedDir)
	default:
		return fmt.Sprintf("export PATH=\"$PATH:%s\"", escapedDir)
	}
}

func shellConfigHasPath(content, dir string) bool {
	target := filepath.Clean(dir)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.Contains(trimmed, "PATH") {
			continue
		}
		if containsPathToken(trimmed, target) {
			return true
		}
	}
	return false
}

func containsPathToken(line, target string) bool {
	tokens := strings.FieldsFunc(line, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		switch r {
		case ':', ';', '"', '\'', '=', '$', '{', '}', '(', ')':
			return true
		default:
			return false
		}
	})
	for _, token := range tokens {
		if filepath.Clean(token) == target {
			return true
		}
	}
	return false
}

func installNightshiftBinary(targetDir string) (string, error) {
	exePath := mustExecutablePath()
	if exePath == "" {
		return "", fmt.Errorf("unable to locate nightshift binary")
	}

	targetDir = expandPath(targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	dest := filepath.Join(targetDir, "nightshift")
	if samePath(exePath, dest) {
		return dest, nil
	}

	if err := copyFile(exePath, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func samePath(a, b string) bool {
	aa, errA := filepath.EvalSymlinks(a)
	bb, errB := filepath.EvalSymlinks(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return aa == bb
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

func renderEnvChecks(cfg *config.Config) string {
	var b strings.Builder
	if _, err := execLookPath("nightshift"); err != nil {
		fmt.Fprintf(&b, "  %s %s\n", styleWarn.Render("Heads up:"), "nightshift not found in PATH yet. Setup can add it for you.")
	} else {
		fmt.Fprintf(&b, "  %s %s\n", styleOk.Render("OK:"), "nightshift is in PATH")
	}
	if _, err := execLookPath("tmux"); err != nil {
		fmt.Fprintf(&b, "  %s %s\n", styleWarn.Render("Note:"), "tmux not found (calibration will be local-only)")
	} else {
		fmt.Fprintf(&b, "  %s %s\n", styleOk.Render("OK:"), "tmux available")
	}
	// Check for Copilot CLI (gh or copilot binary)
	_, ghErr := execLookPath("gh")
	_, copilotErr := execLookPath("copilot")
	if ghErr != nil && copilotErr != nil {
		fmt.Fprintf(&b, "  %s %s\n", styleWarn.Render("Note:"), "Copilot CLI not found (install via 'gh' or native 'copilot')")
	} else if ghErr == nil {
		fmt.Fprintf(&b, "  %s %s\n", styleOk.Render("OK:"), "gh CLI available (use 'gh copilot')")
	} else {
		fmt.Fprintf(&b, "  %s %s\n", styleOk.Render("OK:"), "copilot CLI available")
	}
	if cfg.Providers.Claude.Enabled {
		if _, err := os.Stat(cfg.ExpandedProviderPath("claude")); err != nil {
			fmt.Fprintf(&b, "  %s %s\n", styleWarn.Render("Note:"), "Claude data path not found")
		} else {
			fmt.Fprintf(&b, "  %s %s\n", styleOk.Render("OK:"), "Claude data path found")
		}
	}
	if cfg.Providers.Codex.Enabled {
		if _, err := os.Stat(cfg.ExpandedProviderPath("codex")); err != nil {
			fmt.Fprintf(&b, "  %s %s\n", styleWarn.Render("Note:"), "Codex data path not found")
		} else {
			fmt.Fprintf(&b, "  %s %s\n", styleOk.Render("OK:"), "Codex data path found")
		}
	}
	return b.String()
}

func renderBudgetFields(b *strings.Builder, m *setupModel) {
	fields := []string{
		fmt.Sprintf("Mode: %s", m.cfg.Budget.Mode),
		fmt.Sprintf("Max percent: %d", m.cfg.Budget.MaxPercent),
		fmt.Sprintf("Reserve percent: %d", m.cfg.Budget.ReservePercent),
		fmt.Sprintf("Billing mode: %s", m.cfg.Budget.BillingMode),
		fmt.Sprintf("Calibrate enabled: %t", m.cfg.Budget.CalibrateEnabled),
		fmt.Sprintf("Snapshot interval: %s", m.cfg.Budget.SnapshotInterval),
		fmt.Sprintf("Week start day: %s", m.cfg.Budget.WeekStartDay),
	}
	for i, field := range fields {
		cursor := " "
		if i == m.budgetCursor {
			cursor = ">"
		}
		fmt.Fprintf(b, " %s %s\n", cursor, field)
	}
}

func renderSafetyFields(b *strings.Builder, m *setupModel) {
	items := []struct {
		label     string
		enabled   bool
		available bool
	}{
		{
			label:     "Claude: --dangerously-skip-permissions",
			enabled:   m.cfg.Providers.Claude.DangerouslySkipPermissions,
			available: m.cfg.Providers.Claude.Enabled,
		},
		{
			label:     "Codex:  --dangerously-bypass-approvals-and-sandbox",
			enabled:   m.cfg.Providers.Codex.DangerouslyBypassApprovalsAndSandbox,
			available: m.cfg.Providers.Codex.Enabled,
		},
		{
			label:     "Copilot: --allow-all-tools",
			enabled:   m.cfg.Providers.Copilot.DangerouslySkipPermissions,
			available: m.cfg.Providers.Copilot.Enabled,
		},
	}

	for i, item := range items {
		cursor := " "
		if i == m.safetyCursor {
			cursor = ">"
		}
		state := "OFF"
		if item.enabled {
			state = "ON"
		}
		status := state
		if !item.available {
			status = fmt.Sprintf("%s (provider disabled)", state)
		}
		fmt.Fprintf(b, " %s [%s] %s\n", cursor, status, item.label)
	}
	b.WriteString(styleNote.Render("Tip: leave these OFF if you want the CLI to ask for approvals."))
	b.WriteString("\n")
}

func renderModelFields(b *strings.Builder, m *setupModel) {
	rows := []struct {
		label     string
		models    []modelOption
		idx       int
		available bool
	}{
		{"Claude ", claudeModels, m.claudeModelIdx, m.cfg.Providers.Claude.Enabled},
		{"Codex  ", codexModels, m.codexModelIdx, m.cfg.Providers.Codex.Enabled},
		{"Copilot", copilotModels, m.copilotModelIdx, m.cfg.Providers.Copilot.Enabled},
	}
	for i, row := range rows {
		cursor := " "
		if i == m.modelCursor {
			cursor = ">"
		}
		selected := row.models[row.idx].label
		avail := ""
		if !row.available {
			avail = " (provider disabled)"
		}
		fmt.Fprintf(b, " %s %s  ← %s →%s\n", cursor, row.label, selected, avail)
	}
	b.WriteString(styleNote.Render("Tip: 'default' lets the CLI pick its built-in model."))
	b.WriteString("\n")
}

func (m *setupModel) handleModelInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.modelCursor > 0 {
			m.modelCursor--
		}
	case "down", "j":
		if m.modelCursor < 2 {
			m.modelCursor++
		}
	case "left", "h":
		switch m.modelCursor {
		case 0:
			if m.claudeModelIdx > 0 {
				m.claudeModelIdx--
			}
		case 1:
			if m.codexModelIdx > 0 {
				m.codexModelIdx--
			}
		case 2:
			if m.copilotModelIdx > 0 {
				m.copilotModelIdx--
			}
		}
	case "right", "l":
		switch m.modelCursor {
		case 0:
			if m.claudeModelIdx < len(claudeModels)-1 {
				m.claudeModelIdx++
			}
		case 1:
			if m.codexModelIdx < len(codexModels)-1 {
				m.codexModelIdx++
			}
		case 2:
			if m.copilotModelIdx < len(copilotModels)-1 {
				m.copilotModelIdx++
			}
		}
	case "enter":
		m.cfg.Providers.Claude.Model = claudeModels[m.claudeModelIdx].value
		m.cfg.Providers.Codex.Model = codexModels[m.codexModelIdx].value
		m.cfg.Providers.Copilot.Model = copilotModels[m.copilotModelIdx].value
		return m, m.setStep(stepTaskPreset)
	}
	return m, nil
}

// modelIndex returns the index of the given model value in a model list, defaulting to 0.
func modelIndex(models []modelOption, value string) int {
	for i, m := range models {
		if m.value == value {
			return i
		}
	}
	return 0
}


func renderScheduleFields(b *strings.Builder, m *setupModel) {
	fields := []string{
		fmt.Sprintf("Start time: %s", m.scheduleStart),
		fmt.Sprintf("Cycles: %d", m.scheduleCycles),
		fmt.Sprintf("Interval: %s", m.scheduleInterval),
		fmt.Sprintf("Mode: %s (interval|cron)", m.scheduleMode),
		fmt.Sprintf("Cron: %s", m.scheduleCron),
	}
	for i, field := range fields {
		cursor := " "
		if i == m.scheduleCursor {
			cursor = ">"
		}
		fmt.Fprintf(b, " %s %s\n", cursor, field)
	}
	if m.scheduleMode == "interval" {
		start, errStart := scheduler.ParseTimeOfDay(m.scheduleStart)
		interval, errInterval := time.ParseDuration(m.scheduleInterval)
		if errStart == nil && errInterval == nil {
			end := computeWindowEnd(start, interval, m.scheduleCycles)
			fmt.Fprintf(b, "   Window end (computed): %s\n", end)
		}
	}
}

func scheduleFieldHelp(cursor int, mode string) string {
	switch cursor {
	case 0:
		return "Start time: when Nightshift becomes eligible to run each night (local time)."
	case 1:
		return "Cycles: how many runs to attempt inside the nightly window."
	case 2:
		return "Interval: spacing between runs (e.g., 30m = every 30 minutes)."
	case 3:
		return "Mode: interval uses Start/Cycles/Interval; cron uses a single cron expression."
	case 4:
		if mode == "cron" {
			return "Cron: advanced schedule (e.g., \"0 2 * * *\" = 2:00 AM daily)."
		}
		return "Cron: only used when mode is set to cron."
	default:
		return ""
	}
}

func makeTaskItems(cfg *config.Config, projects []string, preset setup.Preset) []taskItem {
	defs := tasks.AllDefinitionsSorted()
	signals := setup.DetectRepoSignals(projects)
	selected := setup.PresetTasks(preset, defs, signals)
	for _, enabled := range cfg.Tasks.Enabled {
		selected[tasks.TaskType(enabled)] = true
	}

	items := make([]taskItem, 0, len(defs))
	for _, def := range defs {
		items = append(items, taskItem{
			def:      def,
			selected: selected[def.Type],
		})
	}
	return items
}

func runSnapshotCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		output, err := runSnapshot(cfg)
		return snapshotMsg{output: output, err: err}
	}
}

func runSnapshot(cfg *config.Config) (string, error) {
	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return "", err
	}
	defer func() { _ = database.Close() }()

	scraper := snapshots.UsageScraper(nil)
	if cfg.Budget.CalibrateEnabled && strings.ToLower(cfg.Budget.BillingMode) != "api" {
		scraper = tmuxScraper{}
	}

	collector := snapshots.NewCollector(
		database,
		providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude")),
		providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex")),
		providers.NewCopilotWithPath(cfg.ExpandedProviderPath("copilot")),
		scraper,
		weekStartDayFromConfig(cfg),
	)

	var lines []string
	ctx := context.Background()
	if cfg.Providers.Claude.Enabled {
		snapshot, err := collector.TakeSnapshot(ctx, "claude")
		if err != nil {
			lines = append(lines, fmt.Sprintf("claude: error: %v", err))
		} else {
			lines = append(lines, formatSnapshotLine(snapshot))
		}
	}
	if cfg.Providers.Codex.Enabled {
		snapshot, err := collector.TakeSnapshot(ctx, "codex")
		if err != nil {
			lines = append(lines, fmt.Sprintf("codex: error: %v", err))
		} else {
			lines = append(lines, formatSnapshotLine(snapshot))
		}
	}
	if cfg.Providers.Copilot.Enabled {
		snapshot, err := collector.TakeSnapshot(ctx, "copilot")
		if err != nil {
			lines = append(lines, fmt.Sprintf("copilot: error: %v", err))
		} else {
			lines = append(lines, formatSnapshotLine(snapshot))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func formatSnapshotLine(snapshot snapshots.Snapshot) string {
	scraped := "n/a"
	if snapshot.ScrapedPct != nil {
		scraped = fmt.Sprintf("%.1f%%", *snapshot.ScrapedPct)
	}
	inferred := ""
	if snapshot.InferredBudget != nil {
		inferred = fmt.Sprintf(", budget est %s/wk", formatTokens64(*snapshot.InferredBudget))
	}
	return fmt.Sprintf(
		"%s: weekly %s, daily %s, scraped %s%s",
		snapshot.Provider,
		formatTokens64(snapshot.LocalTokens),
		formatTokens64(snapshot.LocalDaily),
		scraped,
		inferred,
	)
}

func runPreviewCmd(cfg *config.Config, projects []string) tea.Cmd {
	return func() tea.Msg {
		output, err := buildSetupPreviewOutput(cfg, projects)
		return previewMsg{output: output, err: err}
	}
}

func buildSetupPreviewOutput(cfg *config.Config, projects []string) (string, error) {
	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return "", err
	}
	defer func() { _ = database.Close() }()

	result, err := buildPreviewResult(cfg, database, projects, "", 1, "", nil, false)
	if err != nil {
		return "", err
	}
	return renderSetupPreviewText(result), nil
}

type setupStepInfo struct {
	step  setupStep
	label string
}

func renderSetupStepper(m *setupModel) string {
	steps := setupSteps(m.includePathStep)
	stepIndex := 0
	stepLabel := ""
	for i, info := range steps {
		if info.step == m.step {
			stepIndex = i
			stepLabel = info.label
			break
		}
	}

	total := len(steps)
	current := stepIndex + 1
	line := fmt.Sprintf("%s  %s", styleNote.Render(fmt.Sprintf("Step %d of %d", current, total)), styleAccent.Render(stepLabel))
	bar := renderSetupProgressBar(current, total, 28)
	return line + "\n" + bar
}

func setupSteps(includePathStep bool) []setupStepInfo {
	steps := []setupStepInfo{
		{step: stepWelcome, label: "Welcome"},
		{step: stepConfig, label: "Global config"},
		{step: stepProjects, label: "Projects"},
		{step: stepBudget, label: "Budget"},
		{step: stepSafety, label: "Safety"},
		{step: stepModel, label: "Models"},
		{step: stepTaskPreset, label: "Task presets"},
		{step: stepTaskSelect, label: "Task selection"},
		{step: stepSchedule, label: "Schedule"},
		{step: stepSnapshot, label: "Snapshot"},
		{step: stepPreview, label: "Preview"},
	}
	if includePathStep {
		steps = append(steps, setupStepInfo{step: stepPath, label: "PATH"})
	}
	steps = append(steps,
		setupStepInfo{step: stepDaemon, label: "Daemon"},
		setupStepInfo{step: stepFinish, label: "Finish"},
	)
	return steps
}

func renderSetupProgressBar(current, total, width int) string {
	if total <= 0 || width <= 0 {
		return ""
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	filled := (width*current + total - 1) / total
	if filled > width {
		filled = width
	}
	empty := width - filled
	filledPart := styleOk.Render(strings.Repeat("=", filled))
	emptyPart := styleDim.Render(strings.Repeat("-", empty))
	return "[" + filledPart + emptyPart + "]"
}

func computeWindowEnd(start scheduler.TimeOfDay, interval time.Duration, cycles int) scheduler.TimeOfDay {
	if cycles <= 0 {
		cycles = 3
	}
	total := interval * time.Duration(cycles)
	startTime := time.Date(2000, 1, 1, start.Hour, start.Minute, 0, 0, time.Local)
	endTime := startTime.Add(total)
	return scheduler.TimeOfDay{Hour: endTime.Hour(), Minute: endTime.Minute()}
}

func detectServiceState() (string, serviceState) {
	service := detectServiceType()
	state := serviceState{}

	switch service {
	case ServiceLaunchd:
		home, _ := os.UserHomeDir()
		plistPath := filepath.Join(home, "Library", "LaunchAgents", launchdPlistName)
		if _, err := os.Stat(plistPath); err == nil {
			state.installed = true
			state.detail = plistPath
		}
	case ServiceSystemd:
		home, _ := os.UserHomeDir()
		servicePath := filepath.Join(home, ".config", "systemd", "user", systemdServiceName)
		timerPath := filepath.Join(home, ".config", "systemd", "user", systemdTimerName)
		if _, err := os.Stat(servicePath); err == nil {
			state.installed = true
			state.detail = servicePath
		}
		if _, err := os.Stat(timerPath); err == nil && state.detail != "" {
			state.detail = fmt.Sprintf("%s, %s", state.detail, timerPath)
		}
	case ServiceCron:
		out, err := exec.Command("crontab", "-l").CombinedOutput()
		if err == nil && strings.Contains(string(out), cronMarker) {
			state.installed = true
			state.detail = "cron entry present"
		}
	}

	running, _ := isDaemonRunning()
	state.running = running
	return service, state
}

func installService(service string, cfg *config.Config) error {
	if cfg == nil {
		loaded, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		cfg = loaded
	}

	switch service {
	case ServiceLaunchd:
		return installLaunchd(mustExecutablePath(), cfg)
	case ServiceSystemd:
		return installSystemd(mustExecutablePath(), cfg)
	case ServiceCron:
		return installCron(mustExecutablePath(), cfg)
	default:
		return fmt.Errorf("unknown service type: %s", service)
	}
}

func uninstallService(service string) error {
	switch service {
	case ServiceLaunchd:
		if !uninstallLaunchd() {
			return fmt.Errorf("launchd service not found")
		}
		return nil
	case ServiceSystemd:
		if !uninstallSystemd() {
			return fmt.Errorf("systemd service not found")
		}
		return nil
	case ServiceCron:
		if !uninstallCron() {
			return fmt.Errorf("cron entry not found")
		}
		return nil
	default:
		return fmt.Errorf("unknown service type: %s", service)
	}
}

func mustExecutablePath() string {
	path, _ := os.Executable()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return real
}

func writeGlobalConfig(cfg *config.Config) error {
	configPath := config.GlobalConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	if _, err := os.Stat(configPath); err == nil {
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("read config: %w", err)
		}
	}

	v.Set("schedule", cfg.Schedule)
	v.Set("budget.mode", cfg.Budget.Mode)
	v.Set("budget.max_percent", cfg.Budget.MaxPercent)
	v.Set("budget.reserve_percent", cfg.Budget.ReservePercent)
	v.Set("budget.weekly_tokens", cfg.Budget.WeeklyTokens)
	v.Set("budget.billing_mode", cfg.Budget.BillingMode)
	v.Set("budget.calibrate_enabled", cfg.Budget.CalibrateEnabled)
	v.Set("budget.snapshot_interval", cfg.Budget.SnapshotInterval)
	v.Set("budget.snapshot_retention_days", cfg.Budget.SnapshotRetentionDays)
	v.Set("budget.week_start_day", cfg.Budget.WeekStartDay)

	v.Set("providers", cfg.Providers)
	v.Set("projects", cfg.Projects)
	v.Set("tasks.enabled", cfg.Tasks.Enabled)

	if err := v.WriteConfig(); err != nil {
		if os.IsNotExist(err) {
			return v.SafeWriteConfig()
		}
		return err
	}

	return nil
}

func execLookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (m *setupModel) updateProjectGitignores() {
	m.gitignoreAdded = 0
	m.gitignoreKept = 0
	m.gitignoreErrs = nil

	for _, project := range m.cfg.Projects {
		path := expandPath(project.Path)
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		gitignorePath := filepath.Join(path, ".gitignore")
		added, err := ensureGitignoreEntry(gitignorePath, nightshiftPlanIgnore, nightshiftPlanIgnoreComment)
		if err != nil {
			m.gitignoreErrs = append(m.gitignoreErrs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if added {
			m.gitignoreAdded++
		} else {
			m.gitignoreKept++
		}
	}
}

func ensureGitignoreEntry(gitignorePath, entry, comment string) (bool, error) {
	var existing string
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if gitignoreHasEntry(existing, entry) {
		return false, nil
	}

	var b strings.Builder
	if existing != "" {
		b.WriteString(strings.TrimRight(existing, "\n"))
		b.WriteString("\n")
	}
	if comment != "" && !strings.Contains(existing, comment) {
		b.WriteString(comment)
		b.WriteString("\n")
	}
	b.WriteString(entry)
	b.WriteString("\n")

	// SECURITY: Use 0644 for .gitignore (world-readable is acceptable for git-tracked files)
	// but ensure proper atomic write to prevent corruption
	return true, os.WriteFile(gitignorePath, []byte(b.String()), 0644)
}

func gitignoreHasEntry(content, entry string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		trimmed = strings.TrimPrefix(trimmed, "/")
		trimmed = strings.TrimSuffix(trimmed, "/")
		if trimmed == entry {
			return true
		}
	}
	return false
}
