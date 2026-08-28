package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if len(cfg.CriticalPaths) != 0 {
		t.Errorf("CriticalPaths = %v, want empty", cfg.CriticalPaths)
	}
	if cfg.HistoryWindow.Days != 0 || cfg.HistoryWindow.MaxCommits != 0 {
		t.Errorf("HistoryWindow = %+v, want zero value", cfg.HistoryWindow)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	content := `
criticalPaths:
  - auth
  - billing
  - checkout
highRiskPaths:
  - "**/migrations/**"
  - protocol.ts
historyWindow:
  days: 30
  maxCommits: 50
`
	writeConfig(t, dir, content)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}

	wantPaths := []string{"auth", "billing", "checkout"}
	if len(cfg.CriticalPaths) != len(wantPaths) {
		t.Fatalf("CriticalPaths = %v, want %v", cfg.CriticalPaths, wantPaths)
	}
	for i, p := range wantPaths {
		if cfg.CriticalPaths[i] != p {
			t.Errorf("CriticalPaths[%d] = %q, want %q", i, cfg.CriticalPaths[i], p)
		}
	}
	wantHighRisk := []string{"**/migrations/**", "protocol.ts"}
	if len(cfg.HighRiskPaths) != len(wantHighRisk) {
		t.Fatalf("HighRiskPaths = %v, want %v", cfg.HighRiskPaths, wantHighRisk)
	}
	for i, p := range wantHighRisk {
		if cfg.HighRiskPaths[i] != p {
			t.Errorf("HighRiskPaths[%d] = %q, want %q", i, cfg.HighRiskPaths[i], p)
		}
	}
	if cfg.HistoryWindow.Days != 30 {
		t.Errorf("HistoryWindow.Days = %d, want 30", cfg.HistoryWindow.Days)
	}
	if cfg.HistoryWindow.MaxCommits != 50 {
		t.Errorf("HistoryWindow.MaxCommits = %d, want 50", cfg.HistoryWindow.MaxCommits)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "criticalPaths: [unterminated")

	if _, err := Load(dir); err == nil {
		t.Fatal("Load: expected error for invalid YAML, got nil")
	}
}

func TestConfig_Overrides(t *testing.T) {
	empty := Config{}
	if got := empty.CriticalPathsOr([]string{"default"}); len(got) != 1 || got[0] != "default" {
		t.Errorf("CriticalPathsOr with empty config = %v, want [default]", got)
	}
	if got := empty.HighRiskPathsOr([]string{"**/default/**"}); len(got) != 1 || got[0] != "**/default/**" {
		t.Errorf("HighRiskPathsOr with empty config = %v, want [**/default/**]", got)
	}
	if got := empty.HistoryWindowDaysOr(90); got != 90 {
		t.Errorf("HistoryWindowDaysOr with empty config = %d, want 90", got)
	}
	if got := empty.HistoryWindowMaxCommitsOr(200); got != 200 {
		t.Errorf("HistoryWindowMaxCommitsOr with empty config = %d, want 200", got)
	}

	set := Config{CriticalPaths: []string{"custom"}, HighRiskPaths: []string{"protocol.ts"}}
	set.HistoryWindow.Days = 10
	set.HistoryWindow.MaxCommits = 20
	if got := set.CriticalPathsOr([]string{"default"}); len(got) != 1 || got[0] != "custom" {
		t.Errorf("CriticalPathsOr with set config = %v, want [custom]", got)
	}
	if got := set.HighRiskPathsOr([]string{"**/default/**"}); len(got) != 1 || got[0] != "protocol.ts" {
		t.Errorf("HighRiskPathsOr with set config = %v, want [protocol.ts]", got)
	}
	if got := set.HistoryWindowDaysOr(90); got != 10 {
		t.Errorf("HistoryWindowDaysOr with set config = %d, want 10", got)
	}
	if got := set.HistoryWindowMaxCommitsOr(200); got != 20 {
		t.Errorf("HistoryWindowMaxCommitsOr with set config = %d, want 20", got)
	}
}

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", FileName, err)
	}
}
