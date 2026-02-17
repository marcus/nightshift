// Package config handles loading and validating nightshift configuration.
// Supports YAML config files and environment variable overrides.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all nightshift configuration.
type Config struct {
	Schedule     ScheduleConfig     `mapstructure:"schedule"`
	Budget       BudgetConfig       `mapstructure:"budget"`
	Providers    ProvidersConfig    `mapstructure:"providers"`
	Projects     []ProjectConfig    `mapstructure:"projects"`
	Tasks        TasksConfig        `mapstructure:"tasks"`
	Integrations IntegrationsConfig `mapstructure:"integrations"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	Reporting    ReportingConfig    `mapstructure:"reporting"`
}

// ScheduleConfig defines when nightshift runs.
type ScheduleConfig struct {
	Cron     string        `mapstructure:"cron"`     // Cron expression (e.g., "0 2 * * *")
	Interval string        `mapstructure:"interval"` // Alternative: duration (e.g., "1h")
	Window   *WindowConfig `mapstructure:"window"`   // Optional time window constraint
}

// WindowConfig defines a time window for execution.
type WindowConfig struct {
	Start    string `mapstructure:"start"`    // Start time (e.g., "22:00")
	End      string `mapstructure:"end"`      // End time (e.g., "06:00")
	Timezone string `mapstructure:"timezone"` // Timezone (e.g., "America/Denver")
}

// BudgetConfig controls token budget allocation.
type BudgetConfig struct {
	Mode                  string         `mapstructure:"mode"`                    // daily | weekly
	MaxPercent            int            `mapstructure:"max_percent"`             // Max % of budget per run
	AggressiveEndOfWeek   bool           `mapstructure:"aggressive_end_of_week"`  // Ramp up in last 2 days
	ReservePercent        int            `mapstructure:"reserve_percent"`         // Always keep in reserve
	WeeklyTokens          int            `mapstructure:"weekly_tokens"`           // Fallback weekly budget
	PerProvider           map[string]int `mapstructure:"per_provider"`            // Per-provider overrides
	BillingMode           string         `mapstructure:"billing_mode"`            // subscription | api
	CalibrateEnabled      bool           `mapstructure:"calibrate_enabled"`       // Enable budget calibration
	SnapshotInterval      string         `mapstructure:"snapshot_interval"`       // Interval for snapshots
	SnapshotRetentionDays int            `mapstructure:"snapshot_retention_days"` // Snapshot retention in days
	WeekStartDay          string         `mapstructure:"week_start_day"`          // monday | sunday
	DBPath                string         `mapstructure:"db_path"`                 // Override DB path
}

// ProvidersConfig defines AI provider settings.
type ProvidersConfig struct {
	Claude    ProviderConfig `mapstructure:"claude"`
	Codex     ProviderConfig `mapstructure:"codex"`
	Ollama    ProviderConfig `mapstructure:"ollama"`
	// Preference sets provider order (e.g., ["claude", "codex", "ollama"]).
	Preference []string `mapstructure:"preference"`
}

// ProviderConfig defines settings for a single AI provider.
type ProviderConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	DataPath string `mapstructure:"data_path"` // Path to provider data directory
	// DangerouslySkipPermissions tells the CLI to skip interactive permission prompts.
	DangerouslySkipPermissions bool `mapstructure:"dangerously_skip_permissions"`
	// DangerouslyBypassApprovalsAndSandbox tells the CLI to bypass approvals and sandboxing.
	DangerouslyBypassApprovalsAndSandbox bool `mapstructure:"dangerously_bypass_approvals_and_sandbox"`
}

// ProjectConfig defines a project to manage.
type ProjectConfig struct {
	Path     string   `mapstructure:"path"`
	Priority int      `mapstructure:"priority"`
	Tasks    []string `mapstructure:"tasks"`   // Task overrides for this project
	Config   string   `mapstructure:"config"`  // Per-project config file
	Pattern  string   `mapstructure:"pattern"` // Glob pattern for discovery
	Exclude  []string `mapstructure:"exclude"` // Paths to exclude
}

// TasksConfig defines task selection settings.
type TasksConfig struct {
	Enabled    []string           `mapstructure:"enabled"`    // Enabled task types
	Priorities map[string]int     `mapstructure:"priorities"` // Priority per task type
	Disabled   []string           `mapstructure:"disabled"`   // Explicitly disabled tasks
	Intervals  map[string]string  `mapstructure:"intervals"`  // Per-task interval overrides (duration strings)
	Custom     []CustomTaskConfig `mapstructure:"custom"`     // User-defined custom tasks
}

// CustomTaskConfig defines a user-defined custom task.
type CustomTaskConfig struct {
	Type        string `mapstructure:"type"`        // Task type slug, e.g. "my-review"
	Name        string `mapstructure:"name"`        // Human-readable name
	Description string `mapstructure:"description"` // Agent prompt text
	Category    string `mapstructure:"category"`    // One of: pr, analysis, options, safe, map, emergency
	CostTier    string `mapstructure:"cost_tier"`   // One of: low, medium, high, very-high
	RiskLevel   string `mapstructure:"risk_level"`  // One of: low, medium, high
	Interval    string `mapstructure:"interval"`    // Duration string, e.g. "48h"
}

// IntegrationsConfig defines external integrations.
type IntegrationsConfig struct {
	ClaudeMD    bool              `mapstructure:"claude_md"`    // Read claude.md
	AgentsMD    bool              `mapstructure:"agents_md"`    // Read agents.md
	TaskSources []TaskSourceEntry `mapstructure:"task_sources"` // Task sources
}

// TaskSourceEntry represents a task source configuration.
type TaskSourceEntry struct {
	TD           *TDConfig `mapstructure:"td"`
	GithubIssues bool      `mapstructure:"github_issues"`
	File         string    `mapstructure:"file"`
}

// TDConfig defines td task management integration.
type TDConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	TeachAgent bool `mapstructure:"teach_agent"` // Include td usage in prompts
}

// LoggingConfig defines logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug | info | warn | error
	Path   string `mapstructure:"path"`   // Log directory
	Format string `mapstructure:"format"` // json | text
}

// ReportingConfig defines reporting settings.
type ReportingConfig struct {
	MorningSummary bool    `mapstructure:"morning_summary"`
	Email          *string `mapstructure:"email"`         // Optional email notification
	SlackWebhook   *string `mapstructure:"slack_webhook"` // Optional Slack webhook
}

// Default values for configuration.
const (
	DefaultBudgetMode        = "daily"
	DefaultMaxPercent        = 75
	DefaultReservePercent    = 5
	DefaultWeeklyTokens      = 700000
	DefaultBillingMode       = "subscription"
	DefaultSnapshotInterval  = "30m"
	DefaultSnapshotRetention = 90
	DefaultWeekStartDay      = "monday"
	DefaultLogLevel          = "info"
	DefaultLogFormat         = "json"
	DefaultClaudeDataPath    = "~/.claude"
	DefaultCodexDataPath     = "~/.codex"
)

// DefaultLogPath returns the default log path.
func DefaultLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "logs")
}

// DefaultDBPath returns the default database path.
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "nightshift.db")
}

// GlobalConfigPath returns the global config path.
func GlobalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nightshift", "config.yaml")
}

// ProjectConfigName is the per-project config filename.
const ProjectConfigName = "nightshift.yaml"

// Load reads configuration from file and environment.
// Order: global config -> project config -> environment overrides
func Load() (*Config, error) {
	return LoadFromPaths("", "")
}

// LoadFromPaths reads configuration from specific paths.
// If projectPath is empty, looks in current directory.
// If globalPath is empty, uses default global path.
func LoadFromPaths(projectPath, globalPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Load global config first
	if globalPath == "" {
		globalPath = GlobalConfigPath()
	}
	if err := loadConfigFile(v, globalPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading global config: %w", err)
	}

	// Load project config (overrides global)
	if projectPath == "" {
		projectPath, _ = os.Getwd()
	}
	projectConfigPath := filepath.Join(projectPath, ProjectConfigName)
	if err := loadConfigFile(v, projectConfigPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	// Bind environment variables
	bindEnvVars(v)

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	normalizeBudgetConfig(&cfg)

	return &cfg, nil
}

// setDefaults configures default values.
func setDefaults(v *viper.Viper) {
	// Budget defaults
	v.SetDefault("budget.mode", DefaultBudgetMode)
	v.SetDefault("budget.max_percent", DefaultMaxPercent)
	v.SetDefault("budget.reserve_percent", DefaultReservePercent)
	v.SetDefault("budget.weekly_tokens", DefaultWeeklyTokens)
	v.SetDefault("budget.aggressive_end_of_week", false)
	v.SetDefault("budget.billing_mode", DefaultBillingMode)
	v.SetDefault("budget.calibrate_enabled", true)
	v.SetDefault("budget.snapshot_interval", DefaultSnapshotInterval)
	v.SetDefault("budget.snapshot_retention_days", DefaultSnapshotRetention)
	v.SetDefault("budget.week_start_day", DefaultWeekStartDay)
	v.SetDefault("budget.db_path", DefaultDBPath())

	// Provider defaults
	v.SetDefault("providers.preference", []string{"claude", "codex"})
	v.SetDefault("providers.claude.enabled", true)
	v.SetDefault("providers.claude.data_path", DefaultClaudeDataPath)
	// SECURITY: Default to false to require explicit opt-in for permission bypassing
	v.SetDefault("providers.claude.dangerously_skip_permissions", false)
	v.SetDefault("providers.codex.enabled", true)
	v.SetDefault("providers.codex.data_path", DefaultCodexDataPath)
	// SECURITY: Default to false to require explicit opt-in for bypassing approvals/sandbox
	v.SetDefault("providers.codex.dangerously_bypass_approvals_and_sandbox", false)

	// Logging defaults
	v.SetDefault("logging.level", DefaultLogLevel)
	v.SetDefault("logging.path", DefaultLogPath())
	v.SetDefault("logging.format", DefaultLogFormat)

	// Reporting defaults
	v.SetDefault("reporting.morning_summary", true)

	// Integration defaults
	v.SetDefault("integrations.claude_md", true)
	v.SetDefault("integrations.agents_md", true)
}

// loadConfigFile merges a YAML config file into viper.
func loadConfigFile(v *viper.Viper, path string) error {
	// Expand home directory
	path = expandPath(path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	return v.MergeInConfig()
}

// bindEnvVars binds environment variables to config keys.
func bindEnvVars(v *viper.Viper) {
	v.SetEnvPrefix("NIGHTSHIFT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicit bindings for nested config
	_ = v.BindEnv("budget.max_percent", "NIGHTSHIFT_BUDGET_MAX_PERCENT")
	_ = v.BindEnv("budget.mode", "NIGHTSHIFT_BUDGET_MODE")
	_ = v.BindEnv("logging.level", "NIGHTSHIFT_LOG_LEVEL")
	_ = v.BindEnv("logging.path", "NIGHTSHIFT_LOG_PATH")
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// Validation errors
var (
	ErrCronAndInterval          = errors.New("cron and interval are mutually exclusive")
	ErrInvalidBudgetMode        = errors.New("budget mode must be 'daily' or 'weekly'")
	ErrInvalidBillingMode       = errors.New("billing mode must be 'subscription' or 'api'")
	ErrInvalidWeekStartDay      = errors.New("week_start_day must be 'monday' or 'sunday'")
	ErrInvalidMaxPercent        = errors.New("max_percent must be between 1 and 100")
	ErrInvalidReservePercent    = errors.New("reserve_percent must be between 0 and 100")
	ErrInvalidSnapshotRetention = errors.New("snapshot_retention_days must be >= 0")
	ErrInvalidLogLevel          = errors.New("log level must be debug, info, warn, or error")
	ErrInvalidLogFormat         = errors.New("log format must be json or text")
	ErrNoSchedule               = errors.New("either cron or interval must be specified")

	ErrCustomTaskMissingType        = errors.New("custom task: type is required")
	ErrCustomTaskMissingName        = errors.New("custom task: name is required")
	ErrCustomTaskMissingDescription = errors.New("custom task: description is required")
	ErrCustomTaskInvalidType        = errors.New("custom task: type must match [a-z0-9-]+")
	ErrCustomTaskInvalidCategory    = errors.New("custom task: invalid category")
	ErrCustomTaskInvalidCostTier    = errors.New("custom task: invalid cost_tier")
	ErrCustomTaskInvalidRiskLevel   = errors.New("custom task: invalid risk_level")
	ErrCustomTaskDuplicateType      = errors.New("custom task: duplicate type")
)

var customTaskTypeRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Validate checks configuration for errors.
func Validate(cfg *Config) error {
	// Schedule validation: cron and interval are mutually exclusive
	if cfg.Schedule.Cron != "" && cfg.Schedule.Interval != "" {
		return ErrCronAndInterval
	}

	// Budget mode validation
	if cfg.Budget.Mode != "" && cfg.Budget.Mode != "daily" && cfg.Budget.Mode != "weekly" {
		return ErrInvalidBudgetMode
	}

	// Billing mode validation
	if cfg.Budget.BillingMode != "" {
		mode := strings.ToLower(cfg.Budget.BillingMode)
		if mode != "subscription" && mode != "api" {
			return ErrInvalidBillingMode
		}
	}

	// Week start day validation
	if cfg.Budget.WeekStartDay != "" {
		day := strings.ToLower(cfg.Budget.WeekStartDay)
		if day != "monday" && day != "sunday" {
			return ErrInvalidWeekStartDay
		}
	}

	// MaxPercent validation
	if cfg.Budget.MaxPercent < 0 || cfg.Budget.MaxPercent > 100 {
		return ErrInvalidMaxPercent
	}

	// ReservePercent validation
	if cfg.Budget.ReservePercent < 0 || cfg.Budget.ReservePercent > 100 {
		return ErrInvalidReservePercent
	}

	if cfg.Budget.SnapshotRetentionDays < 0 {
		return ErrInvalidSnapshotRetention
	}

	// Log level validation
	if cfg.Logging.Level != "" {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[cfg.Logging.Level] {
			return ErrInvalidLogLevel
		}
	}

	// Log format validation
	if cfg.Logging.Format != "" {
		if cfg.Logging.Format != "json" && cfg.Logging.Format != "text" {
			return ErrInvalidLogFormat
		}
	}

	// Task intervals validation
	for taskType, dur := range cfg.Tasks.Intervals {
		if _, err := time.ParseDuration(dur); err != nil {
			return fmt.Errorf("tasks.intervals[%q]: invalid duration %q: %w", taskType, dur, err)
		}
	}

	// Provider preference validation
	if len(cfg.Providers.Preference) > 0 {
		seen := map[string]bool{}
		validProviders := map[string]bool{"claude": true, "codex": true, "ollama": true}
		for _, pref := range cfg.Providers.Preference {
			name := strings.ToLower(strings.TrimSpace(pref))
			if name == "" {
				continue
			}
			if !validProviders[name] {
				return fmt.Errorf("providers.preference contains unknown provider: %s", pref)
			}
			if seen[name] {
				return fmt.Errorf("providers.preference contains duplicate provider: %s", pref)
			}
			seen[name] = true
		}
	}

	// Custom task validation
	if err := validateCustomTasks(cfg.Tasks.Custom); err != nil {
		return err
	}

	return nil
}

func validateCustomTasks(tasks []CustomTaskConfig) error {
	validCategories := map[string]bool{
		"pr": true, "analysis": true, "options": true,
		"safe": true, "map": true, "emergency": true,
	}
	validCostTiers := map[string]bool{
		"low": true, "medium": true, "high": true, "very-high": true,
	}
	validRiskLevels := map[string]bool{
		"low": true, "medium": true, "high": true,
	}

	seenTypes := map[string]bool{}
	for _, task := range tasks {
		if task.Type == "" {
			return ErrCustomTaskMissingType
		}
		if !customTaskTypeRe.MatchString(task.Type) {
			return ErrCustomTaskInvalidType
		}
		if task.Name == "" {
			return ErrCustomTaskMissingName
		}
		if task.Description == "" {
			return ErrCustomTaskMissingDescription
		}
		if task.Category != "" && !validCategories[strings.ToLower(task.Category)] {
			return ErrCustomTaskInvalidCategory
		}
		if task.CostTier != "" && !validCostTiers[strings.ToLower(task.CostTier)] {
			return ErrCustomTaskInvalidCostTier
		}
		if task.RiskLevel != "" && !validRiskLevels[strings.ToLower(task.RiskLevel)] {
			return ErrCustomTaskInvalidRiskLevel
		}
		if task.Interval != "" {
			if _, err := time.ParseDuration(task.Interval); err != nil {
				return fmt.Errorf("custom task %q: invalid interval %q: %w", task.Type, task.Interval, err)
			}
		}
		if seenTypes[task.Type] {
			return ErrCustomTaskDuplicateType
		}
		seenTypes[task.Type] = true
	}
	return nil
}

func normalizeBudgetConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if strings.EqualFold(cfg.Budget.BillingMode, "api") {
		cfg.Budget.CalibrateEnabled = false
	}
}

// Helper methods for accessing configuration

// GetProviderBudget returns the weekly token budget for a provider.
func (c *Config) GetProviderBudget(provider string) int {
	if c.Budget.PerProvider != nil {
		if budget, ok := c.Budget.PerProvider[provider]; ok {
			return budget
		}
	}
	return c.Budget.WeeklyTokens
}

// IsTaskEnabled checks if a task type is enabled.
func (c *Config) IsTaskEnabled(task string) bool {
	// Check if explicitly disabled
	if slices.Contains(c.Tasks.Disabled, task) {
		return false
	}
	// If enabled list is empty, all tasks are enabled
	if len(c.Tasks.Enabled) == 0 {
		return true
	}
	// Check if in enabled list
	return slices.Contains(c.Tasks.Enabled, task)
}

// IsTaskExplicitlyEnabled returns true if the task is in the explicit Enabled list.
func (c *Config) IsTaskExplicitlyEnabled(task string) bool {
	return slices.Contains(c.Tasks.Enabled, task)
}

// GetTaskInterval returns the configured interval override for a task type.
// Returns 0 if no override is set (caller should fall back to TaskDefinition.DefaultInterval).
func (c *Config) GetTaskInterval(taskType string) time.Duration {
	if c.Tasks.Intervals != nil {
		if raw, ok := c.Tasks.Intervals[taskType]; ok {
			d, err := time.ParseDuration(raw)
			if err == nil {
				return d
			}
		}
	}
	return 0
}

// GetTaskPriority returns the priority for a task (higher = more important).
func (c *Config) GetTaskPriority(task string) int {
	if c.Tasks.Priorities != nil {
		if priority, ok := c.Tasks.Priorities[task]; ok {
			return priority
		}
	}
	return 0 // Default priority
}

// ExpandedLogPath returns the log path with ~ expanded.
func (c *Config) ExpandedLogPath() string {
	return expandPath(c.Logging.Path)
}

// ExpandedDBPath returns the database path with ~ expanded.
func (c *Config) ExpandedDBPath() string {
	return expandPath(c.Budget.DBPath)
}

// ExpandedProviderPath returns the provider data path with ~ expanded.
func (c *Config) ExpandedProviderPath(provider string) string {
	switch provider {
	case "claude":
		return expandPath(c.Providers.Claude.DataPath)
	case "codex":
		return expandPath(c.Providers.Codex.DataPath)
	case "ollama":
		return expandPath(c.Providers.Ollama.DataPath)
	default:
		return ""
	}
}
