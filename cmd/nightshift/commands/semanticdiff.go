package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/analysis/semanticdiff"
	"github.com/marcus/nightshift/internal/logging"
)

var semanticDiffCmd = &cobra.Command{
	Use:   "semantic-diff [path]",
	Short: "Explain the semantic meaning of code changes",
	Long: `Analyze git changes between two refs and classify each change by semantic
category (feature, bugfix, refactor, dependency update, config, test, docs, cleanup).

Produces a structured report grouping changes by category with impact highlights
for API surface, schema, and security-sensitive files.

Examples:
  nightshift semantic-diff                        # last commit in current repo
  nightshift semantic-diff --since 7d             # changes in the last 7 days
  nightshift semantic-diff --base main --head dev # compare two branches
  nightshift semantic-diff --json                 # output as JSON`,
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

		baseRef, _ := cmd.Flags().GetString("base")
		headRef, _ := cmd.Flags().GetString("head")
		since, _ := cmd.Flags().GetString("since")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		save, _ := cmd.Flags().GetString("save")

		return runSemanticDiff(path, baseRef, headRef, since, jsonOutput, save)
	},
}

func init() {
	semanticDiffCmd.Flags().StringP("path", "p", "", "Repository path")
	semanticDiffCmd.Flags().String("base", "", "Base ref (commit, branch, or tag)")
	semanticDiffCmd.Flags().String("head", "", "Head ref (default: HEAD)")
	semanticDiffCmd.Flags().String("since", "", "Analyze changes since duration (e.g. 7d, 24h, 30d)")
	semanticDiffCmd.Flags().Bool("json", false, "Output as JSON")
	semanticDiffCmd.Flags().String("save", "", "Save report to file path")
	rootCmd.AddCommand(semanticDiffCmd)
}

func runSemanticDiff(path, baseRef, headRef, since string, jsonOutput bool, savePath string) error {
	logger := logging.Component("semantic-diff")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Verify git repo
	if _, err := os.Stat(filepath.Join(absPath, ".git")); err != nil {
		return fmt.Errorf("not a git repository: %s", absPath)
	}

	var sinceDur time.Duration
	if since != "" {
		sinceDur, err = parseDuration(since)
		if err != nil {
			return fmt.Errorf("parsing --since: %w", err)
		}
	}

	analyzer := semanticdiff.NewAnalyzer(semanticdiff.Options{
		RepoPath: absPath,
		BaseRef:  baseRef,
		HeadRef:  headRef,
		Since:    sinceDur,
	})

	report, err := analyzer.Run()
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	md := semanticdiff.RenderMarkdown(report)
	fmt.Print(md)

	if savePath != "" {
		if err := os.WriteFile(savePath, []byte(md), 0o644); err != nil {
			logger.Errorf("saving report: %v", err)
		} else {
			logger.Infof("report saved to %s", savePath)
		}
	}

	return nil
}

// parseDuration parses human-friendly durations like "7d", "24h", "30d".
func parseDuration(s string) (time.Duration, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty duration")
	}

	// Handle day suffix which Go's time.ParseDuration doesn't support.
	last := s[len(s)-1]
	if last == 'd' || last == 'D' {
		s = s[:len(s)-1] + "h"
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		return d * 24, nil
	}

	// Handle week suffix.
	if last == 'w' || last == 'W' {
		s = s[:len(s)-1] + "h"
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		return d * 24 * 7, nil
	}

	return time.ParseDuration(s)
}
