package golang_test

import (
	"testing"

	"github.com/AlbertoBarrago/impactline/internal/analyzer/golang"
)

func TestExtractImports_SingleAndBlock(t *testing.T) {
	src := []byte(`package foo

import "fmt"

import (
	"os"
	alias "path/filepath"
	_ "database/sql/driver"
	. "some/dot/import"
)

// import "commented/out"
/* import "block/commented" */

func main() {
	s := "import \"fake/in/string\""
	r := ` + "`" + `import "fake/in/raw/string"` + "`" + `
	_, _ = s, r
}
`)

	a := golang.New()
	imports, err := a.ExtractImports("main.go", src)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, imp := range imports {
		got[imp.Specifier] = true
	}

	for _, want := range []string{"fmt", "os", "path/filepath", "database/sql/driver", "some/dot/import"} {
		if !got[want] {
			t.Errorf("expected import %q to be found, got %+v", want, got)
		}
	}

	for _, unwanted := range []string{"commented/out", "block/commented", "fake/in/string", "fake/in/raw/string"} {
		if got[unwanted] {
			t.Errorf("expected %q to NOT be extracted (comment/string), got %+v", unwanted, got)
		}
	}
}

func TestCanHandle(t *testing.T) {
	a := golang.New()
	if !a.CanHandle("main.go") {
		t.Error("expected CanHandle(main.go) to be true")
	}
	if a.CanHandle("main.ts") {
		t.Error("expected CanHandle(main.ts) to be false")
	}
}
