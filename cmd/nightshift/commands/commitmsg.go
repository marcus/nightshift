package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/marcus/nightshift/internal/commitmsg"
	"github.com/spf13/cobra"
)

var commitMsgCheck bool

var commitMsgCmd = &cobra.Command{
	Use:   "commit-msg [--check] <file>",
	Short: "Normalize a Git commit message file",
	Args:  cobra.ExactArgs(1),
	RunE:  runCommitMsg,
}

func init() {
	commitMsgCmd.Flags().BoolVar(&commitMsgCheck, "check", false, "validate without changing the file")
	rootCmd.AddCommand(commitMsgCmd)
}

func runCommitMsg(cmd *cobra.Command, args []string) error {
	path := args[0]
	input, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read commit message %s: %w", path, err)
	}
	if commitMsgCheck {
		if err := commitmsg.Validate(string(input)); err != nil {
			return fmt.Errorf("validate commit message: %w", err)
		}
		return nil
	}

	normalized, err := commitmsg.Normalize(string(input))
	if err != nil {
		return fmt.Errorf("normalize commit message: %w", err)
	}
	if normalized == string(input) {
		return nil
	}
	if err := writeCommitMessageAtomically(path, []byte(normalized)); err != nil {
		return fmt.Errorf("write commit message %s: %w", path, err)
	}
	return nil
}

func writeCommitMessageAtomically(path string, content []byte) (err error) {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".commit-msg-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := temp.Close(); err == nil && closeErr != nil {
				err = closeErr
			}
		}
		if err != nil {
			_ = os.Remove(tempName)
		}
	}()
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if _, err := temp.Write(content); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	closed = true
	if err := os.Rename(tempName, path); err != nil {
		return err
	}
	return nil
}
