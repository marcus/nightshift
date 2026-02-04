package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_CronAndInterval(t *testing.T) {
	cfg := &Config{
		Schedule: ScheduleConfig{
			Cron:     "0 2 * * *",
			Interval: "1h",
		},
	}
	err := Validate(cfg)
	if err != ErrCronAndInterval {
		t.Errorf("expected ErrCronAndInterval, got %v", err)
	}
}

func TestValidate_InvalidBudgetMode(t *testing.T) {
	cfg := &Config{
		Budget: BudgetConfig{
			Mode: "invalid",
		},
	}
	err := Validate(cfg)
	if err != ErrInvalidBudgetMode {
		t.Errorf("expected ErrInvalidBudgetMode, got %v", err)
	}
}

func TestValidate_InvalidMaxPercent(t *testing.T) {
	cfg := &Config{
		Budget: BudgetConfig{
			MaxPercent: 150,
		},
	}
	err := Validate(cfg)
	if err != ErrInvalidMaxPercent {
		t.Errorf("expected ErrInvalidMaxPercent, got %v", err)
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		Logging: LoggingConfig{
			Level: "verbose",
		},
	}
	err := Validate(cfg)
	if err != ErrInvalidLogLevel {
		t.Errorf("expected ErrInvalidLogLevel, got %v", err)
	}
}

func TestValidate_InvalidLogFormat(t *testing.T) {
	cfg := &Config{
		Logging: LoggingConfig{
			Format: "xml",
		},
	}
	err := Validate(cfg)
	if err != ErrInvalidLogFormat {
		t.Errorf("expected ErrInvalidLogFormat, got %v", err)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Schedule: ScheduleConfig{
			Cron: "0 2 * * *",
		},
		Budget: BudgetConfig{
			Mode:           "daily",
			MaxPercent:     10,
			ReservePercent: 5,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
	err := Validate(cfg)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}
	for _, tc := range tests {
		result := expandPath(tc.input)
		if result != tc.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestGetProviderBudget(t *testing.T) {
	cfg := &Config{
		Budget: BudgetConfig{
			WeeklyTokens: 700000,
			PerProvider: map[string]int{
				"claude": 800000,
			},
		},
	}

	// Test with per-provider override
	if got := cfg.GetProviderBudget("claude"); got != 800000 {
		t.Errorf("GetProviderBudget(claude) = %d, want 800000", got)
	}

	// Test fallback to weekly tokens
	if got := cfg.GetProviderBudget("codex"); got != 700000 {
		t.Errorf("GetProviderBudget(codex) = %d, want 700000", got)
	}
}

func TestIsTaskEnabled(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Enabled:  []string{"lint", "docs"},
			Disabled: []string{"idea-generator"},
		},
	}

	tests := []struct {
		task     string
		expected bool
	}{
		{"lint", true},
		{"docs", true},
		{"idea-generator", false},
		{"security", false},
	}

	for _, tc := range tests {
		if got := cfg.IsTaskEnabled(tc.task); got != tc.expected {
			t.Errorf("IsTaskEnabled(%q) = %v, want %v", tc.task, got, tc.expected)
		}
	}
}

func TestIsTaskEnabled_EmptyEnabled(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Disabled: []string{"idea-generator"},
		},
	}

	// With empty enabled list, all non-disabled tasks are enabled
	if !cfg.IsTaskEnabled("lint") {
		t.Error("expected lint to be enabled")
	}
	if cfg.IsTaskEnabled("idea-generator") {
		t.Error("expected idea-generator to be disabled")
	}
}

func TestGetTaskPriority(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Priorities: map[string]int{
				"lint":     1,
				"security": 2,
			},
		},
	}

	if got := cfg.GetTaskPriority("lint"); got != 1 {
		t.Errorf("GetTaskPriority(lint) = %d, want 1", got)
	}
	if got := cfg.GetTaskPriority("docs"); got != 0 {
		t.Errorf("GetTaskPriority(docs) = %d, want 0", got)
	}
}

func TestLoadFromPaths_WithYAML(t *testing.T) {
	// Create temp dir with config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nightshift.yaml")

	configContent := `
schedule:
  cron: "0 3 * * *"
budget:
  mode: weekly
  max_percent: 20
logging:
  level: debug
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load with non-existent global config
	cfg, err := LoadFromPaths(tmpDir, filepath.Join(tmpDir, "nonexistent", "global.yaml"))
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	if cfg.Schedule.Cron != "0 3 * * *" {
		t.Errorf("Schedule.Cron = %q, want %q", cfg.Schedule.Cron, "0 3 * * *")
	}
	if cfg.Budget.Mode != "weekly" {
		t.Errorf("Budget.Mode = %q, want %q", cfg.Budget.Mode, "weekly")
	}
	if cfg.Budget.MaxPercent != 20 {
		t.Errorf("Budget.MaxPercent = %d, want 20", cfg.Budget.MaxPercent)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
}

func TestLoadFromPaths_MergeConfigs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create global config
	globalDir := filepath.Join(tmpDir, "global")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := filepath.Join(globalDir, "config.yaml")
	globalContent := `
budget:
  mode: daily
  max_percent: 10
logging:
  level: info
`
	if err := os.WriteFile(globalConfig, []byte(globalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create project config
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	projectConfig := filepath.Join(projectDir, "nightshift.yaml")
	projectContent := `
budget:
  max_percent: 15
logging:
  level: debug
`
	if err := os.WriteFile(projectConfig, []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(projectDir, globalConfig)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Project config should override global
	if cfg.Budget.MaxPercent != 15 {
		t.Errorf("Budget.MaxPercent = %d, want 15 (project override)", cfg.Budget.MaxPercent)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want debug (project override)", cfg.Logging.Level)
	}
	// Global value should still be present for non-overridden fields
	if cfg.Budget.Mode != "daily" {
		t.Errorf("Budget.Mode = %q, want daily (from global)", cfg.Budget.Mode)
	}
}

func TestLoadFromPaths_Defaults(t *testing.T) {
	tmpDir := t.TempDir()

	// Load with no config files
	cfg, err := LoadFromPaths(tmpDir, filepath.Join(tmpDir, "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Check defaults are applied
	if cfg.Budget.Mode != DefaultBudgetMode {
		t.Errorf("Budget.Mode = %q, want %q", cfg.Budget.Mode, DefaultBudgetMode)
	}
	if cfg.Budget.MaxPercent != DefaultMaxPercent {
		t.Errorf("Budget.MaxPercent = %d, want %d", cfg.Budget.MaxPercent, DefaultMaxPercent)
	}
	if cfg.Budget.WeeklyTokens != DefaultWeeklyTokens {
		t.Errorf("Budget.WeeklyTokens = %d, want %d", cfg.Budget.WeeklyTokens, DefaultWeeklyTokens)
	}
	if cfg.Logging.Level != DefaultLogLevel {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, DefaultLogLevel)
	}
	if cfg.Providers.Claude.DataPath != DefaultClaudeDataPath {
		t.Errorf("Providers.Claude.DataPath = %q, want %q", cfg.Providers.Claude.DataPath, DefaultClaudeDataPath)
	}
}
