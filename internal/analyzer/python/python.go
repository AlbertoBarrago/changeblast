// Package python implements the analyzer.Analyzer contract for Python
// source files.
//
// v0.1 scope (see docs/architecture.md for the full rationale):
//   - plain imports: import a, import a.b.c, import a.b.c as x, and
//     comma-separated forms (import a, b.c as x)
//   - from-imports: from a.b import c, from a.b import c as d, and the
//     parenthesized multi-line form (from a.b import (c, d))
//   - relative imports: from . import x, from .pkg import y,
//     from ..pkg import z (leading dots preserved in the specifier;
//     resolution happens in internal/repository)
//   - star imports (from x import *) are not resolved: there is no
//     name to look up, and v0.1 does not flatten wildcard re-exports
package python

import (
	"regexp"
	"strings"

	"github.com/AlbertoBarrago/blast/internal/analyzer"
)

// Matches a parenthesized from-import name list; its capture groups are
// the module (with any leading dots) and everything between the parens.
var reFromBlock = regexp.MustCompile(`(?ms)^\s*from\s+([.\w]*)\s+import\s*\(([^)]*)\)`)

// Matches a single-line from-import: from <module> import <names>
var reFromSingle = regexp.MustCompile(`(?m)^\s*from\s+([.\w]*)\s+import\s+([^(\n]+)$`)

// Matches a plain import statement: import <names>
var reImport = regexp.MustCompile(`(?m)^\s*import\s+([^\n]+)$`)

// Analyzer is the Python implementation of analyzer.Analyzer.
type Analyzer struct{}

// New returns a Python analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// CanHandle reports whether path is a Python source file.
func (a *Analyzer) CanHandle(path string) bool {
	return strings.HasSuffix(path, ".py")
}

// ExtractImports scans content with the same lightweight regex-based
// approach used for JS/TS and Go, for the same reasons (see
// docs/architecture.md): comments and string contents are stripped
// before matching to eliminate the most common false-positive source.
func (a *Analyzer) ExtractImports(path string, content []byte) ([]analyzer.RawImport, error) {
	src := stripComments(string(content))

	seen := make(map[string]bool)
	var out []analyzer.RawImport

	add := func(specifier string, fromImport bool) {
		if specifier == "" || specifier == "*" {
			return
		}
		key := specifier
		if fromImport {
			key = "from:" + key
		}
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, analyzer.RawImport{Specifier: specifier, FromImport: fromImport})
	}

	for _, m := range reFromBlock.FindAllStringSubmatch(src, -1) {
		module := m[1]
		for _, name := range splitNames(m[2]) {
			if name == "*" {
				continue
			}
			add(joinModuleName(module, name), true)
		}
	}
	for _, m := range reFromSingle.FindAllStringSubmatch(src, -1) {
		module := m[1]
		for _, name := range splitNames(m[2]) {
			if name == "*" {
				continue
			}
			add(joinModuleName(module, name), true)
		}
	}
	for _, m := range reImport.FindAllStringSubmatch(src, -1) {
		for _, name := range splitNames(m[1]) {
			add(stripAlias(name), false)
		}
	}

	return out, nil
}

// splitNames splits a comma-separated import name list, stripping
// whitespace/newlines (from a parenthesized multi-line block) and any
// trailing "as alias" on each entry.
func splitNames(blob string) []string {
	var out []string
	for _, part := range strings.Split(blob, ",") {
		name := stripAlias(part)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// stripAlias trims whitespace and removes a trailing "as <alias>" from
// a single import name.
func stripAlias(part string) string {
	part = strings.TrimSpace(part)
	if idx := strings.Index(part, " as "); idx != -1 {
		part = part[:idx]
	}
	return strings.TrimSpace(part)
}

// joinModuleName joins a from-import's module (leading dots preserved)
// with a single imported name into the "<module>.<name>" specifier
// shape internal/repository's Python resolver expects.
func joinModuleName(module, name string) string {
	if module == "" {
		return name
	}
	if strings.HasSuffix(module, ".") {
		return module + name
	}
	return module + "." + name
}

// stripComments removes "#" line comments, treating single/double-quoted
// strings (including triple-quoted docstrings) as opaque so a "#" inside
// a string literal is not mistaken for a comment.
func stripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))

	for i := 0; i < len(src); i++ {
		c := src[i]

		switch c {
		case '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				b.WriteByte('\n')
			}
		case '"', '\'':
			end := skipString(src, i)
			for j := i; j < end; j++ {
				if src[j] == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
			}
			i = end - 1
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// skipString returns the index just past the end of the string literal
// starting at i (src[i] is a quote character), handling triple-quoted
// strings and backslash escapes.
func skipString(src string, i int) int {
	quote := src[i]
	triple := i+2 < len(src) && src[i+1] == quote && src[i+2] == quote
	if triple {
		i += 3
		for i+2 < len(src) {
			if src[i] == quote && src[i+1] == quote && src[i+2] == quote {
				return i + 3
			}
			if src[i] == '\\' {
				i += 2
				continue
			}
			i++
		}
		return len(src)
	}

	i++
	for i < len(src) {
		if src[i] == '\\' {
			i += 2
			continue
		}
		if src[i] == quote || src[i] == '\n' {
			return i + 1
		}
		i++
	}
	return len(src)
}
