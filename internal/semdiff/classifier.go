package semdiff

import (
	"regexp"
	"sort"
	"strings"
)

// ChangeKind is a high-level label for a semantic change.
type ChangeKind string

const (
	ChangeAddFunction      ChangeKind = "AddFunction"
	ChangeRemoveFunction   ChangeKind = "RemoveFunction"
	ChangeChangeSignature  ChangeKind = "ChangeSignature"
	ChangeRenameSymbol     ChangeKind = "RenameSymbol"
	ChangeAddTest          ChangeKind = "AddTest"
	ChangeModifyTest       ChangeKind = "ModifyTest"
	ChangeAddImport        ChangeKind = "AddImport"
	ChangeRemoveImport     ChangeKind = "RemoveImport"
	ChangeAddErrorHandling ChangeKind = "AddErrorHandling"
	ChangeModifyComment    ChangeKind = "ModifyComment"
	ChangeFormatOnly       ChangeKind = "FormatOnly"
	ChangeAddType          ChangeKind = "AddType"
	ChangeOther            ChangeKind = "Other"
)

var (
	reFuncDecl   = regexp.MustCompile(`^\s*func(?:\s+\([^)]*\))?\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`)
	reTypeDecl   = regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+`)
	reImportLine = regexp.MustCompile(`^\s*(?:import\s+)?"[^"]+"\s*$`)
	reCommentSL  = regexp.MustCompile(`^\s*//`)
	reErrCheck   = regexp.MustCompile(`^\s*if\s+err\s*(!=|==)\s*nil\s*\{?`)
)

type funcInfo struct {
	name string
	args string
}

// ClassifyFile applies heuristics to a single FileDiff and returns the set of
// detected ChangeKinds, in priority order with duplicates removed.
func ClassifyFile(f FileDiff) []ChangeKind {
	seen := map[ChangeKind]bool{}
	var order []ChangeKind
	add := func(k ChangeKind) {
		if !seen[k] {
			seen[k] = true
			order = append(order, k)
		}
	}

	isGo := strings.HasSuffix(f.NewPath, ".go") || strings.HasSuffix(f.OldPath, ".go")
	isTest := strings.HasSuffix(f.NewPath, "_test.go") || strings.HasSuffix(f.OldPath, "_test.go")

	for _, h := range f.Hunks {
		classifyHunk(h, isGo, isTest, add)
	}

	if len(order) == 0 {
		add(ChangeOther)
	}
	return order
}

func classifyHunk(h Hunk, isGo, isTest bool, add func(ChangeKind)) {
	adds := h.Additions()
	dels := h.Deletions()

	// FormatOnly: same non-whitespace content on add/delete sides.
	if len(adds) > 0 && len(adds) == len(dels) && allMatchIgnoringWhitespace(adds, dels) {
		add(ChangeFormatOnly)
		return
	}

	// Comment-only changes.
	if onlyComments(adds) && onlyComments(dels) && (len(adds)+len(dels)) > 0 {
		add(ChangeModifyComment)
		return
	}

	if isGo {
		addedFuncs := collectFuncs(adds)
		removedFuncs := collectFuncs(dels)

		// Renames: same arg list, different name on the same hunk.
		if len(addedFuncs) > 0 && len(removedFuncs) > 0 {
			for _, a := range addedFuncs {
				for _, r := range removedFuncs {
					switch {
					case a.name == r.name && normalizeArgs(a.args) != normalizeArgs(r.args):
						add(ChangeChangeSignature)
					case a.name != r.name && normalizeArgs(a.args) == normalizeArgs(r.args):
						add(ChangeRenameSymbol)
					}
				}
			}
		}

		switch {
		case len(addedFuncs) > len(removedFuncs):
			if isTest {
				for _, f := range addedFuncs {
					if strings.HasPrefix(f.name, "Test") || strings.HasPrefix(f.name, "Benchmark") || strings.HasPrefix(f.name, "Example") {
						add(ChangeAddTest)
					} else {
						add(ChangeAddFunction)
					}
				}
			} else {
				add(ChangeAddFunction)
			}
		case len(removedFuncs) > len(addedFuncs):
			add(ChangeRemoveFunction)
		}

		// Imports.
		if hasImport(adds) {
			add(ChangeAddImport)
		}
		if hasImport(dels) {
			add(ChangeRemoveImport)
		}

		// Type declarations.
		for _, line := range adds {
			if reTypeDecl.MatchString(line) {
				add(ChangeAddType)
			}
		}

		// Error handling.
		for _, line := range adds {
			if reErrCheck.MatchString(line) {
				add(ChangeAddErrorHandling)
			}
		}

		// Test modifications (changes inside Test funcs already covered by other rules).
		if isTest && len(addedFuncs) == 0 && len(removedFuncs) == 0 && (len(adds) > 0 || len(dels) > 0) {
			add(ChangeModifyTest)
		}
	}
}

func collectFuncs(lines []string) []funcInfo {
	var out []funcInfo
	for _, line := range lines {
		m := reFuncDecl.FindStringSubmatch(line)
		if m != nil {
			out = append(out, funcInfo{name: m[1], args: m[2]})
		}
	}
	return out
}

func hasImport(lines []string) bool {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if reImportLine.MatchString(line) {
			return true
		}
		if strings.HasPrefix(trimmed, "import ") {
			return true
		}
	}
	return false
}

func onlyComments(lines []string) bool {
	any := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		any = true
		if !reCommentSL.MatchString(line) && !strings.HasPrefix(trimmed, "/*") && !strings.HasPrefix(trimmed, "*") {
			return false
		}
	}
	return any
}

func allMatchIgnoringWhitespace(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	na := make([]string, len(a))
	nb := make([]string, len(b))
	for i := range a {
		na[i] = collapseWS(a[i])
		nb[i] = collapseWS(b[i])
	}
	sort.Strings(na)
	sort.Strings(nb)
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}

func collapseWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func normalizeArgs(args string) string {
	return collapseWS(args)
}
