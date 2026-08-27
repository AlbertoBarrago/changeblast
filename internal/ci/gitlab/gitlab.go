// Package gitlab implements the ci.Provider contract for GitLab CI
// (`.gitlab-ci.yml`).
package gitlab

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AlbertoBarrago/changeblast/internal/ci"
)

// configFile is the pipeline config's file name, always at the
// repository root in v0.1: `include:` (splitting config across
// multiple files) is not followed, the same "single file, no
// cross-file resolution" scope JS/TS module resolution uses for
// tsconfig.json.
const configFile = ".gitlab-ci.yml"

// reservedKeys are top-level keywords that configure the pipeline as a
// whole rather than declaring a job, so they're never treated as one.
var reservedKeys = map[string]bool{
	"stages":         true,
	"variables":      true,
	"default":        true,
	"workflow":       true,
	"include":        true,
	"image":          true,
	"services":       true,
	"before_script":  true,
	"after_script":   true,
	"cache":          true,
	"pages":          true,
	"workflow_rules": true,
}

// Provider discovers GitLab CI jobs.
type Provider struct{}

// New returns a GitLab CI ci.Provider.
func New() *Provider { return &Provider{} }

// Name identifies this provider.
func (p *Provider) Name() string { return "gitlab-ci" }

// Discover parses .gitlab-ci.yml at repoRoot and returns one
// ci.Workflow per job. Jobs whose name starts with "." are GitLab's
// convention for hidden/template jobs (meant to be reused via
// `extends:`, not run directly) and are skipped, same as they would be
// by GitLab itself absent an `extends:` reference — v0.1 does not
// follow `extends:` to merge a template's rules into the job that
// references it.
func (p *Provider) Discover(repoRoot string) ([]ci.Workflow, error) {
	path := filepath.Join(repoRoot, configFile)

	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		// A malformed pipeline file is repository evidence, not a blast
		// failure: skip it rather than aborting the whole scan.
		return nil, nil
	}

	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	var workflows []ci.Workflow
	for name, v := range doc {
		if reservedKeys[name] || strings.HasPrefix(name, ".") {
			continue
		}
		job, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		workflows = append(workflows, ci.Workflow{
			Name:        name,
			Path:        rel,
			PathFilters: extractPathFilters(job),
		})
	}

	return workflows, nil
}

// extractPathFilters looks for `changes:` path globs on job, via either
// the modern `rules:` list or the older `only:` map form. If job has no
// rules/only at all, or any rule lacks a `changes:` key, it's treated as
// unfiltered (relevant to every change) — evaluating `if:`/`when:`
// conditions to know which rule actually applies is out of scope for
// v0.1, the same "false relevant is safer than a missed one" stance the
// GitHub Actions provider takes for a trigger without `paths`.
func extractPathFilters(job map[string]interface{}) []string {
	if rules, ok := job["rules"].([]interface{}); ok {
		return filtersFromRules(rules)
	}
	if only, ok := job["only"].(map[string]interface{}); ok {
		if changes, ok := only["changes"]; ok {
			return stringList(changes)
		}
	}
	return nil
}

func filtersFromRules(rules []interface{}) []string {
	var filters []string
	for _, r := range rules {
		rule, ok := r.(map[string]interface{})
		if !ok {
			return nil
		}
		changes, ok := rule["changes"]
		if !ok {
			return nil
		}
		filters = append(filters, stringList(changes)...)
	}
	return filters
}

func stringList(v interface{}) []string {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
