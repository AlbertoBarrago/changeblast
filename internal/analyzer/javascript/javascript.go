// Package javascript implements the analyzer.Analyzer contract for
// JavaScript and TypeScript source files (including JSX/TSX).
//
// v0.1 scope (see docs/architecture.md for the full rationale):
//   - relative ESM imports/exports: import ... from './x', export ... from '../y'
//   - CommonJS require('./x')
//   - dynamic import() is recorded as Dynamic, not resolved
//   - bare specifiers (no leading '.' or '/') are treated as external and
//     are not resolved into node_modules
package javascript

import (
	"github.com/AlbertoBarrago/serval/internal/strip"
	"regexp"
	"strings"

	"github.com/AlbertoBarrago/serval/internal/analyzer"
)

// Extensions handled by this analyzer.
var extensions = map[string]struct{}{
	".js":  {},
	".jsx": {},
	".ts":  {},
	".tsx": {},
	".mjs": {},
	".cjs": {},
}

// Matches ESM static imports/exports with a from-clause, and bare
// `import './x'` side-effect imports.
var reESMFrom = regexp.MustCompile(`(?m)^\s*(?:import|export)(?:[^'"]*?\bfrom\b)?\s*['"]([^'"]+)['"]`)

// Matches CommonJS require('x') calls.
var reRequire = regexp.MustCompile(`\brequire\(\s*['"]([^'"]+)['"]\s*\)`)

// Matches dynamic import('x') expressions.
var reDynamicImport = regexp.MustCompile(`\bimport\(\s*['"]([^'"]+)['"]\s*\)`)

// Analyzer is the JS/TS implementation of analyzer.Analyzer.
type Analyzer struct{}

// New returns a JS/TS analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// CanHandle reports whether path has a JS/TS-family extension.
func (a *Analyzer) CanHandle(path string) bool {
	ext := extOf(path)
	_, ok := extensions[ext]
	return ok
}

// ExtractImports scans content with a lightweight regex-based approach
// rather than a full AST parser. This trades perfect accuracy (it can be
// fooled by specifiers inside string/template literals or comments that
// happen to match the shape of an import) for zero extra dependencies and
// speed; see docs/architecture.md for the tradeoff discussion.
func (a *Analyzer) ExtractImports(path string, content []byte) ([]analyzer.RawImport, error) {
	src := string(content)
	src = strip.Comments(src, strip.JSQuotes)

	seen := make(map[string]bool)
	var out []analyzer.RawImport

	add := func(spec string, dynamic bool) {
		key := spec
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, analyzer.RawImport{Specifier: spec, Dynamic: dynamic})
	}

	for _, m := range reESMFrom.FindAllStringSubmatch(src, -1) {
		add(m[1], false)
	}
	for _, m := range reRequire.FindAllStringSubmatch(src, -1) {
		add(m[1], false)
	}
	for _, m := range reDynamicImport.FindAllStringSubmatch(src, -1) {
		add(m[1], true)
	}

	return out, nil
}

func extOf(path string) string {
	idx := strings.LastIndexByte(path, '.')
	if idx < 0 {
		return ""
	}
	return path[idx:]
}
