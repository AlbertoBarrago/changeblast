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
	src := stripComments(string(content))
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
	src := stripComments(string(content))

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

// stripComments removes "//" line comments and "/* */" block comments,
// treating double-quoted strings and single-quoted char literals as
// opaque so a comment-like or import-like sequence inside one is not
// mistaken for real source. Java 15+ text blocks ("""...""") are not
// specially handled: a rare, documented gap consistent with v0.1's
// regex-based approach elsewhere.
func stripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))

	inBlock, inLine, inString, inChar := false, false, false, false
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
		case inChar:
			b.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				b.WriteByte(src[i+1])
				i++
				continue
			}
			if c == '\'' {
				inChar = false
			}
			continue
		}

		switch {
		case c == '"':
			inString = true
			b.WriteByte(c)
		case c == '\'':
			inChar = true
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
