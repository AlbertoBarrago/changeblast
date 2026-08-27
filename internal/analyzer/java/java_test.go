package java_test

import (
	"testing"

	"github.com/AlbertoBarrago/impactline/internal/analyzer/java"
)

func TestPackage(t *testing.T) {
	src := []byte(`package com.example.auth;

import com.example.util.Logger;
`)
	if got := java.Package(src); got != "com.example.auth" {
		t.Errorf("Package() = %q, want %q", got, "com.example.auth")
	}

	if got := java.Package([]byte("class Foo {}")); got != "" {
		t.Errorf("Package() with no declaration = %q, want empty", got)
	}
}

func TestExtractImports(t *testing.T) {
	src := []byte(`package com.example.auth;

import com.example.util.Logger;
import com.example.util.*;
import static com.example.util.Constants.MAX_RETRIES;
import static com.example.util.Constants.*;

// import com.example.commented.Out;
/* import com.example.block.Commented; */

class Token {
    String s = "import com.example.fake.InString;";
    char c = '"';
}
`)

	a := java.New()
	imports, err := a.ExtractImports("Token.java", src)
	if err != nil {
		t.Fatal(err)
	}

	type key struct {
		specifier string
		static    bool
	}
	got := map[key]bool{}
	for _, imp := range imports {
		got[key{imp.Specifier, imp.Static}] = true
	}

	want := []key{
		{"com.example.util.Logger", false},
		{"com.example.util.*", false},
		{"com.example.util.Constants.MAX_RETRIES", true},
		{"com.example.util.Constants.*", true},
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("expected import %+v to be found, got %+v", w, got)
		}
	}

	for _, unwanted := range []string{"com.example.commented.Out", "com.example.block.Commented", "com.example.fake.InString"} {
		if got[key{unwanted, false}] {
			t.Errorf("expected %q to NOT be extracted (comment/string), got %+v", unwanted, got)
		}
	}
}

func TestCanHandle(t *testing.T) {
	a := java.New()
	if !a.CanHandle("Main.java") {
		t.Error("expected CanHandle(Main.java) to be true")
	}
	if a.CanHandle("Main.go") {
		t.Error("expected CanHandle(Main.go) to be false")
	}
}
