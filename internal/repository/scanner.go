package repository

import (
	"os"
	"path/filepath"

	"github.com/AlbertoBarrago/changeblast/internal/analyzer"
	"github.com/AlbertoBarrago/changeblast/internal/analyzer/javascript"
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

// Scanner walks a repository and builds a dependency graph using the
// registered language analyzers.
type Scanner struct {
	root      string
	resolver  *Resolver
	analyzers []analyzer.Analyzer
}

// NewScanner builds a scanner rooted at root with the default set of
// analyzers (currently JS/TS only).
func NewScanner(root string) (*Scanner, error) {
	resolver, err := NewResolver(root)
	if err != nil {
		return nil, err
	}
	return &Scanner{
		root:      root,
		resolver:  resolver,
		analyzers: []analyzer.Analyzer{javascript.New()},
	}, nil
}

// Scan walks the repository tree and returns the resulting dependency
// graph. Unresolved and external imports are recorded as graph nodes with
// the raw specifier so callers can distinguish them, but are not traversed
// further.
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

		a := s.analyzerFor(path)
		if a == nil {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		g.AddNode(path)

		imports, err := a.ExtractImports(path, content)
		if err != nil {
			return err
		}

		for _, imp := range imports {
			s.addImportEdge(g, path, imp)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (s *Scanner) addImportEdge(g *graph.Graph, fromFile string, imp analyzer.RawImport) {
	if imp.Dynamic {
		// Recorded as evidence, not traversed (v0.1 scope).
		return
	}
	if !javascript.IsRelative(imp.Specifier) {
		// Bare specifier: external dependency, not traversed into
		// node_modules (v0.1 scope), unless a tsconfig alias matches.
		if resolved, ok := s.resolver.Resolve(fromFile, imp.Specifier); ok {
			g.AddEdge(fromFile, resolved)
			return
		}
		return
	}

	resolved, ok := s.resolver.Resolve(fromFile, imp.Specifier)
	if !ok {
		return
	}
	g.AddEdge(fromFile, resolved)
}

func (s *Scanner) analyzerFor(path string) analyzer.Analyzer {
	for _, a := range s.analyzers {
		if a.CanHandle(path) {
			return a
		}
	}
	return nil
}
