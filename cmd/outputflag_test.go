package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFormat(t *testing.T) {
	cases := []struct {
		name    string
		flags   analysisFlags
		want    string
		wantErr bool
	}{
		{"default is text", analysisFlags{}, "text", false},
		{"json bool", analysisFlags{json: true}, "json", false},
		{"output-format json", analysisFlags{outputFormat: "json"}, "json", false},
		{"output-format sarif", analysisFlags{outputFormat: "sarif"}, "sarif", false},
		{"output-format text", analysisFlags{outputFormat: "text"}, "text", false},
		{"json bool agrees with output-format json", analysisFlags{json: true, outputFormat: "json"}, "json", false},
		{"json bool conflicts with output-format sarif", analysisFlags{json: true, outputFormat: "sarif"}, "", true},
		{"unknown output-format", analysisFlags{outputFormat: "xml"}, "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveFormat(tc.flags)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got format %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveFormat: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

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
