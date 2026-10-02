package coverage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// RenderText writes a human-readable coverage report to w.
func RenderText(w io.Writer, r *OverallCoverage, showGaps int) error {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "Metrics Coverage Report\n")
	fmt.Fprintf(&buf, "Root: %s\n\n", r.Root)

	fmt.Fprintf(&buf, "Overall: %d/%d functions instrumented (%.1f%%)\n\n",
		r.InstrumentedFuncs, r.TotalFuncs, r.Percent)

	if len(r.Packages) == 0 {
		buf.WriteString("No Go packages analyzed.\n")
		_, err := w.Write(buf.Bytes())
		return err
	}

	pkgs := make([]PackageCoverage, len(r.Packages))
	copy(pkgs, r.Packages)
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].Percent < pkgs[j].Percent
	})

	buf.WriteString("Per-package coverage (lowest first):\n")
	for _, pc := range pkgs {
		fmt.Fprintf(&buf, "  %-50s %3d/%-3d  %5.1f%%\n",
			pc.Dir+" ("+pc.Package+")", pc.InstrumentedFuncs, pc.TotalFuncs, pc.Percent)
	}
	buf.WriteString("\n")

	if showGaps > 0 {
		buf.WriteString("Uninstrumented functions:\n")
		for _, pc := range pkgs {
			if len(pc.UninstrumentedFuncs) == 0 {
				continue
			}
			fmt.Fprintf(&buf, "  %s (%s):\n", pc.Dir, pc.Package)
			limit := len(pc.UninstrumentedFuncs)
			if showGaps > 0 && limit > showGaps {
				limit = showGaps
			}
			for i := 0; i < limit; i++ {
				g := pc.UninstrumentedFuncs[i]
				fmt.Fprintf(&buf, "    %s:%d  %s\n", g.File, g.Line, g.Name)
			}
			if limit < len(pc.UninstrumentedFuncs) {
				fmt.Fprintf(&buf, "    ... and %d more\n", len(pc.UninstrumentedFuncs)-limit)
			}
		}
	}

	_, err := w.Write(buf.Bytes())
	return err
}

// RenderJSON writes the coverage report as indented JSON.
func RenderJSON(w io.Writer, r *OverallCoverage) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
