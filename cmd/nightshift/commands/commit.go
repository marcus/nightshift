package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/marcus/nightshift/internal/commits"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Conventional Commits helpers",
	Long: `Tools for working with Conventional Commits messages.

Use "commit normalize" to validate and reformat a commit message so it
follows the project's rules (type prefix, lowercase type, subject length,
and wrapped body).`,
}

var commitNormalizeCmd = &cobra.Command{
	Use:   "normalize [MESSAGE]",
	Short: "Normalize a commit message to Conventional Commits format",
	Long: `Validate and rewrite a commit message into canonical Conventional
Commits form.

The message is read from a positional argument, from a file passed via
--file (typically .git/COMMIT_EDITMSG by a commit-msg hook), or from stdin
when no argument and no --file are given.

  nightshift commit normalize "feat: add login"
  nightshift commit normalize --file .git/COMMIT_EDITMSG
  git log -1 --pretty=%B | nightshift commit normalize

With --file the normalized message is written back to the file; otherwise it
is printed to stdout.

Use --check to only validate without rewriting; a diff-style report is
printed and the exit code is non-zero when the message is not in canonical
form or cannot be normalized.`,
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		file, _ := cmd.Flags().GetString("file")

		raw, err := readCommitMessage(args, file)
		if err != nil {
			return err
		}

		normalized, err := commits.Normalize(raw)
		if err != nil {
			return err
		}

		if check {
			current := strings.Join(commits.StripComments(raw), "\n")
			if current != normalized {
				printDiff(os.Stderr, current, normalized)
				return commits.ErrNotCanonical
			}
			fmt.Fprintln(cmd.OutOrStdout(), normalized)
			return nil
		}

		if file != "" {
			if err := os.WriteFile(file, []byte(normalized+"\n"), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", file, err)
			}
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), normalized)
		return nil
	},
}

func init() {
	commitNormalizeCmd.Flags().BoolP("check", "c", false, "Only validate; do not rewrite")
	commitNormalizeCmd.Flags().StringP("file", "f", "", "Read the message from this file (use by the commit-msg hook)")
	commitCmd.AddCommand(commitNormalizeCmd)
	rootCmd.AddCommand(commitCmd)
}

// readCommitMessage resolves the message source in order: positional arg,
// --file, then stdin.
func readCommitMessage(args []string, file string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", file, err)
		}
		return string(b), nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return string(b), nil
}

// printDiff writes a diff-style report of the changes between oldText and
// newText to w: removed lines prefixed with '-', added lines with '+', and
// unchanged context lines with a space.
func printDiff(w io.Writer, oldText, newText string) {
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	// lcs[i][j] is the length of the longest common subsequence of
	// oldLines[i:] and newLines[j:].
	lcs := make([][]int, len(oldLines)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(newLines)+1)
	}
	for i := len(oldLines) - 1; i >= 0; i-- {
		for j := len(newLines) - 1; j >= 0; j-- {
			switch {
			case oldLines[i] == newLines[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	fmt.Fprintln(w, "--- current")
	fmt.Fprintln(w, "+++ normalized")
	i, j := 0, 0
	for i < len(oldLines) && j < len(newLines) {
		switch {
		case oldLines[i] == newLines[j]:
			fmt.Fprintf(w, "  %s\n", oldLines[i])
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			fmt.Fprintf(w, "- %s\n", oldLines[i])
			i++
		default:
			fmt.Fprintf(w, "+ %s\n", newLines[j])
			j++
		}
	}
	for ; i < len(oldLines); i++ {
		fmt.Fprintf(w, "- %s\n", oldLines[i])
	}
	for ; j < len(newLines); j++ {
		fmt.Fprintf(w, "+ %s\n", newLines[j])
	}
}
