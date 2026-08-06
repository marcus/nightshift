package commands

import (
	"fmt"
	"io"
	"os"

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

By default the normalized message is written to stdout. Pass --check to
validate only: nothing is written to stdout, and the exit code is non-zero
when the message does not conform.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		file, _ := cmd.Flags().GetString("file")

		raw, err := readCommitMessage(args, file)
		if err != nil {
			return err
		}

		normalized, err := commits.Normalize(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return err
		}

		if check {
			// Validate only: do not rewrite or emit the message. A single
			// confirmation line on stdout keeps the success path observable
			// for callers that do not capture stderr.
			fmt.Fprintln(os.Stdout, "ok")
			return nil
		}
		fmt.Fprintln(os.Stdout, normalized)
		return nil
	},
}

func init() {
	commitNormalizeCmd.Flags().BoolP("check", "c", false, "Only validate; do not rewrite (nothing is written to stdout)")
	commitNormalizeCmd.Flags().StringP("file", "f", "", "Read the message from this file (used by the commit-msg hook)")
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
