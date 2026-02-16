package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunReleaseNotesInvalidPath(t *testing.T) {
	err := runReleaseNotes("/nonexistent/path", "", "", false, false, false, false)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestRunReleaseNotesNotGitRepo(t *testing.T) {
	dir := t.TempDir()
	err := runReleaseNotes(dir, "", "", false, false, false, false)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestRunReleaseNotesWithRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")

	writeTestFile(t, dir, "README.md", "# Test\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "feat: initial commit")
	gitRun(t, dir, "tag", "v0.1.0")

	writeTestFile(t, dir, "main.go", "package main\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "feat: add main")
	gitRun(t, dir, "tag", "v0.2.0")

	t.Run("markdown output", func(t *testing.T) {
		err := runReleaseNotes(dir, "v0.2.0", "", false, false, false, false)
		if err != nil {
			t.Fatalf("runReleaseNotes: %v", err)
		}
	})

	t.Run("json output", func(t *testing.T) {
		err := runReleaseNotes(dir, "v0.2.0", "", false, true, false, false)
		if err != nil {
			t.Fatalf("runReleaseNotes JSON: %v", err)
		}
	})

	t.Run("flat output", func(t *testing.T) {
		err := runReleaseNotes(dir, "v0.2.0", "", true, false, false, false)
		if err != nil {
			t.Fatalf("runReleaseNotes flat: %v", err)
		}
	})

	t.Run("with authors", func(t *testing.T) {
		err := runReleaseNotes(dir, "v0.2.0", "", false, false, false, true)
		if err != nil {
			t.Fatalf("runReleaseNotes authors: %v", err)
		}
	})

	t.Run("no hashes", func(t *testing.T) {
		err := runReleaseNotes(dir, "v0.2.0", "", false, false, true, false)
		if err != nil {
			t.Fatalf("runReleaseNotes no-hashes: %v", err)
		}
	})
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2026-02-16T00:00:00Z",
		"GIT_COMMITTER_DATE=2026-02-16T00:00:00Z",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}
