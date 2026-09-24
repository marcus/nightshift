package commitmsg

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"conventional input is idempotent", "feat(parser): add support\n", "feat(parser): add support\n"},
		{"bare subject", "Improve the setup flow\n", "chore: improve the setup flow\n"},
		{"bare subject with colon", "URL: preserve the endpoint\n", "chore: url: preserve the endpoint\n"},
		{"inferred types", "Fix broken login\n", "fix: broken login\n"},
		{"punctuation cleanup", "feat: Add the API!\n", "feat: add the api\n"},
		{"body spacing and wrapping", "fix: repair login\nbody line   with extra spaces\n\n\n", "fix: repair login\n\nbody line   with extra spaces\n"},
		{"breaking changes", "feat!: replace the config format\n\nBREAKING CHANGE: migrate existing files\n", "feat!: replace the config format\n\nBREAKING CHANGE: migrate existing files\n"},
		{"trailer preservation", "fix: repair login\n\nbody\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://github.com/marcus/nightshift\n", "fix: repair login\n\nbody\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://github.com/marcus/nightshift\n"},
		{"trailer block has a blank line after the body", "fix: repair login\n\nbody\n\nNightshift-Task: commit-normalize\n", "fix: repair login\n\nbody\n\nNightshift-Task: commit-normalize\n"},
		{"duplicate trailer removal", "fix: repair login\n\nSigned-off-by: A\nSigned-off-by: A\n", "fix: repair login\n\nSigned-off-by: A\n"},
		{"preserve leading description punctuation", "fix: #123 crash!\n", "fix: #123 crash\n"},
		{"preserve distinct repeated trailers", "feat: add authors\n\nCo-authored-by: A <a@example.com>\nCo-authored-by: B <b@example.com>\n", "feat: add authors\n\nCo-authored-by: A <a@example.com>\nCo-authored-by: B <b@example.com>\n"},
		{"preserve body formatting", "fix: repair login\n\nUse two  spaces:\n  code:  value\n", "fix: repair login\n\nUse two  spaces:\n  code:  value\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Normalize(test.input)
			if err != nil {
				t.Fatalf("Normalize() error = %v", err)
			}
			if got != test.want {
				t.Errorf("Normalize() = %q, want %q", got, test.want)
			}
			again, err := Normalize(got)
			if err != nil {
				t.Fatalf("second Normalize() error = %v", err)
			}
			if again != got {
				t.Errorf("Normalize() is not idempotent: second result = %q", again)
			}
		})
	}
}

func TestNormalizeRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   error
	}{
		{"empty input", "\n\t\n", ErrEmptyMessage},
		{"long subject", "feat: " + strings.Repeat("x", 70) + "\n", ErrSubjectTooLong},
		{"unknown conventional type", "unknown: do the thing\n", ErrInvalidMessage},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Normalize(test.input)
			if !errors.Is(err, test.err) {
				t.Fatalf("Normalize() error = %v, want errors.Is(..., %v)", err, test.err)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	if err := Validate("fix: repair login\n"); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := Validate("Fix broken login\n"); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("Validate() error = %v, want ErrInvalidMessage", err)
	}
}

func TestFormat(t *testing.T) {
	formatted, err := Format(Message{
		Type:        "feat",
		Scope:       "CLI",
		Breaking:    true,
		Description: "Add commit message support",
		Body:        []string{"This body is wrapped when it contains enough words to exceed the configured line length for commit messages."},
		Trailers:    []Trailer{{Token: "Nightshift-Task", Value: "commit-normalize"}},
	})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if !strings.HasPrefix(formatted, "feat(cli)!: add commit message support\n\n") {
		t.Fatalf("Format() = %q, missing canonical subject", formatted)
	}
	for _, line := range strings.Split(strings.TrimSuffix(formatted, "\n"), "\n") {
		if !strings.Contains(line, "Nightshift-Task:") && len([]rune(line)) > SubjectLimit {
			t.Errorf("formatted line exceeds limit: %q", line)
		}
	}
}
