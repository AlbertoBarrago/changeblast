package output

import (
	"io"
	"os"
)

// colorEnabled reports whether ANSI color output should be used for w,
// per the documented terminal conventions: only when stdout is a TTY,
// and never when the NO_COLOR environment variable is set (see
// https://no-color.org), regardless of TTY state.
func colorEnabled(w io.Writer) bool {
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		return false
	}

	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
	ansiGreen  = "\x1b[32m"
	ansiDim    = "\x1b[2m"
	ansiBold   = "\x1b[1m"
)

// colorize wraps s in the given ANSI code, unless enabled is false.
func colorize(enabled bool, code, s string) string {
	if !enabled {
		return s
	}
	return code + s + ansiReset
}

// StatusSymbol renders a doctor-style status marker (✓/✗/-), colored to
// match its meaning when w is a TTY and NO_COLOR is unset.
type StatusSymbol int

const (
	StatusOK StatusSymbol = iota
	StatusFail
	StatusInfo
)

// Colorize renders symbol (✓, ✗, or -) with the appropriate color for w.
func (s StatusSymbol) Colorize(w io.Writer) string {
	enabled := colorEnabled(w)
	switch s {
	case StatusOK:
		return colorize(enabled, ansiGreen, "✓")
	case StatusFail:
		return colorize(enabled, ansiRed, "✗")
	default:
		return colorize(enabled, ansiDim, "-")
	}
}
