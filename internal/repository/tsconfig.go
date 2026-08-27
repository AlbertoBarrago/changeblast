package repository

import (
	"encoding/json"
	"github.com/AlbertoBarrago/serval/internal/strip"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TSConfig holds the subset of tsconfig.json relevant to module resolution.
type TSConfig struct {
	BaseURL string
	Paths   map[string][]string
	// dir is the directory containing the tsconfig.json file; baseUrl and
	// paths are resolved relative to it.
	dir string
}

type tsconfigJSON struct {
	CompilerOptions struct {
		BaseURL string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

// FindTSConfig looks for tsconfig.json starting at root and searching
// upward through ancestor directories, per v0.1 scope (single tsconfig.json
// at repo root or nearest ancestor). Returns nil if none is found.
func FindTSConfig(root string) (*TSConfig, error) {
	dir, err := findUpward(root, "tsconfig.json")
	if err != nil || dir == "" {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "tsconfig.json"))
	if err != nil {
		return nil, err
	}
	return parseTSConfig(data, dir)
}

// findUpward walks up from root looking for filename and returns the
// directory containing it, or "" when no ancestor has one. Shared by the
// manifest lookups (tsconfig.json, go.mod), which follow the same
// nearest-ancestor rule.
func findUpward(root, filename string) (string, error) {
	dir, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, filename)); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func parseTSConfig(data []byte, dir string) (*TSConfig, error) {
	// tsconfig.json commonly allows comments; strip them defensively since
	// encoding/json does not support JSONC.
	clean := strip.Comments(string(data), strip.JSONC)

	var raw tsconfigJSON
	if err := json.Unmarshal([]byte(clean), &raw); err != nil {
		return nil, err
	}

	return &TSConfig{
		BaseURL: raw.CompilerOptions.BaseURL,
		Paths:   raw.CompilerOptions.Paths,
		dir:     dir,
	}, nil
}

// ResolveAlias attempts to resolve specifier against tsconfig paths/baseUrl.
// It returns the resolved absolute base path (without extension resolution)
// and true if a paths/baseUrl rule matched.
func (c *TSConfig) ResolveAlias(specifier string) (string, bool) {
	if c == nil {
		return "", false
	}

	base := c.dir
	if c.BaseURL != "" {
		base = filepath.Join(c.dir, c.BaseURL)
	}

	// Iterate patterns in a deterministic order (longest prefix first, i.e.
	// most specific pattern wins) so overlapping rules resolve consistently
	// across runs; map iteration order would otherwise be random.
	patterns := make([]string, 0, len(c.Paths))
	for pattern := range c.Paths {
		patterns = append(patterns, pattern)
	}
	sort.Slice(patterns, func(i, j int) bool {
		pi, pj := patterns[i], patterns[j]
		li, lj := len(pi)-strings.Count(pi, "*"), len(pj)-strings.Count(pj, "*")
		if li != lj {
			return li > lj
		}
		return pi < pj
	})

	for _, pattern := range patterns {
		targets := c.Paths[pattern]
		if len(targets) == 0 {
			continue
		}
		if matched, rest := matchPathPattern(pattern, specifier); matched {
			// TypeScript tries each substitution target in order and uses
			// the first that exists on disk; fall back to the last one.
			for _, t := range targets {
				candidate := filepath.Join(base, strings.Replace(t, "*", rest, 1))
				if _, err := os.Stat(candidate); err == nil {
					return candidate, true
				}
			}
			return filepath.Join(base, strings.Replace(targets[len(targets)-1], "*", rest, 1)), true
		}
	}

	// baseUrl alone allows non-relative specifiers to resolve from it.
	if c.BaseURL != "" && !strings.HasPrefix(specifier, ".") {
		return filepath.Join(base, specifier), true
	}

	return "", false
}

// matchPathPattern matches a tsconfig "paths" pattern (which may contain a
// single '*' wildcard) against a specifier, returning the wildcard capture.
func matchPathPattern(pattern, specifier string) (bool, string) {
	if !strings.Contains(pattern, "*") {
		return pattern == specifier, ""
	}

	idx := strings.IndexByte(pattern, '*')
	prefix, suffix := pattern[:idx], pattern[idx+1:]

	// Guard against a negative slice bound when the specifier is shorter
	// than prefix+suffix combined (e.g. "@x/*/y" vs "@x/z").
	if len(specifier) < len(prefix)+len(suffix) {
		return false, ""
	}
	if !strings.HasPrefix(specifier, prefix) || !strings.HasSuffix(specifier, suffix) {
		return false, ""
	}
	rest := specifier[len(prefix) : len(specifier)-len(suffix)]
	return true, rest
}
