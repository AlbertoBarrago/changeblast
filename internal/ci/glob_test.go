package ci

import "testing"

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		// "**" spans whole segments only: "src/**" must not match across
		// a missing separator boundary.
		{"src/**", "src/a/b.ts", true},
		{"src/**", "src/x.ts", true},
		{"src/**", "srcfoo/x.ts", false},
		{"**/*.ts", "src/deep/a.ts", true},
		{"**/*.ts", "a.ts", true},
		{"src/**/gen.ts", "src/a/b/gen.ts", true},
		{"src/**/gen.ts", "src/gen.ts", true},

		// "*" stays within a single segment.
		{"src/*.ts", "src/a.ts", true},
		{"src/*.ts", "src/deep/a.ts", false},

		// "?" matches exactly one non-separator character.
		{"src/a?c.ts", "src/abc.ts", true},
		{"src/a?c.ts", "src/ac.ts", false},
		{"src/a?c.ts", "src/a/c.ts", false},

		// Exact match and regex metacharacters in the pattern.
		{"src/a.ts", "src/a.ts", true},
		{"src/a.ts", "src/b.ts", false},
		{"src/a+b.ts", "src/a+b.ts", true},
		{"src/a+b.ts", "src/aab.ts", false},
	}
	for _, c := range cases {
		if got := matchGlob(c.pattern, c.path); got != c.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}
