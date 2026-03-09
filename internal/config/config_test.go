package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestValidate_InvalidBillingMode(t *testing.T) {
	cfg := &Config{
		Budget: BudgetConfig{
			BillingMode: "metered",
		},
	}
	err := Validate(cfg)
	if err != ErrInvalidBillingMode {
		t.Errorf("expected ErrInvalidBillingMode, got %v", err)
	}
}

func TestValidate_InvalidWeekStartDay(t *testing.T) {
	cfg := &Config{
		Budget: BudgetConfig{
			WeekStartDay: "friday",
		},
	}
	err := Validate(cfg)
	if err != ErrInvalidWeekStartDay {
		t.Errorf("expected ErrInvalidWeekStartDay, got %v", err)
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

func TestGetCopilotMonthlyLimit(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected int64
	}{
		{
			name: "explicit monthly limit takes priority",
			cfg: Config{
				Providers: ProvidersConfig{
					Copilot: ProviderConfig{
						MonthlyLimit: 500,
						CopilotPlan:  "pro", // ignored when MonthlyLimit is set
					},
				},
				Budget: BudgetConfig{WeeklyTokens: 100},
			},
			expected: 500,
		},
		{
			name: "plan preset: free",
			cfg: Config{
				Providers: ProvidersConfig{
					Copilot: ProviderConfig{CopilotPlan: "free"},
				},
			},
			expected: 50,
		},
		{
			name: "plan preset: pro",
			cfg: Config{
				Providers: ProvidersConfig{
					Copilot: ProviderConfig{CopilotPlan: "pro"},
				},
			},
			expected: 300,
		},
		{
			name: "plan preset: pro_plus",
			cfg: Config{
				Providers: ProvidersConfig{
					Copilot: ProviderConfig{CopilotPlan: "pro_plus"},
				},
			},
			expected: 1500,
		},
		{
			name: "plan preset: business",
			cfg: Config{
				Providers: ProvidersConfig{
					Copilot: ProviderConfig{CopilotPlan: "business"},
				},
			},
			expected: 300,
		},
		{
			name: "plan preset: enterprise",
			cfg: Config{
				Providers: ProvidersConfig{
					Copilot: ProviderConfig{CopilotPlan: "enterprise"},
				},
			},
			expected: 1000,
		},
		{
			name: "fallback to weekly tokens * 4",
			cfg: Config{
				Budget: BudgetConfig{WeeklyTokens: 200},
			},
			expected: 800,
		},
		{
			name: "unknown plan falls back to weekly tokens * 4",
			cfg: Config{
				Providers: ProvidersConfig{
					Copilot: ProviderConfig{CopilotPlan: "unknown_plan"},
				},
				Budget: BudgetConfig{WeeklyTokens: 150},
			},
			expected: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetCopilotMonthlyLimit()
			if got != tt.expected {
				t.Errorf("GetCopilotMonthlyLimit() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestNormalizeBudgetConfig(t *testing.T) {
	cfg := &Config{
		Budget: BudgetConfig{
			BillingMode:      "api",
			CalibrateEnabled: true,
		},
	}

	normalizeBudgetConfig(cfg)

	if cfg.Budget.CalibrateEnabled {
		t.Errorf("expected CalibrateEnabled=false for api billing mode")
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

func TestIsTaskExplicitlyEnabled(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Enabled: []string{"lint", "docs"},
		},
	}

	if !cfg.IsTaskExplicitlyEnabled("lint") {
		t.Error("expected lint to be explicitly enabled")
	}
	if cfg.IsTaskExplicitlyEnabled("security") {
		t.Error("expected security to not be explicitly enabled")
	}

	// Empty list
	cfgEmpty := &Config{}
	if cfgEmpty.IsTaskExplicitlyEnabled("lint") {
		t.Error("expected lint to not be explicitly enabled with empty list")
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
  max_percent: 75
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

func TestGetTaskInterval_Override(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Intervals: map[string]string{
				"lint": "30m",
				"docs": "2h",
			},
		},
	}

	if got := cfg.GetTaskInterval("lint"); got != 30*time.Minute {
		t.Errorf("GetTaskInterval(lint) = %v, want 30m", got)
	}
	if got := cfg.GetTaskInterval("docs"); got != 2*time.Hour {
		t.Errorf("GetTaskInterval(docs) = %v, want 2h", got)
	}
}

func TestGetTaskInterval_NotSet(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Intervals: map[string]string{
				"lint": "30m",
			},
		},
	}

	if got := cfg.GetTaskInterval("security"); got != 0 {
		t.Errorf("GetTaskInterval(security) = %v, want 0", got)
	}

	// Also test with nil map
	cfgNil := &Config{}
	if got := cfgNil.GetTaskInterval("lint"); got != 0 {
		t.Errorf("GetTaskInterval(lint) with nil map = %v, want 0", got)
	}
}

func TestValidate_InvalidTaskInterval(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Intervals: map[string]string{
				"lint": "not-a-duration",
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid interval duration, got nil")
	}
	if !strings.Contains(err.Error(), "tasks.intervals") {
		t.Errorf("error should mention tasks.intervals, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not-a-duration") {
		t.Errorf("error should mention the bad value, got: %v", err)
	}
}

func TestValidate_ValidTaskInterval(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Intervals: map[string]string{
				"lint": "30m",
				"docs": "2h30m",
			},
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("expected nil for valid intervals, got %v", err)
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

func TestValidate_CustomTaskValid(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{
					Type:        "my-review",
					Name:        "My Review",
					Description: "Review all the things",
					Category:    "pr",
					CostTier:    "medium",
					RiskLevel:   "low",
					Interval:    "48h",
				},
			},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_CustomTaskMinimal(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{
					Type:        "simple-task",
					Name:        "Simple",
					Description: "A simple task",
				},
			},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_CustomTaskMissingType(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Name: "No Type", Description: "desc"},
			},
		},
	}
	if err := Validate(cfg); !errors.Is(err, ErrCustomTaskMissingType) {
		t.Errorf("expected ErrCustomTaskMissingType, got %v", err)
	}
}

func TestValidate_CustomTaskInvalidType(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "My Task!", Name: "Bad Type", Description: "desc"},
			},
		},
	}
	if err := Validate(cfg); !errors.Is(err, ErrCustomTaskInvalidType) {
		t.Errorf("expected ErrCustomTaskInvalidType, got %v", err)
	}
}

func TestValidate_CustomTaskMissingName(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "good-type", Description: "desc"},
			},
		},
	}
	if err := Validate(cfg); !errors.Is(err, ErrCustomTaskMissingName) {
		t.Errorf("expected ErrCustomTaskMissingName, got %v", err)
	}
}

func TestValidate_CustomTaskMissingDescription(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "good-type", Name: "Good Name"},
			},
		},
	}
	if err := Validate(cfg); !errors.Is(err, ErrCustomTaskMissingDescription) {
		t.Errorf("expected ErrCustomTaskMissingDescription, got %v", err)
	}
}

func TestValidate_CustomTaskInvalidCategory(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "t", Name: "n", Description: "d", Category: "invalid"},
			},
		},
	}
	if err := Validate(cfg); !errors.Is(err, ErrCustomTaskInvalidCategory) {
		t.Errorf("expected ErrCustomTaskInvalidCategory, got %v", err)
	}
}

func TestValidate_CustomTaskInvalidCostTier(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "t", Name: "n", Description: "d", CostTier: "extreme"},
			},
		},
	}
	if err := Validate(cfg); !errors.Is(err, ErrCustomTaskInvalidCostTier) {
		t.Errorf("expected ErrCustomTaskInvalidCostTier, got %v", err)
	}
}

func TestValidate_CustomTaskInvalidRiskLevel(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "t", Name: "n", Description: "d", RiskLevel: "extreme"},
			},
		},
	}
	if err := Validate(cfg); !errors.Is(err, ErrCustomTaskInvalidRiskLevel) {
		t.Errorf("expected ErrCustomTaskInvalidRiskLevel, got %v", err)
	}
}

func TestValidate_CustomTaskInvalidInterval(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "my-task", Name: "n", Description: "d", Interval: "not-a-duration"},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid interval, got nil")
	}
	if !strings.Contains(err.Error(), "my-task") {
		t.Errorf("error should contain task type, got: %v", err)
	}
}

func TestValidate_CustomTaskDuplicateType(t *testing.T) {
	cfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "dup-task", Name: "First", Description: "d1"},
				{Type: "dup-task", Name: "Second", Description: "d2"},
			},
		},
	}
	if err := Validate(cfg); !errors.Is(err, ErrCustomTaskDuplicateType) {
		t.Errorf("expected ErrCustomTaskDuplicateType, got %v", err)
	}
}
