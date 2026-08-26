package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	dir, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	for {
		candidate := filepath.Join(dir, "tsconfig.json")
		if data, err := os.ReadFile(candidate); err == nil {
			return parseTSConfig(data, dir)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
}

func parseTSConfig(data []byte, dir string) (*TSConfig, error) {
	// tsconfig.json commonly allows comments; strip them defensively since
	// encoding/json does not support JSONC.
	clean := stripJSONComments(data)

	var raw tsconfigJSON
	if err := json.Unmarshal(clean, &raw); err != nil {
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

	for pattern, targets := range c.Paths {
		if len(targets) == 0 {
			continue
		}
		if matched, rest := matchPathPattern(pattern, specifier); matched {
			target := strings.Replace(targets[0], "*", rest, 1)
			return filepath.Join(base, target), true
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

	if !strings.HasPrefix(specifier, prefix) || !strings.HasSuffix(specifier, suffix) {
		return false, ""
	}
	rest := specifier[len(prefix) : len(specifier)-len(suffix)]
	return true, rest
}

// stripJSONComments removes // and /* */ comments from JSONC content.
func stripJSONComments(data []byte) []byte {
	src := string(data)
	var b strings.Builder
	b.Grow(len(src))

	inBlock, inLine, inString := false, false, false
	for i := 0; i < len(src); i++ {
		c := src[i]

		if inLine {
			if c == '\n' {
				inLine = false
				b.WriteByte(c)
			}
			continue
		}
		if inBlock {
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if inString {
			b.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				b.WriteByte(src[i+1])
				i++
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			b.WriteByte(c)
			continue
		}
		if c == '/' && i+1 < len(src) {
			if src[i+1] == '/' {
				inLine = true
				i++
				continue
			}
			if src[i+1] == '*' {
				inBlock = true
				i++
				continue
			}
		}
		b.WriteByte(c)
	}
	return []byte(b.String())
}
