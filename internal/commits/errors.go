package commits

import "errors"

// Sentinel errors returned by Parse.
var (
	// ErrEmptyMessage is returned when the input contains no non-blank lines.
	ErrEmptyMessage = errors.New("commits: empty commit message")
)
