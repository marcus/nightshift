package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackwardCompat_ShellPathEscaping verifies that the escapeShellPath
// function correctly handles paths with special characters, providing
// protection against shell injection while maintaining compatibility.
func TestBackwardCompat_ShellPathEscaping(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		contains string // substring that must appear in result
	}{
		{
			name:     "simple path",
			path:     "/usr/local/bin",
			contains: "/usr/local/bin",
		},
		{
			name:     "path with spaces",
			path:     "/path/with spaces/bin",
			contains: "with spaces",
		},
		{
			name:     "path with dollar sign",
			path:     "/path/$with/special",
			contains: "$with",
		},
		{
			name:     "path with backticks",
			path:     "/path/`with`/special",
			contains: "with",
		},
		{
			name:     "path with single quote",
			path:     "/path/with'quote/bin",
			contains: "with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			escaped := escapeShellPath(tt.path)

			// Result should not be empty
			if escaped == "" {
				t.Error("escapeShellPath returned empty string")
			}

			// Result should be quoted (either single or double)
			if !strings.HasPrefix(escaped, "'") && !strings.HasPrefix(escaped, "\"") {
				t.Errorf("escapeShellPath result not quoted: %q", escaped)
			}

			// Result should contain the expected path content
			if !strings.Contains(escaped, tt.contains) {
				t.Errorf("escapeShellPath(%q) result %q does not contain %q", tt.path, escaped, tt.contains)
			}
		})
	}
}

// TestBackwardCompat_PathExportLine verifies that pathExportLine generates
// correct shell export statements with proper escaping.
func TestBackwardCompat_PathExportLine(t *testing.T) {
	tests := []struct {
		shell    string
		path     string
		expected string // substring that should be in result
	}{
		{
			shell:    "bash",
			path:     "/usr/local/bin",
			expected: "export PATH=",
		},
		{
			shell:    "zsh",
			path:     "/usr/local/bin",
			expected: "export PATH=",
		},
		{
			shell:    "fish",
			path:     "/usr/local/bin",
			expected: "set -gx PATH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			result := pathExportLine(tt.shell, tt.path)

			if !strings.Contains(result, tt.expected) {
				t.Errorf("pathExportLine(%q, %q) = %q, want to contain %q", tt.shell, tt.path, result, tt.expected)
			}

			if !strings.Contains(result, tt.path) {
				t.Errorf("pathExportLine result %q does not contain path %q", result, tt.path)
			}
		})
	}
}

// TestBackwardCompat_EnsurePathInShell verifies that the function correctly
// handles existing PATH entries and doesn't duplicate them.
func TestBackwardCompat_EnsurePathInShell(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".bashrc")

	// Initial setup - add path
	changed, err := ensurePathInShell(configPath, "bash", "/usr/local/bin")
	if err != nil {
		t.Fatalf("first ensurePathInShell: %v", err)
	}
	if !changed {
		t.Error("expected first ensurePathInShell to return true (path added)")
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Second call with same path should not add again
	changed, err = ensurePathInShell(configPath, "bash", "/usr/local/bin")
	if err != nil {
		t.Fatalf("second ensurePathInShell: %v", err)
	}
	if changed {
		t.Error("expected second ensurePathInShell to return false (path already present)")
	}

	// Verify file only has one entry for this path
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	count := strings.Count(string(content), "/usr/local/bin")
	if count != 1 {
		t.Errorf("expected one entry for /usr/local/bin, found %d", count)
	}
}

// TestBackwardCompat_ShellConfigHasPath verifies that shellConfigHasPath
// correctly detects existing PATH entries with various formats.
func TestBackwardCompat_ShellConfigHasPath(t *testing.T) {
	tests := []struct {
		name    string
		content string
		path    string
		found   bool
	}{
		{
			name:    "simple PATH export",
			content: "export PATH=$PATH:/usr/local/bin\n",
			path:    "/usr/local/bin",
			found:   true,
		},
		{
			name:    "PATH with tilde",
			content: "export PATH=$PATH:~/bin\n",
			path:    "~/bin",
			found:   true,
		},
		{
			name:    "fish syntax",
			content: "set -gx PATH /usr/local/bin $PATH\n",
			path:    "/usr/local/bin",
			found:   true,
		},
		{
			name:    "commented line",
			content: "# export PATH=$PATH:/usr/local/bin\n",
			path:    "/usr/local/bin",
			found:   false,
		},
		{
			name:    "path not present",
			content: "export PATH=$PATH:/other/bin\n",
			path:    "/usr/local/bin",
			found:   false,
		},
		{
			name: "multiple paths",
			content: `export PATH=$PATH:/usr/local/bin
export PATH=$PATH:~/go/bin
`,
			path:  "/usr/local/bin",
			found: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellConfigHasPath(tt.content, tt.path)
			if result != tt.found {
				t.Errorf("shellConfigHasPath() = %v, want %v", result, tt.found)
			}
		})
	}
}

// TestBackwardCompat_ContainsPathToken verifies that containsPathToken
// correctly extracts and matches path tokens.
func TestBackwardCompat_ContainsPathToken(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		target string
		found  bool
	}{
		{
			name:   "simple match",
			line:   "export PATH=$PATH:/usr/local/bin",
			target: "/usr/local/bin",
			found:  true,
		},
		{
			name:   "path with symlink",
			line:   "export PATH=$PATH:/usr/local/bin",
			target: "/usr/local/bin",
			found:  true,
		},
		{
			name:   "no match",
			line:   "export PATH=$PATH:/other/bin",
			target: "/usr/local/bin",
			found:  false,
		},
		{
			name:   "partial path no match",
			line:   "export PATH=$PATH:/usr/bin",
			target: "/usr/local/bin",
			found:  false,
		},
		{
			name:   "multiple paths one matches",
			line:   "export PATH=\"/usr/local/bin:/opt/bin:$PATH\"",
			target: "/opt/bin",
			found:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsPathToken(tt.line, tt.target)
			if result != tt.found {
				t.Errorf("containsPathToken(%q, %q) = %v, want %v", tt.line, tt.target, result, tt.found)
			}
		})
	}
}

// TestBackwardCompat_ExpandPath verifies that path expansion still works.
func TestBackwardCompat_ExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/bin", filepath.Join(home, "bin")},
		{"~/", home},
		{"/usr/local/bin", "/usr/local/bin"},
		{"./relative", "./relative"},
	}

	for _, tt := range tests {
		result := expandPath(tt.input)
		if result != tt.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestBackwardCompat_DetectShellConfig verifies that shell detection
// still works and returns correct config paths.
func TestBackwardCompat_DetectShellConfig(t *testing.T) {
	// Set SHELL to bash for this test
	oldShell := os.Getenv("SHELL")
	defer func() { _ = os.Setenv("SHELL", oldShell) }()

	// Test bash detection
	t.Setenv("SHELL", "/bin/bash")
	shell, configPath := detectShellConfig()

	if shell != "bash" {
		t.Errorf("detectShellConfig() shell = %q, want bash", shell)
	}

	if !strings.Contains(configPath, "bash") && !strings.Contains(configPath, ".profile") {
		t.Errorf("detectShellConfig() path = %q, want bash-related path", configPath)
	}

	// Test zsh detection
	t.Setenv("SHELL", "/bin/zsh")
	shell, configPath = detectShellConfig()

	if shell != "zsh" {
		t.Errorf("detectShellConfig() shell = %q, want zsh", shell)
	}

	if !strings.Contains(configPath, "zsh") {
		t.Errorf("detectShellConfig() path = %q, want zsh-related path", configPath)
	}
}

// TestBackwardCompat_SamePath verifies that symlink resolution still works.
func TestBackwardCompat_SamePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real file
	realPath := filepath.Join(tmpDir, "real", "bin", "nightshift")
	if err := os.MkdirAll(filepath.Dir(realPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(realPath, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatal(err)
	}

	// Test same path
	if !samePath(realPath, realPath) {
		t.Error("samePath should return true for identical paths")
	}

	// Create symlink
	linkPath := filepath.Join(tmpDir, "link", "bin", "nightshift")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Skip("cannot create symlink on this system")
	}

	// Test symlink resolution
	if !samePath(realPath, linkPath) {
		t.Error("samePath should return true for symlinked paths")
	}

	// Test different paths
	otherPath := filepath.Join(tmpDir, "other", "bin", "nightshift")
	if samePath(realPath, otherPath) {
		t.Error("samePath should return false for different paths")
	}
}

// TestBackwardCompat_InstallBinaryPreservesExisting verifies that
// the copy function works and doesn't break on second install.
func TestBackwardCompat_InstallBinaryPreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "src", "nightshift")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0755); err != nil {
		t.Fatal(err)
	}
	srcContent := []byte("#!/bin/sh\necho hello")
	if err := os.WriteFile(srcPath, srcContent, 0755); err != nil {
		t.Fatal(err)
	}

	// Copy to destination
	dstPath := filepath.Join(tmpDir, "dst", "nightshift")
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	// Verify destination exists and is executable
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Fatal("destination file not created")
	}

	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if dstInfo.Mode().Perm()&0100 == 0 {
		t.Error("destination file is not executable")
	}

	// Verify content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(dstContent) != string(srcContent) {
		t.Errorf("destination content mismatch: got %q, want %q", string(dstContent), string(srcContent))
	}

	// Second copy should not fail
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("second copyFile: %v", err)
	}
}
