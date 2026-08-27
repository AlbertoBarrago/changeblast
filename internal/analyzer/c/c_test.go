package c_test

import (
	"testing"

	"github.com/AlbertoBarrago/blast/internal/analyzer/c"
)

func TestExtractImports_QuotedOnly(t *testing.T) {
	src := []byte(`#include <stdio.h>
#include "token.h"
#include "../util/log.h"

// #include "commented/out.h"
/* #include "block/commented.h" */

int main() {
    char *s = "#include \"fake/in/string.h\"";
    char c = '"';
    return 0;
}
`)

	a := c.New()
	imports, err := a.ExtractImports("main.c", src)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, imp := range imports {
		got[imp.Specifier] = true
	}

	for _, want := range []string{"token.h", "../util/log.h"} {
		if !got[want] {
			t.Errorf("expected include %q to be found, got %+v", want, got)
		}
	}

	for _, unwanted := range []string{"stdio.h", "commented/out.h", "block/commented.h", "fake/in/string.h"} {
		if got[unwanted] {
			t.Errorf("expected %q to NOT be extracted, got %+v", unwanted, got)
		}
	}
}

func TestCanHandle(t *testing.T) {
	a := c.New()
	if !a.CanHandle("main.c") {
		t.Error("expected CanHandle(main.c) to be true")
	}
	if !a.CanHandle("token.h") {
		t.Error("expected CanHandle(token.h) to be true")
	}
	if a.CanHandle("main.go") {
		t.Error("expected CanHandle(main.go) to be false")
	}
}
