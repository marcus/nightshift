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
wrapped body, and verbatim trailer preservation).`,
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

Use --check to only validate without rewriting; the exit code is non-zero
when the message does not conform.`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		file, _ := cmd.Flags().GetString("file")

		raw, err := readCommitMessage(args, file)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return err
		}

		normalized, err := commits.Normalize(raw)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return err
		}

		switch {
		case check:
			// Validate-only: print nothing on success and never rewrite the
			// file. The non-zero exit from the error above is what makes
			// --check usable as a pre-receive / CI gate.
			return nil
		case file != "":
			// Rewrite the message file in place so a commit-msg hook can
			// normalize the actual commit. Only touch the file when the result
			// differs, to avoid churning mtime on already-canonical messages.
			return writeCommitMessageFile(file, normalized)
		default:
			fmt.Fprintln(cmd.OutOrStdout(), normalized)
			return nil
		}
	},
}

func init() {
	commitNormalizeCmd.Flags().BoolP("check", "c", false, "Only validate; do not rewrite")
	commitNormalizeCmd.Flags().StringP("file", "f", "", "Normalize this message file in place (used by the commit-msg hook)")
	commitCmd.AddCommand(commitNormalizeCmd)
	rootCmd.AddCommand(commitCmd)
}

// writeCommitMessageFile writes normalized to path followed by a trailing
// newline, but only when it differs from the file's current contents. This
// keeps already-canonical messages untouched so commit-msg hooks don't churn
// file mtimes on every commit.
func writeCommitMessageFile(path, normalized string) error {
	want := normalized + "\n"
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == want {
			return nil
		}
	}
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
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
