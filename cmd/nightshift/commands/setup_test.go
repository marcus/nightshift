package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/setup"
	"github.com/marcus/nightshift/internal/tasks"
)

func TestApplyBudgetEdit_MaxPercentBounds(t *testing.T) {
	m := &setupModel{
		cfg:         &config.Config{},
		budgetInput: textinput.New(),
	}

	m.budgetCursor = 1
	m.budgetInput.SetValue("101")
	if err := m.applyBudgetEdit(); err == nil {
		t.Fatal("expected max_percent > 100 to fail")
	}

	m.budgetInput.SetValue("100")
	if err := m.applyBudgetEdit(); err != nil {
		t.Fatalf("expected max_percent=100 to pass: %v", err)
	}
}

func TestApplyBudgetEdit_ReservePercentBounds(t *testing.T) {
	m := &setupModel{
		cfg:         &config.Config{},
		budgetInput: textinput.New(),
	}

	m.budgetCursor = 2
	m.budgetInput.SetValue("101")
	if err := m.applyBudgetEdit(); err == nil {
		t.Fatal("expected reserve_percent > 100 to fail")
	}

	m.budgetInput.SetValue("100")
	if err := m.applyBudgetEdit(); err != nil {
		t.Fatalf("expected reserve_percent=100 to pass: %v", err)
	}
}

func TestHandleProjectsInput_RejectsFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-dir.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	m := &setupModel{
		projectEditing: true,
		projectInput:   textinput.New(),
	}
	m.projectInput.SetValue(filePath)

	model, _ := m.handleProjectsInput(tea.KeyMsg{Type: tea.KeyEnter})
	got, ok := model.(*setupModel)
	if !ok {
		t.Fatal("expected *setupModel type")
	}
	if got.projectErr != "path must be a directory" {
		t.Fatalf("projectErr = %q, want %q", got.projectErr, "path must be a directory")
	}
}

func TestHandleTaskInput_RequiresSelection(t *testing.T) {
	m := &setupModel{
		taskItems: []taskItem{
			{
				def:      tasks.TaskDefinition{Type: tasks.TaskType("unit-test-task")},
				selected: false,
			},
		},
	}

	model, cmd := m.handleTaskInput(tea.KeyMsg{Type: tea.KeyEnter})
	got, ok := model.(*setupModel)
	if !ok {
		t.Fatal("expected *setupModel type")
	}
	if cmd != nil {
		t.Fatal("expected no transition cmd when no tasks selected")
	}
	if got.taskErr != "select at least one task" {
		t.Fatalf("taskErr = %q, want %q", got.taskErr, "select at least one task")
	}
}

func TestHandleTaskInput_NoTasksDoesNotPanic(t *testing.T) {
	m := &setupModel{}
	if _, _ = m.handleTaskInput(tea.KeyMsg{Type: tea.KeySpace}); m.taskErr != "" {
		t.Fatalf("taskErr = %q, want empty for non-enter input", m.taskErr)
	}
}

func TestEnsurePathInShell_SubstringDoesNotBlockInsert(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".zshrc")
	if err := os.WriteFile(cfgPath, []byte("export PATH=\"$PATH:/opt/bin2\"\n"), 0644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	changed, err := ensurePathInShell(cfgPath, "zsh", "/opt/bin")
	if err != nil {
		t.Fatalf("ensurePathInShell: %v", err)
	}
	if !changed {
		t.Fatal("expected config to change when only substring path exists")
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !shellConfigHasPath(string(data), "/opt/bin") {
		t.Fatal("expected new path token to be present")
	}
}

func TestEnsurePathInShell_ExactPathNoChange(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".zshrc")
	if err := os.WriteFile(cfgPath, []byte("export PATH=\"$PATH:/opt/bin\"\n"), 0644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	changed, err := ensurePathInShell(cfgPath, "zsh", "/opt/bin")
	if err != nil {
		t.Fatalf("ensurePathInShell: %v", err)
	}
	if changed {
		t.Fatal("expected no change when exact path token exists")
	}
}

func TestMakeTaskItems_UsesSortedDefinitions(t *testing.T) {
	cfg := &config.Config{}
	items := makeTaskItems(cfg, nil, setup.PresetBalanced)
	defs := tasks.AllDefinitionsSorted()

	if len(items) != len(defs) {
		t.Fatalf("len(items)=%d len(defs)=%d", len(items), len(defs))
	}

	for i := range defs {
		if items[i].def.Type != defs[i].Type {
			t.Fatalf("item[%d].Type=%q want %q", i, items[i].def.Type, defs[i].Type)
		}
	}
}

func TestMakeTaskItems_PreservesExplicitEnabledTasks(t *testing.T) {
	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{string(tasks.TaskBugFinder)},
		},
	}

	items := makeTaskItems(cfg, nil, setup.PresetBalanced)
	found := false
	for _, item := range items {
		if item.def.Type != tasks.TaskBugFinder {
			continue
		}
		found = true
		if !item.selected {
			t.Fatal("expected explicitly enabled task to remain selected")
		}
	}
	if !found {
		t.Fatal("expected bug-finder task to exist in setup list")
	}
}

func TestWriteGlobalConfig_ProviderYAMLKeys(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Seed an initial config so WriteConfig can overwrite it.
	if err := os.WriteFile(cfgPath, []byte("# nightshift config\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	cfg := &config.Config{}
	cfg.Providers.Claude.Enabled = true
	cfg.Providers.Claude.DataPath = "/tmp/claude-data"
	cfg.Providers.Claude.DangerouslySkipPermissions = true
	cfg.Providers.Claude.DangerouslyBypassApprovalsAndSandbox = true
	cfg.Providers.Codex.DataPath = "/tmp/codex-data"

	if err := writeGlobalConfigToPath(cfg, cfgPath); err != nil {
		t.Fatalf("writeGlobalConfigToPath: %v", err)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(raw)

	// Must use snake_case mapstructure tag names, not lowercased Go field names.
	wantKeys := []string{
		"data_path",
		"dangerously_skip_permissions",
		"dangerously_bypass_approvals_and_sandbox",
	}
	for _, key := range wantKeys {
		if !containsStr(content, key) {
			t.Errorf("expected YAML key %q in config output, got:\n%s", key, content)
		}
	}

	// Must NOT contain the lowercased Go field name variants.
	badKeys := []string{
		"datapath",
		"dangerouslyskippermissions",
		"dangerouslybypassapprovalsandsandbox",
	}
	for _, key := range badKeys {
		if containsStr(content, key) {
			t.Errorf("found incorrect lowercased key %q in config output, got:\n%s", key, content)
		}
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
