package repository

import (
	"os"
	"path/filepath"
	"strings"
)

// PythonResolver resolves Python import specifiers to a single file
// within the repository. v0.1 treats the repository root as the sole
// entry on sys.path (no src/ layout auto-detection, no venv/site-packages
// resolution, no PYTHONPATH); an absolute import whose top-level segment
// isn't a directory/module directly under root is recorded as external,
// exactly like a JS/TS bare specifier into node_modules or a Go stdlib
// import.
type PythonResolver struct {
	root string
}

// NewPythonResolver builds a resolver rooted at root.
func NewPythonResolver(root string) *PythonResolver {
	return &PythonResolver{root: root}
}

// Resolve returns the absolute path specifier refers to from fromFile,
// or nil if it can't be resolved within the v0.1 scope described above.
//
// For a plain `import` specifier (fromImport is false), the dotted path
// must resolve exactly to a module or package (a.b.c -> a/b/c.py or
// a/b/c/__init__.py); no fallback is attempted, since the statement
// names a module, not an attribute within one.
//
// For a `from <module> import <name>` specifier (fromImport is true,
// Specifier is "<module>.<name>"), <name> may be a submodule (resolved
// the same way as a plain import) or an attribute defined inside
// <module> (not a file on its own) — that isn't decidable without
// deeper analysis, so Resolve tries the full path first and falls back
// to <module> (dropping the last segment) if that doesn't exist.
func (r *PythonResolver) Resolve(fromFile string, specifier string, fromImport bool) []string {
	level, parts := splitSpecifier(specifier)

	base := r.root
	if level > 0 {
		base = filepath.Dir(fromFile)
		for i := 1; i < level; i++ {
			base = filepath.Dir(base)
		}
	}

	if target, ok := resolveModule(base, parts); ok {
		return []string{target}
	}

	if fromImport && len(parts) > 0 {
		if target, ok := resolveModule(base, parts[:len(parts)-1]); ok {
			return []string{target}
		}
	}

	return nil
}

// splitSpecifier separates a specifier's leading relative-import dots
// (its level: 0 for an absolute import) from its dotted module parts.
// A bare relative import with no module name (e.g. the "." in
// "from . import x") has level 1 and no parts of its own.
func splitSpecifier(specifier string) (level int, parts []string) {
	i := 0
	for i < len(specifier) && specifier[i] == '.' {
		i++
	}
	level = i
	rest := specifier[i:]
	if rest == "" {
		return level, nil
	}
	return level, strings.Split(rest, ".")
}

// resolveModule looks up parts as a module (dir/__init__.py) or as a
// .py file, relative to base.
func resolveModule(base string, parts []string) (string, bool) {
	dir := base
	if len(parts) > 0 {
		dir = filepath.Join(base, filepath.Join(parts...))
	}

	if fi, err := os.Stat(dir + ".py"); err == nil && !fi.IsDir() {
		return dir + ".py", true
	}

	initPath := filepath.Join(dir, "__init__.py")
	if fi, err := os.Stat(initPath); err == nil && !fi.IsDir() {
		return initPath, true
	}

	return "", false
}
