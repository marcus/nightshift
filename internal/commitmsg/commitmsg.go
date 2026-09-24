// Package commitmsg parses, normalizes, and validates commit messages.
package commitmsg

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const (
	// SubjectLimit is the maximum length of a formatted commit subject.
	SubjectLimit = 72
)

var (
	ErrEmptyMessage   = errors.New("commit message is empty")
	ErrInvalidMessage = errors.New("commit message is invalid")
	ErrSubjectTooLong = errors.New("commit subject exceeds 72 characters")

	conventionalPattern = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*)(?:\(([^()\r\n]+)\))?(!)?:[ \t]*(.*)$`)
	trailerPattern      = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*(?: [A-Za-z][A-Za-z0-9-]*)*):[ \t]*(.+)$`)
)

var allowedTypes = map[string]struct{}{
	"build":    {},
	"chore":    {},
	"ci":       {},
	"docs":     {},
	"feat":     {},
	"fix":      {},
	"perf":     {},
	"refactor": {},
	"revert":   {},
	"style":    {},
	"test":     {},
}

// Trailer is a Git trailer at the end of a commit message.
type Trailer struct {
	Token string
	Value string
}

// Message is the structured form of a commit message.
type Message struct {
	Type        string
	Scope       string
	Breaking    bool
	Description string
	Body        []string
	Trailers    []Trailer
}

// Parse parses a commit message and repairs fields that can be normalized safely.
func Parse(input string) (Message, error) {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	lines := strings.Split(input, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return Message{}, ErrEmptyMessage
	}

	subject := strings.TrimSpace(lines[0])
	message := Message{}
	if matches := conventionalPattern.FindStringSubmatch(subject); matches != nil {
		message.Type = strings.ToLower(matches[1])
		if _, ok := allowedTypes[message.Type]; !ok {
			return Message{}, fmt.Errorf("%w: unsupported type %q", ErrInvalidMessage, matches[1])
		}
		message.Scope = normalizeScope(matches[2])
		message.Breaking = matches[3] == "!"
		message.Description = normalizeDescription(matches[4])
		if message.Description == "" {
			return Message{}, fmt.Errorf("%w: subject description is empty", ErrInvalidMessage)
		}
	} else {
		if strings.Contains(subject, ":") && looksLikeConventionalSubject(subject) {
			return Message{}, fmt.Errorf("%w: malformed conventional subject", ErrInvalidMessage)
		}
		message.Type = inferType(subject)
		message.Description = normalizeDescription(inferredDescription(subject, message.Type))
		if message.Description == "" {
			return Message{}, fmt.Errorf("%w: subject description is empty", ErrInvalidMessage)
		}
	}

	bodyLines := append([]string(nil), lines[1:]...)
	message.Trailers, bodyLines = parseTrailers(bodyLines)
	message.Body = normalizeBody(bodyLines)
	for _, trailer := range message.Trailers {
		if strings.EqualFold(trailer.Token, "BREAKING CHANGE") {
			message.Breaking = true
			break
		}
	}
	return message, nil
}

// Normalize returns the canonical commit message, including a final newline.
func Normalize(input string) (string, error) {
	message, err := Parse(input)
	if err != nil {
		return "", err
	}
	return Format(message)
}

// Validate checks that input already uses the canonical format.
func Validate(input string) error {
	normalized, err := Normalize(input)
	if err != nil {
		return err
	}
	if input != normalized {
		return fmt.Errorf("%w: run the commit-msg normalizer", ErrInvalidMessage)
	}
	return nil
}

// Format formats a parsed message with the canonical subject, body, and trailers.
func Format(message Message) (string, error) {
	message.Type = strings.ToLower(strings.TrimSpace(message.Type))
	if _, ok := allowedTypes[message.Type]; !ok {
		return "", fmt.Errorf("%w: unsupported type %q", ErrInvalidMessage, message.Type)
	}
	message.Scope = normalizeScope(message.Scope)
	message.Description = normalizeDescription(message.Description)
	if message.Description == "" {
		return "", fmt.Errorf("%w: subject description is empty", ErrInvalidMessage)
	}

	subject := message.Type
	if message.Scope != "" {
		subject += "(" + message.Scope + ")"
	}
	if message.Breaking {
		subject += "!"
	}
	subject += ": " + message.Description
	if runeLen(subject) > SubjectLimit {
		return "", ErrSubjectTooLong
	}

	trailers := deduplicateTrailers(message.Trailers)
	for _, trailer := range trailers {
		if trailer.Token == "" || trailer.Value == "" || !trailerPattern.MatchString(trailer.Token+": value") {
			return "", fmt.Errorf("%w: malformed trailer %q", ErrInvalidMessage, trailer.Token)
		}
	}

	sections := []string{subject}
	body := normalizeBody(message.Body)
	if len(body) > 0 {
		sections = append(sections, strings.Join(body, "\n"))
	}
	if len(trailers) > 0 {
		trailerLines := make([]string, 0, len(trailers))
		for _, trailer := range trailers {
			trailerLines = append(trailerLines, trailer.Token+": "+trailer.Value)
		}
		sections = append(sections, strings.Join(trailerLines, "\n"))
	}
	return strings.Join(sections, "\n\n") + "\n", nil
}

func looksLikeConventionalSubject(subject string) bool {
	colon := strings.IndexByte(subject, ':')
	if colon <= 0 {
		return false
	}
	prefix := strings.TrimSpace(subject[:colon])
	if prefix == "" {
		return false
	}
	return !strings.ContainsAny(prefix, " \t")
}

func normalizeScope(scope string) string {
	scope = strings.ToLower(strings.Join(strings.Fields(scope), "-"))
	return strings.Trim(scope, "-_")
}

func normalizeDescription(description string) string {
	description = strings.ToLower(strings.Join(strings.Fields(description), " "))
	description = strings.TrimLeftFunc(description, unicode.IsPunct)
	description = strings.TrimRightFunc(description, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSpace(r)
	})
	return description
}

func inferType(subject string) string {
	lower := strings.ToLower(strings.TrimSpace(subject))
	first := lower
	if fields := strings.Fields(lower); len(fields) > 0 {
		first = strings.Trim(fields[0], "[]():,;")
	}
	switch {
	case first == "test" || first == "tests" || strings.HasPrefix(lower, "add test"):
		return "test"
	case first == "fix" || first == "bug" || first == "repair" || first == "resolve" || first == "handle":
		return "fix"
	case first == "add" || first == "implement" || first == "introduce" || first == "support":
		return "feat"
	case first == "doc" || first == "docs" || first == "document" || first == "readme":
		return "docs"
	case first == "refactor" || first == "rework":
		return "refactor"
	case first == "perf" || first == "optimize":
		return "perf"
	case first == "style" || first == "format":
		return "style"
	case first == "build" || first == "compile":
		return "build"
	case first == "ci" || first == "pipeline":
		return "ci"
	case first == "revert":
		return "revert"
	default:
		return "chore"
	}
}

func inferredDescription(subject, messageType string) string {
	if messageType == "chore" {
		return subject
	}
	fields := strings.Fields(subject)
	if len(fields) < 2 {
		return subject
	}
	first := strings.ToLower(strings.Trim(fields[0], "[]():,;"))
	if messageType == "test" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(subject)), "add test") {
		return subject
	}
	if _, ok := map[string]string{
		"add": "feat", "implement": "feat", "introduce": "feat", "support": "feat",
		"fix": "fix", "bug": "fix", "repair": "fix", "resolve": "fix", "handle": "fix",
		"doc": "docs", "docs": "docs", "document": "docs", "readme": "docs",
		"refactor": "refactor", "rework": "refactor", "perf": "perf", "optimize": "perf",
		"style": "style", "format": "style", "build": "build", "compile": "build",
		"ci": "ci", "pipeline": "ci", "revert": "revert", "test": "test", "tests": "test",
	}[first]; ok {
		return strings.Join(fields[1:], " ")
	}
	return subject
}

func parseTrailers(lines []string) ([]Trailer, []string) {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	start := end
	foundTrailer := false
	separatorSeen := false
	for start > 0 {
		if trailerPattern.MatchString(lines[start-1]) {
			start--
			foundTrailer = true
			separatorSeen = false
			continue
		}
		if foundTrailer && !separatorSeen && strings.TrimSpace(lines[start-1]) == "" {
			start--
			separatorSeen = true
			continue
		}
		if foundTrailer && (strings.HasPrefix(lines[start-1], " ") || strings.HasPrefix(lines[start-1], "\t")) {
			start--
			continue
		}
		break
	}
	if start == end {
		return nil, lines
	}

	trailers := make([]Trailer, 0, end-start)
	for _, line := range lines[start:end] {
		if matches := trailerPattern.FindStringSubmatch(line); matches != nil {
			trailers = append(trailers, Trailer{Token: matches[1], Value: strings.TrimSpace(matches[2])})
		} else if len(trailers) > 0 {
			trailers[len(trailers)-1].Value += "\n" + strings.TrimSpace(line)
		}
	}
	return deduplicateTrailers(trailers), lines[:start]
}

func deduplicateTrailers(trailers []Trailer) []Trailer {
	result := make([]Trailer, 0, len(trailers))
	seen := make(map[string]struct{}, len(trailers))
	for _, trailer := range trailers {
		trailer.Token = strings.TrimSpace(trailer.Token)
		trailer.Value = strings.TrimSpace(trailer.Value)
		key := strings.ToLower(trailer.Token)
		if trailer.Token == "" || trailer.Value == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trailer)
	}
	return result
}

func normalizeBody(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil
	}

	result := make([]string, 0, len(lines))
	paragraph := make([]string, 0)
	flush := func() {
		if len(paragraph) == 0 {
			return
		}
		result = append(result, wrapParagraph(paragraph)...)
		paragraph = paragraph[:0]
	}
	for _, line := range lines {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		if strings.TrimSpace(line) == "" {
			flush()
			if len(result) > 0 && result[len(result)-1] != "" {
				result = append(result, "")
			}
			continue
		}
		paragraph = append(paragraph, line)
	}
	flush()
	for len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}
	return result
}

func wrapParagraph(lines []string) []string {
	joined := strings.Join(lines, " ")
	joined = strings.Join(strings.Fields(joined), " ")
	if joined == "" {
		return nil
	}
	prefix := ""
	content := joined
	if len(lines) == 1 {
		original := strings.TrimSpace(lines[0])
		if len(original) >= 2 && (strings.HasPrefix(original, "- ") || strings.HasPrefix(original, "* ") || strings.HasPrefix(original, "+ ")) {
			prefix, content = original[:2], original[2:]
		} else {
			for i, r := range original {
				if r == '.' || r == ')' {
					if i+1 < len(original) && original[i+1] == ' ' {
						prefix, content = original[:i+2], original[i+2:]
					}
					break
				}
			}
		}
	}
	words := strings.Fields(content)
	if len(words) == 0 {
		return nil
	}
	firstWidth := SubjectLimit - runeLen(prefix)
	if firstWidth < 1 {
		firstWidth = SubjectLimit
	}
	wrapped := wrapWords(words, firstWidth)
	if prefix == "" || len(wrapped) == 0 {
		return wrapped
	}
	wrapped[0] = prefix + wrapped[0]
	indent := strings.Repeat(" ", runeLen(prefix))
	for i := 1; i < len(wrapped); i++ {
		wrapped[i] = indent + wrapped[i]
	}
	return wrapped
}

func wrapWords(words []string, width int) []string {
	result := make([]string, 0, len(words))
	line := ""
	for _, word := range words {
		if line == "" {
			line = word
			continue
		}
		if runeLen(line)+1+runeLen(word) <= width {
			line += " " + word
			continue
		}
		result = append(result, line)
		line = word
	}
	if line != "" {
		result = append(result, line)
	}
	return result
}

func runeLen(value string) int {
	return len([]rune(value))
}
