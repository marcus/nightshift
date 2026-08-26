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
			name: "revert type accepted",
			in:   "revert: feat(api): drop v2 endpoint",
			want: "revert: feat(api): drop v2 endpoint",
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
			name: "keeps trailer lines verbatim and unwrapped",
			in:   "chore: bump deps\n\nupdate everything to latest and here is a long explanatory paragraph that goes past the wrap width so it has to be wrapped by the normalizer\n\nSigned-off-by: Jane <jane@example.com>\nNightshift-Task: commit-normalize",
			want: "chore: bump deps\n\n" +
				"update everything to latest and here is a long explanatory paragraph\n" +
				"that goes past the wrap width so it has to be wrapped by the normalizer" +
				"\n\n" +
				"Signed-off-by: Jane <jane@example.com>\n" +
				"Nightshift-Task: commit-normalize",
		},
		{
			name: "trailer-shaped line mid-body stays prose in place",
			in:   "fix(api): retry on 429\n\nbefore the note this paragraph has some words\nNote: this line looks like a trailer but is mid-body prose\nand the paragraph continues after it unchanged in order",
			want: "fix(api): retry on 429\n\n" +
				"before the note this paragraph has some words Note: this line looks like\n" +
				"a trailer but is mid-body prose and the paragraph continues after it\n" +
				"unchanged in order",
		},
		{
			name: "trailer-shaped block followed by prose is wrapped as prose",
			in:   "chore: tidy\n\nSigned-off-by: Jane <jane@example.com>\n\none closing paragraph of prose",
			want: "chore: tidy\n\n" +
				"Signed-off-by: Jane <jane@example.com>" +
				"\n\n" +
				"one closing paragraph of prose",
		},
		{
			name: "trailing period trimmed before subject length is checked",
			in:   "feat: " + strings.Repeat("a", MaxSubjectLength) + ".",
			want: "feat: " + strings.Repeat("a", MaxSubjectLength),
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
			name:    "malformed scope with unbalanced parens rejected",
			in:      "feat(a)b): add thing",
			wantErr: ErrInvalidScope,
		},
		{
			name:    "empty scope rejected",
			in:      "feat(): add thing",
			wantErr: ErrInvalidScope,
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
		"chore: tidy\n\nsome prose that wraps because it is long enough to need it across columns\n\nSigned-off-by: Jane <jane@example.com>",
		"fix: keep prose order\n\na paragraph mentioning Note: inline that is long enough that the wrapping code has to run over it and fold it across several lines",
		"chore: trailers last\n\nprose paragraph\n\nSigned-off-by: Jane <jane@example.com>\nReviewed-by: Bob <bob@example.com>",
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
	for _, typ := range []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"} {
		if !isAllowedType(typ) {
			t.Errorf("expected %q to be an allowed type", typ)
		}
	}
	if isAllowedType("wip") {
		t.Error("did not expect wip to be allowed")
	}
}
