package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/AlbertoBarrago/changeblast/internal/output"
)

func TestStripMarkdown_NoColorUnwrapsBold(t *testing.T) {
	var buf bytes.Buffer
	got := output.StripMarkdown(&buf, "The file has a **MEDIUM** risk score (31/100) due to **+28 points**.")

	if strings.Contains(got, "*") {
		t.Errorf("expected no literal asterisks left, got: %q", got)
	}
	if !strings.Contains(got, "MEDIUM risk score") {
		t.Errorf("expected bold text preserved unwrapped, got: %q", got)
	}
}

func TestStripMarkdown_StripsBackticksHeadersBullets(t *testing.T) {
	var buf bytes.Buffer
	got := output.StripMarkdown(&buf, "### Summary\nUse `foo.ts` carefully.\n- first point\n* second point")

	if strings.Contains(got, "`") {
		t.Errorf("expected backticks stripped, got: %q", got)
	}
	if strings.Contains(got, "###") {
		t.Errorf("expected header marker stripped, got: %q", got)
	}
	if !strings.Contains(got, "- first point") || !strings.Contains(got, "- second point") {
		t.Errorf("expected bullets normalized to '- ', got: %q", got)
	}
}

func TestStripMarkdown_PlainTextUnchanged(t *testing.T) {
	var buf bytes.Buffer
	in := "This is plain prose with no markdown at all."
	if got := output.StripMarkdown(&buf, in); got != in {
		t.Errorf("StripMarkdown(plain) = %q, want unchanged %q", got, in)
	}
}
