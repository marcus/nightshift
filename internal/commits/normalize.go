// Package commits parses, normalizes, and validates Git commit messages
// following the Conventional Commits specification.
//
// The standard message shape enforced by this package is:
//
//	<type>(<scope>): <subject>
//
//	<body wrapped to <=72 columns>
//
//	<Trailers>
//
// where:
//   - type is a lowercase, known Conventional Commits type,
//   - scope is optional, lowercase, and trimmed,
//   - subject is a <=72 character imperative-mood phrase with no trailing period,
//   - the body is hard-wrapped to <=72 columns, and
//   - trailers are preserved verbatim and emitted in a stable order.
package commits

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// MaxSubjectWidth is the hard limit on the header subject length.
const MaxSubjectWidth = 72

// MaxBodyWidth is the column at which the body is hard-wrapped.
const MaxBodyWidth = 72

// AllowedTypes is the set of recognized Conventional Commits types.
var AllowedTypes = map[string]bool{
	"feat":     true,
	"fix":      true,
	"docs":     true,
	"style":    true,
	"refactor": true,
	"perf":     true,
	"test":     true,
	"build":    true,
	"ci":       true,
	"chore":    true,
	"revert":   true,
}

// AllowedScopes is the set of known scopes. A scope that is not in this set is
// not rejected — it is only lower-cased and trimmed — but listing the canonical
// scopes here lets tooling warn on typos.
var AllowedScopes = map[string]bool{
	"commits":      true,
	"config":       true,
	"scheduler":    true,
	"providers":    true,
	"agents":       true,
	"cli":          true,
	"docs":         true,
	"website":      true,
	"db":           true,
	"reporting":    true,
	"security":     true,
	"setup":        true,
	"orchestrator": true,
	"budget":       true,
}

// imperativeFixes maps common non-imperative verb forms to their imperative
// base. Used both to auto-fix the subject and to detect non-imperative mood.
var imperativeFixes = map[string]string{
	"added": "add", "adds": "add", "adding": "add",
	"fixed": "fix", "fixes": "fix", "fixing": "fix",
	"updated": "update", "updates": "update", "updating": "update",
	"removed": "remove", "removes": "remove", "removing": "remove",
	"created": "create", "creates": "create", "creating": "create",
	"changed": "change", "changes": "change", "changing": "change",
	"renamed": "rename", "renames": "rename", "renaming": "rename",
	"moved": "move", "moves": "move", "moving": "move",
	"deleted": "delete", "deletes": "delete", "deleting": "delete",
	"improved": "improve", "improves": "improve", "improving": "improve",
	"refactored": "refactor", "refactors": "refactor", "refactoring": "refactor",
	"implemented": "implement", "implements": "implement", "implementing": "implement",
	"introduced": "introduce", "introduces": "introduce", "introducing": "introduce",
	"supported": "support", "supports": "support", "supporting": "support",
	"handled": "handle", "handles": "handle", "handling": "handle",
}

var (
	// headerRe captures the optional type, scope, breaking marker, and subject
	// from the first non-blank line of a commit message.
	headerRe = regexp.MustCompile(`^\s*([A-Za-z]+)(?:\(([^)]*)\))?(!)?:\s*(.*)$`)
	// trailerRe matches a single git trailer line "Key: value" or "Key #value".
	trailerRe = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*[:#]\s*(.+)$`)
)

// Trailer is a single git trailer (e.g. "Signed-off-by: Alice <a@x>").
type Trailer struct {
	Key   string
	Value string
}

// Parsed is the structured representation of a raw commit message.
type Parsed struct {
	Type     string
	Scope    string
	Breaking bool
	Subject  string
	Body     string // body text without trailers
	Trailers []Trailer
	Raw      string // original input
}

// Parse splits a raw commit message into its header fields, body, and trailers.
// It returns an error only when no header can be identified at all.
func Parse(raw string) (*Parsed, error) {
	p := &Parsed{Raw: raw}

	// Normalize CRLF to LF and split.
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	// Locate the first non-blank line — that is the header.
	headerIdx := -1
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return p, ErrEmptyMessage
	}

	header := lines[headerIdx]
	m := headerRe.FindStringSubmatch(header)
	if m == nil {
		// No recognizable header: keep the raw line as the subject so callers can
		// still normalize/validate.
		p.Subject = strings.TrimSpace(header)
	} else {
		p.Type = m[1]
		p.Scope = strings.TrimSpace(m[2])
		p.Breaking = m[3] == "!"
		p.Subject = strings.TrimSpace(m[4])
	}

	// Remaining lines form the body + trailer block.
	rest := lines[headerIdx+1:]
	p.Body, p.Trailers = splitBodyAndTrailers(rest)

	return p, nil
}

// splitBodyAndTrailers separates trailing trailer lines from the prose body.
// A contiguous run of trailer-formatted lines at the very end of the message
// (preceded by a blank line, or immediately following the header) is treated as
// the trailer block.
func splitBodyAndTrailers(lines []string) (string, []Trailer) {
	// Drop a single leading blank line (the separator after the header).
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	// Trim trailing blank lines.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return "", nil
	}

	// Walk backward collecting contiguous trailer lines.
	var trailers []Trailer
	i := len(lines)
	for i > 0 {
		idx := i - 1
		line := lines[idx]
		if strings.TrimSpace(line) == "" {
			break
		}
		m := trailerRe.FindStringSubmatch(line)
		if m == nil {
			break
		}
		trailers = append([]Trailer{{Key: m[1], Value: strings.TrimSpace(m[2])}}, trailers...)
		i = idx
	}

	bodyLines := lines[:i]
	// Trim trailing blank lines left between body and trailers.
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[len(bodyLines)-1]) == "" {
		bodyLines = bodyLines[:len(bodyLines)-1]
	}
	return strings.TrimRight(strings.Join(bodyLines, "\n"), " \t"), trailers
}

// Normalize rewrites the parsed message into the standard format. It is
// idempotent: normalizing an already-canonical message yields the same text.
func (p *Parsed) Normalize() string {
	var b strings.Builder

	// Header.
	typ := strings.ToLower(strings.TrimSpace(p.Type))
	scope := strings.ToLower(strings.TrimSpace(p.Scope))
	b.WriteString(typ)
	if scope != "" {
		b.WriteString("(")
		b.WriteString(scope)
		b.WriteString(")")
	}
	if p.Breaking {
		b.WriteString("!")
	}
	b.WriteString(": ")
	b.WriteString(normalizeSubject(p.Subject))

	// Body.
	body := wrapBody(strings.TrimSpace(p.Body))
	if body != "" {
		b.WriteString("\n\n")
		b.WriteString(body)
	}

	// Trailers (stable order: external trailers first in original order, then
	// Nightshift-* trailers sorted alphabetically).
	if len(p.Trailers) > 0 {
		b.WriteString("\n\n")
		b.WriteString(strings.Join(formatTrailers(p.Trailers), "\n"))
	}

	return b.String()
}

// Normalize is a convenience that parses raw input and returns the canonical
// message.
func Normalize(raw string) (string, error) {
	p, err := Parse(raw)
	if err != nil {
		return "", err
	}
	return p.Normalize(), nil
}

// normalizeSubject trims, drops trailing punctuation, coerces the leading verb
// into imperative mood, lower-cases the first letter (the dominant Conventional
// Commits style), and clamps to MaxSubjectWidth.
func normalizeSubject(subject string) string {
	s := strings.TrimSpace(subject)
	s = strings.TrimRight(s, ".!?")
	s = toImperative(s)
	s = lowercaseFirst(s)
	if len([]rune(s)) > MaxSubjectWidth {
		runes := []rune(s)
		s = string(runes[:MaxSubjectWidth])
	}
	return s
}

// toImperative rewrites the leading word if it is a known non-imperative form.
func toImperative(s string) string {
	if s == "" {
		return s
	}
	firstSpace := strings.IndexAny(s, " \t")
	var first, rest string
	if firstSpace < 0 {
		first, rest = s, ""
	} else {
		first, rest = s[:firstSpace], s[firstSpace:]
	}
	lower := strings.ToLower(first)
	if base, ok := imperativeFixes[lower]; ok {
		return base + rest
	}
	return s
}

// lowercaseFirst lower-cases the first rune of s, leaving the rest untouched.
func lowercaseFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// wrapBody hard-wraps each paragraph of the body to MaxBodyWidth columns,
// preserving blank-line paragraph breaks.
func wrapBody(body string) string {
	if body == "" {
		return ""
	}
	paragraphs := strings.Split(body, "\n\n")
	var out []string
	for _, para := range paragraphs {
		// Collapse internal newlines within a paragraph into spaces so wrapping
		// produces even lines, then re-wrap.
		flat := strings.Join(strings.Fields(para), " ")
		if flat == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrap(flat, MaxBodyWidth))
	}
	return strings.Join(out, "\n\n")
}

// wrap hard-wraps a single line (no internal newlines) to width columns on word
// boundaries.
func wrap(s string, width int) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if len(line)+1+len(w) <= width {
			line += " " + w
		} else {
			lines = append(lines, line)
			line = w
		}
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n")
}

// formatTrailers renders trailers with a stable ordering: external trailers
// keep their relative input order, then Nightshift-* trailers are appended
// sorted alphabetically by key.
func formatTrailers(trailers []Trailer) []string {
	var external []string
	var nightshift []string
	for _, t := range trailers {
		line := t.Key + ": " + t.Value
		if strings.HasPrefix(t.Key, "Nightshift-") {
			nightshift = append(nightshift, line)
		} else {
			external = append(external, line)
		}
	}
	sort.Strings(nightshift)
	return append(external, nightshift...)
}
