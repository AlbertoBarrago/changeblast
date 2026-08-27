package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/impactline/internal/git"
)

func TestAnalyze_ChurnAndCoChange(t *testing.T) {
	root := t.TempDir()
	run(t, root, "init", "-q", "-b", "main")
	run(t, root, "config", "user.email", "test@example.com")
	run(t, root, "config", "user.name", "Test")

	tokenPath := filepath.Join(root, "token.ts")
	middlewarePath := filepath.Join(root, "middleware.ts")

	writeFile(t, tokenPath, "export const a = 1;")
	run(t, root, "add", ".")
	run(t, root, "commit", "-q", "-m", "initial")

	writeFile(t, tokenPath, "export const a = 2;")
	writeFile(t, middlewarePath, "export const b = 1;")
	run(t, root, "add", ".")
	run(t, root, "commit", "-q", "-m", "touch both")

	writeFile(t, tokenPath, "export const a = 3;")
	run(t, root, "add", ".")
	run(t, root, "commit", "-q", "-m", "touch token only")

	h, err := git.Analyze(root, tokenPath)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if h.Changes != 3 {
		t.Errorf("expected 3 changes, got %d", h.Changes)
	}
	if h.Window != git.DefaultWindow {
		t.Errorf("expected default window, got %+v", h.Window)
	}

	if len(h.CoChanged) != 1 {
		t.Fatalf("expected 1 co-changed file, got %d: %+v", len(h.CoChanged), h.CoChanged)
	}
	if h.CoChanged[0].Path != middlewarePath || h.CoChanged[0].Count != 1 {
		t.Errorf("unexpected co-change entry: %+v", h.CoChanged[0])
	}
}

func TestAnalyzeWithWindow_MaxCommitsOverride(t *testing.T) {
	root := t.TempDir()
	run(t, root, "init", "-q", "-b", "main")
	run(t, root, "config", "user.email", "test@example.com")
	run(t, root, "config", "user.name", "Test")

	tokenPath := filepath.Join(root, "token.ts")

	for i := 0; i < 3; i++ {
		writeFile(t, tokenPath, "export const a = "+string(rune('0'+i))+";")
		run(t, root, "add", ".")
		run(t, root, "commit", "-q", "-m", "touch token")
	}

	narrow := git.Window{Days: git.HistoryWindowDays, MaxCommits: 1}
	h, err := git.AnalyzeWithWindow(root, tokenPath, narrow)
	if err != nil {
		t.Fatalf("AnalyzeWithWindow: %v", err)
	}

	if h.Changes != 1 {
		t.Errorf("expected 1 change with maxCommits=1 override, got %d", h.Changes)
	}
	if h.Window != narrow {
		t.Errorf("expected reported window %+v, got %+v", narrow, h.Window)
	}
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
