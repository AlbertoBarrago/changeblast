// Package java implements the analyzer.Analyzer contract for Java
// source files.
//
// v0.1 scope (see docs/architecture.md for the full rationale):
//   - a single `package a.b.c;` declaration per file, used by
//     internal/repository's JavaResolver to derive that file's source
//     root (no repository-wide manifest like go.mod/tsconfig.json to
//     anchor resolution against, unlike Go/JS/TS)
//   - plain imports (`import a.b.C;`), type wildcard imports
//     (`import a.b.*;`), and static imports, including static wildcard
//     (`import static a.b.C.member;`, `import static a.b.C.*;`)
package java

import (
	"github.com/AlbertoBarrago/serval/internal/strip"
	"regexp"
	"strings"

	"github.com/AlbertoBarrago/serval/internal/analyzer"
)

// Matches the (single, expected) package declaration.
var rePackage = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`)

// Matches an import statement, capturing an optional "static" keyword
// and the dotted specifier (which may end in ".*").
var reImport = regexp.MustCompile(`(?m)^\s*import\s+(static\s+)?([\w.]+(?:\.\*)?)\s*;`)

// Analyzer is the Java implementation of analyzer.Analyzer.
type Analyzer struct{}

// New returns a Java analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// CanHandle reports whether path is a Java source file.
func (a *Analyzer) CanHandle(path string) bool {
	return strings.HasSuffix(path, ".java")
}

// Package returns the package declared by content, or "" for the
// default (unnamed) package. Exposed separately from ExtractImports
// since internal/repository's JavaResolver needs it to derive the
// file's source root, not just its imports.
func Package(content []byte) string {
	src := strip.Comments(string(content), strip.JavaQuotes)
	if m := rePackage.FindStringSubmatch(src); m != nil {
		return m[1]
	}
	return ""
}

// ExtractImports scans content with the same lightweight regex-based
// approach used for JS/TS, Go, and Python, for the same reasons (see
// docs/architecture.md): comments and string/char literal contents are
// stripped before matching to eliminate the most common false-positive
// source.
func (a *Analyzer) ExtractImports(path string, content []byte) ([]analyzer.RawImport, error) {
	src := strip.Comments(string(content), strip.JavaQuotes)

	seen := make(map[string]bool)
	var out []analyzer.RawImport

	for _, m := range reImport.FindAllStringSubmatch(src, -1) {
		static := m[1] != ""
		specifier := m[2]

		key := specifier
		if static {
			key = "static:" + key
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		out = append(out, analyzer.RawImport{Specifier: specifier, Static: static})
	}

	return out, nil
}
