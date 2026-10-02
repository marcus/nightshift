// Package commits implements Conventional Commits message normalization and
// validation. It exposes pure, well-tested functions used by the CLI and by the
// commit-msg git hook to keep the project's history consistent.
//
// The supported format follows the Conventional Commits 1.0.0 specification:
//
//	<type>(<scope>)!: <subject>
//
//	<body>
//
// The optional "!" after the type (or scope) marks a breaking change, and the
// special "BREAKING CHANGE:" footer in the body is preserved verbatim.
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
	"revert":   {},
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
// A trailing "!" on the type/scope (e.g. "feat!:" or "feat(scope)!:") is parsed
// as the Conventional Commits breaking-change marker and reported via the
// breaking return value.
func parseHeader(header string) (typ, scope, subject string, breaking bool, err error) {
	header = strings.TrimSpace(header)
	colon := strings.Index(header, ":")
	if colon <= 0 {
		return "", "", "", false, ErrMissingType
	}
	prefix := header[:colon]
	subject = strings.TrimSpace(header[colon+1:])

	// Detect the breaking-change marker "!" that sits between the type/scope
	// and the colon before any further parsing.
	prefix = strings.TrimSpace(prefix)
	if strings.HasSuffix(prefix, "!") {
		breaking = true
		prefix = prefix[:len(prefix)-1]
	}

	// Split an optional "(scope)" from the type.
	if strings.HasPrefix(prefix, "(") {
		// A leading "(" with no type is not a valid conventional header.
		return "", "", "", false, ErrMissingType
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

// formatHeader reassembles a canonical header line from its components. A
// breaking-change marker ("!") is emitted after the type or scope when set.
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
// line) are preserved. A trailing footer block — one or more paragraphs whose
// every line is a git footer (a "<Key>: <value>" or "<Key>#<value>" line, or a
// "BREAKING CHANGE:" line) — is emitted verbatim rather than reflowed, so that
// structured metadata such as "Fixes #123", "BREAKING CHANGE: ...", and the
// project's own trailers survive normalization untouched.
func wrapBody(body []string, width int) string {
	paragraphs, footerStart := bodyParagraphs(body)

	var b strings.Builder
	for i, p := range paragraphs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		if i >= footerStart {
			// Footer block: preserve internal line breaks, do not reflow.
			b.WriteString(strings.Join(p, "\n"))
		} else {
			b.WriteString(wrapParagraph(strings.Join(p, " "), width))
		}
	}
	return b.String()
}

// bodyParagraphs splits the body into paragraphs (maximal runs of non-blank
// lines). It also returns footerStart, the index of the first paragraph of a
// trailing contiguous footer block; it equals len(paragraphs) when the body has
// no footer block.
func bodyParagraphs(body []string) (paragraphs [][]string, footerStart int) {
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

	footerStart = len(paragraphs)
	for i := len(paragraphs) - 1; i >= 0; i-- {
		if isFooterParagraph(paragraphs[i]) {
			footerStart = i
		} else {
			break
		}
	}
	return paragraphs, footerStart
}

// isFooterParagraph reports whether every line in the paragraph is a git footer
// line. A footer paragraph is emitted verbatim rather than reflowed.
func isFooterParagraph(lines []string) bool {
	if len(lines) == 0 {
		return false
	}
	for _, l := range lines {
		if !isFooterLine(l) {
			return false
		}
	}
	return true
}

// isFooterLine reports whether l has the shape of a git trailer footer:
//   - "BREAKING CHANGE: <value>", or
//   - "<Token>: <value>" with a colon separator, or
//   - "<Token> #<value>" issue-reference style (e.g. "Fixes #123"),
//
// where Token is a non-empty run of ASCII letters, digits, and dashes, and a
// non-empty value follows the separator.
func isFooterLine(l string) bool {
	if rest := strings.TrimPrefix(l, "BREAKING CHANGE:"); rest != l {
		return strings.TrimSpace(rest) != ""
	}
	if i := strings.Index(l, ": "); i > 0 && isFooterToken(l[:i]) && strings.TrimSpace(l[i+2:]) != "" {
		return true
	}
	if i := strings.Index(l, " #"); i > 0 && isFooterToken(l[:i]) && strings.TrimSpace(l[i+2:]) != "" {
		return true
	}
	return false
}

// isFooterToken reports whether s is a valid footer token: a non-empty run of
// ASCII letters, digits, and dashes.
func isFooterToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
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
