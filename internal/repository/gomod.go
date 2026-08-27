package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GoModule holds the module path declared by go.mod, used to resolve
// local Go imports (see GoResolver).
type GoModule struct {
	// Path is the module path from the "module" directive, e.g.
	// "github.com/AlbertoBarrago/serval".
	Path string
	// dir is the directory containing go.mod (the module root); local
	// import paths are resolved relative to it.
	dir string
}

// FindGoModule looks for go.mod starting at root and searching upward
// through ancestor directories, mirroring FindTSConfig's behavior for
// tsconfig.json. Returns nil if none is found. Go workspaces (go.work,
// multi-module repositories) are out of scope for v0.1 — only the
// nearest single go.mod is considered.
func FindGoModule(root string) (*GoModule, error) {
	dir, err := findUpward(root, "go.mod")
	if err != nil || dir == "" {
		return nil, err
	}
	candidate := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(candidate)
	if err != nil {
		return nil, err
	}
	path, err := parseModulePath(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", candidate, err)
	}
	return &GoModule{Path: path, dir: dir}, nil
}

// parseModulePath extracts the module path from a go.mod file's
// "module <path>" directive, which by Go spec must appear as the first
// non-comment, non-blank line.
func parseModulePath(data []byte) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			// Strip a trailing line comment; module paths contain no spaces,
			// so cutting at the first space is safe.
			rest = strings.TrimSpace(rest)
			if i := strings.IndexAny(rest, " \t"); i >= 0 {
				rest = strings.TrimSpace(rest[:i])
			}
			return rest, nil
		}
		return "", fmt.Errorf("expected \"module\" directive, found: %q", line)
	}
	return "", fmt.Errorf("no module directive found")
}
