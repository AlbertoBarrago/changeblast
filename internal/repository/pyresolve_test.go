package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/impactline/internal/repository"
)

func TestPythonResolver_Absolute(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "pkg", "auth", "token.py"), "")
	mustWriteFile(t, filepath.Join(root, "pkg", "auth", "__init__.py"), "")
	mustWriteFile(t, filepath.Join(root, "main.py"), "")

	r := repository.NewPythonResolver(root)
	fromFile := filepath.Join(root, "main.py")

	// import pkg.auth.token -> exact module, no fallback
	got := r.Resolve(fromFile, "pkg.auth.token", false)
	want := filepath.Join(root, "pkg", "auth", "token.py")
	if len(got) != 1 || got[0] != want {
		t.Errorf("Resolve(pkg.auth.token) = %v, want [%s]", got, want)
	}

	// from pkg.auth import token -> submodule wins over package fallback
	got = r.Resolve(fromFile, "pkg.auth.token", true)
	if len(got) != 1 || got[0] != want {
		t.Errorf("Resolve(from pkg.auth import token) = %v, want [%s]", got, want)
	}

	// from pkg.auth import sign -> not a submodule, falls back to pkg/auth/__init__.py
	got = r.Resolve(fromFile, "pkg.auth.sign", true)
	wantInit := filepath.Join(root, "pkg", "auth", "__init__.py")
	if len(got) != 1 || got[0] != wantInit {
		t.Errorf("Resolve(from pkg.auth import sign) = %v, want [%s]", got, wantInit)
	}

	// import pkg.auth.sign (plain import, no fallback) -> unresolved
	got = r.Resolve(fromFile, "pkg.auth.sign", false)
	if got != nil {
		t.Errorf("Resolve(pkg.auth.sign) = %v, want nil (no fallback for plain import)", got)
	}

	// external/stdlib module -> unresolved
	got = r.Resolve(fromFile, "os.path", false)
	if got != nil {
		t.Errorf("Resolve(os.path) = %v, want nil (external)", got)
	}
}

func TestPythonResolver_Relative(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "pkg", "__init__.py"), "")
	mustWriteFile(t, filepath.Join(root, "pkg", "sibling.py"), "")
	fromFile := filepath.Join(root, "pkg", "main.py")
	mustWriteFile(t, fromFile, "")

	r := repository.NewPythonResolver(root)

	// from . import sibling -> pkg/sibling.py
	got := r.Resolve(fromFile, ".sibling", true)
	want := filepath.Join(root, "pkg", "sibling.py")
	if len(got) != 1 || got[0] != want {
		t.Errorf("Resolve(.sibling) = %v, want [%s]", got, want)
	}

	// from . import x -> not found as submodule, falls back to pkg/__init__.py
	got = r.Resolve(fromFile, ".x", true)
	wantInit := filepath.Join(root, "pkg", "__init__.py")
	if len(got) != 1 || got[0] != wantInit {
		t.Errorf("Resolve(.x) = %v, want [%s]", got, wantInit)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
