package localcli_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/AlbertoBarrago/impactline/internal/ai"
	"github.com/AlbertoBarrago/impactline/internal/ai/localcli"
)

func TestClaudeProvider_Explain(t *testing.T) {
	fakeBinOnPath(t, "claude", `#!/bin/sh
echo "explained by claude"
`)

	p := localcli.NewClaude("")
	got, err := p.Explain(context.Background(), ai.Finding{Target: "a.ts", RiskLevel: "HIGH"})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if got != "explained by claude" {
		t.Errorf("Explain() = %q, want %q", got, "explained by claude")
	}
	if p.Name() != "claude" {
		t.Errorf("Name() = %q, want claude", p.Name())
	}
}

func TestGeminiProvider_Explain(t *testing.T) {
	fakeBinOnPath(t, "gemini", `#!/bin/sh
echo "explained by gemini"
`)

	p := localcli.NewGemini("")
	got, err := p.Explain(context.Background(), ai.Finding{Target: "a.ts", RiskLevel: "HIGH"})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if got != "explained by gemini" {
		t.Errorf("Explain() = %q, want %q", got, "explained by gemini")
	}
}

func TestCodexProvider_Explain(t *testing.T) {
	// codex writes its response to the file passed after
	// --output-last-message rather than stdout.
	fakeBinOnPath(t, "codex", `#!/bin/sh
for i in "$@"; do
  if [ "$prev" = "--output-last-message" ]; then
    echo "explained by codex" > "$i"
  fi
  prev="$i"
done
`)

	p := localcli.NewCodex("")
	got, err := p.Explain(context.Background(), ai.Finding{Target: "a.ts", RiskLevel: "HIGH"})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if got != "explained by codex" {
		t.Errorf("Explain() = %q, want %q", got, "explained by codex")
	}
}

func TestExplain_BinaryNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	p := localcli.NewClaude("")
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error when the binary is not on PATH")
	}
}

func TestExplain_NonZeroExit(t *testing.T) {
	fakeBinOnPath(t, "claude", `#!/bin/sh
echo "something went wrong" >&2
exit 1
`)

	p := localcli.NewClaude("")
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error on non-zero exit")
	}
}

// fakeBinOnPath writes a shell script named name (with script as its
// content) into a temp dir prepended to PATH for the duration of the
// test, so localcli's exec.LookPath/exec.Command calls resolve to it
// instead of any real CLI that might also be installed.
func fakeBinOnPath(t *testing.T, name, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell-script binaries are POSIX-shell only")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
