// Command commit-lint validates a Git commit message against the Nightshift
// Conventional Commits convention. It reads the message from the file path
// passed as its first argument (the contract used by Git's commit-msg hook) and
// exits with a non-zero status if any violations are found.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/marcus/nightshift/internal/commits"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: commit-lint <message-file>")
		os.Exit(2)
	}
	path := os.Args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "commit-lint: read %s: %v\n", path, err)
		os.Exit(2)
	}

	// Strip trailing comment lines that Git appends for the user's reference.
	msg := stripComments(string(data))

	violations := commits.Validate(msg)
	if len(violations) == 0 {
		return
	}

	fmt.Fprintln(os.Stderr, "commit message does not follow the Conventional Commits convention:")
	for _, v := range violations {
		fmt.Fprintf(os.Stderr, "  - %s\n", v.Error())
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Run the normalizer to auto-fix, or edit your message to match:")
	fmt.Fprintln(os.Stderr, "  <type>(<scope>): <subject>  (imperative, <=72 chars)")
	os.Exit(1)
}

// stripComments removes Git's trailing "#" comment block from the commit
// message file and trims trailing blank lines.
func stripComments(s string) string {
	rawLines := splitLines(s)
	var keep []string
	for _, line := range rawLines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		keep = append(keep, line)
	}
	for len(keep) > 0 && keep[len(keep)-1] == "" {
		keep = keep[:len(keep)-1]
	}
	out := ""
	for i, l := range keep {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
