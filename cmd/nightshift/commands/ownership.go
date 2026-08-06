package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/analysis"
	"github.com/marcus/nightshift/internal/logging"
)

var ownershipCmd = &cobra.Command{
	Use:   "ownership [path]",
	Short: "Analyze code ownership boundaries per directory",
	Long: `Analyze git history to identify code ownership boundaries per directory.

Parses commit history to find dominant contributors per directory, detects
co-change cohesion between directories, and suggests ownership boundaries.
Can output a GitHub CODEOWNERS file or a human-readable markdown report.

Flags:
  --depth      Directory depth to analyze (default 2)
  --min-commits  Minimum commits for a directory to be included (default 10)
  --codeowners   Output in GitHub CODEOWNERS format
  --json         Output as JSON
  --since/--until  Filter by date range`,
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
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		depth, _ := cmd.Flags().GetInt("depth")
		minCommits, _ := cmd.Flags().GetInt("min-commits")
		codeowners, _ := cmd.Flags().GetBool("codeowners")

		return runOwnership(path, jsonOutput, since, until, depth, minCommits, codeowners)
	},
}

func init() {
	ownershipCmd.Flags().StringP("path", "p", "", "Repository path")
	ownershipCmd.Flags().Bool("json", false, "Output as JSON")
	ownershipCmd.Flags().String("since", "", "Start date (RFC3339 or YYYY-MM-DD)")
	ownershipCmd.Flags().String("until", "", "End date (RFC3339 or YYYY-MM-DD)")
	ownershipCmd.Flags().Int("depth", 2, "Directory depth to analyze")
	ownershipCmd.Flags().Int("min-commits", 10, "Minimum commits threshold per directory")
	ownershipCmd.Flags().Bool("codeowners", false, "Output in GitHub CODEOWNERS format")
	rootCmd.AddCommand(ownershipCmd)
}

func runOwnership(path string, jsonOutput bool, since, until string, depth, minCommits int, codeowners bool) error {
	logger := logging.Component("ownership")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !analysis.RepositoryExists(absPath) {
		return fmt.Errorf("not a git repository: %s", absPath)
	}

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

	opts := analysis.ParseOptions{
		Since: sinceTime,
		Until: untilTime,
	}

	parser := analysis.NewGitParser(absPath)

	dirs, err := parser.ParseDirectoryOwnership(opts, depth, minCommits)
	if err != nil {
		return fmt.Errorf("parsing directory ownership: %w", err)
	}

	if len(dirs) == 0 {
		logger.Warnf("no directories with sufficient commits found in %s", absPath)
		return nil
	}

	boundaries := analysis.SuggestBoundaries(dirs)
	unclear := analysis.FindUnclearOwnership(dirs, 0.5)

	cohesion, err := parser.DetectCohesion(opts, depth, 3)
	if err != nil {
		logger.Warnf("could not detect cohesion: %v", err)
	}

	report := &analysis.OwnershipReport{
		Timestamp:   time.Now(),
		RepoPath:    absPath,
		Directories: dirs,
		Boundaries:  boundaries,
		Cohesion:    cohesion,
		Unclear:     unclear,
	}

	if codeowners {
		fmt.Print(analysis.GenerateCODEOWNERS(boundaries))
		return nil
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Print(analysis.RenderOwnershipMarkdown(report))
	return nil
}
