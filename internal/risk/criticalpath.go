package risk

import (
	"path/filepath"
	"strings"
)

// DefaultCriticalPathKeywords is the v0.1 fixed heuristic for "critical
// path" matching: a path segment matches (case-insensitively) one of
// these keywords. This is a documented default, not a hidden constant —
// see docs/architecture.md for the rationale and known limitation (it
// will false-positive/false-negative on domain-specific critical code
// until .changeblast.yml's `criticalPaths` override lands, which is not
// implemented in v0.1).
var DefaultCriticalPathKeywords = []string{"auth", "payment", "billing", "security"}

// MatchCriticalPath reports whether path contains a critical-path
// keyword as a path segment (case-insensitive), and returns the matched
// keyword. Matching is done per path segment, not as a substring of the
// whole path, so e.g. "src/author/bio.ts" does not match "auth".
func MatchCriticalPath(path string) (keyword string, matched bool) {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for _, seg := range segments {
		lower := strings.ToLower(seg)
		for _, kw := range DefaultCriticalPathKeywords {
			if strings.Contains(lower, kw) {
				return kw, true
			}
		}
	}
	return "", false
}
