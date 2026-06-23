package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/normalizer"
)

var commitNormalizeCmd = &cobra.Command{
	Use:   "commit-normalize",
	Short: "Normalize a commit message",
	Long: `Standardize a Conventional Commits message.

Reads a commit message from a file path argument, --file, or stdin and
prints the normalized message. The subject is lowercased, trailing periods
are stripped, a blank line separates the subject and body, the type(scope):
shape is enforced, and body paragraphs are wrapped at 100 columns.

Warnings (e.g. an oversized or untyped subject) are printed to stderr.
Exits non-zero if the input is empty or cannot be read.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileFlag, _ := cmd.Flags().GetString("file")
		return runCommitNormalize(cmd, args, fileFlag)
	},
}

func init() {
	commitNormalizeCmd.Flags().String("file", "", "Read the commit message from a file (use - for stdin)")
	rootCmd.AddCommand(commitNormalizeCmd)
}

func runCommitNormalize(cmd *cobra.Command, args []string, fileFlag string) error {
	data, err := readCommitMessage(args, fileFlag)
	if err != nil {
		return err
	}

	result, err := normalizer.Normalize(string(data))
	if err != nil {
		return err
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
	}
	fmt.Fprintln(cmd.OutOrStdout(), result.Message)
	return nil
}

// readCommitMessage resolves the message source: a positional file path takes
// precedence, then --file, then stdin when no source is given.
func readCommitMessage(args []string, fileFlag string) ([]byte, error) {
	switch {
	case len(args) == 1:
		return readMessageSource(args[0])
	case fileFlag != "":
		return readMessageSource(fileFlag)
	default:
		return io.ReadAll(os.Stdin)
	}
}

// readMessageSource reads from path, or from stdin when path is "-".
func readMessageSource(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading commit message %s: %w", path, err)
	}
	return data, nil
}
