// Package impact computes downstream impact of a target file within a
// dependency graph: files that directly import it, and the transitive
// closure beyond that.
package impact

import "github.com/AlbertoBarrago/impactline/internal/graph"

// Result is the impact analysis outcome for a single target file.
type Result struct {
	Target   string
	Direct   []string
	Indirect []string
}

// Compute returns the direct dependents (files that import target) and the
// indirect (transitive) dependents beyond the direct set.
func Compute(g *graph.Graph, target string) Result {
	direct := g.Dependents(target)

	directSet := make(map[string]bool, len(direct))
	for _, d := range direct {
		directSet[d] = true
	}

	visited := map[string]bool{target: true}
	for _, d := range direct {
		visited[d] = true
	}

	var indirect []string
	queue := append([]string{}, direct...)
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

	return Result{
		Target:   target,
		Direct:   direct,
		Indirect: indirect,
	}
}
