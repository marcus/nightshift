package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/analysis"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/logging"
)

var busFactorCmd = &cobra.Command{
	Use:   "busfactor [path]",
	Short: "Analyze code ownership concentration (bus factor)",
	Long: `Analyze code ownership concentration in a repository or directory.

The bus factor measures how many key contributors are critical to project continuity.
High concentration indicates risk - loss of few people could impact the project significantly.

Metrics:
  - Bus Factor: Minimum contributors needed for 50% of commits
  - Herfindahl Index: Ownership concentration (0=diverse, 1=concentrated)
  - Gini Coefficient: Knowledge inequality (0=equal, 1=unequal)
  - Risk Level: critical/high/medium/low assessment`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			return err
		}

		// Use current directory if path not specified
		if path == "" && len(args) > 0 {
			path = args[0]
		}
		if path == "" {
			var err error
			path, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		filePath, _ := cmd.Flags().GetString("file")
		saveReport, _ := cmd.Flags().GetBool("save")
		dbPath, _ := cmd.Flags().GetString("db")

		return runBusFactor(path, jsonOutput, since, until, filePath, saveReport, dbPath)
	},
}

func init() {
	busFactorCmd.Flags().StringP("path", "p", "", "Repository or directory path")
	busFactorCmd.Flags().Bool("json", false, "Output as JSON")
	busFactorCmd.Flags().String("since", "", "Start date (RFC3339 or YYYY-MM-DD)")
	busFactorCmd.Flags().String("until", "", "End date (RFC3339 or YYYY-MM-DD)")
	busFactorCmd.Flags().StringP("file", "f", "", "Analyze specific file or pattern")
	busFactorCmd.Flags().Bool("save", false, "Save results to database")
	busFactorCmd.Flags().String("db", "", "Database path (uses config if not set)")
	rootCmd.AddCommand(busFactorCmd)
}

func runBusFactor(path string, jsonOutput bool, since, until, filePath string, saveReport bool, dbPath string) error {
	logger := logging.Component("busfactor")

	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Check if valid git repo
	if !analysis.RepositoryExists(absPath) {
		return fmt.Errorf("not a git repository: %s", absPath)
	}

	// Parse dates
	var sinceTime, untilTime time.Time
	if since != "" {
		t, err := parseDate(since)
		if err != nil {
			return fmt.Errorf("parsing since date: %w", err)
		}
		sinceTime = t
	}
	if until != "" {
		t, err := parseDate(until)
		if err != nil {
			return fmt.Errorf("parsing until date: %w", err)
		}
		untilTime = t
	}

	// Parse git history
	parser := analysis.NewGitParser(absPath)
	opts := analysis.ParseOptions{
		Since:    sinceTime,
		Until:    untilTime,
		FilePath: filePath,
	}

	authors, err := parser.ParseAuthors(opts)
	if err != nil {
		return fmt.Errorf("parsing git history: %w", err)
	}

	if len(authors) == 0 {
		logger.Warnf("no commits found in %s", absPath)
		return nil
	}

	// Calculate metrics
	metrics := analysis.CalculateMetrics(authors)

	// Generate report
	gen := analysis.NewReportGenerator()
	component := filepath.Base(absPath)
	report := gen.Generate(component, authors, metrics)

	// Output results
	if jsonOutput {
		return outputJSON(report)
	}

	// Human-readable output
	markdown := gen.RenderMarkdown(report)
	fmt.Println(markdown)

	// Save if requested
	if saveReport {
		if dbPath == "" {
			cfg, err := config.Load()
			if err != nil {
				logger.Warnf("could not load config for db path: %v", err)
			} else {
				dbPath = cfg.ExpandedDBPath()
			}
		}

		if dbPath != "" {
			database, err := db.Open(dbPath)
			if err != nil {
				logger.Errorf("opening database: %v", err)
			} else {
				defer func() { _ = database.Close() }()

				result := &analysis.BusFactorResult{
					Component:    component,
					Timestamp:    time.Now(),
					Metrics:      metrics,
					Contributors: authors,
					RiskLevel:    metrics.RiskLevel,
				}

				if err := result.Store(database.SQL()); err != nil {
					logger.Errorf("storing result: %v", err)
				} else {
					logger.Infof("results saved (ID: %d)", result.ID)
				}
			}
		}
	}

	return nil
}

func outputJSON(report *analysis.Report) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func parseDate(dateStr string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t, nil
	}

	// Try YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s (use RFC3339 or YYYY-MM-DD)", dateStr)
}
