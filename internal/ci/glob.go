package ci

import (
	"regexp"
	"strings"
	"sync"
)

// matchGlob matches path against a GitHub Actions-style path filter glob:
// "**" spans whole path segments, "*" matches within a single segment.
// This is a small hand-rolled matcher rather than filepath.Match because
// filepath.Match does not support "**".
func matchGlob(pattern, path string) bool {
	return globToRegexp(pattern).MatchString(path)
}

// globCache memoizes compiled globs so workflow filters are compiled once
// per pattern instead of on every path check.
var globCache sync.Map

func globToRegexp(pattern string) *regexp.Regexp {
	if v, ok := globCache.Load(pattern); ok {
		return v.(*regexp.Regexp)
	}

	var b strings.Builder
	b.WriteString("^")

	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == '*' && i+1 < len(runes) && runes[i+1] == '*':
			i++
			// "prefix/**/" matches zero or more whole segments between
			// the separators; at pattern start ("**/x") the same holds
			// with no leading separator. The separator after "**" is
			// swallowed and becomes optional.
			if i+1 < len(runes) && runes[i+1] == '/' &&
				(strings.HasSuffix(b.String(), "/") || b.String() == "^") {
				i++
				b.WriteString("(?:.*/)?")
			} else {
				// Trailing "**" (or a non-segment-adjacent "**") matches
				// anything from here, always below a real separator.
				b.WriteString(".*")
			}
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		case strings.ContainsRune(`.+()|[]{}^$\`, c):
			b.WriteString(regexp.QuoteMeta(string(c)))
		default:
			b.WriteRune(c)
		}
	}
	b.WriteString("$")

	// Patterns are trusted input from the repository's own workflow
	// files, and are built entirely from a fixed character whitelist
	// above, so this cannot fail to compile.
	re := regexp.MustCompile(b.String())
	globCache.Store(pattern, re)
	return re
}
