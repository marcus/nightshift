package semdiff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FileExplanation describes the semantic changes of a single file.
type FileExplanation struct {
	Path    string       `json:"path"`
	Kinds   []ChangeKind `json:"kinds"`
	Added   int          `json:"added"`
	Removed int          `json:"removed"`
	Note    string       `json:"note,omitempty"`
}

// Explanation aggregates classifications across all files in a diff.
type Explanation struct {
	Summary    string             `json:"summary"`
	Categories map[ChangeKind]int `json:"categories"`
	Files      []FileExplanation  `json:"files"`
}

// Explain produces a high-level Explanation from a slice of FileDiffs.
func Explain(files []FileDiff) Explanation {
	exp := Explanation{Categories: map[ChangeKind]int{}}
	for _, f := range files {
		kinds := ClassifyFile(f)
		fe := FileExplanation{
			Path:    pickPath(f),
			Kinds:   kinds,
			Added:   countLines(f, LineAdd),
			Removed: countLines(f, LineDelete),
		}
		switch {
		case f.IsNew:
			fe.Note = "new file"
		case f.IsDelete:
			fe.Note = "deleted file"
		case f.IsRename:
			fe.Note = fmt.Sprintf("renamed from %s", f.OldPath)
		}
		exp.Files = append(exp.Files, fe)
		for _, k := range kinds {
			exp.Categories[k]++
		}
	}
	exp.Summary = summarize(exp)
	return exp
}

func pickPath(f FileDiff) string {
	if f.NewPath != "" {
		return f.NewPath
	}
	return f.OldPath
}

func countLines(f FileDiff, kind LineKind) int {
	n := 0
	for _, h := range f.Hunks {
		for _, l := range h.Lines {
			if l.Kind == kind {
				n++
			}
		}
	}
	return n
}

func summarize(exp Explanation) string {
	if len(exp.Files) == 0 {
		return "No changes detected."
	}
	keys := make([]ChangeKind, 0, len(exp.Categories))
	for k := range exp.Categories {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if exp.Categories[keys[i]] != exp.Categories[keys[j]] {
			return exp.Categories[keys[i]] > exp.Categories[keys[j]]
		}
		return string(keys[i]) < string(keys[j])
	})
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s×%d", k, exp.Categories[k]))
	}
	return fmt.Sprintf("%d file(s) changed: %s", len(exp.Files), strings.Join(parts, ", "))
}

// Render returns a human-readable plain-text explanation.
func (e Explanation) Render() string {
	var b strings.Builder
	fmt.Fprintln(&b, e.Summary)
	for _, f := range e.Files {
		fmt.Fprintf(&b, "\n%s (+%d/-%d)", f.Path, f.Added, f.Removed)
		if f.Note != "" {
			fmt.Fprintf(&b, " [%s]", f.Note)
		}
		fmt.Fprintln(&b)
		for _, k := range f.Kinds {
			fmt.Fprintf(&b, "  - %s\n", k)
		}
	}
	return b.String()
}

// RenderJSON serializes the explanation as indented JSON.
func (e Explanation) RenderJSON() (string, error) {
	out, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
