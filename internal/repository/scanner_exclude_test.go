package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/repository"
)

// TestScan_ExcludesVCSAndVirtualEnvDirs guards the exclusion list: venv
// trees (duplicated site-packages) and non-Git VCS metadata (Mercurial,
// Jujutsu) must not be scanned, or a Python project with a checked-in
// .venv balloons the graph with thousands of irrelevant nodes.
func TestScan_ExcludesVCSAndVirtualEnvDirs(t *testing.T) {
	root := t.TempDir()

	appPath := filepath.Join(root, "app.py")
	if err := os.WriteFile(appPath, []byte("import os\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, dir := range []string{".venv", "venv", ".hg", ".jj"} {
		nested := filepath.Join(root, dir, "lib", "app.py")
		if err := os.MkdirAll(filepath.Dir(nested), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(nested, []byte("import os\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	scanner, err := repository.NewScanner(root)
	if err != nil {
		t.Fatalf("NewScanner: %v", err)
	}

	g, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if !g.HasNode(appPath) {
		t.Errorf("expected graph to contain %s", appPath)
	}
	for _, dir := range []string{".venv", "venv", ".hg", ".jj"} {
		excluded := filepath.Join(root, dir, "lib", "app.py")
		if g.HasNode(excluded) {
			t.Errorf("expected %s to be excluded from the scan", excluded)
		}
	}
}
