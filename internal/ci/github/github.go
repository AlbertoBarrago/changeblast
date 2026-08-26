// Package github implements the ci.Provider contract for GitHub Actions
// workflow files under .github/workflows.
package github

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AlbertoBarrago/changeblast/internal/ci"
)

// workflowFile is the subset of a GitHub Actions workflow file's schema
// relevant to determining path relevance. Trigger keys can be a bare
// string, a list of strings, or a map (with optional "paths"/"paths-ignore"),
// so "on" is decoded generically and inspected by hand.
type workflowFile struct {
	Name string      `yaml:"name"`
	On   interface{} `yaml:"on"`
}

// Provider discovers GitHub Actions workflows.
type Provider struct{}

// New returns a GitHub Actions ci.Provider.
func New() *Provider { return &Provider{} }

// Name identifies this provider.
func (p *Provider) Name() string { return "github-actions" }

// Discover finds workflow YAML files under .github/workflows.
func (p *Provider) Discover(repoRoot string) ([]ci.Workflow, error) {
	dir := filepath.Join(repoRoot, ".github", "workflows")

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var workflows []ci.Workflow
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		path := filepath.Join(dir, e.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var wf workflowFile
		if err := yaml.Unmarshal(content, &wf); err != nil {
			// A malformed workflow file is repository evidence, not a
			// blast failure: skip it rather than aborting the whole scan.
			continue
		}

		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			rel = path
		}

		name := wf.Name
		if name == "" {
			name = e.Name()
		}

		workflows = append(workflows, ci.Workflow{
			Name:        name,
			Path:        filepath.ToSlash(rel),
			PathFilters: extractPathFilters(wf.On),
		})
	}

	return workflows, nil
}

// extractPathFilters walks the decoded "on" trigger value looking for
// "paths" filters on any trigger. A workflow with multiple triggers that
// only some of which declare "paths" is treated as unfiltered (relevant
// to any change) — narrowing that correctly would require modeling which
// trigger actually fires, which is out of scope for v0.1.
func extractPathFilters(on interface{}) []string {
	m, ok := on.(map[string]interface{})
	if !ok {
		return nil
	}

	var filters []string
	sawFilteredTrigger := false
	sawUnfilteredTrigger := false

	for _, v := range m {
		triggerMap, ok := v.(map[string]interface{})
		if !ok {
			sawUnfilteredTrigger = true
			continue
		}

		paths, ok := triggerMap["paths"]
		if !ok {
			sawUnfilteredTrigger = true
			continue
		}

		sawFilteredTrigger = true
		list, ok := paths.([]interface{})
		if !ok {
			continue
		}
		for _, p := range list {
			if s, ok := p.(string); ok {
				filters = append(filters, strings.TrimSpace(s))
			}
		}
	}

	if sawUnfilteredTrigger || !sawFilteredTrigger {
		return nil
	}
	return filters
}
