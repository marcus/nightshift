package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCommitNormalize executes `nightshift commit normalize` with the given args
// and captured stdout/stderr, returning the command error (nil on success).
// Cobra persists flag values across Execute() calls on a shared command, so the
// --check/--file flags are reset to their defaults before each invocation to
// keep tests independent.
func runCommitNormalize(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	for _, name := range []string{"check", "file"} {
		if f := commitNormalizeCmd.Flags().Lookup(name); f != nil {
			f.Changed = false
			_ = f.Value.Set(f.DefValue)
		}
	}
	rootCmd.SetArgs(append([]string{"commit", "normalize"}, args...))
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	err := rootCmd.Execute()
	return out.String(), errOut.String(), err
}

func TestCommitNormalizeCheckRejectsNonConforming(t *testing.T) {
	_, errOut, err := runCommitNormalize(t, "--check", "just a plain message")
	if err == nil {
		t.Fatal("expected --check to reject a non-conforming message, got nil error")
	}
	if !strings.Contains(errOut, "conventional commit type") {
		t.Errorf("expected diagnostic about missing type on stderr, got: %q", errOut)
	}
}

func TestCommitNormalizeCheckSilentOnSuccess(t *testing.T) {
	out, errOut, err := runCommitNormalize(t, "--check", "feat: add login")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if out != "" {
		t.Errorf("expected no stdout in --check mode, got: %q", out)
	}
	if errOut != "" {
		t.Errorf("expected no stderr in --check mode, got: %q", errOut)
	}
}

func TestCommitNormalizePrintsToStdoutByDefault(t *testing.T) {
	out, _, err := runCommitNormalize(t, "  docs: update README.  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "docs: update README\n" {
		t.Errorf("expected normalized message on stdout, got: %q", out)
	}
}

func TestCommitNormalizeFileRewritesInPlace(t *testing.T) {
	dir := t.TempDir()
	msgFile := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgFile, []byte("FEAT(ui): render button.\n\nbody here\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, err := runCommitNormalize(t, "--file", msgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected no stdout when rewriting --file in place, got: %q", out)
	}

	got, err := os.ReadFile(msgFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "feat(ui): render button\n\nbody here\n"
	if string(got) != want {
		t.Errorf("file not rewritten in place\n got: %q\nwant: %q", got, want)
	}
}

func TestCommitNormalizeFileIdempotentNoMtimeChurn(t *testing.T) {
	dir := t.TempDir()
	msgFile := filepath.Join(dir, "COMMIT_EDITMSG")
	canonical := "feat: already canonical\n"
	if err := os.WriteFile(msgFile, []byte(canonical), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(msgFile)
	if err != nil {
		t.Fatal(err)
	}
	before := info.ModTime()

	if _, _, err := runCommitNormalize(t, "--file", msgFile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(msgFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != canonical {
		t.Errorf("already-canonical file should be unchanged\n got: %q\nwant: %q", got, canonical)
	}
	if mt, _ := os.Stat(msgFile); !mt.ModTime().Equal(before) {
		t.Errorf("expected mtime to be untouched on already-canonical message")
	}
}

func TestCommitNormalizeFileCheckDoesNotRewrite(t *testing.T) {
	dir := t.TempDir()
	msgFile := filepath.Join(dir, "COMMIT_EDITMSG")
	original := "feat: keep me\n"
	if err := os.WriteFile(msgFile, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runCommitNormalize(t, "--check", "--file", msgFile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(msgFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Errorf("--check must not rewrite the file\n got: %q\nwant: %q", got, original)
	}
}
