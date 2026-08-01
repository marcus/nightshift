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
		{
			name: "breaking-change bang indicator",
			in:   "feat!: drop support for Go 1.20",
			want: "feat!: drop support for Go 1.20",
		},
		{
			name: "breaking-change bang indicator with scope",
			in:   "fix(api)!: change response shape",
			want: "fix(api)!: change response shape",
		},
		{
			name: "breaking-change footer preserved verbatim",
			in:   "feat: redesign API\n\nBREAKING CHANGE: the old /v1 endpoints are gone",
			want: "feat: redesign API\n\nBREAKING CHANGE: the old /v1 endpoints are gone",
		},
		{
			name: "breaking-change hyphen footer preserved verbatim",
			in:   "feat: redesign API\n\nBREAKING-CHANGE: the old /v1 endpoints are gone",
			want: "feat: redesign API\n\nBREAKING-CHANGE: the old /v1 endpoints are gone",
		},
		{
			name: "breaking footer not hard-wrapped",
			in:   "feat: redesign API\n\nBREAKING CHANGE: " + strings.Repeat("word ", 40),
			want: "feat: redesign API\n\nBREAKING CHANGE: " + strings.Join(strings.Fields(strings.Repeat("word ", 40)), " "),
		},
		// --- git trailer preservation (the core safety guarantee) ---
		{
			name: "project trailers preserved verbatim, one per line",
			in:   "feat: do thing\n\nbody.\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://github.com/marcus/nightshift",
			want: "feat: do thing\n\nbody.\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://github.com/marcus/nightshift",
		},
		{
			name: "signed-off-by and co-authored-by trailers preserved",
			in:   "feat: pair on thing\n\nbody text here\n\nSigned-off-by: Alice <alice@example.com>\nCo-authored-by: Bob <bob@example.com>",
			want: "feat: pair on thing\n\nbody text here\n\nSigned-off-by: Alice <alice@example.com>\nCo-authored-by: Bob <bob@example.com>",
		},
		{
			name: "github issue trailer without colon preserved",
			in:   "fix: handle nil\n\nbody.\n\nFixes #123",
			want: "fix: handle nil\n\nbody.\n\nFixes #123",
		},
		{
			name: "trailer value containing a colon is not split",
			in:   "feat: x\n\nReviewed-by: https://example.com/reviews/42",
			want: "feat: x\n\nReviewed-by: https://example.com/reviews/42",
		},
		// --- subject-length validation runs after the trailing period is stripped ---
		{
			name: "subject one over limit but in range after period stripped",
			in:   "feat: " + strings.Repeat("a", MaxSubjectLength-4) + ".",
			want: "feat: " + strings.Repeat("a", MaxSubjectLength-4),
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
		"feat!: drop the legacy CLI\n\nBREAKING CHANGE: the legacy CLI is removed",
		"fix(api)!: change response shape\n\nBREAKING-CHANGE: payloads no longer include legacy fields",
		// A message with the project's own trailers must round-trip unchanged.
		"feat: do thing\n\nbody.\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://github.com/marcus/nightshift",
		"feat: pair\n\nbody.\n\nSigned-off-by: Alice <alice@example.com>\nCo-authored-by: Bob <bob@example.com>",
		"fix: handle nil\n\nbody.\n\nFixes #123",
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
