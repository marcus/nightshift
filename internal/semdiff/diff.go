// Package semdiff provides semantic analysis of git diffs.
//
// It parses unified diff output into structured hunks and applies rule-based
// heuristics — tuned for Go — to classify each hunk into a ChangeKind such as
// AddFunction, RenameSymbol, AddTest, ChangeSignature, or FormatOnly. The
// results are then aggregated into a human-readable explanation.
package semdiff

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Line represents a single line within a diff hunk.
type Line struct {
	Kind    LineKind
	Content string
}

// LineKind describes how a line participates in a diff hunk.
type LineKind int

const (
	// LineContext is an unchanged context line.
	LineContext LineKind = iota
	// LineAdd is an added line.
	LineAdd
	// LineDelete is a removed line.
	LineDelete
)

// Hunk represents a single contiguous region of a unified diff.
type Hunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []Line
}

// FileDiff represents all changes to a single file.
type FileDiff struct {
	OldPath  string
	NewPath  string
	IsRename bool
	IsNew    bool
	IsDelete bool
	Hunks    []Hunk
}

// Additions returns added lines (without the leading "+").
func (h Hunk) Additions() []string {
	out := make([]string, 0, len(h.Lines))
	for _, l := range h.Lines {
		if l.Kind == LineAdd {
			out = append(out, l.Content)
		}
	}
	return out
}

// Deletions returns deleted lines (without the leading "-").
func (h Hunk) Deletions() []string {
	out := make([]string, 0, len(h.Lines))
	for _, l := range h.Lines {
		if l.Kind == LineDelete {
			out = append(out, l.Content)
		}
	}
	return out
}

// Options controls how the diff is gathered from git.
type Options struct {
	RepoPath string
	Staged   bool
	Range    string
}

// Gather runs git diff according to opts and returns the parsed FileDiffs.
func Gather(opts Options) ([]FileDiff, error) {
	args := []string{"diff", "--no-color", "--no-ext-diff", "-U3"}
	switch {
	case opts.Staged:
		args = append(args, "--cached")
	case opts.Range != "":
		args = append(args, opts.Range)
	}

	cmd := exec.Command("git", args...)
	if opts.RepoPath != "" {
		cmd.Dir = opts.RepoPath
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running git diff: %w", err)
	}
	return Parse(string(out))
}

// Parse converts a unified diff into structured FileDiffs.
func Parse(diff string) ([]FileDiff, error) {
	var files []FileDiff
	var current *FileDiff
	var hunk *Hunk

	flushHunk := func() {
		if current != nil && hunk != nil {
			current.Hunks = append(current.Hunks, *hunk)
			hunk = nil
		}
	}
	flushFile := func() {
		flushHunk()
		if current != nil {
			files = append(files, *current)
			current = nil
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(diff))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flushFile()
			current = &FileDiff{}
			old, new := parseDiffHeader(line)
			current.OldPath = old
			current.NewPath = new
		case current == nil:
			// Ignore lines outside of any file (e.g. summary text).
			continue
		case strings.HasPrefix(line, "new file mode"):
			current.IsNew = true
		case strings.HasPrefix(line, "deleted file mode"):
			current.IsDelete = true
		case strings.HasPrefix(line, "rename from "):
			current.IsRename = true
			current.OldPath = strings.TrimPrefix(line, "rename from ")
		case strings.HasPrefix(line, "rename to "):
			current.IsRename = true
			current.NewPath = strings.TrimPrefix(line, "rename to ")
		case strings.HasPrefix(line, "--- "):
			p := strings.TrimPrefix(line, "--- ")
			if p != "/dev/null" {
				current.OldPath = stripPrefix(p)
			}
		case strings.HasPrefix(line, "+++ "):
			p := strings.TrimPrefix(line, "+++ ")
			if p != "/dev/null" {
				current.NewPath = stripPrefix(p)
			}
		case strings.HasPrefix(line, "@@"):
			flushHunk()
			h, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			hunk = &h
		case hunk != nil && len(line) > 0:
			switch line[0] {
			case '+':
				hunk.Lines = append(hunk.Lines, Line{Kind: LineAdd, Content: line[1:]})
			case '-':
				hunk.Lines = append(hunk.Lines, Line{Kind: LineDelete, Content: line[1:]})
			case ' ':
				hunk.Lines = append(hunk.Lines, Line{Kind: LineContext, Content: line[1:]})
			case '\\':
				// "\ No newline at end of file" — ignore.
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning diff: %w", err)
	}
	flushFile()
	return files, nil
}

func parseDiffHeader(line string) (oldPath, newPath string) {
	rest := strings.TrimPrefix(line, "diff --git ")
	parts := strings.Fields(rest)
	if len(parts) != 2 {
		return "", ""
	}
	return stripPrefix(parts[0]), stripPrefix(parts[1])
}

func stripPrefix(p string) string {
	if strings.HasPrefix(p, "a/") || strings.HasPrefix(p, "b/") {
		return p[2:]
	}
	return p
}

// parseHunkHeader parses lines like "@@ -10,5 +10,7 @@ funcname".
func parseHunkHeader(line string) (Hunk, error) {
	end := strings.Index(line[2:], "@@")
	if end < 0 {
		return Hunk{}, fmt.Errorf("invalid hunk header: %q", line)
	}
	inner := strings.TrimSpace(line[2 : 2+end])
	parts := strings.Fields(inner)
	if len(parts) < 2 {
		return Hunk{}, fmt.Errorf("invalid hunk header: %q", line)
	}
	oldStart, oldLines, err := parseRange(parts[0])
	if err != nil {
		return Hunk{}, err
	}
	newStart, newLines, err := parseRange(parts[1])
	if err != nil {
		return Hunk{}, err
	}
	return Hunk{
		OldStart: oldStart,
		OldLines: oldLines,
		NewStart: newStart,
		NewLines: newLines,
	}, nil
}

func parseRange(s string) (start, count int, err error) {
	s = strings.TrimLeft(s, "-+")
	count = 1
	if idx := strings.Index(s, ","); idx >= 0 {
		start, err = strconv.Atoi(s[:idx])
		if err != nil {
			return 0, 0, err
		}
		count, err = strconv.Atoi(s[idx+1:])
		if err != nil {
			return 0, 0, err
		}
		return start, count, nil
	}
	start, err = strconv.Atoi(s)
	if err != nil {
		return 0, 0, err
	}
	return start, count, nil
}
