// Package output renders analysis results for human (text) and
// machine (JSON) consumption. It has no knowledge of how results were
// computed — it only formats them.
package output

import (
	"fmt"
	"io"

	"github.com/AlbertoBarrago/serval/internal/impact"
)

// RenderInspectText writes a human-readable rendering of an inspect result
// to w. root is used to render paths relative to the repository for
// readability.
func RenderInspectText(w io.Writer, root string, result impact.Result) {

	fmt.Fprintln(w, "Target")
	fmt.Fprintf(w, "  %s\n\n", relPath(root, result.Target))

	fmt.Fprintln(w, "Direct impact")
	if len(result.Direct) == 0 {
		fmt.Fprintln(w, "  (none found)")
	}
	for _, d := range result.Direct {
		fmt.Fprintf(w, "  %s\n", relPath(root, d))
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Indirect impact")
	if len(result.Indirect) == 0 {
		fmt.Fprintln(w, "  (none found)")
	}
	for _, d := range result.Indirect {
		fmt.Fprintf(w, "  %s\n", relPath(root, d))
	}
}
