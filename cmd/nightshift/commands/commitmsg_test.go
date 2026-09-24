package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcus/nightshift/internal/commitmsg"
)

func TestRunCommitMsgNormalizesFileAtomically(t *testing.T) {
	messagePath := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(messagePath, []byte("Improve setup\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	commitMsgCheck = false
	t.Cleanup(func() { commitMsgCheck = false })
	if err := runCommitMsg(nil, []string{messagePath}); err != nil {
		t.Fatalf("runCommitMsg() error = %v", err)
	}

	content, err := os.ReadFile(messagePath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "chore: improve setup\n"; got != want {
		t.Errorf("normalized content = %q, want %q", got, want)
	}
	info, err := os.Stat(messagePath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Errorf("file mode = %o, want %o", got, want)
	}
}

func TestRunCommitMsgCheckDoesNotChangeFile(t *testing.T) {
	messagePath := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	original := []byte("Improve setup\n")
	if err := os.WriteFile(messagePath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	commitMsgCheck = true
	t.Cleanup(func() { commitMsgCheck = false })
	err := runCommitMsg(nil, []string{messagePath})
	if !errors.Is(err, commitmsg.ErrInvalidMessage) {
		t.Fatalf("runCommitMsg() error = %v, want ErrInvalidMessage", err)
	}
	content, err := os.ReadFile(messagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(original) {
		t.Errorf("check mode changed content to %q", content)
	}
}
