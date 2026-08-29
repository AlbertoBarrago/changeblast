// Package bitbucket implements the ci.Provider contract for Bitbucket
// Pipelines (`bitbucket-pipelines.yml`).
package bitbucket

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/AlbertoBarrago/serval/internal/ci"
)

// configFile is the pipeline config's file name, always at the
// repository root.
const configFile = "bitbucket-pipelines.yml"

// namedSections are the `pipelines:` subsections that group multiple
// named pipelines (as opposed to `default:`, a single anonymous one).
var namedSections = []string{"branches", "pull-requests", "tags", "custom"}

// Provider discovers Bitbucket Pipelines definitions.
type Provider struct{}

// New returns a Bitbucket Pipelines ci.Provider.
func New() *Provider { return &Provider{} }

// Name identifies this provider.
func (p *Provider) Name() string { return "bitbucket-pipelines" }

// Discover parses bitbucket-pipelines.yml at repoRoot and returns one
// ci.Workflow per pipeline definition: "default" for the anonymous
// pipeline (if present), and "<section>:<pattern>" for each entry under
// branches/pull-requests/tags/custom.
//
// Bitbucket Pipelines does support a `condition.changesets.includePaths`
// glob on individual steps, but resolving it correctly means walking
// each step in a pipeline's sequence and deciding whether an unfiltered
// step anywhere makes the whole pipeline unfiltered — the same
// trigger-evaluation depth the CircleCI provider's doc comment defers,
// out of scope for v0.1. Every discovered workflow is reported with no
// PathFilters (relevant to any change).
func (p *Provider) Discover(repoRoot string) ([]ci.Workflow, error) {
	path := filepath.Join(repoRoot, configFile)

	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Every pipeline section (default/branches/pull-requests/tags/custom)
	// has a different value shape, so the top-level "pipelines" key is
	// decoded loosely and inspected by hand, the same approach the GitHub
	// Actions provider uses for its polymorphic "on" trigger.
	var raw struct {
		Pipelines map[string]interface{} `yaml:"pipelines"`
	}
	if err := yaml.Unmarshal(content, &raw); err != nil {
		// A malformed pipeline file is repository evidence, not a serval
		// failure: skip it rather than aborting the whole scan, but make
		// the omission visible.
		fmt.Fprintf(os.Stderr, "warning: ignoring malformed Bitbucket Pipelines config %s: %v\n", path, err)
		return nil, nil
	}

	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	var workflows []ci.Workflow
	if _, ok := raw.Pipelines["default"]; ok {
		workflows = append(workflows, ci.Workflow{Name: "default", Path: rel})
	}

	for _, section := range namedSections {
		entries, ok := raw.Pipelines[section].(map[string]interface{})
		if !ok {
			continue
		}
		for name := range entries {
			workflows = append(workflows, ci.Workflow{
				Name: fmt.Sprintf("%s:%s", section, name),
				Path: rel,
			})
		}
	}

	return workflows, nil
}
