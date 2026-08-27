// Package golang implements the analyzer.Analyzer contract for Go source
// files.
//
// v0.1 scope (see docs/architecture.md for the full rationale):
//   - single-line imports: import "fmt"
//   - grouped imports: import ( "fmt"; "os" ) including aliased
//     (foo "path"), blank (_ "path"), and dot (. "path") forms
//   - only imports whose path is the current module (from go.mod) or a
//     subpackage of it are resolved; standard library and external
//     module imports are recorded as external, not traversed
//   - a single go.mod at the repository root or nearest ancestor
//     (consistent with the tsconfig.json handling for JS/TS); Go
//     workspaces (go.work, multi-module repos) are out of scope for v0.1
package golang

import (
	"regexp"
	"strings"

	"github.com/AlbertoBarrago/impactline/internal/analyzer"
)

// Matches a fully parenthesized import block; its capture group is
// everything between the parens.
var reImportBlock = regexp.MustCompile(`(?s)import\s*\(([^)]*)\)`)

// Matches a single-line import: import [alias|.|_] "path"
var reImportSingle = regexp.MustCompile(`(?m)^\s*import\s+(?:[\w.]+\s+)?"([^"]+)"`)

// Matches a quoted import path within an import block's contents.
var reQuoted = regexp.MustCompile(`"([^"]+)"`)

// Analyzer is the Go implementation of analyzer.Analyzer.
type Analyzer struct{}

// New returns a Go analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// CanHandle reports whether path is a Go source file.
func (a *Analyzer) CanHandle(path string) bool {
	return strings.HasSuffix(path, ".go")
}

// ExtractImports scans content with the same lightweight regex-based
// approach used for JS/TS, for the same reasons (see
// docs/architecture.md): it can be fooled by an import-shaped string
// inside a string or comment that survives stripComments, but Go's
// import declarations are syntactically simple enough that this is a
// minor, documented risk rather than a real accuracy problem in practice.
func (a *Analyzer) ExtractImports(path string, content []byte) ([]analyzer.RawImport, error) {
	src := stripComments(string(content))

	seen := make(map[string]bool)
	var out []analyzer.RawImport

	add := func(spec string) {
		if seen[spec] {
			return
		}
		seen[spec] = true
		out = append(out, analyzer.RawImport{Specifier: spec})
	}

	for _, block := range reImportBlock.FindAllStringSubmatch(src, -1) {
		for _, q := range reQuoted.FindAllStringSubmatch(block[1], -1) {
			add(q[1])
		}
	}
	for _, m := range reImportSingle.FindAllStringSubmatch(src, -1) {
		add(m[1])
	}

	return out, nil
}

// stripComments removes // line comments and /* */ block comments,
// treating both double-quoted and backtick-raw string contents as
// opaque so a comment-like sequence inside a string literal is not
// mistaken for an actual comment.
func stripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))

	inBlock, inLine, inString, inRaw := false, false, false, false
	for i := 0; i < len(src); i++ {
		c := src[i]

		switch {
		case inLine:
			if c == '\n' {
				inLine = false
				b.WriteByte(c)
			}
			continue
		case inBlock:
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		case inRaw:
			b.WriteByte(c)
			if c == '`' {
				inRaw = false
			}
			continue
		case inString:
			b.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				b.WriteByte(src[i+1])
				i++
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}

		switch {
		case c == '`':
			inRaw = true
			b.WriteByte(c)
		case c == '"':
			inString = true
			b.WriteByte(c)
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			inLine = true
			i++
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			inBlock = true
			i++
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
