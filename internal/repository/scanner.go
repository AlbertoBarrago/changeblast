package repository

import (
	"os"
	"path/filepath"

	"github.com/AlbertoBarrago/changeblast/internal/analyzer"
	"github.com/AlbertoBarrago/changeblast/internal/analyzer/golang"
	"github.com/AlbertoBarrago/changeblast/internal/analyzer/javascript"
	"github.com/AlbertoBarrago/changeblast/internal/analyzer/python"
	"github.com/AlbertoBarrago/changeblast/internal/graph"
)

// excludedDirs are never scanned, regardless of language.
var excludedDirs = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	"dist":         {},
	"build":        {},
	"coverage":     {},
	"vendor":       {},
}

// languageSupport pairs a language's Analyzer with the function that
// resolves its raw imports to zero or more absolute file paths. Keeping
// these together is what lets Scanner itself stay free of any
// language-specific branching beyond selecting which pair handles a
// given file.
type languageSupport struct {
	analyzer analyzer.Analyzer
	resolve  func(fromFile string, imp analyzer.RawImport) []string
}

// Scanner walks a repository and builds a dependency graph using the
// registered language analyzers.
type Scanner struct {
	root      string
	languages []languageSupport
}

// NewScanner builds a scanner rooted at root with the default set of
// analyzers (currently JS/TS and Go).
func NewScanner(root string) (*Scanner, error) {
	jsResolver, err := NewResolver(root)
	if err != nil {
		return nil, err
	}

	goModule, err := FindGoModule(root)
	if err != nil {
		return nil, err
	}
	goResolver := NewGoResolver(goModule)
	pyResolver := NewPythonResolver(root)

	return &Scanner{
		root: root,
		languages: []languageSupport{
			{
				analyzer: javascript.New(),
				resolve:  resolveJS(jsResolver),
			},
			{
				analyzer: golang.New(),
				resolve: func(fromFile string, imp analyzer.RawImport) []string {
					return goResolver.Resolve(fromFile, imp.Specifier)
				},
			},
			{
				analyzer: python.New(),
				resolve: func(fromFile string, imp analyzer.RawImport) []string {
					return pyResolver.Resolve(fromFile, imp.Specifier, imp.FromImport)
				},
			},
		},
	}, nil
}

// resolveJS adapts the JS/TS Resolver (single target, tsconfig-aware) to
// the languageSupport.resolve shape.
func resolveJS(r *Resolver) func(string, analyzer.RawImport) []string {
	return func(fromFile string, imp analyzer.RawImport) []string {
		if imp.Dynamic {
			// Recorded as evidence, not traversed (v0.1 scope).
			return nil
		}
		if resolved, ok := r.Resolve(fromFile, imp.Specifier); ok {
			return []string{resolved}
		}
		return nil
	}
}

// Scan walks the repository tree and returns the resulting dependency
// graph. Unresolved and external imports are not traversed further.
func (s *Scanner) Scan() (*graph.Graph, error) {
	g := graph.New()

	err := filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, excluded := excludedDirs[d.Name()]; excluded && path != s.root {
				return filepath.SkipDir
			}
			return nil
		}

		lang := s.languageFor(path)
		if lang == nil {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		g.AddNode(path)

		imports, err := lang.analyzer.ExtractImports(path, content)
		if err != nil {
			return err
		}

		for _, imp := range imports {
			for _, target := range lang.resolve(path, imp) {
				g.AddEdge(path, target)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (s *Scanner) languageFor(path string) *languageSupport {
	for i := range s.languages {
		if s.languages[i].analyzer.CanHandle(path) {
			return &s.languages[i]
		}
	}
	return nil
}
