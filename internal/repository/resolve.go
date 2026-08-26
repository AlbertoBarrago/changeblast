package repository

import (
	"os"
	"path/filepath"
)

// candidateExtensions are tried, in order, when a specifier has no
// extension of its own.
var candidateExtensions = []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}

// indexFiles are tried when a specifier resolves to a directory.
var indexFiles = []string{"index.ts", "index.tsx", "index.js", "index.jsx"}

// Resolver resolves import specifiers found in a file to absolute paths
// on disk, within the v0.1 scope: relative specifiers, tsconfig
// paths/baseUrl aliases, extension and index resolution. Bare specifiers
// are not resolved (treated as external by the caller).
type Resolver struct {
	tsconfig *TSConfig
}

// NewResolver builds a resolver for a repository root, loading tsconfig.json
// if present.
func NewResolver(root string) (*Resolver, error) {
	cfg, err := FindTSConfig(root)
	if err != nil {
		return nil, err
	}
	return &Resolver{tsconfig: cfg}, nil
}

// Resolve attempts to resolve specifier as imported from fromFile (an
// absolute path). It returns the resolved absolute file path and true on
// success.
func (r *Resolver) Resolve(fromFile, specifier string) (string, bool) {
	var basePath string

	switch {
	case isRelative(specifier):
		basePath = filepath.Join(filepath.Dir(fromFile), specifier)
	default:
		aliased, ok := r.tsconfig.ResolveAlias(specifier)
		if !ok {
			return "", false
		}
		basePath = aliased
	}

	return resolveOnDisk(basePath)
}

func resolveOnDisk(basePath string) (string, bool) {
	if fileExists(basePath) && !isDir(basePath) {
		return basePath, true
	}

	for _, ext := range candidateExtensions {
		candidate := basePath + ext
		if fileExists(candidate) {
			return candidate, true
		}
	}

	if isDir(basePath) {
		for _, idx := range indexFiles {
			candidate := filepath.Join(basePath, idx)
			if fileExists(candidate) {
				return candidate, true
			}
		}
	}

	return "", false
}

func isRelative(specifier string) bool {
	return len(specifier) > 0 && (specifier[0] == '.' || specifier[0] == '/')
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
