// Package ci discovers CI workflow definitions and determines which ones
// are relevant to a given set of changed files. v0.1 supports GitHub
// Actions only; the Provider interface exists so GitLab CI, Azure
// DevOps, and Jenkins can be added later without touching callers.
package ci

// Workflow is a single CI workflow definition, provider-agnostic.
type Workflow struct {
	// Name is the workflow's declared name, or its file name if unset.
	Name string
	// Path is the workflow file path, relative to the repository root.
	Path string
	// PathFilters are the trigger path globs that scope when this
	// workflow runs (e.g. GitHub Actions `on.push.paths`). An empty list
	// means the workflow has no path filter and is considered relevant
	// to any change.
	PathFilters []string
}

// Provider discovers workflows for a specific CI system.
type Provider interface {
	// Name identifies the provider (e.g. "github-actions").
	Name() string
	// Discover finds workflow definitions under repoRoot.
	Discover(repoRoot string) ([]Workflow, error)
}

// Relevant returns the subset of workflows whose path filters match at
// least one of changedFiles (paths relative to the repository root), or
// that have no path filters at all.
func Relevant(workflows []Workflow, changedFiles []string) []Workflow {
	var out []Workflow
	for _, wf := range workflows {
		if len(wf.PathFilters) == 0 {
			out = append(out, wf)
			continue
		}
		for _, f := range changedFiles {
			if matchesAny(wf.PathFilters, f) {
				out = append(out, wf)
				break
			}
		}
	}
	return out
}

func matchesAny(patterns []string, path string) bool {
	for _, p := range patterns {
		if matchGlob(p, path) {
			return true
		}
	}
	return false
}
