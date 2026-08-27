package javascript_test

import (
	"testing"

	"github.com/AlbertoBarrago/serval/internal/analyzer/javascript"
)

func TestExtractImports(t *testing.T) {
	src := []byte(`
import { a } from './a';
import b from "../b";
import './side-effect';
export { c } from './c';
const x = require('./d');
const y = import('./e');
// import { fake } from './commented';
/* import { fake2 } from './blockcommented'; */
import react from 'react';
`)

	a := javascript.New()
	imports, err := a.ExtractImports("test.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, imp := range imports {
		got[imp.Specifier] = imp.Dynamic
	}

	wantStatic := []string{"./a", "../b", "./side-effect", "./c", "./d", "react"}
	for _, spec := range wantStatic {
		if dyn, ok := got[spec]; !ok || dyn {
			t.Errorf("expected static import %q to be present, got ok=%v dynamic=%v", spec, ok, dyn)
		}
	}

	if dyn, ok := got["./e"]; !ok || !dyn {
		t.Errorf("expected dynamic import './e' to be marked dynamic, got ok=%v dynamic=%v", ok, dyn)
	}

	for _, spec := range []string{"./commented", "./blockcommented"} {
		if _, ok := got[spec]; ok {
			t.Errorf("expected commented-out import %q to be ignored", spec)
		}
	}
}

func TestCanHandle(t *testing.T) {
	a := javascript.New()
	for _, path := range []string{"x.ts", "x.tsx", "x.js", "x.jsx", "x.mjs", "x.cjs"} {
		if !a.CanHandle(path) {
			t.Errorf("expected CanHandle(%q) to be true", path)
		}
	}
	if a.CanHandle("x.go") {
		t.Errorf("expected CanHandle(x.go) to be false")
	}
}
