package risk

import (
	"path/filepath"
	"strings"
)

// DefaultCriticalPathKeywords is the v0.1 fixed heuristic for "critical
// path" matching: a path segment matches (case-insensitively) one of
// these keywords. This is a documented default, not a hidden constant —
// see docs/architecture.md for the rationale and known limitation. A
// repository can override this list via .serval.yml's
// `criticalPaths` (internal/config); callers pass the resolved keyword
// list into MatchCriticalPath rather than this package reading the
// config itself.
var DefaultCriticalPathKeywords = []string{"auth", "payment", "billing", "security"}

// MatchCriticalPath reports whether path contains a critical-path
// keyword as a path segment (case-insensitive), and returns the matched
// keyword. Matching is done per path segment, not as a substring of the
// whole path or of another segment, so e.g. "src/author/bio.ts" does not
// match "auth", while "src/auth/bio.ts" and "src/Auth/bio.ts" do.
// keywords is normally DefaultCriticalPathKeywords, or a repository's
// .serval.yml override.
func MatchCriticalPath(path string, keywords []string) (keyword string, matched bool) {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for _, seg := range segments {
		for _, kw := range keywords {
			if strings.EqualFold(seg, kw) {
				return kw, true
			}
		}
	}
	return "", false
}
