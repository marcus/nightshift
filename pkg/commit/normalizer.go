package commit

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// CommitType represents valid conventional commit types.
type CommitType string

const (
	TypeFeat     CommitType = "feat"
	TypeFix      CommitType = "fix"
	TypeDocs     CommitType = "docs"
	TypeStyle    CommitType = "style"
	TypeRefactor CommitType = "refactor"
	TypePerf     CommitType = "perf"
	TypeTest     CommitType = "test"
	TypeBuild    CommitType = "build"
	TypeCI       CommitType = "ci"
	TypeChore    CommitType = "chore"
	TypeRevert   CommitType = "revert"
)

const (
	// MaxSubjectLength is the maximum length for a commit subject line.
	MaxSubjectLength = 72
	// MaxBodyLineLength is the maximum length for body lines.
	MaxBodyLineLength = 100
)

var (
	// validTypes contains all valid commit types.
	validTypes = map[string]CommitType{
		"feat":     TypeFeat,
		"fix":      TypeFix,
		"docs":     TypeDocs,
		"style":    TypeStyle,
		"refactor": TypeRefactor,
		"perf":     TypePerf,
		"test":     TypeTest,
		"build":    TypeBuild,
		"ci":       TypeCI,
		"chore":    TypeChore,
		"revert":   TypeRevert,
	}

	// commitPattern matches conventional commit format: type(scope): subject
	// Also supports breaking change indicator with !
	commitPattern = regexp.MustCompile(`^([a-z]+)(\(([^)]+)\))?!?: (.+)$`)
)

// CommitMessage represents a parsed commit message.
type CommitMessage struct {
	Type          CommitType
	Scope         string
	Subject       string
	Body          string
	Footer        string
	BreakingChange bool
}

// ValidationError represents a commit message validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Parse parses a commit message string into a CommitMessage struct.
func Parse(message string) (*CommitMessage, error) {
	lines := strings.Split(message, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return nil, &ValidationError{Field: "message", Message: "commit message cannot be empty"}
	}

	subject := strings.TrimSpace(lines[0])
	matches := commitPattern.FindStringSubmatch(subject)

	if matches == nil {
		return nil, &ValidationError{
			Field:   "subject",
			Message: "subject must follow format: type(scope): description",
		}
	}

	typeStr := matches[1]
	scope := matches[3]
	description := matches[4]

	// Validate type
	commitType, valid := validTypes[typeStr]
	if !valid {
		return nil, &ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("invalid type '%s', must be one of: %s", typeStr, getValidTypesString()),
		}
	}

	// Parse body and footer
	var body, footer string
	var breakingChange bool

	if len(lines) > 1 {
		// Skip the blank line after subject if present
		bodyStart := 1
		if bodyStart < len(lines) && strings.TrimSpace(lines[bodyStart]) == "" {
			bodyStart = 2
		}

		// Find footer start (lines starting with BREAKING CHANGE: or other tokens)
		footerStart := -1
		for i := bodyStart; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "BREAKING CHANGE:") ||
				strings.HasPrefix(line, "Fixes:") ||
				strings.HasPrefix(line, "Closes:") ||
				strings.HasPrefix(line, "Refs:") {
				footerStart = i
				break
			}
		}

		switch {
		case footerStart > bodyStart:
			body = strings.Join(lines[bodyStart:footerStart], "\n")
			footer = strings.Join(lines[footerStart:], "\n")
		case footerStart == bodyStart:
			footer = strings.Join(lines[footerStart:], "\n")
		case bodyStart < len(lines):
			body = strings.Join(lines[bodyStart:], "\n")
		}

		body = strings.TrimSpace(body)
		footer = strings.TrimSpace(footer)

		if strings.Contains(footer, "BREAKING CHANGE:") || strings.Contains(subject, "!") {
			breakingChange = true
		}
	}

	return &CommitMessage{
		Type:           commitType,
		Scope:          scope,
		Subject:        description,
		Body:           body,
		Footer:         footer,
		BreakingChange: breakingChange,
	}, nil
}

// Normalize takes a commit message and returns a normalized version
// following conventional commits format.
func Normalize(message string) (string, error) {
	cm, err := Parse(message)
	if err != nil {
		return "", err
	}

	return cm.Format(), nil
}

// Format formats the CommitMessage into a conventional commit string.
func (cm *CommitMessage) Format() string {
	var parts []string

	// Format subject line
	subject := formatSubject(cm.Type, cm.Scope, cm.Subject)
	parts = append(parts, subject)

	// Add blank line before body if body exists
	if cm.Body != "" {
		parts = append(parts, "")
		parts = append(parts, wrapText(cm.Body, MaxBodyLineLength))
	}

	// Add blank line before footer if footer exists
	if cm.Footer != "" {
		parts = append(parts, "")
		parts = append(parts, cm.Footer)
	}

	return strings.Join(parts, "\n")
}

// Validate validates the commit message against conventional commits rules.
func (cm *CommitMessage) Validate() error {
	// Validate type
	if _, valid := validTypes[string(cm.Type)]; !valid {
		return &ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("invalid type '%s'", cm.Type),
		}
	}

	// Validate subject
	if cm.Subject == "" {
		return &ValidationError{Field: "subject", Message: "subject cannot be empty"}
	}

	// Check subject capitalization (should not start with capital letter)
	if unicode.IsUpper(rune(cm.Subject[0])) {
		return &ValidationError{
			Field:   "subject",
			Message: "subject should start with lowercase letter",
		}
	}

	// Check subject doesn't end with period
	if strings.HasSuffix(cm.Subject, ".") {
		return &ValidationError{
			Field:   "subject",
			Message: "subject should not end with period",
		}
	}

	// Check subject length
	subjectLine := formatSubject(cm.Type, cm.Scope, cm.Subject)
	if len(subjectLine) > MaxSubjectLength {
		return &ValidationError{
			Field:   "subject",
			Message: fmt.Sprintf("subject line too long (%d > %d)", len(subjectLine), MaxSubjectLength),
		}
	}

	return nil
}

// formatSubject formats the subject line with type and optional scope.
func formatSubject(commitType CommitType, scope, subject string) string {
	if scope != "" {
		return fmt.Sprintf("%s(%s): %s", commitType, scope, subject)
	}
	return fmt.Sprintf("%s: %s", commitType, subject)
}

// wrapText wraps text to the specified line length.
func wrapText(text string, maxLength int) string {
	if text == "" {
		return text
	}

	var result []string
	paragraphs := strings.Split(text, "\n\n")

	for i, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}

		// Handle already-formatted lines (like lists)
		if strings.HasPrefix(paragraph, "- ") || strings.HasPrefix(paragraph, "* ") {
			result = append(result, paragraph)
			if i < len(paragraphs)-1 {
				result = append(result, "")
			}
			continue
		}

		words := strings.Fields(paragraph)
		if len(words) == 0 {
			continue
		}

		var line string
		for _, word := range words {
			switch {
			case line == "":
				line = word
			case len(line)+1+len(word) <= maxLength:
				line = line + " " + word
			default:
				result = append(result, line)
				line = word
			}
		}
		if line != "" {
			result = append(result, line)
		}

		// Add blank line between paragraphs
		if i < len(paragraphs)-1 {
			result = append(result, "")
		}
	}

	return strings.Join(result, "\n")
}

// getValidTypesString returns a comma-separated list of valid types.
func getValidTypesString() string {
	types := make([]string, 0, len(validTypes))
	for t := range validTypes {
		types = append(types, t)
	}
	return strings.Join(types, ", ")
}

// IsValidType checks if a string is a valid commit type.
func IsValidType(t string) bool {
	_, valid := validTypes[t]
	return valid
}
