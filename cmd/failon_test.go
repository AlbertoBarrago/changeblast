package cmd

import (
	"testing"

	"github.com/AlbertoBarrago/serval/internal/risk"
)

func TestNormalizeLevel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"low", string(risk.LevelLow)},
		{"LOW", string(risk.LevelLow)},
		{"Low", string(risk.LevelLow)},
		{"  HIGH  ", string(risk.LevelHigh)},
		{"Medium", string(risk.LevelMedium)},
		{"bogus", "bogus"},
	}
	for _, c := range cases {
		if got := normalizeLevel(c.in); got != c.want {
			t.Errorf("normalizeLevel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidateFailOn(t *testing.T) {
	valid := []string{"", "low", "HIGH", " Medium ", "high"}
	for _, v := range valid {
		if err := validateFailOn(v); err != nil {
			t.Errorf("validateFailOn(%q) = %v, want nil", v, err)
		}
	}
	invalid := []string{"hight", "critical", "1"}
	for _, v := range invalid {
		if err := validateFailOn(v); err == nil {
			t.Errorf("validateFailOn(%q) = nil, want error", v)
		}
	}
}

func TestApplyFailOn(t *testing.T) {
	cases := []struct {
		failOn string
		level  risk.Level
		want   bool // whether a failOnError is expected
	}{
		{"", risk.LevelHigh, false},
		{"high", risk.LevelMedium, false},
		{"high", risk.LevelHigh, true},
		{"medium", risk.LevelMedium, true},
		{"low", risk.LevelLow, true},
		{"low", risk.LevelMedium, true},
	}
	for _, c := range cases {
		err := applyFailOn(c.failOn, c.level)
		if _, ok := err.(failOnError); ok != c.want {
			t.Errorf("applyFailOn(%q, %s) err=%v, want failOnError=%v", c.failOn, c.level, err, c.want)
		}
	}

	if err := applyFailOn("bogus", risk.LevelLow); err == nil {
		t.Error("applyFailOn with invalid threshold should error")
	}
}
