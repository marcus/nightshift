package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/semdiff"
)

var explainDiffCmd = &cobra.Command{
	Use:   "explain-diff",
	Short: "Explain the semantic meaning of pending diff",
	Long: `Parse the current git diff and produce a high-level, semantic explanation
of what changed: added/removed functions, renames, signature changes, new
tests, import shifts, comment-only edits, formatting churn, and more.

By default it inspects the unstaged working tree. Use --staged for the
index, or --range to inspect a commit range like main..HEAD.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		staged, _ := cmd.Flags().GetBool("staged")
		rng, _ := cmd.Flags().GetString("range")
		asJSON, _ := cmd.Flags().GetBool("json")
		path, _ := cmd.Flags().GetString("path")

		files, err := semdiff.Gather(semdiff.Options{
			RepoPath: path,
			Staged:   staged,
			Range:    rng,
		})
		if err != nil {
			return err
		}
		exp := semdiff.Explain(files)
		if asJSON {
			out, err := exp.RenderJSON()
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		}
		fmt.Print(exp.Render())
		return nil
	},
}

func init() {
	explainDiffCmd.Flags().Bool("staged", false, "Inspect staged (index) changes instead of working tree")
	explainDiffCmd.Flags().String("range", "", "Inspect a git revision range, e.g. main..HEAD")
	explainDiffCmd.Flags().Bool("json", false, "Emit output as JSON")
	explainDiffCmd.Flags().String("path", "", "Path to the git repository (default: current directory)")
	rootCmd.AddCommand(explainDiffCmd)
}
