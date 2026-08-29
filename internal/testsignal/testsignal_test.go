package testsignal_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/testsignal"
)

func write(t *testing.T, root, relPath string) {
	t.Helper()
	full := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHasCorrelatedTest(t *testing.T) {
	cases := []struct {
		name       string
		target     string
		extraFiles []string
		want       bool
	}{
		{"go with test", "internal/auth/token.go", []string{"internal/auth/token_test.go"}, true},
		{"go without test", "internal/auth/token.go", nil, false},
		{"go is itself a test", "internal/auth/token_test.go", nil, true},

		{"ts with .test sibling", "src/auth/token.ts", []string{"src/auth/token.test.ts"}, true},
		{"ts with .spec sibling", "src/auth/token.ts", []string{"src/auth/token.spec.ts"}, true},
		{"ts with __tests__ dir", "src/auth/token.ts", []string{"src/auth/__tests__/token.ts"}, true},
		{"ts without test", "src/auth/token.ts", nil, false},
		{"ts is itself a test", "src/auth/token.test.ts", nil, true},
		{"ts inside __tests__", "src/auth/__tests__/token.ts", nil, true},

		{"py with test_ prefix sibling", "pkg/auth.py", []string{"pkg/test_auth.py"}, true},
		{"py with _test suffix sibling", "pkg/auth.py", []string{"pkg/auth_test.py"}, true},
		{"py with tests/ dir", "pkg/auth.py", []string{"pkg/tests/test_auth.py"}, true},
		{"py without test", "pkg/auth.py", nil, false},
		{"py is itself a test", "pkg/test_auth.py", nil, true},

		{"java with mirrored test", "src/main/java/com/x/Auth.java", []string{"src/test/java/com/x/AuthTest.java"}, true},
		{"java without test", "src/main/java/com/x/Auth.java", nil, false},
		{"java is itself a test", "src/main/java/com/x/AuthTest.java", nil, true},
		{"java outside maven layout", "pkg/Auth.java", nil, true},

		{"c always true", "src/auth.c", nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, f := range tc.extraFiles {
				write(t, root, f)
			}
			got := testsignal.HasCorrelatedTest(root, tc.target)
			if got != tc.want {
				t.Errorf("HasCorrelatedTest(%q) = %v, want %v", tc.target, got, tc.want)
			}
		})
	}
}
