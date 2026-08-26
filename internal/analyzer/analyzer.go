// Package analyzer defines the language-analyzer contract used by the
// repository scanner. Concrete implementations live in subpackages
// (e.g. internal/analyzer/javascript) so new languages can be added
// without touching the core pipeline.
package analyzer

// RawImport is a single import/require statement as found in source,
// before resolution against the filesystem.
type RawImport struct {
	// Specifier is the raw string used in the import/require statement,
	// e.g. "./token" or "react".
	Specifier string
	// Dynamic marks import() expressions, which v0.1 does not resolve.
	Dynamic bool
}

// Analyzer extracts raw imports from a single source file. It must not
// touch the filesystem beyond the given path/content and must not perform
// path resolution — that is the resolver's job.
type Analyzer interface {
	// CanHandle reports whether this analyzer knows how to parse path,
	// based on its extension.
	CanHandle(path string) bool
	// ExtractImports parses content and returns the raw imports found.
	ExtractImports(path string, content []byte) ([]RawImport, error)
}
