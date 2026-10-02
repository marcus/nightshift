package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/analysis/coverage"
)

var metricsCoverageCmd = &cobra.Command{
	Use:   "metrics-coverage [path]",
	Short: "Analyze metrics instrumentation coverage",
	Long: `Statically scan a Go codebase for instrumentation calls (logging,
stats, metrics, telemetry) and report per-package coverage.

A function is considered "instrumented" if its body contains at least one
call expression matching the configured patterns. The default pattern set
covers log/logging/stats/metrics/otel/telemetry packages and common method
names like .Info, .Error, .Record, .Observe, and .Inc.

Use --min-coverage in CI to fail the build when overall coverage drops
below a threshold.`,
	Example: `  nightshift metrics-coverage
  nightshift metrics-coverage --path ./internal --format json
  nightshift metrics-coverage --exclude 'vendor/**' --exclude '**/mocks/**'
  nightshift metrics-coverage --min-coverage 60`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		if path == "" && len(args) > 0 {
			path = args[0]
		}
		if path == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			path = cwd
		}

		format, _ := cmd.Flags().GetString("format")
		minCov, _ := cmd.Flags().GetFloat64("min-coverage")
		excludes, _ := cmd.Flags().GetStringArray("exclude")
		output, _ := cmd.Flags().GetString("output")
		includeTests, _ := cmd.Flags().GetBool("include-tests")
		patterns, _ := cmd.Flags().GetStringArray("pattern")
		maxGaps, _ := cmd.Flags().GetInt("max-gaps")

		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}

		opts := coverage.Options{
			Root:         absPath,
			Patterns:     patterns,
			Excludes:     excludes,
			IncludeTests: includeTests,
		}

		report, err := coverage.New(opts).Analyze()
		if err != nil {
			return fmt.Errorf("running coverage analysis: %w", err)
		}

		var out io.Writer = os.Stdout
		if output != "" {
			f, err := os.Create(output)
			if err != nil {
				return fmt.Errorf("opening output file: %w", err)
			}
			defer func() { _ = f.Close() }()
			out = f
		}

		switch strings.ToLower(format) {
		case "json":
			if err := coverage.RenderJSON(out, report); err != nil {
				return fmt.Errorf("rendering JSON: %w", err)
			}
		case "", "text":
			if err := coverage.RenderText(out, report, maxGaps); err != nil {
				return fmt.Errorf("rendering text: %w", err)
			}
		default:
			return fmt.Errorf("unknown format %q (expected text or json)", format)
		}

		if minCov > 0 && report.Percent < minCov {
			return fmt.Errorf("metrics coverage %.1f%% is below threshold %.1f%%", report.Percent, minCov)
		}
		return nil
	},
}

func init() {
	metricsCoverageCmd.Flags().StringP("path", "p", "", "Directory to analyze (default: current dir)")
	metricsCoverageCmd.Flags().String("format", "text", "Output format: text or json")
	metricsCoverageCmd.Flags().Float64("min-coverage", 0, "Fail if overall coverage is below this percent (0 disables)")
	metricsCoverageCmd.Flags().StringArray("exclude", nil, "Glob patterns of relative paths to exclude (repeatable)")
	metricsCoverageCmd.Flags().StringArray("pattern", nil, "Override default instrumentation call-site patterns (repeatable)")
	metricsCoverageCmd.Flags().StringP("output", "o", "", "Write output to file instead of stdout")
	metricsCoverageCmd.Flags().Bool("include-tests", false, "Include *_test.go files in analysis")
	metricsCoverageCmd.Flags().Int("max-gaps", 10, "Max uninstrumented functions to list per package in text output (0 to hide)")
	rootCmd.AddCommand(metricsCoverageCmd)
}
