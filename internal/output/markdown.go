package output

import (
	"io"
	"regexp"
	"strings"
)

var (
	reBold   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reCode   = regexp.MustCompile("`([^`]+)`")
	reHeader = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reBullet = regexp.MustCompile(`(?m)^\s*[-*]\s+`)
)

// StripMarkdown removes common Markdown formatting from s (bold,
// inline code, headers, bullet markers), for rendering AI-generated
// explanations in a terminal that doesn't interpret Markdown: without
// this, a model that ignores the "plain prose only" prompt instruction
// prints its literal "**word**" asterisks instead of emphasis. Bold
// spans become ANSI bold when w supports color, plain text otherwise;
// everything else is just unwrapped.
func StripMarkdown(w io.Writer, s string) string {
	enabled := colorEnabled(w)

	s = reBold.ReplaceAllStringFunc(s, func(m string) string {
		inner := reBold.FindStringSubmatch(m)[1]
		return colorize(enabled, ansiBold, inner)
	})
	s = reCode.ReplaceAllString(s, "$1")
	s = reHeader.ReplaceAllString(s, "")
	s = reBullet.ReplaceAllString(s, "- ")

	return strings.TrimSpace(s)
}
