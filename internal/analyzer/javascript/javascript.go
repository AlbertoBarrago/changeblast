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
	src = stripComments(src)

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

// stripComments removes // line comments and /* */ block comments to
// reduce false-positive import matches. It is intentionally simple and
// does not understand strings containing comment-like sequences.
func stripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))

	inBlock := false
	inLine := false
	for i := 0; i < len(src); i++ {
		c := src[i]

		if inLine {
			if c == '\n' {
				inLine = false
				b.WriteByte(c)
			}
			continue
		}
		if inBlock {
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) {
			if src[i+1] == '/' {
				inLine = true
				i++
				continue
			}
			if src[i+1] == '*' {
				inBlock = true
				i++
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

func extOf(path string) string {
	idx := strings.LastIndexByte(path, '.')
	if idx < 0 {
		return ""
	}
	return path[idx:]
}
