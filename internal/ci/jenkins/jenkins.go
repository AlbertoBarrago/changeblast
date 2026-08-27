// Package jenkins implements the ci.Provider contract for Jenkins
// declarative pipelines (`Jenkinsfile`).
//
// Unlike the other CI providers, a Jenkinsfile is Groovy, not YAML, so
// there is no structured parser to lean on: path relevance is extracted
// with the same regex-based, comment-stripping approach v0.1 uses for
// its source-language analyzers, not gopkg.in/yaml.v3.
package jenkins

import (
	"github.com/AlbertoBarrago/serval/internal/strip"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AlbertoBarrago/serval/internal/ci"
)

// configFile is the pipeline file's conventional name at the repository
// root. Jenkins can be configured to look elsewhere (e.g. a
// multibranch pipeline's custom script path); v0.1 only checks the
// default, same "one default location" scope the Azure DevOps provider
// uses.
const configFile = "Jenkinsfile"

// Matches a declarative pipeline stage declaration: stage('Name') {
var reStage = regexp.MustCompile(`stage\s*\(\s*['"]([^'"]+)['"]\s*\)\s*\{`)

// Matches a `changeset` when-condition, with or without parens:
// changeset "pattern" / changeset(pattern: "pattern")
var reChangeset = regexp.MustCompile(`changeset\s*\(?[^'")]*['"]([^'"]+)['"]`)

// Provider discovers stages in a declarative Jenkins pipeline.
type Provider struct{}

// New returns a Jenkins ci.Provider.
func New() *Provider { return &Provider{} }

// Name identifies this provider.
func (p *Provider) Name() string { return "jenkins" }

// Discover parses Jenkinsfile at repoRoot's declarative pipeline
// stages, one ci.Workflow per stage.
//
// A stage's body is approximated as the text between its opening `{`
// and the start of the next `stage(...)` declaration (or end of file),
// not by balancing braces — a real Groovy parser is out of scope for
// v0.1, the same regex-based tradeoff made for source-language import
// extraction (see docs/architecture.md). This can misattribute a
// changeset condition that lives inside a deeply nested block to the
// wrong stage in unusual pipeline shapes; documented, not silently
// accepted.
//
// Scripted pipelines (raw Groovy without a `pipeline { stages { ... } }`
// structure) have no stages to find and so discover nothing — not an
// error, just zero workflows.
func (p *Provider) Discover(repoRoot string) ([]ci.Workflow, error) {
	path := filepath.Join(repoRoot, configFile)

	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	src := strip.Comments(string(content), strip.GroovyQuotes)

	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	stages := reStage.FindAllStringSubmatchIndex(src, -1)

	var workflows []ci.Workflow
	for i, s := range stages {
		name := src[s[2]:s[3]]

		bodyEnd := len(src)
		if i+1 < len(stages) {
			bodyEnd = stages[i+1][0]
		}
		body := src[s[1]:bodyEnd]

		var filters []string
		for _, m := range reChangeset.FindAllStringSubmatch(body, -1) {
			filters = append(filters, strings.TrimSpace(m[1]))
		}

		workflows = append(workflows, ci.Workflow{
			Name:        name,
			Path:        rel,
			PathFilters: filters,
		})
	}

	return workflows, nil
}
