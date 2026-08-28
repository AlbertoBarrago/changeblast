package risk

import (
	"path/filepath"
	"strings"
)

// DefaultHighRiskPaths is the v0.2 built-in floor list: zones that are
// "touch with care" in nearly any repository regardless of computed
// score, independent of the application domain. A repository can
// override this list via .serval.yml's `highRiskPaths` (internal/config);
// callers pass the resolved pattern list into MatchHighRiskPath rather
// than this package reading the config itself.
var DefaultHighRiskPaths = []string{
	"**/migrations/**",
	"**/*.env*",
	"**/secrets/**",
	"**/.github/workflows/**",
	"**/infra/**",
	"**/terraform/**",
}

// MatchHighRiskPath reports whether path matches one of patterns, and
// returns the matched pattern. Unlike MatchCriticalPath (segment keyword
// matching), this is glob matching against the whole path, since
// high-risk paths name specific locations rather than generic keywords.
//
// Patterns are matched segment by segment (split on "/"): a "**"
// segment matches zero or more path segments, any other segment is
// matched against the corresponding path segment with filepath.Match
// (so "*"/"?" work within a single segment, same semantics as
// gitignore-style globs). The stdlib's filepath.Match has no native
// "**" support, hence this small matcher.
func MatchHighRiskPath(path string, patterns []string) (pattern string, matched bool) {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range patterns {
		if matchSegments(strings.Split(p, "/"), segments) {
			return p, true
		}
	}
	return "", false
}

func matchSegments(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}
	if pattern[0] == "**" {
		if matchSegments(pattern[1:], path) {
			return true
		}
		if len(path) == 0 {
			return false
		}
		return matchSegments(pattern, path[1:])
	}
	if len(path) == 0 {
		return false
	}
	if ok, err := filepath.Match(pattern[0], path[0]); err != nil || !ok {
		return false
	}
	return matchSegments(pattern[1:], path[1:])
}
