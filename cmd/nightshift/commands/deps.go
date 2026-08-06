package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/deps"
)

var depsCmd = &cobra.Command{
	Use:   "deps [path]",
	Short: "Scan dependencies for security and maintenance risks",
	Long: `Analyze Go module dependencies for:
  - Security vulnerabilities (via OSV.dev)
  - Maintenance health (via GitHub API)
  - License risks (via heuristic matching)

Results are scored by severity and optionally saved to the database.
Set GITHUB_TOKEN for enhanced GitHub API rate limits.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, _ := cmd.Flags().GetString("project")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		save, _ := cmd.Flags().GetBool("save")
		dbPath, _ := cmd.Flags().GetString("db")

		if project == "" && len(args) > 0 {
			project = args[0]
		}
		if project == "" {
			var err error
			project, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
		}

		return runDeps(project, jsonOutput, save, dbPath)
	},
}

func init() {
	depsCmd.Flags().StringP("project", "p", "", "Project path containing go.mod")
	depsCmd.Flags().Bool("json", false, "Output as JSON")
	depsCmd.Flags().Bool("save", false, "Save results to database")
	depsCmd.Flags().String("db", "", "Database path (uses config if not set)")
	rootCmd.AddCommand(depsCmd)
}

func runDeps(project string, jsonOutput, save bool, dbPath string) error {
	scanner := deps.NewScanner(deps.ScanOptions{
		ProjectPath: project,
		Concurrency: 5,
	})

	result, err := scanner.Scan(cmdContext())
	if err != nil {
		return fmt.Errorf("scanning dependencies: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Terminal output
	printDepsResult(result)

	// Save if requested
	if save {
		if err := saveDepsResult(result, dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save results: %v\n", err)
		}
	}

	return nil
}

func printDepsResult(result *deps.ScanResult) {
	fmt.Printf("Dependency Risk Scan: %s\n", result.Project)
	fmt.Printf("Duration: %s | Dependencies: %d (%d direct)\n\n", result.Duration, result.TotalDeps, result.DirectDeps)

	if len(result.Findings) == 0 {
		fmt.Println("No risk findings detected.")
		return
	}

	fmt.Printf("Findings: %d total\n", len(result.Findings))
	for level, count := range result.Summary {
		fmt.Printf("  %s: %d\n", riskLabel(level), count)
	}
	fmt.Println()

	for _, f := range result.Findings {
		fmt.Printf("  %s [%s] %s\n", riskLabel(f.Risk), f.Category, f.Title)
		fmt.Printf("    %s@%s\n", f.Module, f.Version)
		if f.Description != "" {
			fmt.Printf("    %s\n", f.Description)
		}
		fmt.Println()
	}
}

func riskLabel(level deps.RiskLevel) string {
	switch level {
	case deps.RiskCritical:
		return "\033[31mCRITICAL\033[0m"
	case deps.RiskHigh:
		return "\033[91mHIGH\033[0m"
	case deps.RiskMedium:
		return "\033[33mMEDIUM\033[0m"
	case deps.RiskLow:
		return "\033[36mLOW\033[0m"
	default:
		return "NONE"
	}
}

func saveDepsResult(result *deps.ScanResult, dbPath string) error {
	if dbPath == "" {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		dbPath = cfg.ExpandedDBPath()
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = database.Close() }()

	store := deps.NewStore(database.SQL())
	scanID, err := store.SaveScanResult(result)
	if err != nil {
		return fmt.Errorf("saving result: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Results saved (scan ID: %d)\n", scanID)
	return nil
}

func cmdContext() context.Context {
	return context.Background()
}
