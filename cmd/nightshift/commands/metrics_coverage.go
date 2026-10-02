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

var metricsCoverageCmd = &cobra.Command{
	Use:   "metrics-coverage [path]",
	Short: "Analyze metrics instrumentation coverage in Go source code",
	Long: `Analyze metrics instrumentation coverage across Go packages.

Scans Go source files using go/ast to detect metrics instrumentation patterns
including Prometheus, OpenTelemetry, StatsD, expvar, and custom metric types.

For each package, computes the ratio of instrumented exported functions to total
exported functions and identifies gaps (uninstrumented HTTP handlers, error paths,
and public API functions).

Coverage levels:
  - Low risk (>=80%): Well-instrumented codebase
  - Medium risk (50-80%): Moderate coverage
  - High risk (20-50%): Significant gaps
  - Critical (<20%): Minimal instrumentation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			return err
		}

		if path == "" && len(args) > 0 {
			path = args[0]
		}
		if path == "" {
			path, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		saveReport, _ := cmd.Flags().GetBool("save")
		dbPath, _ := cmd.Flags().GetString("db")

		return runMetricsCoverage(path, jsonOutput, saveReport, dbPath)
	},
}

func init() {
	metricsCoverageCmd.Flags().StringP("path", "p", "", "Directory path to scan")
	metricsCoverageCmd.Flags().Bool("json", false, "Output as JSON")
	metricsCoverageCmd.Flags().Bool("save", false, "Save results to database")
	metricsCoverageCmd.Flags().String("db", "", "Database path (uses config if not set)")
	rootCmd.AddCommand(metricsCoverageCmd)
}

func runMetricsCoverage(path string, jsonOutput bool, saveReport bool, dbPath string) error {
	logger := logging.Component("metrics-coverage")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Verify directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("accessing path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", absPath)
	}

	// Scan packages
	scanner := analysis.NewMetricsCoverageScanner(absPath)
	packages, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("scanning packages: %w", err)
	}

	if len(packages) == 0 {
		logger.Warnf("no Go packages with exported functions found in %s", absPath)
		return nil
	}

	// Generate report
	gen := analysis.NewMetricsCoverageReportGenerator()
	component := filepath.Base(absPath)
	report := gen.GenerateCoverageReport(component, packages)

	// Output results
	if jsonOutput {
		return outputMetricsCoverageJSON(report)
	}

	markdown := gen.RenderCoverageMarkdown(report)
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

				result := &analysis.MetricsCoverageResult{
					Component:   component,
					Timestamp:   time.Now(),
					Summary:     report.Summary,
					Packages:    report.Packages,
					CoveragePct: report.Summary.OverallCoveragePct,
					GapCount:    report.Summary.GapCount,
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

func outputMetricsCoverageJSON(report *analysis.MetricsCoverageReport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
