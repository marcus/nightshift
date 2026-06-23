package normalizer

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantMsg  string
		wantWarn []string // each substring must appear in some warning
		wantErr  bool
	}{
		{
			name:    "clean already-normalized message",
			input:   "feat: add login",
			wantMsg: "feat: add login",
		},
		{
			name:    "lowercase subject and strip trailing period",
			input:   "Fix: Add login.",
			wantMsg: "fix: add login",
		},
		{
			name:    "normalize scope and breaking marker",
			input:   "Feat(Auth): Add OAuth support.",
			wantMsg: "feat(auth): add oauth support",
		},
		{
			name:     "missing type prefix warns and lowercases",
			input:    "Update the README",
			wantMsg:  "update the readme",
			wantWarn: []string{"missing a Conventional Commits type prefix"},
		},
		{
			name: "oversized subject warns",
			input: "feat: this is an intentionally very long subject that obviously " +
				"exceeds the seventy-two character guideline by a wide margin",
			wantMsg: "feat: this is an intentionally very long subject that obviously " +
				"exceeds the seventy-two character guideline by a wide margin",
			wantWarn: []string{"subject exceeds 72 characters"},
		},
		{
			name:    "trim trailing whitespace and blank lines",
			input:   "Fix: Trim me.   \n\n  \n",
			wantMsg: "fix: trim me",
		},
		{
			name:    "insert blank line between subject and body",
			input:   "feat: add login\nmore detail here\nand more",
			wantMsg: "feat: add login\n\nmore detail here and more",
		},
		{
			name:    "preserve subject, body, and footer",
			input:   "fix: handle nil pointer in scheduler\n\nThe scheduler panicked when a run had no tasks.\n\nFixes #123",
			wantMsg: "fix: handle nil pointer in scheduler\n\nThe scheduler panicked when a run had no tasks.\n\nFixes #123",
		},
		{
			name:    "empty input errors",
			input:   "   \n",
			wantErr: true,
		},
		{
			name:    "blank-only input errors",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Normalize(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q) error = nil, want error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) unexpected error: %v", tc.input, err)
			}
			if res.Message != tc.wantMsg {
				t.Errorf("Normalize(%q) message =\n%q\nwant\n%q", tc.input, res.Message, tc.wantMsg)
			}
			for _, want := range tc.wantWarn {
				if !warnContains(res.Warnings, want) {
					t.Errorf("Normalize(%q) warnings %q, want one containing %q", tc.input, res.Warnings, want)
				}
			}
			// Absence of expected warnings must be exact too.
			if len(tc.wantWarn) == 0 && len(res.Warnings) != 0 {
				t.Errorf("Normalize(%q) warnings = %q, want none", tc.input, res.Warnings)
			}
		})
	}
}

// TestNormalizeWrapsBody verifies body paragraphs are wrapped to at most WrapLimit
// columns while preserving every word.
func TestNormalizeWrapsBody(t *testing.T) {
	paragraph := strings.Repeat("word ", 40) // a single 200-character run
	res, err := Normalize("docs: rewrite readme\n\n" + paragraph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(res.Message, "\n")
	if lines[0] != "docs: rewrite readme" {
		t.Fatalf("subject = %q, want %q", lines[0], "docs: rewrite readme")
	}
	if lines[1] != "" {
		t.Fatalf("expected a blank line after the subject, got %q", lines[1])
	}
	for i, line := range lines[2:] {
		if n := utf8.RuneCountInString(line); n > WrapLimit {
			t.Errorf("body line %d is %d runes, want <= %d", i, n, WrapLimit)
		}
	}
	if got := strings.Count(res.Message, "word"); got != 40 {
		t.Errorf("preserved %d words, want 40", got)
	}
}

// TestNormalizePreservesFooters verifies footer lines are not wrapped and remain
// separated from the body by a blank line.
func TestNormalizePreservesFooters(t *testing.T) {
	footer := "Signed-off-by: A User with a Very Long Display Name <alongaddress@example.com>"
	res, err := Normalize("chore: bump deps\n\n" + footer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(res.Message, "\n")
	// Lines: [0] subject, [1] "", [2] footer (single line, not wrapped).
	if len(lines) != 3 {
		t.Fatalf("message has %d lines, want 3:\n%s", len(lines), res.Message)
	}
	if lines[1] != "" {
		t.Errorf("expected blank line before footer, got %q", lines[1])
	}
	if lines[2] != footer {
		t.Errorf("footer = %q, want %q", lines[2], footer)
	}
}

// warnContains reports whether any warning contains substr.
func warnContains(warnings []string, substr string) bool {
	for _, w := range warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}
