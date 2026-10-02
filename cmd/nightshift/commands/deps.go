package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/deps"
	"github.com/marcus/nightshift/internal/logging"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Scan dependencies for security, maintenance, and license risks",
	Long: `Scan Go module dependencies across managed projects for:
  - Security vulnerabilities (via OSV.dev)
  - Maintenance risks (archived repos, stale projects)
  - License concerns (copyleft, unknown licenses)

Results are scored Critical/High/Medium/Low and stored in the database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, _ := cmd.Flags().GetString("project")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		dbPath, _ := cmd.Flags().GetString("db")

		return runDeps(project, jsonOutput, dbPath)
	},
}

func init() {
	depsCmd.Flags().StringP("project", "p", "", "Scan a specific project path")
	depsCmd.Flags().Bool("json", false, "Output results as JSON")
	depsCmd.Flags().String("db", "", "Database path (uses config if not set)")
	rootCmd.AddCommand(depsCmd)
}

func runDeps(project string, jsonOutput bool, dbPath string) error {
	logger := logging.Component("deps")

	if dbPath == "" {
		cfg, err := config.Load()
		if err != nil {
			logger.Warnf("could not load config for db path, using default: %v", err)
		} else {
			dbPath = cfg.ExpandedDBPath()
		}
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = database.Close() }()

	store := deps.NewStore(database.SQL())
	if err := store.InitTables(); err != nil {
		return fmt.Errorf("initializing tables: %w", err)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	scanner := deps.NewScanner(store, githubToken, logger)

	ctx := context.Background()

	var paths []string
	if project != "" {
		paths = []string{project}
	} else {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		for _, p := range cfg.Projects {
			paths = append(paths, p.Path)
		}
		if len(paths) == 0 {
			return fmt.Errorf("no projects configured; use --project to specify a path")
		}
	}

	results, scanErr := scanner.ScanAll(ctx, paths)

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
	} else {
		renderResults(results, scanErr)
	}

	for _, r := range results {
		for _, f := range r.Findings {
			if f.RiskLevel == deps.RiskCritical {
				if scanErr != nil {
					fmt.Fprintf(os.Stderr, "\nWarning: scan completed with errors: %v\n", scanErr)
				}
				os.Exit(1)
			}
		}
	}

	if scanErr != nil {
		fmt.Fprintf(os.Stderr, "\nWarning: scan completed with errors: %v\n", scanErr)
	}

	return nil
}

func renderResults(results []*deps.ScanResult, scanErr error) {
	criticalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	highStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	mediumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	lowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	headerStyle := lipgloss.NewStyle().Bold(true).Underline(true)

	for _, result := range results {
		critical, high, medium, low := deps.SummarizeResults(result)
		fmt.Printf("\n%s (%d deps)\n", headerStyle.Render(result.Project), len(result.Deps))
		fmt.Printf("  Critical: %s  High: %s  Medium: %s  Low: %s\n",
			criticalStyle.Render(fmt.Sprintf("%d", critical)),
			highStyle.Render(fmt.Sprintf("%d", high)),
			mediumStyle.Render(fmt.Sprintf("%d", medium)),
			lowStyle.Render(fmt.Sprintf("%d", low)),
		)

		if len(result.Findings) == 0 {
			fmt.Println("  No issues found.")
			continue
		}

		fmt.Println()
		for _, f := range result.Findings {
			var styled string
			badge := strings.ToUpper(string(f.RiskLevel))
			switch f.RiskLevel {
			case deps.RiskCritical:
				styled = criticalStyle.Render(fmt.Sprintf("  [%s]", badge))
			case deps.RiskHigh:
				styled = highStyle.Render(fmt.Sprintf("  [%s]", badge))
			case deps.RiskMedium:
				styled = mediumStyle.Render(fmt.Sprintf("  [%s]", badge))
			default:
				styled = lowStyle.Render(fmt.Sprintf("  [%s]", badge))
			}
			fmt.Printf("%s %s\n", styled, f.Detail)
		}
	}

	if scanErr != nil {
		fmt.Fprintf(os.Stderr, "\nWarning: %v\n", scanErr)
	}

	fmt.Println()
}
