package repository

import (
	"os"
	"path/filepath"
	"strings"
)

// GoResolver resolves Go import paths to the files of the local package
// they refer to. Unlike JS/TS (one specifier -> at most one file), a Go
// import targets a package directory that may contain several files, so
// Resolve returns every matching file rather than a single path.
type GoResolver struct {
	module *GoModule
}

// NewGoResolver builds a resolver for the given module (nil if the
// repository has no go.mod — every import is then treated as
// unresolved/external).
func NewGoResolver(module *GoModule) *GoResolver {
	return &GoResolver{module: module}
}

// Resolve returns the absolute paths of every non-test .go file in the
// package that specifier refers to, excluding fromFile itself. It
// returns nil for standard library imports, external module
// dependencies, or any specifier outside the current module — v0.1 does
// not traverse into the module cache or GOPATH, matching the "external
// dependency, not traversed" treatment used for JS/TS bare imports.
func (r *GoResolver) Resolve(fromFile, specifier string) []string {
	if r.module == nil {
		return nil
	}

	var pkgDir string
	switch {
	case specifier == r.module.Path:
		pkgDir = r.module.dir
	case strings.HasPrefix(specifier, r.module.Path+"/"):
		rest := strings.TrimPrefix(specifier, r.module.Path+"/")
		pkgDir = filepath.Join(r.module.dir, filepath.FromSlash(rest))
	default:
		return nil
	}

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		target := filepath.Join(pkgDir, e.Name())
		if target == fromFile {
			continue
		}
		files = append(files, target)
	}
	return files
}
