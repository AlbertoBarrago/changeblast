// Package c implements the analyzer.Analyzer contract for C source and
// header files.
//
// v0.1 scope (see docs/architecture.md for the full rationale):
//   - only quoted includes (`#include "foo.h"`) are extracted as raw
//     imports; angle-bracket includes (`#include <stdio.h>`) are always
//     treated as system/library headers and skipped entirely, since
//     v0.1 never traverses into them (same treatment as a JS/TS bare
//     specifier or a Go standard-library import, just decided at
//     extraction time rather than resolution time, since there is
//     nothing else a quoted-vs-angle distinction could mean)
package c

import (
	"regexp"
	"strings"

	"github.com/AlbertoBarrago/changeblast/internal/analyzer"
)

// Matches a quoted #include directive, capturing the path between the
// quotes. Angle-bracket includes are deliberately not matched here.
var reIncludeQuoted = regexp.MustCompile(`(?m)^\s*#\s*include\s*"([^"]+)"`)

// Analyzer is the C implementation of analyzer.Analyzer.
type Analyzer struct{}

// New returns a C analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// CanHandle reports whether path is a C source or header file.
func (a *Analyzer) CanHandle(path string) bool {
	return strings.HasSuffix(path, ".c") || strings.HasSuffix(path, ".h")
}

// ExtractImports scans content with the same lightweight regex-based
// approach used for the other languages, for the same reasons (see
// docs/architecture.md): comments and string/char literal contents are
// stripped before matching to eliminate the most common false-positive
// source.
func (a *Analyzer) ExtractImports(path string, content []byte) ([]analyzer.RawImport, error) {
	src := stripComments(string(content))

	seen := make(map[string]bool)
	var out []analyzer.RawImport

	for _, m := range reIncludeQuoted.FindAllStringSubmatch(src, -1) {
		specifier := m[1]
		if seen[specifier] {
			continue
		}
		seen[specifier] = true
		out = append(out, analyzer.RawImport{Specifier: specifier})
	}

	return out, nil
}

// stripComments removes "//" line comments and "/* */" block comments,
// treating double-quoted strings and single-quoted char literals as
// opaque so a comment-like or include-like sequence inside one is not
// mistaken for real source. Preprocessor conditionals (#ifdef/#endif)
// are not evaluated: an #include inside a disabled branch is still
// extracted, a documented over-approximation consistent with v0.1's
// "false positive over missed edge" stance elsewhere (see the CI
// analyzer's path-filter handling).
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
