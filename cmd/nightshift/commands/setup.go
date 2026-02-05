package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/providers"
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
	stepTaskPreset
	stepTaskSelect
	stepSchedule
	stepSnapshot
	stepPreview
	stepDaemon
	stepFinish
)

type setupModel struct {
	step setupStep

	cfg         *config.Config
	configPath  string
	configExist bool

	projects       []string
	projectCursor  int
	projectInput   textinput.Model
	projectEditing bool
	projectErr     string

	budgetCursor  int
	budgetInput   textinput.Model
	budgetEditing bool
	budgetErr     string

	taskPresetCursor int
	taskCursor       int
	taskItems        []taskItem
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

	daemonCursor int
	serviceType  string
	serviceState serviceState

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

func newSetupModel() (*setupModel, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
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

	model := &setupModel{
		step:             stepWelcome,
		cfg:              cfg,
		configPath:       configPath,
		configExist:      configExist,
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
	}

	return model, nil
}

func (m *setupModel) Init() tea.Cmd {
	return spinner.Tick
}

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
				return m, m.setStep(stepDaemon)
			}
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

func (m *setupModel) View() string {
	var b strings.Builder
	b.WriteString("Nightshift Setup\n")
	b.WriteString("================\n\n")

	switch m.step {
	case stepWelcome:
		b.WriteString("This wizard will configure Nightshift end-to-end.\n\n")
		b.WriteString("Checks:\n")
		b.WriteString(renderEnvChecks(m.cfg))
		b.WriteString("\nPress Enter to continue.\n")
	case stepConfig:
		b.WriteString("Global config:\n")
		b.WriteString(fmt.Sprintf("  %s\n", m.configPath))
		if m.configExist {
			b.WriteString("  Status: found (will update in place)\n")
		} else {
			b.WriteString("  Status: will create\n")
		}
		b.WriteString("\nThis wizard only writes the global config. Per-project configs are optional.\n")
		b.WriteString("\nPress Enter to continue.\n")
	case stepProjects:
		b.WriteString("Projects (global config)\n")
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
			b.WriteString(fmt.Sprintf(" %s %s\n", cursor, label))
		}
		if m.projectErr != "" {
			b.WriteString("\nError: " + m.projectErr + "\n")
		}
		b.WriteString("\nPress Enter to continue.\n")
	case stepBudget:
		b.WriteString("Budget defaults (edit with e)\n")
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
	case stepTaskPreset:
		b.WriteString("Task presets (derived from registry)\n")
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
			b.WriteString(fmt.Sprintf(" %s %s\n", cursor, label))
		}
	case stepTaskSelect:
		b.WriteString("Tasks (space to toggle, ↑/↓ to move)\n\n")
		for i, item := range m.taskItems {
			cursor := " "
			if i == m.taskCursor {
				cursor = ">"
			}
			check := " "
			if item.selected {
				check = "x"
			}
			b.WriteString(fmt.Sprintf(" %s [%s] %-22s %s\n", cursor, check, item.def.Type, item.def.Name))
		}
		b.WriteString("\nPress Enter to continue.\n")
	case stepSchedule:
		b.WriteString("Schedule\n")
		b.WriteString("Use ↑/↓ to select, e to edit.\n\n")
		renderScheduleFields(&b, m)
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
		b.WriteString("Running snapshot (required)...\n")
		if m.snapshotRunning {
			b.WriteString(m.spinner.View() + "\n")
		} else {
			if m.snapshotErr != nil {
				b.WriteString("Snapshot error: " + m.snapshotErr.Error() + "\n")
			} else {
				b.WriteString(m.snapshotOutput + "\n")
			}
			b.WriteString("\nPress Enter to continue.\n")
		}
	case stepPreview:
		b.WriteString("Previewing next run...\n")
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
	case stepDaemon:
		b.WriteString("Daemon setup\n\n")
		b.WriteString(fmt.Sprintf("Service: %s\n", m.serviceType))
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
			b.WriteString(fmt.Sprintf(" %s %s\n", cursor, label))
		}
		b.WriteString("\nPress Enter to apply.\n")
	case stepFinish:
		b.WriteString("Setup complete.\n")
		b.WriteString("Nightshift is configured and ready to run.\n")
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
	case stepDaemon:
		m.serviceType, m.serviceState = detectServiceState()
	}
	return nil
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
			if _, err := os.Stat(path); err != nil {
				m.projectErr = "path not found"
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
		return m, m.setStep(stepTaskPreset)
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

func (m *setupModel) handleTaskInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "enter":
		m.applyTasks()
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
		if err != nil || v <= 0 {
			return fmt.Errorf("max_percent must be positive")
		}
		m.cfg.Budget.MaxPercent = v
	case 2:
		v, err := strconv.Atoi(value)
		if err != nil || v < 0 {
			return fmt.Errorf("reserve_percent must be >= 0")
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
		if err := runDaemonStop(nil, nil); err != nil {
			// ignore if not running
		}
		return uninstallService(m.serviceType)
	default:
		return nil
	}
}

func renderEnvChecks(cfg *config.Config) string {
	var b strings.Builder
	if _, err := execLookPath("nightshift"); err != nil {
		b.WriteString("  FAIL: nightshift not found in PATH\n")
	} else {
		b.WriteString("  OK: nightshift in PATH\n")
	}
	if _, err := execLookPath("tmux"); err != nil {
		b.WriteString("  WARN: tmux not found (calibration will be local-only)\n")
	} else {
		b.WriteString("  OK: tmux available\n")
	}
	if cfg.Providers.Claude.Enabled {
		if _, err := os.Stat(cfg.ExpandedProviderPath("claude")); err != nil {
			b.WriteString("  WARN: Claude data path not found\n")
		} else {
			b.WriteString("  OK: Claude data path\n")
		}
	}
	if cfg.Providers.Codex.Enabled {
		if _, err := os.Stat(cfg.ExpandedProviderPath("codex")); err != nil {
			b.WriteString("  WARN: Codex data path not found\n")
		} else {
			b.WriteString("  OK: Codex data path\n")
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
		b.WriteString(fmt.Sprintf(" %s %s\n", cursor, field))
	}
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
		b.WriteString(fmt.Sprintf(" %s %s\n", cursor, field))
	}
	if m.scheduleMode == "interval" {
		start, errStart := scheduler.ParseTimeOfDay(m.scheduleStart)
		interval, errInterval := time.ParseDuration(m.scheduleInterval)
		if errStart == nil && errInterval == nil {
			end := computeWindowEnd(start, interval, m.scheduleCycles)
			b.WriteString(fmt.Sprintf("   Window end (computed): %s\n", end))
		}
	}
}

func makeTaskItems(cfg *config.Config, projects []string, preset setup.Preset) []taskItem {
	defs := tasks.AllDefinitions()
	signals := setup.DetectRepoSignals(projects)
	selected := setup.PresetTasks(preset, defs, signals)

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
	defer database.Close()

	scraper := snapshots.UsageScraper(nil)
	if cfg.Budget.CalibrateEnabled && strings.ToLower(cfg.Budget.BillingMode) != "api" {
		scraper = tmuxScraper{}
	}

	collector := snapshots.NewCollector(
		database,
		providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude")),
		providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex")),
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
		output, err := buildPreviewOutput(cfg, projects, 1, false, "")
		return previewMsg{output: output, err: err}
	}
}

func buildPreviewOutput(cfg *config.Config, projects []string, runs int, longPrompt bool, writeDir string) (string, error) {
	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return "", err
	}
	defer database.Close()

	var buf bytes.Buffer
	if err := renderPreview(&buf, cfg, database, projects, "", runs, longPrompt, writeDir); err != nil {
		return "", err
	}
	return buf.String(), nil
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
