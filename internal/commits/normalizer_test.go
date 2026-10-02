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
			name: "lowercases a capitalized subject",
			in:   "feat: Add login screen",
			want: "feat: add login screen",
		},
		{
			name: "revert type is allowed",
			in:   "revert: feat: add login screen",
			want: "revert: feat: add login screen",
		},
		{
			name: "breaking-change marker is preserved",
			in:   "feat!: drop support for v1",
			want: "feat!: drop support for v1",
		},
		{
			name: "breaking-change marker with scope is preserved",
			in:   "FEAT(API)!: Drop support for v1",
			want: "feat(API)!: drop support for v1",
		},
		{
			name: "subject shrinking to the limit after trimming the trailing period is accepted",
			in:   "feat: " + strings.Repeat("a", MaxSubjectLength) + ".",
			want: "feat: " + strings.Repeat("a", MaxSubjectLength),
		},
		{
			name: "trailer block in the last paragraph is preserved verbatim",
			in: "chore: release v1.2.3\n\nbody text that is long enough to be wrapped when the normalizer reflows this paragraph onto multiple lines at the configured width\n" +
				"Nightshift-Task: release\nNightshift-Ref: https://github.com/marcus/nightshift",
			want: "chore: release v1.2.3\n\n" +
				"body text that is long enough to be wrapped when the normalizer reflows\n" +
				"this paragraph onto multiple lines at the configured width\n" +
				"Nightshift-Task: release\nNightshift-Ref: https://github.com/marcus/nightshift",
		},
		{
			name: "breaking change footer is preserved verbatim",
			in:   "feat!: drop the legacy API\n\nBREAKING CHANGE: removes /v1 endpoints entirely, use /v2 instead",
			want: "feat!: drop the legacy API\n\nBREAKING CHANGE: removes /v1 endpoints entirely, use /v2 instead",
		},
		{
			name: "body wrapping counts runes not bytes",
			// Three 10-rune words of 3-byte CJK characters: 32 runes but 92
			// bytes. At rune width they fit on one line; byte counting would
			// split them.
			in:   "docs: wrapping\n\n" + strings.Repeat("日", 10) + " " + strings.Repeat("日", 10) + " " + strings.Repeat("日", 10),
			want: "docs: wrapping\n\n" + strings.Repeat("日", 10) + " " + strings.Repeat("日", 10) + " " + strings.Repeat("日", 10),
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

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{
			name: "canonical message passes",
			in:   "fix(api): handle nil response\n\nbody wrapped at seventy-two columns by hand\n",
		},
		{
			name: "comment lines ignored",
			in:   "feat: add login\n# git template comment\n",
		},
		{
			name:    "uppercase type fails",
			in:      "FEAT: add login",
			wantErr: ErrNotCanonical,
		},
		{
			name:    "capitalized subject fails",
			in:      "feat: Add login",
			wantErr: ErrNotCanonical,
		},
		{
			name:    "unwrapped body fails",
			in:      "feat: add login\n\nthis body paragraph is intentionally far longer than the configured wrap width so it must be hard wrapped by the normalizer",
			wantErr: ErrNotCanonical,
		},
		{
			name:    "unknown type fails",
			in:      "wip: halfway done",
			wantErr: ErrUnknownType,
		},
		{
			name:    "empty message fails",
			in:      "\n\n# only comments\n",
			wantErr: ErrEmptyMessage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.in)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate(%q): unexpected error: %v", tc.in, err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Validate(%q): expected error to wrap %v, got %v", tc.in, tc.wantErr, err)
			}
		})
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	cases := []string{
		"feat: add login screen",
		"fix(api): handle nil response\n\nLong body that explains the fix in more detail than the subject alone can manage so that we exercise the wrapping path too and then some more words here.",
		"docs: update README\n\nfirst paragraph\n\nsecond paragraph stays separate",
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
