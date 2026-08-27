// Package azure implements the ci.Provider contract for Azure DevOps
// Pipelines (`azure-pipelines.yml`).
package azure

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AlbertoBarrago/impactline/internal/ci"
)

// configFile is the pipeline config's default file name, at the
// repository root. Azure DevOps lets a pipeline be renamed/relocated in
// the project's UI, which v0.1 has no way to discover — only the
// conventional default name is checked, the same "one default location,
// no project-specific override" scope GitLab CI's provider uses.
const configFile = "azure-pipelines.yml"

// pipelineFile is the subset of an Azure Pipelines YAML file relevant
// to determining path relevance. trigger/pr can each be `none`, a bare
// branch list, or a map with `paths.include`, so both are decoded
// generically and inspected by hand, the same approach the GitHub
// Actions provider uses for its "on" trigger value.
type pipelineFile struct {
	Name    string      `yaml:"name"`
	Trigger interface{} `yaml:"trigger"`
	PR      interface{} `yaml:"pr"`
}

// Provider discovers the Azure Pipelines config.
type Provider struct{}

// New returns an Azure DevOps ci.Provider.
func New() *Provider { return &Provider{} }

// Name identifies this provider.
func (p *Provider) Name() string { return "azure-pipelines" }

// Discover parses azure-pipelines.yml at repoRoot, returning it as a
// single ci.Workflow (unlike GitHub Actions' one-file-per-workflow or
// GitLab CI's one-job-per-workflow, an Azure Pipelines file declares
// one pipeline; its `trigger`/`pr` path filters apply to the whole
// thing, not to individual stages/jobs within it).
func (p *Provider) Discover(repoRoot string) ([]ci.Workflow, error) {
	path := filepath.Join(repoRoot, configFile)

	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var pf pipelineFile
	if err := yaml.Unmarshal(content, &pf); err != nil {
		// A malformed pipeline file is repository evidence, not an impactline
		// failure: skip it rather than aborting the whole scan.
		return nil, nil
	}

	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		rel = path
	}

	name := pf.Name
	if name == "" {
		name = configFile
	}

	return []ci.Workflow{{
		Name:        name,
		Path:        filepath.ToSlash(rel),
		PathFilters: extractPathFilters(pf.Trigger, pf.PR),
	}}, nil
}

// extractPathFilters looks for `paths.include` on trigger and pr. A
// trigger explicitly set to `none` is disabled and contributes nothing
// (neither a filter nor an "unfiltered" signal, since it never fires).
// Any other trigger shape lacking `paths.include` — including one
// omitted entirely, which defaults to running on every push with no
// path filter — makes the whole pipeline unfiltered, the same
// over-approximation stance the GitHub Actions and GitLab CI providers
// take.
func extractPathFilters(trigger, pr interface{}) []string {
	var filters []string
	sawFilter := false
	sawUnfiltered := false

	for _, t := range []interface{}{trigger, pr} {
		switch v := t.(type) {
		case nil:
			sawUnfiltered = true
		case string:
			if strings.EqualFold(v, "none") {
				continue
			}
			sawUnfiltered = true
		case map[string]interface{}:
			paths, ok := v["paths"].(map[string]interface{})
			if !ok {
				sawUnfiltered = true
				continue
			}
			include, ok := paths["include"].([]interface{})
			if !ok {
				sawUnfiltered = true
				continue
			}
			sawFilter = true
			for _, p := range include {
				if s, ok := p.(string); ok {
					filters = append(filters, strings.TrimSpace(s))
				}
			}
		default:
			// Bare branch list or any other shape without paths.
			sawUnfiltered = true
		}
	}

	if sawUnfiltered || !sawFilter {
		return nil
	}
	return filters
}
