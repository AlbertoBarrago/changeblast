package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryRoot(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "src", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// From inside the repo, including deep subdirectories.
	got, err := repositoryRoot(filepath.Join(repo, "src", "nested"))
	if err != nil {
		t.Fatalf("repositoryRoot: %v", err)
	}
	if got != repo {
		t.Errorf("repositoryRoot = %q, want %q", got, repo)
	}
}

func TestRepositoryRoot_GitFile(t *testing.T) {
	// Worktrees and submodules carry a .git *file*, not a directory.
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := repositoryRoot(repo)
	if err != nil {
		t.Fatalf("repositoryRoot with .git file: %v", err)
	}
	if got != repo {
		t.Errorf("repositoryRoot = %q, want %q", got, repo)
	}
}

func TestRepositoryRoot_OutsideRepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := repositoryRoot(dir); err == nil {
		t.Error("repositoryRoot outside a git repository should error, got nil")
	}
}
