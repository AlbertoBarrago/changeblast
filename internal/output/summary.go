package output

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/AlbertoBarrago/changeblast/internal/risk"
)

// RenderSummary writes a compact, one-line-per-file report for multiple
// InspectResults, sorted by risk score descending, followed by a level
// count footer. Intended for analyses that can touch many files (`blast
// inspect <dir>`, `blast diff`), where the full per-file detail
// RenderInspectFull produces would be unusable at scale.
func RenderSummary(w io.Writer, root, header string, results []InspectResult) {
	if len(results) == 0 {
		fmt.Fprintf(w, "%s: no recognized modules found.\n", header)
		return
	}

	sorted := append([]InspectResult(nil), results...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Risk.Total > sorted[j].Risk.Total
	})

	fmt.Fprintf(w, "%s (%d files)\n\n", header, len(sorted))

	counts := map[risk.Level]int{}
	for _, r := range sorted {
		counts[r.Risk.Level]++

		target := r.Impact.Target
		if rel, err := filepath.Rel(root, target); err == nil {
			target = rel
		}
		downstream := len(r.Impact.Direct) + len(r.Impact.Indirect)

		// Pad the level to a fixed width before colorizing: ANSI escape
		// codes are invisible on screen but count as bytes, so padding
		// after colorizing would misalign columns.
		paddedLevel := fmt.Sprintf("%-6s", string(r.Risk.Level))
		fmt.Fprintf(w, "%s %3d/100  %-50s (%d downstream)\n",
			colorize(colorEnabled(w), levelColor(r.Risk.Level), paddedLevel), r.Risk.Total, target, downstream)
	}

	fmt.Fprintf(w, "\n%d %s, %d %s, %d %s\n",
		counts[risk.LevelHigh], "HIGH",
		counts[risk.LevelMedium], "MEDIUM",
		counts[risk.LevelLow], "LOW",
	)
}
