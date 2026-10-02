// Package commands implements the nightshift CLI commands using cobra.
//
// The commands package is the user-facing interface for nightshift. It provides
// subcommands for running tasks, managing the daemon, checking budget status,
// displaying configuration, running diagnostics, and installing system services.
//
// # Available Commands
//
//   - run       Execute configured tasks on one or more projects
//   - daemon    Manage the background scheduler daemon (start/stop/status)
//   - budget    Show budget status and token usage across providers
//   - config    View and modify nightshift configuration
//   - doctor    Run diagnostics to detect configuration or environment issues
//   - init      Create a default nightshift.yaml configuration file
//   - install   Install a system service (launchd, systemd, or cron)
//   - uninstall Remove a previously installed system service
//   - logs      Tail nightshift log output
//   - preview   Show a dry-run preview of what would execute
//   - report    Display reports from previous runs
//   - snapshot  Manually take a usage snapshot
//   - stats     Show usage statistics
//   - status    Show current nightshift status
//   - task      Manage task definitions
//   - setup     Interactive first-time setup wizard
//   - busfactor Analyze code ownership and bus factor
//
// # Configuration
//
// Commands load configuration from ~/.config/nightshift/config.yaml (global)
// and nightshift.yaml (per-project). Project config overrides global config.
// Use "nightshift config validate" to check for errors.
package commands
