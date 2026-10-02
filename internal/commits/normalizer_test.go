package commits

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{
			name: "valid simple feat",
			in:   "feat: add login screen",
			want: "feat: add login screen",
		},
		{
			name: "valid with scope",
			in:   "fix(api): handle nil response",
			want: "fix(api): handle nil response",
		},
		{
			name: "trims surrounding whitespace and trailing period",
			in:   "  docs: update README.  ",
			want: "docs: update README",
		},
		{
			name: "lowercases an uppercased type",
			in:   "FEAT(ui): render button",
			want: "feat(ui): render button",
		},
		{
			name: "breaking-change marker without scope",
			in:   "feat!: drop support for v1",
			want: "feat!: drop support for v1",
		},
		{
			name: "breaking-change marker with scope",
			in:   "feat(api)!: rename public endpoint",
			want: "feat(api)!: rename public endpoint",
		},
		{
			name: "revert type accepted",
			in:   "revert: roll back the broken release",
			want: "revert: roll back the broken release",
		},
		{
			name: "previously-rejected feat! now round-trips",
			in:   "  FEAT!: remove legacy parser  ",
			want: "feat!: remove legacy parser",
		},
		{
			name: "BREAKING CHANGE footer preserved verbatim",
			in:   "feat!: drop v1\n\nThe old API is gone.\n\nBREAKING CHANGE: removes /v1 entirely",
			want: "feat!: drop v1\n\nThe old API is gone.\n\nBREAKING CHANGE: removes /v1 entirely",
		},
		{
			name: "structured trailers preserved and not reflowed",
			in: "fix: patch leak\n\nSome long explanatory paragraph that would normally be hard wrapped by the normalizer onto several lines so we can confirm it still wraps while the trailers below stay verbatim.\n\n" +
				"Fixes #123\n" +
				"Reviewed-by: Alice\n" +
				"Nightshift-Task: commit-normalize\n" +
				"Nightshift-Ref: https://example.com/repo",
			want: "fix: patch leak\n\nSome long explanatory paragraph that would normally be hard wrapped by\nthe normalizer onto several lines so we can confirm it still wraps while\nthe trailers below stay verbatim.\n\n" +
				"Fixes #123\n" +
				"Reviewed-by: Alice\n" +
				"Nightshift-Task: commit-normalize\n" +
				"Nightshift-Ref: https://example.com/repo",
		},
		{
			name: "preserves body and wraps long lines",
			in:   "feat: add thing\n\nthis is a body paragraph that is intentionally far longer than the configured wrap width so it must be hard wrapped onto multiple lines by the normalizer function",
			want: "feat: add thing\n\n" +
				"this is a body paragraph that is intentionally far longer than the\n" +
				"configured wrap width so it must be hard wrapped onto multiple lines by\n" +
				"the normalizer function",
		},
		{
			name: "strips git comment lines",
			in:   "chore: tidy\n# please enter the commit message\n\nbody here",
			want: "chore: tidy\n\nbody here",
		},
		{
			name:    "missing type rejected",
			in:      "just a plain message",
			wantErr: ErrMissingType,
		},
		{
			name:    "unknown type rejected",
			in:      "wip: halfway done",
			wantErr: ErrUnknownType,
		},
		{
			name:    "missing subject rejected",
			in:      "feat:",
			wantErr: ErrMissingSubject,
		},
		{
			name:    "capitalized subject rejected",
			in:      "feat: Add login screen",
			wantErr: ErrSubjectLowercase,
		},
		{
			name:    "overlong subject rejected",
			in:      "feat: " + strings.Repeat("a", MaxSubjectLength+1),
			wantErr: ErrSubjectTooLong,
		},
		{
			name:    "empty message rejected",
			in:      "\n\n# only comments\n  \n",
			wantErr: ErrEmptyMessage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Normalize(tc.in)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("Normalize(%q): expected error %v, got nil (result %q)", tc.in, tc.wantErr, got)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Normalize(%q): expected error to wrap %v, got %v", tc.in, tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q): unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("Normalize(%q):\n got: %q\nwant: %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	cases := []string{
		"feat: add login screen",
		"fix(api): handle nil response\n\nLong body that explains the fix in more detail than the subject alone can manage so that we exercise the wrapping path too and then some more words here.",
		"docs: update README\n\nfirst paragraph\n\nsecond paragraph stays separate",
		"feat!: drop support for v1",
		"feat(api)!: rename public endpoint",
		"revert: roll back the broken release",
		"fix: patch leak\n\nExplains the fix with a long paragraph that exercises wrapping here.\n\nFixes #123\nNightshift-Task: commit-normalize",
		"feat!: remove v1\n\nBody before footers.\n\nBREAKING CHANGE: removes /v1 entirely",
	}
	for _, in := range cases {
		once, err := Normalize(in)
		if err != nil {
			t.Fatalf("first Normalize(%q) errored: %v", in, err)
		}
		twice, err := Normalize(once)
		if err != nil {
			t.Fatalf("second Normalize(%q) errored: %v", once, err)
		}
		if once != twice {
			t.Errorf("not idempotent for %q\n once:  %q\n twice: %q", in, once, twice)
		}
	}
}

func TestAllowedTypes(t *testing.T) {
	for _, typ := range []string{"feat", "fix", "docs", "style", "refactor", "test", "chore", "perf", "build", "ci", "revert"} {
		if !isAllowedType(typ) {
			t.Errorf("expected %q to be an allowed type", typ)
		}
	}
	if isAllowedType("wip") {
		t.Error("did not expect wip to be allowed")
	}
}

func TestFooterDetection(t *testing.T) {
	t.Run("footer line shapes", func(t *testing.T) {
		good := []string{
			"Fixes #123",
			"BREAKING CHANGE: drop v1",
			"Co-authored-by: A. Uthor <a@example.com>",
			"Nightshift-Task: commit-normalize",
			"Reviewed-by: Alice",
		}
		for _, l := range good {
			if !isFooterLine(l) {
				t.Errorf("expected %q to be a footer line", l)
			}
		}
		bad := []string{
			"plain prose",
			"no separator here",
			": missing key",
			"key:",   // no value after separator
			"Fixes#", // no value after separator
		}
		for _, l := range bad {
			if isFooterLine(l) {
				t.Errorf("did not expect %q to be a footer line", l)
			}
		}
	})

	t.Run("prose paragraph not treated as footer block", func(t *testing.T) {
		// A trailing paragraph that mixes footer-shaped and prose lines is not
		// a footer block and must be reflowed normally.
		prose := "first line is fine yes and this is plainly long prose that should be hard wrapped by the normalizer onto several lines for sure"
		in := "docs: note\n\nfirst line is fine: yes\n" + prose
		got, err := Normalize(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The prose paragraph must not be preserved verbatim; it should be wrapped.
		if strings.Contains(got, prose) {
			t.Errorf("expected prose to be reflowed, got %q", got)
		}
	})
}
