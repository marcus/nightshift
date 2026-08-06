package tasks

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCommitMsgHookAcceptsConventionalSubject(t *testing.T) {
	out, err := runCommitMsgHook(t, "fix(tasks): standardize commit message template\n")
	if err != nil {
		t.Fatalf("commit-msg hook rejected valid message: %v\n%s", err, out)
	}
}

func TestCommitMsgHookRejectsNonStandardSubject(t *testing.T) {
	out, err := runCommitMsgHook(t, "Update commit message handling\n")
	if err == nil {
		t.Fatal("expected commit-msg hook to reject non-standard subject")
	}
	if !strings.Contains(out, "<type>(<optional-scope>): <imperative summary>") {
		t.Fatalf("unexpected hook output:\n%s", out)
	}
}

func TestCommitMsgHookRequiresPairedNightshiftTrailers(t *testing.T) {
	out, err := runCommitMsgHook(t, "fix(tasks): standardize commit message template\n\nNightshift-Task: commit-normalize\n")
	if err == nil {
		t.Fatal("expected commit-msg hook to reject partial Nightshift trailers")
	}
	if !strings.Contains(out, "Nightshift commits must include both trailers") {
		t.Fatalf("unexpected hook output:\n%s", out)
	}
}

func TestCommitMsgHookAcceptsNightshiftTrailers(t *testing.T) {
	msg := strings.Join([]string{
		"docs: add commit message guide",
		"",
		"Nightshift-Task: commit-normalize",
		"Nightshift-Ref: https://github.com/marcus/nightshift",
		"",
	}, "\n")

	out, err := runCommitMsgHook(t, msg)
	if err != nil {
		t.Fatalf("commit-msg hook rejected valid Nightshift trailers: %v\n%s", err, out)
	}
}

func runCommitMsgHook(t *testing.T, message string) (string, error) {
	t.Helper()

	dir := t.TempDir()
	msgFile := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgFile, []byte(message), 0644); err != nil {
		t.Fatalf("write temp commit message: %v", err)
	}

	scriptPath := filepath.Join(repoRoot(t), "scripts", "commit-msg.sh")
	cmd := exec.Command("bash", scriptPath, msgFile)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
