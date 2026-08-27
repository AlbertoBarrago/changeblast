package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunDiff_ChangeSetSection exercises the set-level analysis: two
// changed files where one imports the other, plus one unchanged module
// importing both. The text output must surface the "Change set" section
// with the internal edge and the shared dependent.
func TestRunDiff_ChangeSetSection(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	runGitCmd(t, root, "init", "-q", "-b", "main")
	runGitCmd(t, root, "config", "user.email", "test@example.com")
	runGitCmd(t, root, "config", "user.name", "Test")

	// a imports b; user imports both a and b.
	writeTestFile(t, filepath.Join(root, "b.ts"), "export const b = 1;\n")
	writeTestFile(t, filepath.Join(root, "a.ts"), "import \"./b\";\nexport const a = 1;\n")
	writeTestFile(t, filepath.Join(root, "user.ts"), "import \"./a\";\nimport \"./b\";\nexport const u = 1;\n")
	runGitCmd(t, root, "add", ".")
	runGitCmd(t, root, "commit", "-q", "-m", "initial")

	// Change both a and b.
	writeTestFile(t, filepath.Join(root, "a.ts"), "import \"./b\";\nexport const a = 2;\n")
	writeTestFile(t, filepath.Join(root, "b.ts"), "export const b = 2;\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	diffFlags = analysisFlags{}
	diffExplain = &explainFlags{}
	diffCmd.SetOut(&stdout)
	diffCmd.SetErr(&stderr)
	t.Cleanup(func() {
		diffFlags, diffExplain = analysisFlags{}, &explainFlags{}
		diffCmd.SetOut(nil)
		diffCmd.SetErr(nil)
	})

	if err := diffCmd.RunE(diffCmd, []string{}); err != nil {
		t.Fatalf("runDiff: %v", err)
	}
	out := stdout.String()

	for _, want := range []string{
		"Change set",
		"2 changed files",
		"a.ts imports b.ts",
		"user.ts", // shared dependent
		"Change-set risk",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	_ = stderr
}

// TestRunDiff_ChangeSetJSON verifies the {changeSet, files} envelope and
// that a single-file diff omits changeSet.
func TestRunDiff_ChangeSetJSON(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	runGitCmd(t, root, "init", "-q", "-b", "main")
	runGitCmd(t, root, "config", "user.email", "test@example.com")
	runGitCmd(t, root, "config", "user.name", "Test")

	writeTestFile(t, filepath.Join(root, "b.ts"), "export const b = 1;\n")
	writeTestFile(t, filepath.Join(root, "a.ts"), "import \"./b\";\nexport const a = 1;\n")
	runGitCmd(t, root, "add", ".")
	runGitCmd(t, root, "commit", "-q", "-m", "initial")

	writeTestFile(t, filepath.Join(root, "a.ts"), "import \"./b\";\nexport const a = 2;\n")
	writeTestFile(t, filepath.Join(root, "b.ts"), "export const b = 2;\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	flagOut := filepath.Join(root, "out.json")
	diffFlags = analysisFlags{json: true, output: flagOut}
	diffExplain = &explainFlags{}
	t.Cleanup(func() { diffFlags, diffExplain = analysisFlags{}, &explainFlags{} })

	if err := diffCmd.RunE(diffCmd, []string{}); err != nil {
		t.Fatalf("runDiff: %v", err)
	}

	data, err := os.ReadFile(flagOut)
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		ChangeSet *struct {
			Targets       []string `json:"targets"`
			InternalEdges []struct {
				Importer string `json:"importer"`
				Imported string `json:"imported"`
			} `json:"internalEdges"`
			Risk struct {
				Total int    `json:"total"`
				Level string `json:"level"`
			} `json:"risk"`
		} `json:"changeSet"`
		Files []json.RawMessage `json:"files"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, data)
	}

	if body.ChangeSet == nil {
		t.Fatalf("expected changeSet in output:\n%s", data)
	}
	if len(body.Files) != 2 {
		t.Errorf("files = %d, want 2", len(body.Files))
	}
	if len(body.ChangeSet.InternalEdges) != 1 ||
		body.ChangeSet.InternalEdges[0].Importer != "a.ts" ||
		body.ChangeSet.InternalEdges[0].Imported != "b.ts" {
		t.Errorf("internalEdges = %+v, want [{a.ts b.ts}]", body.ChangeSet.InternalEdges)
	}
}
