package repository

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/AlbertoBarrago/changeblast/internal/analyzer"
)

// JavaResolver resolves Java import specifiers to files within the
// repository. Unlike Go (anchored on go.mod) or JS/TS (anchored on
// tsconfig.json), Java v0.1 has no repository-wide manifest to derive
// resolution from: instead, each importing file's own `package`
// declaration (internal/analyzer/java.Package) is used to derive that
// file's source root, by walking up one directory per package segment
// from the file's own path. This assumes the conventional layout where
// a file's directory path suffix matches its package name (e.g.
// src/main/java/a/b/Foo.java declaring "package a.b;"), true for any
// standard Maven/Gradle layout — the documented v0.1 scope, same
// category of assumption as JS/TS's relative-import-only resolution.
type JavaResolver struct{}

// NewJavaResolver returns a Java resolver.
func NewJavaResolver() *JavaResolver {
	return &JavaResolver{}
}

// Resolve resolves imp (found in fromFile, which declares packageName)
// to zero or more absolute .java file paths: one file for a plain or
// static import, every other .java file in the target package
// directory for a type wildcard import (`import a.b.*;`), matching the
// package-level granularity Go uses for its own imports.
func (r *JavaResolver) Resolve(fromFile, packageName string, imp analyzer.RawImport) []string {
	root := sourceRoot(fromFile, packageName)
	parts := strings.Split(imp.Specifier, ".")
	if len(parts) == 0 {
		return nil
	}

	if imp.Static {
		// import static a.b.C.member / import static a.b.C.*
		// -> the class a.b.C itself; member/"*" isn't a file.
		if len(parts) < 2 {
			return nil
		}
		return resolveClass(root, parts[:len(parts)-1])
	}

	if parts[len(parts)-1] == "*" {
		dir := filepath.Join(root, filepath.Join(parts[:len(parts)-1]...))
		return javaFilesIn(dir, fromFile)
	}

	return resolveClass(root, parts)
}

// sourceRoot derives fromFile's source root by walking up one directory
// per segment of packageName from fromFile's own directory. A file with
// no package declaration is assumed to live at its own source root.
func sourceRoot(fromFile, packageName string) string {
	dir := filepath.Dir(fromFile)
	if packageName == "" {
		return dir
	}
	for i := strings.Count(packageName, ".") + 1; i > 0; i-- {
		dir = filepath.Dir(dir)
	}
	return dir
}

func resolveClass(root string, parts []string) []string {
	target := filepath.Join(root, filepath.Join(parts...)) + ".java"
	if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
		return []string{target}
	}
	return nil
}

func javaFilesIn(dir, fromFile string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".java") {
			continue
		}
		target := filepath.Join(dir, e.Name())
		if target == fromFile {
			continue
		}
		files = append(files, target)
	}
	return files
}
