package config

import (
	"os"
	"path/filepath"
	"testing"
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
