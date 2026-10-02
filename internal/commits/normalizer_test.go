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
			name: "breaking-change marker without scope",
			in:   "feat!: redesign the public API",
			want: "feat!: redesign the public API",
		},
		{
			name: "breaking-change marker with scope",
			in:   "feat(api)!: redesign the public API",
			want: "feat(api)!: redesign the public API",
		},
		{
			name: "breaking-change marker preserved with fix type",
			in:   "fix!: stop crashing on nil input",
			want: "fix!: stop crashing on nil input",
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
			wantErr: ErrSubjectCapitalized,
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
		{
			name:    "breaking marker with unknown type still rejected",
			in:      "wip!: not allowed",
			wantErr: ErrUnknownType,
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
		"feat!: redesign the public API",
		"fix(api)!: handle nil response",
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
	for _, typ := range []string{"feat", "fix", "docs", "style", "refactor", "test", "chore", "perf", "build", "ci"} {
		if !isAllowedType(typ) {
			t.Errorf("expected %q to be an allowed type", typ)
		}
	}
	if isAllowedType("wip") {
		t.Error("did not expect wip to be allowed")
	}
}
