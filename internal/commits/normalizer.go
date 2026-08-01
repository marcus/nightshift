// Package commits implements Conventional Commits message normalization and
// validation. It exposes pure, well-tested functions used by the CLI and by the
// commit-msg git hook to keep the project's history consistent.
//
// The supported format follows the Conventional Commits 1.0.0 specification:
//
//	<type>(<scope>): <subject>
//
//	<body>
//
// The normalizer is intentionally strict but constructive: rather than silently
// accepting malformed input it fixes the trivially fixable (whitespace, type
// casing, trailing punctuation, body wrapping) and rejects anything that needs
// a human decision (missing type, unknown type, missing subject).
package commits

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxSubjectLength is the maximum number of runes allowed in a commit subject.
const MaxSubjectLength = 72

// BodyWrapWidth is the column at which the commit body is wrapped.
const BodyWrapWidth = 72

// allowedTypes is the set of Conventional Commit types this project accepts.
var allowedTypes = map[string]struct{}{
	"feat":     {},
	"fix":      {},
	"docs":     {},
	"style":    {},
	"refactor": {},
	"test":     {},
	"chore":    {},
	"perf":     {},
	"build":    {},
	"ci":       {},
}

// Errors returned by the normalizer. They are wrapped so callers can match on
// the underlying cause with errors.Is.
var (
	// ErrEmptyMessage is returned when the message contains no non-comment,
	// non-whitespace content.
	ErrEmptyMessage = errors.New("commit message is empty")
	// ErrMissingType is returned when the subject line is not a Conventional
	// Commit (no type prefix before the colon).
	ErrMissingType = errors.New("commit message must start with a conventional commit type")
	// ErrUnknownType is returned when the type prefix is not in the allowed set.
	ErrUnknownType = errors.New("commit type is not in the allowed set")
	// ErrMissingSubject is returned when the type prefix is present but no
	// subject text follows the colon.
	ErrMissingSubject = errors.New("commit subject is missing")
	// ErrSubjectTooLong is returned when the subject exceeds MaxSubjectLength.
	ErrSubjectTooLong = fmt.Errorf("commit subject exceeds %d characters", MaxSubjectLength)
	// ErrSubjectLowercase is returned when the subject starts with an uppercase
	// letter (the rule is "do not capitalize the subject").
	ErrSubjectLowercase = errors.New("commit subject must not be capitalized")
)

// Normalize parses, validates, and rewrites a raw commit message so that it
// conforms to the project's Conventional Commits rules. It returns the
// canonical form and a non-nil error describing the first unrecoverable
// problem when the message cannot be normalized.
//
// Normalization is idempotent: Normalize(Normalize(m)) == Normalize(m).
func Normalize(msg string) (string, error) {
	lines := stripComments(msg)
	if len(lines) == 0 {
		return "", ErrEmptyMessage
	}

	header := lines[0]
	body := lines[1:]

	typ, scope, subject, breaking, err := parseHeader(header)
	if err != nil {
		return "", err
	}

	subject = cleanSubject(subject)

	var b strings.Builder
	b.WriteString(formatHeader(typ, scope, subject, breaking))

	wrapped := wrapBody(body, BodyWrapWidth)
	if wrapped != "" {
		b.WriteString("\n\n")
		b.WriteString(wrapped)
	}

	return b.String(), nil
}

// stripComments removes git's commented-out lines (those beginning with "#"),
// trims trailing whitespace from every line, and drops leading/trailing blank
// lines. It returns the meaningful lines of the message.
func stripComments(msg string) []string {
	rawLines := strings.Split(msg, "\n")
	out := make([]string, 0, len(rawLines))
	for _, l := range rawLines {
		l = strings.TrimRight(l, " \t\r")
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		out = append(out, l)
	}
	// Drop leading and trailing blank lines.
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return out
}

// parseHeader splits the first line into its Conventional Commit components and
// validates them. The returned type is lower-cased to match the allowed set.
// The breaking flag is true when the type/scope prefix ends with a "!"
// (e.g. "feat!:" or "feat(api)!:"), per the Conventional Commits 1.0.0 spec.
func parseHeader(header string) (typ, scope, subject string, breaking bool, err error) {
	header = strings.TrimSpace(header)
	colon := strings.Index(header, ":")
	if colon <= 0 {
		return "", "", "", false, ErrMissingType
	}
	prefix := header[:colon]
	subject = strings.TrimSpace(header[colon+1:])

	// Split an optional "(scope)" from the type.
	prefix = strings.TrimSpace(prefix)
	if strings.HasPrefix(prefix, "(") {
		// A leading "(" with no type is not a valid conventional header.
		return "", "", "", false, ErrMissingType
	}
	// A trailing "!" after the type/scope marks a breaking change. It must be
	// stripped before scope validation so "feat(api)!:" is parsed as
	// feat/(api)/breaking rather than rejected.
	if strings.HasSuffix(prefix, "!") {
		breaking = true
		prefix = strings.TrimSuffix(prefix, "!")
	}
	if open := strings.Index(prefix, "("); open > 0 && strings.HasSuffix(prefix, ")") {
		typ = prefix[:open]
		scope = prefix[open+1 : len(prefix)-1]
	} else {
		typ = prefix
	}
	typ = strings.ToLower(strings.TrimSpace(typ))
	scope = strings.TrimSpace(scope)

	if typ == "" {
		return "", "", "", false, ErrMissingType
	}
	if !isAllowedType(typ) {
		return "", "", "", false, fmt.Errorf("%w: %q", ErrUnknownType, typ)
	}
	if strings.TrimSpace(subject) == "" {
		return "", "", "", false, ErrMissingSubject
	}
	if utf8.RuneCountInString(subject) > MaxSubjectLength {
		return "", "", "", false, ErrSubjectTooLong
	}
	if startsUpper(subject) {
		return "", "", "", false, ErrSubjectLowercase
	}
	return typ, scope, subject, breaking, nil
}

// cleanSubject normalizes the subject text: lowercases a leading uppercase
// letter is *not* done here (capitalization is a hard error, not a fix), but
// surrounding whitespace and a trailing period are removed.
func cleanSubject(subject string) string {
	s := strings.TrimSpace(subject)
	s = strings.TrimRight(s, ".")
	return s
}

// formatHeader reassembles a canonical header line from its components. When
// breaking is true the "!" indicator is re-emitted after the type/scope prefix,
// matching the Conventional Commits 1.0.0 breaking-change form.
func formatHeader(typ, scope, subject string, breaking bool) string {
	bang := ""
	if breaking {
		bang = "!"
	}
	if scope != "" {
		return typ + "(" + scope + ")" + bang + ": " + subject
	}
	return typ + bang + ": " + subject
}

// wrapBody collapses runs of blank lines, preserves non-blank paragraphs, and
// hard-wraps each paragraph line to width. Paragraph breaks (a single blank
// line) are preserved.
func wrapBody(body []string, width int) string {
	var paragraphs [][]string
	var cur []string
	for _, l := range body {
		if strings.TrimSpace(l) == "" {
			if len(cur) > 0 {
				paragraphs = append(paragraphs, cur)
				cur = nil
			}
			continue
		}
		cur = append(cur, strings.TrimSpace(l))
	}
	if len(cur) > 0 {
		paragraphs = append(paragraphs, cur)
	}

	var b strings.Builder
	for i, p := range paragraphs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		if isBreakingFooter(p[0]) {
			// The BREAKING CHANGE: footer is a trailer, not prose: emit it
			// verbatim (preserving its original line breaks) instead of
			// joining its lines into a single wrapped paragraph.
			b.WriteString(strings.Join(p, "\n"))
			continue
		}
		b.WriteString(wrapParagraph(strings.Join(p, " "), width))
	}
	return b.String()
}

// isBreakingFooter reports whether the given line opens a Conventional Commits
// "BREAKING CHANGE:" (or "BREAKING-CHANGE:") footer. Both spellings are valid
// per the 1.0.0 spec.
func isBreakingFooter(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING-CHANGE:")
}

// wrapParagraph hard-wraps a single-line paragraph at width, breaking on word
// boundaries. A word longer than width is left intact rather than split.
func wrapParagraph(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	lineLen := 0
	for i, w := range words {
		if i == 0 {
			b.WriteString(w)
			lineLen = len(w)
			continue
		}
		if lineLen+1+len(w) <= width {
			b.WriteByte(' ')
			b.WriteString(w)
			lineLen += 1 + len(w)
		} else {
			b.WriteByte('\n')
			b.WriteString(w)
			lineLen = len(w)
		}
	}
	return b.String()
}

// isAllowedType reports whether typ is one of the accepted Conventional Commit
// types.
func isAllowedType(typ string) bool {
	_, ok := allowedTypes[typ]
	return ok
}

// startsUpper reports whether the first rune of s is an ASCII uppercase letter.
func startsUpper(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r >= 'A' && r <= 'Z'
}
