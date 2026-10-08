package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCommitMessage(t *testing.T) {
	t.Run("positional argument wins", func(t *testing.T) {
		got, err := readCommitMessage([]string{"feat: add thing"}, "ignored-file")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "feat: add thing" {
			t.Errorf("got %q, want %q", got, "feat: add thing")
		}
	})

	t.Run("reads from file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "msg")
		if err := os.WriteFile(path, []byte("fix: correct thing\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := readCommitMessage(nil, path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "fix: correct thing\n" {
			t.Errorf("got %q, want %q", got, "fix: correct thing\n")
		}
	})

	t.Run("missing file errors", func(t *testing.T) {
		if _, err := readCommitMessage(nil, filepath.Join(t.TempDir(), "nope")); err == nil {
			t.Error("expected an error for a missing file")
		}
	})
}

func TestIsCanonical(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		normalized string
		want       bool
	}{
		{"exact match", "feat: add login", "feat: add login", true},
		{"trailing newline ignored", "feat: add login\n", "feat: add login", true},
		{"needs rewriting", "FEAT: add login", "feat: add login", false},
		{"trailing blank lines", "feat: add login\n\n", "feat: add login", false},
		{"extra whitespace", "  feat: add login  ", "feat: add login", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isCanonical(tc.raw, tc.normalized); got != tc.want {
				t.Errorf("isCanonical(%q, %q) = %v, want %v", tc.raw, tc.normalized, got, tc.want)
			}
		})
	}
}
