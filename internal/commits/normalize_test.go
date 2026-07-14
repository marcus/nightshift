package commits

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, raw string) *Parsed {
	t.Helper()
	p, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%q) error: %v", raw, err)
	}
	return p
}

func TestParseHeader(t *testing.T) {
	p := mustParse(t, "feat(commits): add normalizer")
	if p.Type != "feat" {
		t.Errorf("Type = %q, want feat", p.Type)
	}
	if p.Scope != "commits" {
		t.Errorf("Scope = %q, want commits", p.Scope)
	}
	if p.Subject != "add normalizer" {
		t.Errorf("Subject = %q", p.Subject)
	}
	if p.Breaking {
		t.Error("Breaking should be false")
	}
}

func TestParseBreakingHeader(t *testing.T) {
	p := mustParse(t, "feat(api)!: drop v1 endpoints")
	if !p.Breaking {
		t.Error("Breaking should be true")
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	cases := []string{
		"feat(commits): add normalizer",
		"fix: handle empty body",
	}
	for _, in := range cases {
		got, err := Normalize(in)
		if err != nil {
			t.Fatalf("Normalize(%q) error: %v", in, err)
		}
		if got != in {
			t.Errorf("Normalize not idempotent for %q\n got: %q\nwant: %q", in, got, in)
		}
	}
}

func TestNormalizeLowercasesTypeAndScope(t *testing.T) {
	got, _ := Normalize("FEAT(Commits): Add normalizer")
	want := "feat(commits): add normalizer"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeTrimsTrailingPeriod(t *testing.T) {
	got, _ := Normalize("feat: add thing.")
	if got != "feat: add thing" {
		t.Errorf("got %q", got)
	}
}

func TestNormalizeImperativeVerb(t *testing.T) {
	cases := map[string]string{
		"feat: Added new thing":    "feat: add new thing",
		"fix: Fixes the bug":       "fix: fix the bug",
		"docs: Updated the readme": "docs: update the readme",
		"refactor: Refactored X":   "refactor: refactor X",
	}
	for in, want := range cases {
		got, _ := Normalize(in)
		if got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizePreservesCapitalizationWhenNoVerbMatch(t *testing.T) {
	// "Enable" is already imperative; the first letter is lower-cased to the
	// conventional subject style but the rest of the word is preserved.
	got, _ := Normalize("feat: Enable dark mode")
	want := "feat: enable dark mode"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubjectLengthClamping(t *testing.T) {
	long := strings.Repeat("a", 100)
	got, _ := Normalize("feat: " + long)
	header := got
	// The subject portion must be exactly MaxSubjectWidth.
	if subj := strings.TrimPrefix(header, "feat: "); len(subj) != MaxSubjectWidth {
		t.Errorf("clamped subject length %d, want %d; header=%q", len(subj), MaxSubjectWidth, header)
	}
}

func TestNormalizeBodyWrapping(t *testing.T) {
	raw := "feat: add thing\n\n" + strings.Repeat("word ", 30) + "\n\nTrailer-1: value"
	got, _ := Normalize(raw)
	for i, line := range strings.Split(got, "\n") {
		// Header and trailer lines are exempt; only check body paragraph lines.
		if i == 0 || strings.HasPrefix(line, "Trailer-") {
			continue
		}
		if len(line) > MaxBodyWidth {
			t.Errorf("body line exceeds %d cols: %q (len %d)", MaxBodyWidth, line, len(line))
		}
	}
}

func TestNormalizePreservesParagraphBreaks(t *testing.T) {
	raw := "feat: add thing\n\nfirst paragraph here\n\nsecond paragraph here"
	got, _ := Normalize(raw)
	if !strings.Contains(got, "\n\nsecond") {
		t.Errorf("paragraph break not preserved: %q", got)
	}
}

func TestTrailerPreservation(t *testing.T) {
	raw := "feat: add thing\n\nBody text.\n\nSigned-off-by: Alice <a@example.com>\nReviewed-by: Bob"
	p := mustParse(t, raw)
	if len(p.Trailers) != 2 {
		t.Fatalf("got %d trailers, want 2", len(p.Trailers))
	}
	if p.Trailers[0].Key != "Signed-off-by" {
		t.Errorf("first trailer key = %q", p.Trailers[0].Key)
	}
	got := p.Normalize()
	if !strings.Contains(got, "Signed-off-by: Alice <a@example.com>") {
		t.Errorf("trailer value lost:\n%s", got)
	}
}

func TestNightshiftTrailerOrdering(t *testing.T) {
	raw := "feat: add thing\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://example.com\nSigned-off-by: Alice"
	got, _ := Normalize(raw)
	// External trailer must come before Nightshift trailers, and Nightshift
	// trailers must be sorted alphabetically.
	bobIdx := strings.Index(got, "Signed-off-by")
	refIdx := strings.Index(got, "Nightshift-Ref")
	taskIdx := strings.Index(got, "Nightshift-Task")
	if !(bobIdx < refIdx && refIdx < taskIdx) {
		t.Errorf("trailer ordering wrong:\n%s", got)
	}
}

func TestValidateWellFormed(t *testing.T) {
	vs := Validate("feat(commits): add normalizer\n\nBody line one.\n\nSigned-off-by: A")
	if len(vs) != 0 {
		t.Errorf("expected no violations, got: %v", vs)
	}
}

func TestValidateMissingType(t *testing.T) {
	vs := Validate("just a subject with no type")
	if !hasCode(vs, CodeMissingType) && !hasCode(vs, CodeUnknownType) {
		t.Errorf("expected missing/unknown type violation, got: %v", vs)
	}
}

func TestValidateUnknownType(t *testing.T) {
	vs := Validate("wat: something")
	if !hasCode(vs, CodeUnknownType) {
		t.Errorf("expected unknown type, got: %v", vs)
	}
}

func TestValidateOversizeSubject(t *testing.T) {
	vs := Validate("feat: " + strings.Repeat("a", 100))
	if !hasCode(vs, CodeOversizeSubject) {
		t.Errorf("expected oversize subject, got: %v", vs)
	}
}

func TestValidateNonImperative(t *testing.T) {
	vs := Validate("feat: Added the thing")
	if !hasCode(vs, CodeNonImperative) {
		t.Errorf("expected non-imperative, got: %v", vs)
	}
}

func TestValidateTrailingPeriod(t *testing.T) {
	vs := Validate("feat: add thing.")
	if !hasCode(vs, CodeTrailingPeriod) {
		t.Errorf("expected trailing period, got: %v", vs)
	}
}

func TestValidateUnwrappedBody(t *testing.T) {
	long := strings.Repeat("word ", 30)
	vs := Validate("feat: add thing\n\n" + long)
	if !hasCode(vs, CodeUnwrappedBody) {
		t.Errorf("expected unwrapped body, got: %v", vs)
	}
}

func TestValidateEmpty(t *testing.T) {
	vs := Validate("   \n\n  ")
	if !hasCode(vs, CodeMissingHeader) {
		t.Errorf("expected missing header, got: %v", vs)
	}
}

func TestValidateBaseVerbEdEnding(t *testing.T) {
	// "embed" ends in "ed" but is a legitimate base verb.
	vs := Validate("feat: embed the payload")
	if hasCode(vs, CodeNonImperative) {
		t.Errorf("embed should not be flagged: %v", vs)
	}
}

func TestConforms(t *testing.T) {
	if !Conforms("feat: add thing") {
		t.Error("expected to conform")
	}
	if Conforms("nope: bad") {
		t.Error("expected not to conform")
	}
}

func hasCode(vs []Violation, code string) bool {
	for _, v := range vs {
		if v.Code == code {
			return true
		}
	}
	return false
}
