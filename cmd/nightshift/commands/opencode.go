// Package commands implements the nightshift CLI commands using cobra.
package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/spf13/cobra"
)

var opencodeCmd = &cobra.Command{
	Use:   "opencode",
	Short: "Run tasks using the opencode provider",
	Long: `Run tasks using the opencode provider. 
This command allows you to use opencode as an AI coding agent for specific tasks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Validate opencode provider is enabled
		if !cfg.Providers.Opencode.Enabled {
			return fmt.Errorf("opencode provider is not enabled in config")
		}

		// Create opencode provider
		opencodeProvider := providers.NewOpencode()

		// Set up context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		// Example of using opencode - replace with actual implementation
		task := providers.Task{
			Prompt: "Write a hello world program in Go",
			Files:  []string{},
		}

		result, err := opencodeProvider.Execute(ctx, task)
		if err != nil {
			return fmt.Errorf("execute task: %w", err)
		}

		if result.Error != nil {
			return fmt.Errorf("task execution error: %w", result.Error)
		}

		fmt.Println("Opencode output:")
		fmt.Println(result.Output)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(opencodeCmd)

	// Add flags for opencode command
	opencodeCmd.Flags().BoolP("dry-run", "n", false, "Show what would be executed without running")
}
