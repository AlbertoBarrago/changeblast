package output

import (
	"fmt"
	"io"

	"github.com/AlbertoBarrago/serval/internal/impact"
	"github.com/AlbertoBarrago/serval/internal/risk"
)

// RenderSetText writes the human-readable "Change set" section for
// `serval diff`: how the whole diff lands as one set rather than file by
// file.
func RenderSetText(w io.Writer, root string, set impact.SetResult, score risk.Score) {
	fmt.Fprintln(w, "Change set")
	fmt.Fprintf(w, "  %d changed files\n", len(set.Targets))

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Combined impact")
	fmt.Fprintf(w, "  %d direct, %d indirect modules (deduplicated across the set)\n",
		len(set.Direct), len(set.Indirect))
	for _, d := range set.Direct {
		fmt.Fprintf(w, "  %s\n", relPath(root, d))
	}
	for _, d := range set.Indirect {
		fmt.Fprintf(w, "  %s (indirect)\n", relPath(root, d))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Interactions within the change set")
	if len(set.InternalEdges) == 0 {
		fmt.Fprintln(w, "  (none — changed files do not depend on each other)")
	}
	for _, e := range set.InternalEdges {
		fmt.Fprintf(w, "  %s imports %s\n", relPath(root, e.Importer), relPath(root, e.Imported))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Shared dependents")
	if len(set.SharedDependents) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, s := range set.SharedDependents {
		fmt.Fprintf(w, "  %s\n", relPath(root, s))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Change-set risk")
	fmt.Fprintf(w, "  %s: %d/100%s\n", colorizeLevel(w, score.Level), score.Total, forcedSuffix(score))
	for _, e := range score.Breakdown {
		fmt.Fprintf(w, "  +%-3d %s\n", e.Points, e.Reason)
	}
}

// SetJSON is the JSON-serializable shape of a set-level diff result.
type SetJSON struct {
	Targets          []string      `json:"targets"`
	Direct           []string      `json:"direct"`
	Indirect         []string      `json:"indirect"`
	InternalEdges    []impact.Edge `json:"internalEdges"`
	SharedDependents []string      `json:"sharedDependents"`
	Risk             risk.Score    `json:"risk"`
}

// ToSetJSON converts an impact.SetResult and its set-level risk score to
// their JSON representation, with paths made relative to root.
func ToSetJSON(root string, set impact.SetResult, score risk.Score) SetJSON {
	relAll := func(paths []string) []string {
		out := make([]string, len(paths))
		for i, p := range paths {
			out[i] = relPath(root, p)
		}
		return out
	}

	edges := make([]impact.Edge, len(set.InternalEdges))
	for i, e := range set.InternalEdges {
		edges[i] = impact.Edge{
			Importer: relPath(root, e.Importer),
			Imported: relPath(root, e.Imported),
		}
	}

	return SetJSON{
		Targets:          relAll(set.Targets),
		Direct:           relAll(set.Direct),
		Indirect:         relAll(set.Indirect),
		InternalEdges:    edges,
		SharedDependents: relAll(set.SharedDependents),
		Risk:             score,
	}
}
