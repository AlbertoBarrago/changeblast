// Package circleci implements the ci.Provider contract for CircleCI
// (`.circleci/config.yml`).
package circleci

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/AlbertoBarrago/serval/internal/ci"
)

// configFile is the pipeline config's file name, always at the
// repository root in v0.1: `config.yml`'s own `orbs:`/local includes
// are not followed, the same "single file, no cross-file resolution"
// scope GitLab CI's `include:` uses.
const configFile = ".circleci/config.yml"

// defaultWorkflowName is used when config.yml declares jobs but no
// explicit `workflows:` section — CircleCI itself runs every job as one
// implicit workflow in that case.
const defaultWorkflowName = "default"

// Provider discovers CircleCI workflows.
type Provider struct{}

// New returns a CircleCI ci.Provider.
func New() *Provider { return &Provider{} }

// Name identifies this provider.
func (p *Provider) Name() string { return "circleci" }

// Discover parses .circleci/config.yml at repoRoot and returns one
// ci.Workflow per entry under `workflows:` (falling back to a single
// "default" workflow when only `jobs:` is declared).
//
// CircleCI's base config schema has no per-path trigger filter
// equivalent to GitHub Actions' `on.push.paths` or GitLab CI's
// `changes:` — path-based filtering exists only via the separate
// "path-filtering" orb and dynamic config parameters, which would
// require evaluating orb-generated config, out of scope for v0.1. Every
// discovered workflow is therefore reported with no PathFilters
// (relevant to any change), the same stance the Jenkins provider takes
// for a stage with no `changeset` condition.
func (p *Provider) Discover(repoRoot string) ([]ci.Workflow, error) {
	path := filepath.Join(repoRoot, configFile)

	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var doc struct {
		Jobs      map[string]interface{} `yaml:"jobs"`
		Workflows map[string]interface{} `yaml:"workflows"`
	}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		// A malformed pipeline file is repository evidence, not a serval
		// failure: skip it rather than aborting the whole scan, but make
		// the omission visible.
		fmt.Fprintf(os.Stderr, "warning: ignoring malformed CircleCI config %s: %v\n", path, err)
		return nil, nil
	}

	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	var workflows []ci.Workflow
	for name, v := range doc.Workflows {
		// The scalar `version:` key sits alongside workflow names at this
		// level; only map values are actual workflow definitions.
		if _, ok := v.(map[string]interface{}); !ok {
			continue
		}
		workflows = append(workflows, ci.Workflow{Name: name, Path: rel})
	}

	if len(workflows) == 0 && len(doc.Jobs) > 0 {
		workflows = append(workflows, ci.Workflow{Name: defaultWorkflowName, Path: rel})
	}

	return workflows, nil
}
