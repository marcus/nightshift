// Package normalizer standardizes Conventional Commits messages.
//
// A normalized message has the shape:
//
//	type(scope): subject
//
//	<body paragraphs wrapped at 100 columns>
//
//	Footer-Key: value
//
// Normalization lowercases the subject line, strips trailing periods, collapses
// stray whitespace, guarantees a single blank line between the subject and the
// body, enforces the "type(scope)!?: subject" shape, and wraps body paragraphs
// to at most 100 columns.
package normalizer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// SubjectLimit is the recommended maximum length of the subject line.
	SubjectLimit = 72
	// WrapLimit is the maximum column width for body paragraphs.
	WrapLimit = 100
)

var (
	// prefixRe captures a Conventional Commits type prefix — type, type(scope),
	// type!, or type(scope)! — immediately followed by a colon.
	prefixRe = regexp.MustCompile(`^([a-z]+)(?:\(([^)]+)\))?(!)?:`)
	// footerRe matches a footer line: "Token: value" or "Token #123".
	footerRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*:(?:\s.*)?$|^[A-Za-z][A-Za-z0-9_-]*\s#\d+$`)
)

// Result is a normalized commit message together with any non-fatal diagnostics.
type Result struct {
	// Message is the normalized commit message.
	Message string
	// Warnings describes non-fatal issues such as an oversized or untyped subject.
	Warnings []string
}

// Normalize standardizes a Conventional Commits message. It returns an error
// only when the input is empty; structural problems are reported as warnings so
// callers can still emit the cleaned message.
func Normalize(input string) (Result, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return Result{}, errors.New("commit message is empty")
	}

	lines := strings.Split(trimmed, "\n")
	subject, warnings := normalizeSubject(strings.TrimSpace(lines[0]))

	body, footers := splitBody(strings.Join(lines[1:], "\n"))
	body = wrapBody(body)

	sections := []string{subject}
	if body != "" {
		sections = append(sections, body)
	}
	if footers != "" {
		sections = append(sections, footers)
	}

	return Result{
		Message:  strings.Join(sections, "\n\n"),
		Warnings: warnings,
	}, nil
}

// normalizeSubject lowercases the subject, strips trailing periods, and
// normalizes the type(scope)!? shape. It returns the cleaned subject and any
// warnings.
func normalizeSubject(raw string) (string, []string) {
	var warnings []string

	// Lowercase the entire subject line per the normalization rules.
	subject := strings.ToLower(strings.TrimSpace(raw))

	if m := prefixRe.FindStringSubmatch(subject); m != nil {
		// Rebuild a canonical "type(scope)!?: description".
		typ, scope, breaking := m[1], m[2], m[3]
		desc := strings.TrimRight(strings.TrimSpace(strings.TrimPrefix(subject, m[0])), ".")
		subject = typ
		if scope != "" {
			subject += "(" + scope + ")"
		}
		subject += breaking + ":"
		if desc != "" {
			subject += " " + desc
		}
	} else {
		warnings = append(warnings, `subject is missing a Conventional Commits type prefix (expected "type(scope): ...")`)
		subject = strings.TrimRight(subject, ".")
	}

	if n := width(subject); n > SubjectLimit {
		warnings = append(warnings, fmt.Sprintf("subject exceeds %d characters (got %d)", SubjectLimit, n))
	}

	return subject, warnings
}

// splitBody separates the body region into body paragraphs and a trailing footer
// block. Footers ("Token: value" / "Token #123") are returned verbatim; the body
// is returned as blank-line-separated paragraphs for later wrapping.
func splitBody(region string) (body, footers string) {
	region = strings.TrimSpace(region)
	if region == "" {
		return "", ""
	}

	paragraphs := splitParagraphs(region)

	// A footer block is a contiguous run of trailing paragraphs whose lines all
	// look like trailers.
	i := len(paragraphs)
	for i > 0 && allFooters(paragraphs[i-1]) {
		i--
	}

	return strings.Join(paragraphs[:i], "\n\n"), strings.Join(paragraphs[i:], "\n\n")
}

// wrapBody word-wraps each blank-line-separated paragraph to WrapLimit columns.
func wrapBody(body string) string {
	if body == "" {
		return ""
	}
	paragraphs := splitParagraphs(body)
	wrapped := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		wrapped = append(wrapped, strings.Join(wrap(p, WrapLimit), "\n"))
	}
	return strings.Join(wrapped, "\n\n")
}

// wrap performs greedy word wrapping of text to at most limit columns.
func wrap(text string, limit int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := []string{words[0]}
	for _, w := range words[1:] {
		cur := lines[len(lines)-1]
		if width(cur)+1+width(w) <= limit {
			lines[len(lines)-1] = cur + " " + w
		} else {
			lines = append(lines, w)
		}
	}
	return lines
}

// splitParagraphs splits text into paragraphs on blank lines, trimming trailing
// whitespace from each line.
func splitParagraphs(text string) []string {
	var paragraphs, current []string
	flush := func() {
		if len(current) > 0 {
			paragraphs = append(paragraphs, strings.Join(current, "\n"))
			current = nil
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, strings.TrimRight(line, " \t\r"))
	}
	flush()
	return paragraphs
}

// allFooters reports whether every non-blank line in paragraph matches footerRe.
func allFooters(paragraph string) bool {
	for _, line := range strings.Split(paragraph, "\n") {
		if line = strings.TrimSpace(line); line != "" && !footerRe.MatchString(line) {
			return false
		}
	}
	return true
}

// width returns the rune count of s, the conventional measure of commit length.
func width(s string) int {
	return len([]rune(s))
}
