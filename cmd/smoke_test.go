package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunDiff_FromSubdirectory guards the regression where
// ChangedFiles' repo-root-relative paths were stat'ed against the
// process cwd: running `serval diff` from a subdirectory silently
// skipped every changed file.
func TestRunDiff_FromSubdirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	runGitCmd(t, root, "init", "-q", "-b", "main")
	runGitCmd(t, root, "config", "user.email", "test@example.com")
	runGitCmd(t, root, "config", "user.name", "Test")

	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "src", "app.ts"), "export const a = 1;\n")
	runGitCmd(t, root, "add", ".")
	runGitCmd(t, root, "commit", "-q", "-m", "initial")

	// Uncommitted change under src/.
	writeTestFile(t, filepath.Join(root, "src", "app.ts"), "export const a = 2;\n")

	// Run from a subdirectory of the repo, like a user would in a
	// terminal cd'ed into src/.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })
	if err := os.Chdir(filepath.Join(root, "src")); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	inspectFlags = analysisFlags{}
	diffFlags = analysisFlags{}
	diffExplain = &explainFlags{}

	cmd := diffCmd
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("runDiff from subdirectory: %v", err)
	}

	if !strings.Contains(stdout.String(), "app.ts") {
		t.Errorf("expected changed file app.ts in output, got:\n%s", stdout.String())
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
