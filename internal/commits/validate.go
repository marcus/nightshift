package commits

import (
	"fmt"
	"strings"
)

// Violation code constants.
const (
	CodeMissingHeader   = "missing_header"
	CodeMissingType     = "missing_type"
	CodeUnknownType     = "unknown_type"
	CodeOversizeSubject = "oversize_subject"
	CodeNonImperative   = "non_imperative"
	CodeTrailingPeriod  = "trailing_period"
	CodeUnwrappedBody   = "unwrapped_body"
	CodeBadTrailer      = "bad_trailer"
)

// Violation is a single structured validation problem.
type Violation struct {
	Code    string // machine-readable code (Code* constant)
	Field   string // logical field the problem relates to
	Message string // human-readable explanation
}

// Error renders a violation as a single line.
func (v Violation) Error() string {
	return fmt.Sprintf("%s: %s", v.Code, v.Message)
}

// Validate reports structured violations for a raw commit message. An empty
// (nil) slice means the message already conforms to the standard format.
func Validate(raw string) []Violation {
	p, err := Parse(raw)
	if err != nil {
		return []Violation{{
			Code:    CodeMissingHeader,
			Field:   "header",
			Message: "commit message is empty",
		}}
	}

	var vs []Violation

	// Type checks.
	t := strings.ToLower(strings.TrimSpace(p.Type))
	if t == "" {
		vs = append(vs, Violation{
			Code:    CodeMissingType,
			Field:   "type",
			Message: "commit header is missing a type (expected '<type>: <subject>')",
		})
	} else if !AllowedTypes[t] {
		vs = append(vs, Violation{
			Code:  CodeUnknownType,
			Field: "type",
			Message: fmt.Sprintf(
				"unknown type %q; allowed: %s",
				p.Type, joinKeys(AllowedTypes),
			),
		})
	}

	// Subject checks.
	subject := strings.TrimSpace(p.Subject)
	if subject == "" && t != "" {
		vs = append(vs, Violation{
			Code:    CodeMissingType,
			Field:   "subject",
			Message: "commit subject is empty",
		})
	}
	if subject != "" {
		if r := []rune(subject); len(r) > MaxSubjectWidth {
			vs = append(vs, Violation{
				Code:  CodeOversizeSubject,
				Field: "subject",
				Message: fmt.Sprintf(
					"subject is %d characters, exceeds limit of %d",
					len(r), MaxSubjectWidth,
				),
			})
		}
		if reason, bad := imperativeMood(subject); bad {
			vs = append(vs, Violation{
				Code:    CodeNonImperative,
				Field:   "subject",
				Message: reason,
			})
		}
		if endsWithPeriod(subject) {
			vs = append(vs, Violation{
				Code:    CodeTrailingPeriod,
				Field:   "subject",
				Message: "subject must not end with a period",
			})
		}
	}

	// Body wrapping checks.
	if body := strings.TrimSpace(p.Body); body != "" {
		for _, line := range strings.Split(body, "\n") {
			if len([]rune(line)) > MaxBodyWidth {
				vs = append(vs, Violation{
					Code:  CodeUnwrappedBody,
					Field: "body",
					Message: fmt.Sprintf(
						"body line is %d characters, exceeds %d: %q",
						len([]rune(line)), MaxBodyWidth, truncate(line, 40),
					),
				})
				break
			}
		}
	}

	// Trailer format checks.
	for _, tr := range p.Trailers {
		if strings.TrimSpace(tr.Key) == "" || strings.TrimSpace(tr.Value) == "" {
			vs = append(vs, Violation{
				Code:    CodeBadTrailer,
				Field:   "trailers",
				Message: fmt.Sprintf("malformed trailer: %s: %s", tr.Key, tr.Value),
			})
		}
	}

	return vs
}

// imperativeMood reports whether the subject's leading word looks non-imperative
// (past tense, gerund, or third-person singular). Returns a human reason when
// the mood is bad.
func imperativeMood(subject string) (string, bool) {
	first := strings.Fields(subject)
	if len(first) == 0 {
		return "", false
	}
	word := strings.ToLower(strings.TrimRight(first[0], ".!?"))
	if _, bad := imperativeFixes[word]; bad {
		return fmt.Sprintf("subject verb %q is not imperative (use the base form)", word), true
	}
	// Heuristic: words ending in "-ed"/"-ing" are usually non-imperative, unless
	// they are known to be valid base verbs. Skip short words to avoid false
	// positives on words like "red", "fed".
	if len(word) > 4 && strings.HasSuffix(word, "ed") {
		if _, common := commonBaseVerbs[word]; !common {
			return fmt.Sprintf("subject verb %q looks like past tense; use imperative mood", word), true
		}
	}
	if len(word) > 5 && strings.HasSuffix(word, "ing") {
		return fmt.Sprintf("subject verb %q looks like a gerund; use imperative mood", word), true
	}
	return "", false
}

// commonBaseVerbs are verbs that legitimately end in "ed" in their base form and
// must not be flagged as past tense.
var commonBaseVerbs = map[string]bool{
	"embed": true,
	"shed":  true,
	"red":   true,
	"bred":  true,
	"speed": true,
	"feed":  true,
	"breed": true,
	"pled":  true,
}

func endsWithPeriod(s string) bool {
	t := strings.TrimRight(s, " \t")
	if t == "" {
		return false
	}
	r := []rune(t)
	last := r[len(r)-1]
	return last == '.' || last == '。'
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func joinKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return strings.Join(keys, ", ")
}

// sortStrings sorts s in place using a simple insertion sort to avoid pulling
// in the sort package here (it lives in normalize.go's import set already, but
// keeping validate.go dependencies minimal makes the file self-contained).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// Conforms is a convenience predicate for callers that only need a boolean.
func Conforms(raw string) bool {
	return len(Validate(raw)) == 0
}
