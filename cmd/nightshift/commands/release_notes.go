package commands

import (
	"fmt"
	"os"

	"github.com/marcus/nightshift/internal/releasenotes"
	"github.com/spf13/cobra"
)

var releaseNotesCmd = &cobra.Command{
	Use:   "release-notes",
	Short: "Draft release notes from git history",
	Long: `Generate release notes from conventional commits between two git refs.

Collects commits between --from (default: latest tag) and --to (default: HEAD),
parses conventional commit messages, groups them by type (Features, Bug Fixes,
Other), and renders formatted markdown matching the CHANGELOG.md style.

Examples:
  nightshift release-notes
  nightshift release-notes --from v0.3.3 --to v0.3.4
  nightshift release-notes --version v0.4.0 --output RELEASE.md`,
	RunE: runReleaseNotes,
}

func init() {
	releaseNotesCmd.Flags().String("from", "", "Start ref (tag or commit); default: latest tag")
	releaseNotesCmd.Flags().String("to", "", "End ref (tag, commit, or branch); default: HEAD")
	releaseNotesCmd.Flags().String("version", "", "Version string for the header; default: Unreleased")
	releaseNotesCmd.Flags().StringP("output", "o", "", "Write output to file instead of stdout")
	rootCmd.AddCommand(releaseNotesCmd)
}

func runReleaseNotes(cmd *cobra.Command, args []string) error {
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	version, _ := cmd.Flags().GetString("version")
	output, _ := cmd.Flags().GetString("output")

	opts := releasenotes.Options{
		From:    from,
		To:      to,
		Version: version,
	}

	md, err := releasenotes.Generate(opts)
	if err != nil {
		return fmt.Errorf("generate release notes: %w", err)
	}

	if output != "" {
		if err := os.WriteFile(output, []byte(md), 0644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Release notes written to %s\n", output)
		return nil
	}

	fmt.Fprint(cmd.OutOrStdout(), md)
	return nil
}
