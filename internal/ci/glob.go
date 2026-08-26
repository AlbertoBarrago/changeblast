package ci

import (
	"regexp"
	"strings"
)

// matchGlob matches path against a GitHub Actions-style path filter glob:
// "**" matches any number of path segments, "*" matches within a single
// segment. This is a small hand-rolled matcher rather than filepath.Match
// because filepath.Match does not support "**".
func matchGlob(pattern, path string) bool {
	re := globToRegexp(pattern)
	return re.MatchString(path)
}

func globToRegexp(pattern string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")

	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == '*' && i+1 < len(runes) && runes[i+1] == '*':
			b.WriteString(".*")
			i++
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
	return regexp.MustCompile(b.String())
}
