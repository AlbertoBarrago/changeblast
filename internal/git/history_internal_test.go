package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRunGit_Timeout guards the subprocess timeout: a git invocation
// that hangs (here, a commit whose core.editor blocks forever) must
// surface as a "timed out after" error rather than freeze the run.
// Shrink subprocessTimeout from its 30s default; t.Cleanup restores it.
func TestRunGit_Timeout(t *testing.T) {
	original := subprocessTimeout
	subprocessTimeout = 250 * time.Millisecond
	t.Cleanup(func() { subprocessTimeout = original })

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	runSetup := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runSetup("init", "-q", "-b", "main")
	runSetup("config", "user.email", "test@example.com")
	runSetup("config", "user.name", "Test")

	// A blocking editor turns `git commit` (no -m) into a hang: git
	// waits for the editor process, which outlives the timeout.
	editor := filepath.Join(root, "block-editor.sh")
	script := "#!/bin/sh\nsleep 30\n"
	if err := os.WriteFile(editor, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	runSetup("config", "core.editor", editor)

	// GIT_EDITOR (and VISUAL/EDITOR) in the test process's own environment
	// take precedence over core.editor and would make git skip our
	// blocking script entirely, so clear them for this test.
	for _, k := range []string{"GIT_EDITOR", "VISUAL", "EDITOR"} {
		if v, ok := os.LookupEnv(k); ok {
			os.Unsetenv(k)
			t.Cleanup(func() { _ = os.Setenv(k, v) })
		}
	}

	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	runSetup("add", ".")

	start := time.Now()
	_, err := runGit(root, "commit")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a hanging git commit, got nil")
	}
	if !strings.Contains(err.Error(), "timed out after") {
		t.Fatalf("expected a timeout error, got: %v", err)
	}
	if elapsed >= 10*time.Second {
		t.Fatalf("runGit took %s; timeout did not fire", elapsed)
	}
}
