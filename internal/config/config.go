// Package config loads the optional .changeblast.yml repository
// configuration file, which lets a repository override v0.1's fixed
// defaults for critical-path keywords and the Git history window.
//
// The file is looked up at the repository root only (unlike
// tsconfig.json/go.mod, this is project-level configuration, not
// something that varies per subpackage), and is entirely optional: a
// missing file yields a zero-value Config, not an error.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName is the config file name, expected at the repository root.
const FileName = ".changeblast.yml"

// Config is the set of overridable v0.1 defaults. Every field is
// optional; a zero value means "use the built-in default" and is
// resolved by the CriticalPathsOr/HistoryWindowOr helpers below rather
// than by callers checking for zero values themselves.
type Config struct {
	// CriticalPaths overrides risk.DefaultCriticalPathKeywords when
	// non-empty.
	CriticalPaths []string `yaml:"criticalPaths"`
	// HistoryWindow overrides git.DefaultWindow when either field is set.
	HistoryWindow struct {
		Days       int `yaml:"days"`
		MaxCommits int `yaml:"maxCommits"`
	} `yaml:"historyWindow"`
}

// Load reads .changeblast.yml from root. A missing file is not an
// error: it returns a zero-value Config, so callers fall back to
// built-in defaults uniformly whether the file is absent or present but
// empty.
func Load(root string) (Config, error) {
	path := filepath.Join(root, FileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("failed to read %s: %w", FileName, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse %s: %w", FileName, err)
	}

	return cfg, nil
}

// CriticalPathsOr returns c.CriticalPaths if non-empty, otherwise
// def.
func (c Config) CriticalPathsOr(def []string) []string {
	if len(c.CriticalPaths) > 0 {
		return c.CriticalPaths
	}
	return def
}

// HistoryWindowDaysOr returns the configured history window days if
// set (> 0), otherwise def.
func (c Config) HistoryWindowDaysOr(def int) int {
	if c.HistoryWindow.Days > 0 {
		return c.HistoryWindow.Days
	}
	return def
}

// HistoryWindowMaxCommitsOr returns the configured history window max
// commit count if set (> 0), otherwise def.
func (c Config) HistoryWindowMaxCommitsOr(def int) int {
	if c.HistoryWindow.MaxCommits > 0 {
		return c.HistoryWindow.MaxCommits
	}
	return def
}
