// Package commands implements the nightshift CLI commands using cobra.
package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "0.3.0"
)

var rootCmd = &cobra.Command{
	Use:   "nightshift",
	Short: "AI-powered autonomous development assistant",
	Long: `Nightshift runs AI coding agents during off-hours to handle
development tasks like code review, refactoring, testing, and documentation.

Configure tasks in nightshift.yaml and let Nightshift work while you sleep.`,
	Version: Version,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
}
