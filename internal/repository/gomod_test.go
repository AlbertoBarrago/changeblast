package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/repository"
)

func TestFindGoModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/foo\n\ngo 1.22\n")

	mod, err := repository.FindGoModule(root)
	if err != nil {
		t.Fatalf("FindGoModule: %v", err)
	}
	if mod == nil {
		t.Fatal("expected a module to be found")
	}
	if mod.Path != "example.com/foo" {
		t.Errorf("Path = %q, want %q", mod.Path, "example.com/foo")
	}
}

func TestFindGoModule_None(t *testing.T) {
	root := t.TempDir()
	mod, err := repository.FindGoModule(root)
	if err != nil {
		t.Fatalf("FindGoModule: %v", err)
	}
	if mod != nil {
		t.Errorf("expected nil, got %+v", mod)
	}
}

func TestFindGoModule_NearestAncestor(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "internal", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/bar\n")

	mod, err := repository.FindGoModule(nested)
	if err != nil {
		t.Fatalf("FindGoModule: %v", err)
	}
	if mod == nil || mod.Path != "example.com/bar" {
		t.Fatalf("mod = %+v", mod)
	}
}

func TestGoResolver_ResolvesPackageFiles(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "internal", "widget")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(pkgDir, "widget.go"), "package widget")
	writeFile(t, filepath.Join(pkgDir, "widget_test.go"), "package widget")
	writeFile(t, filepath.Join(pkgDir, "README.md"), "not go")

	mod, err := repository.FindGoModule(root)
	if err != nil || mod == nil {
		t.Fatalf("FindGoModule: mod=%+v err=%v", mod, err)
	}

	r := repository.NewGoResolver(mod)
	from := filepath.Join(root, "main.go")
	resolved := r.Resolve(from, "example.com/app/internal/widget")

	// Test files are not production dependents: including them as graph
	// nodes inflated the impact score for every Go import.
	if len(resolved) != 1 {
		t.Fatalf("expected 1 .go file (test files excluded), got %d: %v", len(resolved), resolved)
	}
	if resolved[0] != filepath.Join(pkgDir, "widget.go") {
		t.Errorf("unexpected resolution: %q", resolved[0])
	}
}

func TestGoResolver_ExternalImportUnresolved(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")

	mod, _ := repository.FindGoModule(root)
	r := repository.NewGoResolver(mod)

	if resolved := r.Resolve(filepath.Join(root, "main.go"), "fmt"); resolved != nil {
		t.Errorf("expected stdlib import to be unresolved, got %v", resolved)
	}
	if resolved := r.Resolve(filepath.Join(root, "main.go"), "github.com/other/pkg"); resolved != nil {
		t.Errorf("expected external module import to be unresolved, got %v", resolved)
	}
}

func TestGoResolver_NilModule(t *testing.T) {
	r := repository.NewGoResolver(nil)
	if resolved := r.Resolve("main.go", "example.com/anything"); resolved != nil {
		t.Errorf("expected nil module to never resolve, got %v", resolved)
	}
}

func TestGoResolver_ExcludesSelf(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "widget")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	self := filepath.Join(pkgDir, "widget.go")
	writeFile(t, self, "package widget")

	mod, _ := repository.FindGoModule(root)
	r := repository.NewGoResolver(mod)

	resolved := r.Resolve(self, "example.com/app/widget")
	for _, f := range resolved {
		if f == self {
			t.Error("expected Resolve to exclude the importing file itself")
		}
	}
}
