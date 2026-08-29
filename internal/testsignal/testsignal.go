// Package testsignal answers one question for the risk engine: does a
// changed file have a companion test file, by that language's own
// naming convention? It never parses source — just filesystem
// existence checks against a per-language naming pattern, the same
// "documented approximation over a real analysis" tradeoff the rest of
// v0.1 makes (see docs/architecture.md).
package testsignal

import (
	"os"
	"path/filepath"
	"strings"
)

// HasCorrelatedTest reports whether targetPath (relative to root) has a
// companion test file. A file that is itself already a test (by its own
// language's convention) is reported as true — it can't be "missing its
// own test". A language with no reliable, universal test-naming
// convention (C) always reports true: the signal simply doesn't apply,
// which keeps its absence from being silently misread as "no test
// exists" for a language v0.1 has no way to check.
func HasCorrelatedTest(root, targetPath string) bool {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var candidates []string
	switch ext {
	case ".go":
		if strings.HasSuffix(name, "_test") {
			return true
		}
		candidates = []string{filepath.Join(dir, name+"_test.go")}

	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		if strings.HasSuffix(name, ".test") || strings.HasSuffix(name, ".spec") ||
			filepath.Base(dir) == "__tests__" {
			return true
		}
		candidates = []string{
			filepath.Join(dir, name+".test"+ext),
			filepath.Join(dir, name+".spec"+ext),
			filepath.Join(dir, "__tests__", base),
			filepath.Join(dir, "__tests__", name+".test"+ext),
		}

	case ".py":
		if strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test") {
			return true
		}
		candidates = []string{
			filepath.Join(dir, "test_"+name+".py"),
			filepath.Join(dir, name+"_test.py"),
			filepath.Join(dir, "tests", "test_"+name+".py"),
			filepath.Join(dir, "tests", name+"_test.py"),
		}

	case ".java":
		if strings.HasSuffix(name, "Test") {
			return true
		}
		const srcMain = "src" + string(filepath.Separator) + "main" + string(filepath.Separator) + "java"
		const srcTest = "src" + string(filepath.Separator) + "test" + string(filepath.Separator) + "java"
		if !strings.Contains(dir, srcMain) {
			// No Maven/Gradle layout to map onto a test source root — the
			// signal doesn't apply rather than guessing.
			return true
		}
		testDir := strings.Replace(dir, srcMain, srcTest, 1)
		candidates = []string{filepath.Join(testDir, name+"Test.java")}

	default:
		// No universal test-naming convention for this language (e.g. C):
		// the signal doesn't apply, so it never penalizes what it can't
		// actually check.
		return true
	}

	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(root, c)); err == nil {
			return true
		}
	}
	return false
}
