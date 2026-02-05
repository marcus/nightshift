package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/marcus/nightshift/internal/config"
	"github.com/spf13/cobra"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create configuration file",
	Long: `Initialize a new nightshift configuration file.

By default, creates nightshift.yaml in the current directory.
Use --global to create a global config at ~/.config/nightshift/config.yaml`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().Bool("global", false, "Create global config instead of project config")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing config without prompting")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	global, _ := cmd.Flags().GetBool("global")
	force, _ := cmd.Flags().GetBool("force")

	var configPath string
	var configType string

	if global {
		configPath = config.GlobalConfigPath()
		configType = "global"
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		configPath = filepath.Join(cwd, config.ProjectConfigName)
		configType = "project"
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		if !force {
			fmt.Printf("%sConfig already exists:%s %s\n", colorYellow, colorReset, configPath)
			fmt.Print("Overwrite? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// Create parent directory if needed
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	// Generate and write config
	content := generateDefaultConfig(global)
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Success output
	fmt.Printf("\n%s%sCreated %s config:%s %s\n\n", colorBold, colorGreen, configType, colorReset, configPath)
	fmt.Printf("%sNext steps:%s\n", colorCyan, colorReset)
	if global {
		fmt.Println("  1. Edit the config to set your schedule and budget")
		fmt.Println("  2. Add project paths under 'projects:'")
		fmt.Println("  3. Run 'nightshift config validate' to verify")
		fmt.Println("  4. Run 'nightshift install' to set up system service")
	} else {
		fmt.Println("  1. Edit the config to customize tasks for this project")
		fmt.Println("  2. Run 'nightshift config validate' to verify")
		fmt.Println("  3. Run 'nightshift run --dry-run' to preview")
	}
	fmt.Println()

	return nil
}

// generateDefaultConfig creates the default config YAML with helpful comments.
func generateDefaultConfig(global bool) string {
	if global {
		return generateGlobalConfig()
	}
	return generateProjectConfig()
}

func generateGlobalConfig() string {
	return `# Nightshift Global Configuration
# Location: ~/.config/nightshift/config.yaml
#
# This file configures nightshift's default behavior across all projects.
# Per-project configs (nightshift.yaml) override these settings.

# Schedule configuration
# Choose either cron OR interval (not both)
schedule:
  cron: "0 2 * * *"              # Run at 2 AM daily
  # interval: 1h                 # Alternative: run every N hours
  window:                        # Optional: restrict to specific hours
    start: "22:00"
    end: "06:00"
    timezone: "America/Denver"   # Your timezone

# Budget configuration
#
# How budget modes work:
# - daily: Each night uses up to max_percent of that day's budget (weekly/7)
# - weekly: Each night uses up to max_percent of the REMAINING weekly budget
#
budget:
  mode: daily                    # daily | weekly
  max_percent: 75                # Max % of budget per run (default: 75)
  aggressive_end_of_week: false  # Weekly mode: ramp up in last 2 days
  reserve_percent: 5             # Always keep this % in reserve
  weekly_tokens: 700000          # Fallback weekly budget
  # per_provider:                # Optional per-provider overrides
  #   claude: 700000
  #   codex: 500000

# Provider configuration (usage tracking)
providers:
  claude:
    enabled: true
    data_path: "~/.claude"       # Path to Claude Code data directory
    dangerously_skip_permissions: true
  codex:
    enabled: true
    data_path: "~/.codex"        # Path to Codex data directory
    dangerously_bypass_approvals_and_sandbox: true

# Projects to manage
# Add your project paths here
projects:
  # - path: ~/code/myproject
  #   priority: 1                # Higher = more important
  #   tasks:                     # Override enabled tasks
  #     - lint
  #     - docs
  # - path: ~/code/library
  #   priority: 2
  #   config: .nightshift.yaml   # Per-project override file

# Task configuration
tasks:
  enabled:
    - lint                       # Linter fixes
    - docs                       # Documentation backfill
    - security                   # Security scanning
    - test-gaps                  # Test coverage gaps
    - dead-code                  # Dead code removal
  priorities:
    lint: 1
    security: 2
    docs: 3
  disabled: []                   # Explicitly disabled tasks

# Integration points
integrations:
  claude_md: true                # Read claude.md for project context
  agents_md: true                # Read agents.md for agent hints
  task_sources:
    - td:                        # td task management
        enabled: true
        teach_agent: true        # Include td usage in agent prompts
    # - github_issues            # GitHub issues with "nightshift" label
    # - file: TODO.md            # Custom file

# Logging configuration
logging:
  level: info                    # debug | info | warn | error
  path: ~/.local/share/nightshift/logs
  format: json                   # json | text

# Reporting configuration
reporting:
  morning_summary: true          # Generate morning summary
  # email: user@example.com      # Optional email notification
  # slack_webhook: https://...   # Optional Slack notification
`
}

func generateProjectConfig() string {
	return `# Nightshift Project Configuration
# Location: nightshift.yaml (project root)
#
# This file configures nightshift for this specific project.
# These settings override the global config (~/.config/nightshift/config.yaml).

# Task configuration for this project
tasks:
  enabled:
    - lint                       # Linter fixes
    - docs                       # Documentation backfill
    - security                   # Security scanning
    - test-gaps                  # Test coverage gaps
  priorities:
    lint: 1
    security: 2
    docs: 3
  disabled: []                   # Explicitly disabled tasks

# Integration points
integrations:
  claude_md: true                # Read claude.md for project context
  agents_md: true                # Read agents.md for agent hints
  task_sources:
    - td:                        # td task management
        enabled: true
        teach_agent: true        # Include td usage in agent prompts
    # - github_issues            # GitHub issues with "nightshift" label
    # - file: TODO.md            # Custom task file

# Optional: Override schedule for this project
# schedule:
#   cron: "0 3 * * *"            # Different time for this project

# Optional: Override budget for this project
# budget:
#   max_percent: 5               # Lower budget for this project
`
}
