package repository

import (
	"os"
	"path/filepath"
)

// CResolver resolves a quoted `#include "..."` specifier to a file
// relative to the including file's own directory. v0.1 has no
// awareness of compiler include paths (`-I` flags, `CPATH`, etc.) or
// any build system (Make/CMake); a #include that isn't resolvable
// relative to its including file is recorded as external/unresolved,
// same treatment as a JS/TS bare specifier into node_modules. This is
// the same "relative-only" scope JS/TS uses for its own imports.
type CResolver struct{}

// NewCResolver returns a C resolver.
func NewCResolver() *CResolver {
	return &CResolver{}
}

// Resolve resolves specifier (the quoted path from fromFile's
// #include "...") relative to fromFile's directory.
func (r *CResolver) Resolve(fromFile, specifier string) []string {
	target := filepath.Join(filepath.Dir(fromFile), specifier)
	if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
		return []string{target}
	}
	return nil
}
