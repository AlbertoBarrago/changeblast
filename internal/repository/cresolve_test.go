package repository_test

import (
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/blast/internal/repository"
)

func TestCResolver(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "src", "token.h"), "")
	mustWriteFile(t, filepath.Join(root, "util", "log.h"), "")
	fromFile := filepath.Join(root, "src", "main.c")
	mustWriteFile(t, fromFile, "")

	r := repository.NewCResolver()

	// #include "token.h" -> sibling file
	got := r.Resolve(fromFile, "token.h")
	want := filepath.Join(root, "src", "token.h")
	if len(got) != 1 || got[0] != want {
		t.Errorf("Resolve(token.h) = %v, want [%s]", got, want)
	}

	// #include "../util/log.h" -> relative to including file's dir
	got = r.Resolve(fromFile, "../util/log.h")
	wantLog := filepath.Join(root, "util", "log.h")
	if len(got) != 1 || got[0] != wantLog {
		t.Errorf("Resolve(../util/log.h) = %v, want [%s]", got, wantLog)
	}

	// #include "missing.h" -> unresolved
	got = r.Resolve(fromFile, "missing.h")
	if got != nil {
		t.Errorf("Resolve(missing.h) = %v, want nil", got)
	}
}
