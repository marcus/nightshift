package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/changelog"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Synthesize a changelog from git commits",
	Long: `Generate a changelog from git commits between two refs (tags or SHAs).

Parses conventional commit prefixes (feat/fix/docs/refactor/chore/ci/test),
groups them into human-readable categories (Features, Bug Fixes, Other),
and renders output in markdown matching the existing CHANGELOG.md style.

Examples:
  nightshift changelog
  nightshift changelog --from v0.3.3 --to v0.3.4 --version v0.3.4
  nightshift changelog --format plain --output CHANGELOG.md`,
	RunE: runChangelog,
}

func init() {
	changelogCmd.Flags().String("from", "", "Start ref: tag or SHA (default: latest tag)")
	changelogCmd.Flags().String("to", "HEAD", "End ref: tag or SHA")
	changelogCmd.Flags().String("version", "", "Version label for the header (e.g. v1.0.0)")
	changelogCmd.Flags().String("format", "markdown", "Output format: markdown or plain")
	changelogCmd.Flags().StringP("output", "o", "", "Write output to file (default: stdout)")
	rootCmd.AddCommand(changelogCmd)
}

func runChangelog(cmd *cobra.Command, args []string) error {
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	version, _ := cmd.Flags().GetString("version")
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")

	if format != "markdown" && format != "plain" {
		return fmt.Errorf("unsupported format %q: use markdown or plain", format)
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	gen := changelog.NewGenerator(dir)

	// Default --from to latest tag
	if from == "" {
		tag, err := gen.LatestTag()
		if err != nil {
			return fmt.Errorf("no --from specified and no tags found: %w", err)
		}
		from = tag
	}

	groups, err := gen.Generate(from, to)
	if err != nil {
		return fmt.Errorf("generating changelog: %w", err)
	}

	if len(groups) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No commits found in range", from+".."+to)
		return nil
	}

	var rendered string
	if format == "plain" {
		rendered = changelog.RenderPlain(version, groups)
	} else {
		rendered = changelog.RenderMarkdown(version, groups)
	}

	if output != "" {
		if err := os.WriteFile(output, []byte(rendered), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Changelog written to %s\n", output)
		return nil
	}

	fmt.Fprint(cmd.OutOrStdout(), rendered)
	return nil
}
