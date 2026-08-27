package strip

import "testing"

func TestComments(t *testing.T) {
	cases := []struct {
		name string
		opts Opts
		in   string
		want string
	}{
		{
			name: "line comment removed, newline kept",
			opts: JavaQuotes,
			in:   "import x; // comment\nimport y;",
			want: "import x; \nimport y;",
		},
		{
			name: "block comment removed",
			opts: JavaQuotes,
			in:   "a /* hidden */ b",
			want: "a  b",
		},
		{
			name: "comment marker inside string kept",
			opts: JavaQuotes,
			in:   `url := "http://x/*y"`,
			want: `url := "http://x/*y"`,
		},
		{
			name: "escaped quote does not end string",
			opts: JavaQuotes,
			in:   `s := "a\" // still string"`,
			want: `s := "a\" // still string"`,
		},
		{
			name: "go raw string has no escapes",
			opts: GoQuotes,
			in:   "a := `\\` // comment\n",
			want: "a := `\\` \n",
		},
		{
			name: "multiline block comment keeps newline structure",
			opts: GoQuotes,
			in:   "a /* x\ny */ b\n",
			want: "a  b\n",
		},
		{
			name: "unterminated block comment swallows rest",
			opts: JavaQuotes,
			in:   "a /* no end",
			want: "a ",
		},
		{
			name: "char literal in java",
			opts: JavaQuotes,
			in:   `char c = '/'; // comment`,
			want: `char c = '/'; `,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Comments(c.in, c.opts); got != c.want {
				t.Errorf("Comments(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
