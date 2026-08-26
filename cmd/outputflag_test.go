package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenOutputTarget_EmptyPathReturnsDefault(t *testing.T) {
	var buf bytes.Buffer
	w, closeFn, err := openOutputTarget(&buf, "")
	if err != nil {
		t.Fatalf("openOutputTarget: %v", err)
	}
	defer closeFn()

	if w != io.Writer(&buf) {
		t.Error("expected the default writer to be returned unchanged")
	}
}

func TestOpenOutputTarget_WritesToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.txt")

	w, closeFn, err := openOutputTarget(os.Stdout, path)
	if err != nil {
		t.Fatalf("openOutputTarget: %v", err)
	}

	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := closeFn(); err != nil {
		t.Fatalf("close: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("content = %q, want %q", content, "hello")
	}
}
