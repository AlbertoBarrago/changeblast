package impact

import (
	"sort"

	"github.com/AlbertoBarrago/serval/internal/graph"
)

// Edge is a dependency edge between two files that are both in the
// change set: Importer imports Imported. Such edges mark files whose
// risks interact — a fix to one may have to be validated against the
// other — rather than being independent.
type Edge struct {
	Importer string `json:"importer"`
	Imported string `json:"imported"`
}

// SetResult is the set-level impact of a change set (e.g. all files in a
// diff), as opposed to the per-file result of Compute. Where Compute
// answers "what does this file reach", ComputeSet answers "what does
// this change reach as a whole", with overlaps counted once.
type SetResult struct {
	// Targets are the change-set files the result was computed over, in
	// input order.
	Targets []string
	// Direct is the deduplicated union of direct dependents across all
	// targets, excluding the targets themselves (a target imported by
	// another target shows up in InternalEdges, not here).
	Direct []string
	// Indirect is the deduplicated transitive dependents beyond Direct,
	// again excluding the targets.
	Indirect []string
	// InternalEdges lists dependency edges between two targets, sorted for
	// deterministic output.
	InternalEdges []Edge
	// SharedDependents lists files outside the set that directly import
	// at least two distinct targets — modules hit by multiple parts of
	// the same change. Sorted for deterministic output.
	SharedDependents []string
}

// ComputeSet aggregates the impact of targets against g as one change
// set. Targets unknown to the graph contribute nothing (callers filter
// those beforehand, same as for Compute).
func ComputeSet(g *graph.Graph, targets []string) SetResult {
	inSet := make(map[string]bool, len(targets))
	for _, t := range targets {
		inSet[t] = true
	}

	// Direct union and per-target importer counts, in one pass.
	direct := make(map[string]bool)
	importerCount := make(map[string]int)
	var internal []Edge
	for _, t := range targets {
		for _, dep := range g.Dependents(t) {
			if inSet[dep] {
				internal = append(internal, Edge{Importer: dep, Imported: t})
				continue
			}
			direct[dep] = true
			importerCount[dep]++
		}
	}

	// Transitive closure over the direct union, excluding the set.
	visited := make(map[string]bool, len(direct)+len(targets))
	for t := range inSet {
		visited[t] = true
	}
	var indirect []string
	queue := make([]string, 0, len(direct))
	for d := range direct {
		visited[d] = true
		queue = append(queue, d)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, dep := range g.Dependents(cur) {
			if visited[dep] {
				continue
			}
			visited[dep] = true
			indirect = append(indirect, dep)
			queue = append(queue, dep)
		}
	}
	sort.Strings(indirect)

	var shared []string
	for dep, n := range importerCount {
		if n >= 2 {
			shared = append(shared, dep)
		}
	}
	sort.Strings(shared)

	sort.Slice(internal, func(i, j int) bool {
		if internal[i].Importer != internal[j].Importer {
			return internal[i].Importer < internal[j].Importer
		}
		return internal[i].Imported < internal[j].Imported
	})

	return SetResult{
		Targets:          targets,
		Direct:           sortDirect(direct),
		Indirect:         indirect,
		InternalEdges:    internal,
		SharedDependents: shared,
	}
}

func sortDirect(direct map[string]bool) []string {
	out := make([]string, 0, len(direct))
	for d := range direct {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}
