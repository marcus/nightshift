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

var siloCmd = &cobra.Command{
	Use:   "knowledge-silo [path]",
	Short: "Detect knowledge silos in the codebase",
	Long: `Analyze git history per directory to identify knowledge silos — areas where
only one or two people have contributed.

Directories are ranked by silo risk based on contributor concentration. Use this
to find areas that need knowledge transfer, pairing sessions, or documentation.

The silo score (0-1) combines commit concentration (Herfindahl index) with
contributor count. Higher scores indicate greater knowledge isolation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
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

		depth, _ := cmd.Flags().GetInt("depth")
		minCommits, _ := cmd.Flags().GetInt("min-commits")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		saveReport, _ := cmd.Flags().GetBool("save")
		dbPath, _ := cmd.Flags().GetString("db")

		return runSilo(path, depth, minCommits, jsonOutput, since, until, saveReport, dbPath)
	},
}

func init() {
	siloCmd.Flags().StringP("path", "p", "", "Repository path to analyze")
	siloCmd.Flags().Int("depth", 2, "Directory depth for grouping (default 2)")
	siloCmd.Flags().Int("min-commits", 5, "Minimum commits to include a directory")
	siloCmd.Flags().Bool("json", false, "Output as JSON")
	siloCmd.Flags().String("since", "", "Start date (RFC3339 or YYYY-MM-DD)")
	siloCmd.Flags().String("until", "", "End date (RFC3339 or YYYY-MM-DD)")
	siloCmd.Flags().Bool("save", false, "Save results to database")
	siloCmd.Flags().String("db", "", "Database path (uses config if not set)")
	rootCmd.AddCommand(siloCmd)
}

func runSilo(path string, depth, minCommits int, jsonOutput bool, since, until string, saveReport bool, dbPath string) error {
	logger := logging.Component("knowledge-silo")

	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

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

	// Parse git history by directory
	parser := analysis.NewGitParser(absPath)
	opts := analysis.SiloParseOptions{
		Since:      sinceTime,
		Until:      untilTime,
		Depth:      depth,
		MinCommits: minCommits,
	}

	dirAuthors, err := parser.ParseAuthorsByDirectory(opts)
	if err != nil {
		return fmt.Errorf("parsing git history: %w", err)
	}

	if len(dirAuthors) == 0 {
		logger.Warnf("no directories with commits found in %s", absPath)
		return nil
	}

	// Calculate silo scores
	entries := analysis.CalculateSilos(dirAuthors, minCommits)

	if len(entries) == 0 {
		logger.Warnf("no directories met the minimum commit threshold (%d)", minCommits)
		return nil
	}

	// Generate report
	gen := analysis.NewSiloReportGenerator()
	report := gen.Generate(absPath, depth, entries)

	// Output results
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

				result := &analysis.SiloResult{
					Timestamp: time.Now(),
					RepoPath:  absPath,
					Depth:     depth,
					Results:   entries,
					Summary:   report,
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
