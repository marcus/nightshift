package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBackwardCompat_OldConfigLoadsWithNewDefaults verifies that config files
// from v0.3.0 (which may not specify dangerous_* flags) load correctly
// with v0.3.1 security defaults (dangerous_* = false).
func TestBackwardCompat_OldConfigLoadsWithNewDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate old config from v0.3.0 that doesn't mention dangerous_* flags
	oldConfigContent := `
budget:
  mode: daily
  max_percent: 75
  weekly_tokens: 700000
logging:
  level: info
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(oldConfigContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// v0.3.1 default: dangerous_skip_permissions must be false (security default)
	if cfg.Providers.Claude.DangerouslySkipPermissions {
		t.Error("Claude.DangerouslySkipPermissions should default to false, got true")
	}

	// v0.3.1 default: dangerous_bypass_approvals_and_sandbox must be false
	if cfg.Providers.Codex.DangerouslyBypassApprovalsAndSandbox {
		t.Error("Codex.DangerouslyBypassApprovalsAndSandbox should default to false, got true")
	}
}

// TestBackwardCompat_ExplicitDangerousTrue verifies that users who
// explicitly set dangerous flags to true in their v0.3.0 config
// still get the correct behavior in v0.3.1.
func TestBackwardCompat_ExplicitDangerousTrue(t *testing.T) {
	tmpDir := t.TempDir()

	// Old config that explicitly enabled dangerous_skip_permissions
	oldConfigWithDangerous := `
budget:
  mode: daily
providers:
  claude:
    enabled: true
    dangerously_skip_permissions: true
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(oldConfigWithDangerous), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Explicitly set value should be preserved
	if !cfg.Providers.Claude.DangerouslySkipPermissions {
		t.Error("Claude.DangerouslySkipPermissions should be true (explicitly set), got false")
	}
}

// TestBackwardCompat_ExplicitDangerousFalse verifies that users who
// explicitly set dangerous flags to false in their v0.3.0 config
// still get the correct behavior.
func TestBackwardCompat_ExplicitDangerousFalse(t *testing.T) {
	tmpDir := t.TempDir()

	// Config that explicitly disabled dangerous flags
	configWithSafe := `
providers:
  claude:
    enabled: true
    dangerously_skip_permissions: false
  codex:
    enabled: true
    dangerously_bypass_approvals_and_sandbox: false
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(configWithSafe), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Explicitly false values should be preserved
	if cfg.Providers.Claude.DangerouslySkipPermissions {
		t.Error("Claude.DangerouslySkipPermissions should be false, got true")
	}
	if cfg.Providers.Codex.DangerouslyBypassApprovalsAndSandbox {
		t.Error("Codex.DangerouslyBypassApprovalsAndSandbox should be false, got true")
	}
}

// TestBackwardCompat_MixedConfig verifies handling of configs where
// some dangerous flags are set and some are not.
func TestBackwardCompat_MixedConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Config with only one dangerous flag explicitly set
	mixedConfig := `
providers:
  claude:
    enabled: true
    dangerously_skip_permissions: true
  codex:
    enabled: true
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(mixedConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Explicitly set should be true
	if !cfg.Providers.Claude.DangerouslySkipPermissions {
		t.Error("Claude.DangerouslySkipPermissions should be true, got false")
	}

	// Not set should default to false
	if cfg.Providers.Codex.DangerouslyBypassApprovalsAndSandbox {
		t.Error("Codex.DangerouslyBypassApprovalsAndSandbox should default to false, got true")
	}
}

// TestBackwardCompat_ValidationStillWorks verifies that validation
// rules haven't changed and old valid configs still validate.
func TestBackwardCompat_ValidationStillWorks(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid old-style config",
			config: &Config{
				Schedule: ScheduleConfig{
					Cron: "0 2 * * *",
				},
				Budget: BudgetConfig{
					Mode:           "daily",
					MaxPercent:     75,
					ReservePercent: 5,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid budget mode still invalid",
			config: &Config{
				Budget: BudgetConfig{
					Mode: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level still invalid",
			config: &Config{
				Logging: LoggingConfig{
					Level: "verbose",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBackwardCompat_EnvironmentOverrides verifies that environment
// variable overrides still work correctly.
func TestBackwardCompat_EnvironmentOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	oldConfig := `
budget:
  mode: daily
  max_percent: 75
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Set environment override
	t.Setenv("NIGHTSHIFT_BUDGET_MODE", "weekly")

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Environment should override file
	if cfg.Budget.Mode != "weekly" {
		t.Errorf("Budget.Mode = %q, want weekly (env override)", cfg.Budget.Mode)
	}

	// File value should still be used for unset env vars
	if cfg.Budget.MaxPercent != 75 {
		t.Errorf("Budget.MaxPercent = %d, want 75 (from file)", cfg.Budget.MaxPercent)
	}
}

// TestBackwardCompat_ProjectConfigMerging verifies that project-level
// configs still merge correctly with global config.
func TestBackwardCompat_ProjectConfigMerging(t *testing.T) {
	tmpDir := t.TempDir()

	// Global config
	globalDir := filepath.Join(tmpDir, "global")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfigPath := filepath.Join(globalDir, "config.yaml")
	globalContent := `
budget:
  mode: daily
  max_percent: 75
providers:
  claude:
    enabled: true
    dangerously_skip_permissions: false
`
	if err := os.WriteFile(globalConfigPath, []byte(globalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Project config (partial override)
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	projectConfigPath := filepath.Join(projectDir, "nightshift.yaml")
	projectContent := `
budget:
  max_percent: 15
providers:
  claude:
    dangerously_skip_permissions: true
`
	if err := os.WriteFile(projectConfigPath, []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(projectDir, globalConfigPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Project should override global
	if cfg.Budget.MaxPercent != 15 {
		t.Errorf("Budget.MaxPercent = %d, want 15 (project override)", cfg.Budget.MaxPercent)
	}
	if !cfg.Providers.Claude.DangerouslySkipPermissions {
		t.Errorf("Claude.DangerouslySkipPermissions = %v, want true (project override)", cfg.Providers.Claude.DangerouslySkipPermissions)
	}

	// Global value should still apply for non-overridden fields
	if cfg.Budget.Mode != "daily" {
		t.Errorf("Budget.Mode = %q, want daily (from global)", cfg.Budget.Mode)
	}
}

// TestBackwardCompat_DefaultsPreserved verifies that other defaults
// haven't changed and old configs get all necessary defaults.
func TestBackwardCompat_DefaultsPreserved(t *testing.T) {
	tmpDir := t.TempDir()

	// Minimal old config
	minimalConfig := `
budget:
  mode: daily
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(minimalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Check that defaults still exist
	if cfg.Budget.MaxPercent != DefaultMaxPercent {
		t.Errorf("Budget.MaxPercent = %d, want %d (default)", cfg.Budget.MaxPercent, DefaultMaxPercent)
	}
	if cfg.Budget.ReservePercent != DefaultReservePercent {
		t.Errorf("Budget.ReservePercent = %d, want %d (default)", cfg.Budget.ReservePercent, DefaultReservePercent)
	}
	if cfg.Budget.WeeklyTokens != DefaultWeeklyTokens {
		t.Errorf("Budget.WeeklyTokens = %d, want %d (default)", cfg.Budget.WeeklyTokens, DefaultWeeklyTokens)
	}
	if cfg.Logging.Level != DefaultLogLevel {
		t.Errorf("Logging.Level = %q, want %q (default)", cfg.Logging.Level, DefaultLogLevel)
	}
	if cfg.Logging.Format != DefaultLogFormat {
		t.Errorf("Logging.Format = %q, want %q (default)", cfg.Logging.Format, DefaultLogFormat)
	}
	if cfg.Budget.BillingMode != DefaultBillingMode {
		t.Errorf("Budget.BillingMode = %q, want %q (default)", cfg.Budget.BillingMode, DefaultBillingMode)
	}
}

// TestBackwardCompat_CopilotProviderConfig verifies that configs including
// copilot provider settings (added in v0.3.2) load correctly alongside
// older claude/codex settings.
func TestBackwardCompat_CopilotProviderConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Config that includes all three providers (v0.3.2+ format)
	configWithCopilot := `
providers:
  claude:
    enabled: true
    data_path: ~/.claude
    dangerously_skip_permissions: false
  codex:
    enabled: true
    data_path: ~/.codex
    dangerously_bypass_approvals_and_sandbox: false
  copilot:
    enabled: true
    data_path: ~/.copilot
    dangerously_skip_permissions: false
  preference:
    - claude
    - codex
    - copilot
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(configWithCopilot), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	if !cfg.Providers.Copilot.Enabled {
		t.Error("Copilot.Enabled should be true")
	}
	if cfg.Providers.Copilot.DataPath != "~/.copilot" {
		t.Errorf("Copilot.DataPath = %q, want ~/.copilot", cfg.Providers.Copilot.DataPath)
	}
	if cfg.Providers.Copilot.DangerouslySkipPermissions {
		t.Error("Copilot.DangerouslySkipPermissions should be false")
	}
	if len(cfg.Providers.Preference) != 3 {
		t.Errorf("Providers.Preference length = %d, want 3", len(cfg.Providers.Preference))
	}
}

// TestBackwardCompat_PreCopilotConfigStillWorks verifies that configs from
// before v0.3.2 (that don't mention copilot at all) still load correctly
// and get appropriate defaults for the copilot provider.
func TestBackwardCompat_PreCopilotConfigStillWorks(t *testing.T) {
	tmpDir := t.TempDir()

	// Config from before copilot was added — no copilot section at all
	oldConfig := `
providers:
  claude:
    enabled: true
    data_path: ~/.claude
  codex:
    enabled: true
    data_path: ~/.codex
budget:
  mode: daily
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Copilot should get defaults even when not mentioned in config
	if !cfg.Providers.Copilot.Enabled {
		t.Error("Copilot.Enabled should default to true")
	}
	if cfg.Providers.Copilot.DangerouslySkipPermissions {
		t.Error("Copilot.DangerouslySkipPermissions should default to false")
	}
}

// TestBackwardCompat_ConfigFieldsStable verifies that all expected fields
// exist on the Config struct by constructing a full config. A compilation
// failure here means a field was removed or renamed.
func TestBackwardCompat_ConfigFieldsStable(t *testing.T) {
	// This test verifies the Config struct shape via compilation.
	// If any field is removed or renamed, this won't compile.
	cfg := Config{
		Schedule: ScheduleConfig{
			Cron:        "0 2 * * *",
			Interval:    "",
			Window:      nil,
			MaxProjects: 3,
			MaxTasks:    2,
		},
		Budget: BudgetConfig{
			Mode:                  "daily",
			MaxPercent:            75,
			AggressiveEndOfWeek:   true,
			ReservePercent:        5,
			WeeklyTokens:          700000,
			PerProvider:           map[string]int{"claude": 500000},
			BillingMode:           "subscription",
			CalibrateEnabled:      true,
			SnapshotInterval:      "30m",
			SnapshotRetentionDays: 90,
			WeekStartDay:          "monday",
			DBPath:                "/tmp/test.db",
		},
		Providers: ProvidersConfig{
			Claude: ProviderConfig{
				Enabled:                              true,
				DataPath:                             "~/.claude",
				DangerouslySkipPermissions:           false,
				DangerouslyBypassApprovalsAndSandbox: false,
			},
			Codex: ProviderConfig{
				Enabled:  true,
				DataPath: "~/.codex",
			},
			Copilot: ProviderConfig{
				Enabled:  true,
				DataPath: "~/.copilot",
			},
			Preference: []string{"claude", "codex", "copilot"},
		},
		Projects: []ProjectConfig{
			{
				Path:     "/tmp/project",
				Priority: 1,
				Tasks:    []string{"lint-fix"},
				Config:   "",
				Pattern:  "",
				Exclude:  nil,
			},
		},
		Tasks: TasksConfig{
			Enabled:    []string{"lint-fix"},
			Priorities: map[string]int{"lint-fix": 1},
			Disabled:   nil,
			Intervals:  map[string]string{"lint-fix": "24h"},
			Custom: []CustomTaskConfig{
				{
					Type:        "my-task",
					Name:        "My Task",
					Description: "test",
					Category:    "pr",
					CostTier:    "low",
					RiskLevel:   "low",
					Interval:    "48h",
				},
			},
		},
		Integrations: IntegrationsConfig{
			ClaudeMD: true,
			AgentsMD: true,
			TaskSources: []TaskSourceEntry{
				{GithubIssues: true},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Path:   "/tmp/logs",
			Format: "json",
		},
		Reporting: ReportingConfig{
			MorningSummary: true,
		},
	}

	// Sanity check that it validates
	if err := Validate(&cfg); err != nil {
		t.Fatalf("Full config validation failed: %v", err)
	}
}

// TestBackwardCompat_DefaultConstants verifies that default constant values
// haven't changed, since they affect behavior for users with minimal configs.
func TestBackwardCompat_DefaultConstants(t *testing.T) {
	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"DefaultBudgetMode", DefaultBudgetMode, "daily"},
		{"DefaultMaxPercent", DefaultMaxPercent, 75},
		{"DefaultReservePercent", DefaultReservePercent, 5},
		{"DefaultWeeklyTokens", DefaultWeeklyTokens, 700000},
		{"DefaultBillingMode", DefaultBillingMode, "subscription"},
		{"DefaultSnapshotInterval", DefaultSnapshotInterval, "30m"},
		{"DefaultSnapshotRetention", DefaultSnapshotRetention, 90},
		{"DefaultWeekStartDay", DefaultWeekStartDay, "monday"},
		{"DefaultLogLevel", DefaultLogLevel, "info"},
		{"DefaultLogFormat", DefaultLogFormat, "json"},
		{"DefaultClaudeDataPath", DefaultClaudeDataPath, "~/.claude"},
		{"DefaultCodexDataPath", DefaultCodexDataPath, "~/.codex"},
		{"DefaultCopilotDataPath", DefaultCopilotDataPath, "~/.copilot"},
		{"ProjectConfigName", ProjectConfigName, "nightshift.yaml"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v — changing defaults breaks existing minimal configs",
				c.name, c.got, c.want)
		}
	}
}

// TestBackwardCompat_CustomTaskValidation verifies that custom task
// validation rules are stable.
func TestBackwardCompat_CustomTaskValidation(t *testing.T) {
	// Valid custom task must pass
	validCfg := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{
					Type:        "my-review",
					Name:        "My Review",
					Description: "Custom review task",
					Category:    "pr",
					CostTier:    "medium",
					RiskLevel:   "low",
					Interval:    "48h",
				},
			},
		},
	}
	if err := Validate(validCfg); err != nil {
		t.Errorf("Valid custom task should pass: %v", err)
	}

	// Invalid category must fail
	invalidCat := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Type: "test", Name: "Test", Description: "test", Category: "invalid"},
			},
		},
	}
	if err := Validate(invalidCat); err == nil {
		t.Error("Invalid category should fail validation")
	}

	// Missing required fields must fail
	missingType := &Config{
		Tasks: TasksConfig{
			Custom: []CustomTaskConfig{
				{Name: "Test", Description: "test"},
			},
		},
	}
	if err := Validate(missingType); err == nil {
		t.Error("Missing type should fail validation")
	}
}

// TestBackwardCompat_HelperMethodsStable verifies that Config helper
// methods work as expected with various inputs.
func TestBackwardCompat_HelperMethodsStable(t *testing.T) {
	cfg := &Config{
		Budget: BudgetConfig{
			WeeklyTokens: 700000,
			PerProvider:   map[string]int{"claude": 500000},
		},
		Tasks: TasksConfig{
			Enabled:    []string{"lint-fix", "bug-finder"},
			Disabled:   []string{"dead-code"},
			Priorities: map[string]int{"lint-fix": 10},
			Intervals:  map[string]string{"lint-fix": "24h"},
		},
	}

	// GetProviderBudget: per-provider override
	if got := cfg.GetProviderBudget("claude"); got != 500000 {
		t.Errorf("GetProviderBudget(claude) = %d, want 500000", got)
	}
	// GetProviderBudget: fallback to default
	if got := cfg.GetProviderBudget("codex"); got != 700000 {
		t.Errorf("GetProviderBudget(codex) = %d, want 700000", got)
	}

	// IsTaskEnabled: in enabled list
	if !cfg.IsTaskEnabled("lint-fix") {
		t.Error("lint-fix should be enabled")
	}
	// IsTaskEnabled: not in enabled list
	if cfg.IsTaskEnabled("perf-profile") {
		t.Error("perf-profile should not be enabled (not in list)")
	}
	// IsTaskEnabled: explicitly disabled
	if cfg.IsTaskEnabled("dead-code") {
		t.Error("dead-code should be disabled")
	}

	// GetTaskPriority
	if got := cfg.GetTaskPriority("lint-fix"); got != 10 {
		t.Errorf("GetTaskPriority(lint-fix) = %d, want 10", got)
	}
	if got := cfg.GetTaskPriority("unknown"); got != 0 {
		t.Errorf("GetTaskPriority(unknown) = %d, want 0", got)
	}

	// GetTaskInterval
	if got := cfg.GetTaskInterval("lint-fix"); got != 24*time.Hour {
		t.Errorf("GetTaskInterval(lint-fix) = %v, want 24h", got)
	}
	if got := cfg.GetTaskInterval("unknown"); got != 0 {
		t.Errorf("GetTaskInterval(unknown) = %v, want 0", got)
	}
}

// TestBackwardCompat_ValidationErrorsStable verifies that validation
// error sentinel values haven't changed.
func TestBackwardCompat_ValidationErrorsStable(t *testing.T) {
	// These error values are part of the public API since users may
	// check for specific validation errors.
	errors := []struct {
		name string
		err  error
	}{
		{"ErrCronAndInterval", ErrCronAndInterval},
		{"ErrInvalidBudgetMode", ErrInvalidBudgetMode},
		{"ErrInvalidBillingMode", ErrInvalidBillingMode},
		{"ErrInvalidWeekStartDay", ErrInvalidWeekStartDay},
		{"ErrInvalidMaxPercent", ErrInvalidMaxPercent},
		{"ErrInvalidReservePercent", ErrInvalidReservePercent},
		{"ErrInvalidSnapshotRetention", ErrInvalidSnapshotRetention},
		{"ErrInvalidLogLevel", ErrInvalidLogLevel},
		{"ErrInvalidLogFormat", ErrInvalidLogFormat},
		{"ErrCustomTaskMissingType", ErrCustomTaskMissingType},
		{"ErrCustomTaskMissingName", ErrCustomTaskMissingName},
		{"ErrCustomTaskMissingDescription", ErrCustomTaskMissingDescription},
		{"ErrCustomTaskInvalidType", ErrCustomTaskInvalidType},
		{"ErrCustomTaskInvalidCategory", ErrCustomTaskInvalidCategory},
		{"ErrCustomTaskInvalidCostTier", ErrCustomTaskInvalidCostTier},
		{"ErrCustomTaskInvalidRiskLevel", ErrCustomTaskInvalidRiskLevel},
		{"ErrCustomTaskDuplicateType", ErrCustomTaskDuplicateType},
	}

	for _, e := range errors {
		if e.err == nil {
			t.Errorf("%s is nil — sentinel errors must not be removed", e.name)
		}
	}
}

// TestBackwardCompat_ProviderPathExpansion verifies that provider
// path expansion still works correctly.
func TestBackwardCompat_ProviderPathExpansion(t *testing.T) {
	tmpDir := t.TempDir()

	config := `
providers:
  claude:
    data_path: ~/.claude
  codex:
    data_path: ~/.codex
`
	configPath := filepath.Join(tmpDir, "nightshift.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPaths(tmpDir, configPath)
	if err != nil {
		t.Fatalf("LoadFromPaths error: %v", err)
	}

	// Paths should expand ~ correctly
	claudePath := cfg.ExpandedProviderPath("claude")
	if claudePath == "~/.claude" || claudePath == "" {
		t.Errorf("Claude path not expanded: %q", claudePath)
	}

	codexPath := cfg.ExpandedProviderPath("codex")
	if codexPath == "~/.codex" || codexPath == "" {
		t.Errorf("Codex path not expanded: %q", codexPath)
	}
}
