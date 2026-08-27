package python_test

import (
	"testing"

	"github.com/AlbertoBarrago/impactline/internal/analyzer"
	"github.com/AlbertoBarrago/impactline/internal/analyzer/python"
)

func TestExtractImports_PlainAndFrom(t *testing.T) {
	src := []byte(`import os
import a.b.c
import a, b.c as x

from a.b import c
from a.b import c as d, e
from . import x
from .pkg import y
from ..pkg.sub import z
from a.b import (
    f,
    g as h,
)
from wild import *

# import "commented/out"
"""
import "docstring/out"
# not a comment, still inside string
"""
s = "import 'fake/in/string'"
`)

	a := python.New()
	imports, err := a.ExtractImports("main.py", src)
	if err != nil {
		t.Fatal(err)
	}

	byKey := map[string]analyzer.RawImport{}
	for _, imp := range imports {
		key := imp.Specifier
		if imp.FromImport {
			key = "from:" + key
		}
		byKey[key] = imp
	}

	wantPlain := []string{"os", "a.b.c", "a", "b.c"}
	for _, w := range wantPlain {
		imp, ok := byKey[w]
		if !ok {
			t.Errorf("expected plain import %q to be found, got %+v", w, byKey)
			continue
		}
		if imp.FromImport {
			t.Errorf("expected %q to not be a from-import", w)
		}
	}

	wantFrom := []string{"a.b.c", "a.b.e", ".x", ".pkg.y", "..pkg.sub.z", "a.b.f", "a.b.g"}
	for _, w := range wantFrom {
		imp, ok := byKey["from:"+w]
		if !ok {
			t.Errorf("expected from-import %q to be found, got %+v", w, byKey)
			continue
		}
		if !imp.FromImport {
			t.Errorf("expected %q to be a from-import", w)
		}
	}

	for _, unwanted := range []string{"commented/out", "docstring/out", "fake/in/string", "wild.*"} {
		if _, ok := byKey[unwanted]; ok {
			t.Errorf("expected %q to NOT be extracted, got %+v", unwanted, byKey)
		}
		if _, ok := byKey["from:"+unwanted]; ok {
			t.Errorf("expected %q to NOT be extracted as from-import, got %+v", unwanted, byKey)
		}
	}
}

func TestCanHandle(t *testing.T) {
	a := python.New()
	if !a.CanHandle("main.py") {
		t.Error("expected CanHandle(main.py) to be true")
	}
	if a.CanHandle("main.go") {
		t.Error("expected CanHandle(main.go) to be false")
	}
}
