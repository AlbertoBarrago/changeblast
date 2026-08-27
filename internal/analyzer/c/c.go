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
	"github.com/AlbertoBarrago/serval/internal/strip"
	"regexp"
	"strings"

	"github.com/AlbertoBarrago/serval/internal/analyzer"
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
	src := strip.Comments(string(content), strip.JavaQuotes)

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
