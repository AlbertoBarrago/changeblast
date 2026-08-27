// Package strip removes comments from C-family source code, preserving
// string literal contents so a comment-like sequence inside a string is
// not mistaken for an actual comment. A single parametrized lexer serves
// every language serval parses (Go, JS/TS, Java, C, Groovy, JSONC)
// instead of one near-identical state machine per analyzer.
package strip

import "strings"

// Opts configures the lexer for one language family.
type Opts struct {
	// Quotes lists quote characters that start an ordinary string
	// literal (backslash escapes allowed inside when Escapes is set),
	// e.g. "\"'" for Java/C, "\"" for JSONC.
	Quotes string
	// Raw lists quote characters that start a raw string literal where
	// no escaping applies (Go backtick strings). Empty for languages
	// without raw strings.
	Raw string
	// Escapes enables backslash-escape handling inside ordinary strings.
	Escapes bool
}

// Presets for the languages serval parses.
var (
	// GoQuotes covers Go: "..." with escapes and `...` raw strings.
	GoQuotes = Opts{Quotes: `"`, Raw: "`", Escapes: true}
	// JavaQuotes covers Java and C: "..." strings and '...' char
	// literals, both with escapes.
	JavaQuotes = Opts{Quotes: "\"'", Escapes: true}
	// JSQuotes covers JavaScript/TypeScript: same as Go's set (template
	// literals behave like raw strings for comment-detection purposes).
	JSQuotes = Opts{Quotes: `"`, Raw: "`", Escapes: true}
	// JSONC covers tsconfig-style JSON: only "..." strings with escapes.
	JSONC = Opts{Quotes: `"`, Escapes: true}
	// GroovyQuotes covers Jenkinsfile (Groovy): '...' and "..." with
	// escapes.
	GroovyQuotes = Opts{Quotes: "\"'", Escapes: true}
)

// Comments removes // line comments and /* */ block comments from src,
// keeping string literal contents intact per opts.
func Comments(src string, o Opts) string {
	var b strings.Builder
	b.Grow(len(src))

	const (
		stateCode = iota
		stateLine
		stateBlock
		stateString
	)
	state := stateCode
	// terminator of the current string (the opening quote character);
	// only meaningful in stateString.
	var closeQuote byte
	// raw records whether the current string was opened by a Raw quote
	// (no backslash escaping inside, e.g. Go backtick strings).
	raw := false
	escaped := false

	for i := 0; i < len(src); i++ {
		c := src[i]

		switch state {
		case stateLine:
			if c == '\n' {
				state = stateCode
				b.WriteByte(c)
			}
			continue
		case stateBlock:
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				state = stateCode
				i++
			}
			continue
		case stateString:
			b.WriteByte(c)
			if !raw && o.Escapes {
				if escaped {
					escaped = false
					continue
				}
				if c == '\\' {
					escaped = true
					continue
				}
			}
			if c == closeQuote {
				state = stateCode
			}
			continue
		}

		switch {
		case strings.IndexByte(o.Raw, c) >= 0:
			state = stateString
			closeQuote = c
			raw = true
			b.WriteByte(c)
		case strings.IndexByte(o.Quotes, c) >= 0:
			state = stateString
			closeQuote = c
			raw = false
			b.WriteByte(c)
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			state = stateLine
			i++
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			state = stateBlock
			i++
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
