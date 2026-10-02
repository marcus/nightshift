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

var serviceAdvisorCmd = &cobra.Command{
	Use:   "service-advisor [path]",
	Short: "Analyze Go package structure for service boundary opportunities",
	Long: `Analyze Go package structure to identify service boundary opportunities.

Examines packages by coupling (afferent/efferent import ratios), cohesion
(internal vs cross-package references), size (LOC/file count), and change
coupling (git co-change frequency) to produce a ranked list of packages
with extract/merge/keep recommendations.

Metrics:
  - Coupling: Import dependency connections (afferent + efferent)
  - Cohesion: Internal references vs exported API surface
  - Size: Lines of code relative to project total
  - Churn: Co-change frequency from git history
  - Extract-Worthiness: Composite score for service extraction potential`,
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
		minLOC, _ := cmd.Flags().GetInt("min-loc")

		return runServiceAdvisor(path, jsonOutput, saveReport, dbPath, minLOC)
	},
}

func init() {
	serviceAdvisorCmd.Flags().StringP("path", "p", "", "Repository or directory path")
	serviceAdvisorCmd.Flags().Bool("json", false, "Output as JSON")
	serviceAdvisorCmd.Flags().Bool("save", false, "Save results to database")
	serviceAdvisorCmd.Flags().String("db", "", "Database path (uses config if not set)")
	serviceAdvisorCmd.Flags().Int("min-loc", 10, "Minimum LOC threshold for package analysis")
	rootCmd.AddCommand(serviceAdvisorCmd)
}

func runServiceAdvisor(path string, jsonOutput, saveReport bool, dbPath string, minLOC int) error {
	logger := logging.Component("service-advisor")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !analysis.RepositoryExists(absPath) {
		return fmt.Errorf("not a git repository: %s", absPath)
	}

	// Analyze packages
	analyzer := analysis.NewServiceAnalyzer(absPath)
	pkgs, err := analyzer.Analyze(minLOC)
	if err != nil {
		return fmt.Errorf("analyzing packages: %w", err)
	}

	if len(pkgs) == 0 {
		logger.Warnf("no Go packages found in %s (above %d LOC threshold)", absPath, minLOC)
		return nil
	}

	// Calculate metrics
	analyses := analysis.CalculateServiceMetrics(pkgs)

	// Generate report
	gen := analysis.NewServiceReportGenerator()
	component := filepath.Base(absPath)
	report := gen.Generate(component, analyses)

	// Output
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

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

				metrics := make([]analysis.ServiceMetrics, len(analyses))
				for i, a := range analyses {
					metrics[i] = a.Metrics
				}

				result := &analysis.ServiceAdvisorResult{
					Component:       component,
					Timestamp:       time.Now(),
					Packages:        analyses,
					Metrics:         metrics,
					Recommendations: report.Recommendations,
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
