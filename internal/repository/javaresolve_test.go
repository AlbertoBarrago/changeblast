package repository_test

import (
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/blast/internal/analyzer"
	"github.com/AlbertoBarrago/blast/internal/repository"
)

func TestJavaResolver_PlainAndWildcard(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "src", "main", "java", "com", "example", "util", "Logger.java"), "")
	mustWriteFile(t, filepath.Join(root, "src", "main", "java", "com", "example", "util", "Constants.java"), "")
	fromFile := filepath.Join(root, "src", "main", "java", "com", "example", "auth", "Token.java")
	mustWriteFile(t, fromFile, "")

	r := repository.NewJavaResolver()

	// import com.example.util.Logger;
	got := r.Resolve(fromFile, "com.example.auth", analyzer.RawImport{Specifier: "com.example.util.Logger"})
	want := filepath.Join(root, "src", "main", "java", "com", "example", "util", "Logger.java")
	if len(got) != 1 || got[0] != want {
		t.Errorf("Resolve(Logger) = %v, want [%s]", got, want)
	}

	// import com.example.util.*; -> every other .java file in that dir
	got = r.Resolve(fromFile, "com.example.auth", analyzer.RawImport{Specifier: "com.example.util.*"})
	if len(got) != 2 {
		t.Fatalf("Resolve(util.*) = %v, want 2 files", got)
	}

	// import static com.example.util.Constants.MAX_RETRIES;
	got = r.Resolve(fromFile, "com.example.auth", analyzer.RawImport{
		Specifier: "com.example.util.Constants.MAX_RETRIES",
		Static:    true,
	})
	wantConst := filepath.Join(root, "src", "main", "java", "com", "example", "util", "Constants.java")
	if len(got) != 1 || got[0] != wantConst {
		t.Errorf("Resolve(static Constants.MAX_RETRIES) = %v, want [%s]", got, wantConst)
	}

	// import com.example.util.Missing; -> unresolved
	got = r.Resolve(fromFile, "com.example.auth", analyzer.RawImport{Specifier: "com.example.util.Missing"})
	if got != nil {
		t.Errorf("Resolve(Missing) = %v, want nil", got)
	}

	// java.util.List (JDK standard library) -> unresolved/external
	got = r.Resolve(fromFile, "com.example.auth", analyzer.RawImport{Specifier: "java.util.List"})
	if got != nil {
		t.Errorf("Resolve(java.util.List) = %v, want nil (external)", got)
	}
}

func TestJavaResolver_DefaultPackage(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "Helper.java"), "")
	fromFile := filepath.Join(root, "Main.java")
	mustWriteFile(t, fromFile, "")

	r := repository.NewJavaResolver()

	got := r.Resolve(fromFile, "", analyzer.RawImport{Specifier: "Helper"})
	want := filepath.Join(root, "Helper.java")
	if len(got) != 1 || got[0] != want {
		t.Errorf("Resolve(Helper, default package) = %v, want [%s]", got, want)
	}
}
