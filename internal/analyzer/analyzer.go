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
	// JS/TS-specific; ignored by other languages' resolvers.
	Dynamic bool
	// FromImport marks a Python `from <module> import <name>` statement,
	// with Specifier set to "<module>.<name>" (dots preserved for a
	// relative import, e.g. ".pkg.name"). <name> may be either a
	// submodule or an attribute of <module>, which isn't decidable
	// without the filesystem, so the resolver tries the full path first
	// and falls back to the path without the last segment. Python-
	// specific; ignored by other languages' resolvers.
	FromImport bool
	// Static marks a Java `import static a.b.C.member;` statement, with
	// Specifier set to the full "a.b.C.member" (or "a.b.C.*" for a
	// static wildcard). Unlike FromImport, this is unambiguous: the
	// resolver always drops the last segment to get at the class
	// a.b.C, since Java's import syntax makes that structurally
	// certain rather than a guess. Java-specific; ignored by other
	// languages' resolvers.
	Static bool
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
