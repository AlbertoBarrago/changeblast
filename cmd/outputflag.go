package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// analysisFlags bundles the flags every analysis command shares
// (inspect, diff, and the `serval <path>` root alias). One struct keeps
// registration and access symmetric, mirroring explainFlags.
type analysisFlags struct {
	json         bool
	outputFormat string
	failOn       string
	output       string
}

// addAnalysisFlags registers the shared --json, --output-format,
// --fail-on, and --output flags on c, writing into dest.
func addAnalysisFlags(c *cobra.Command, dest *analysisFlags) {
	c.Flags().BoolVar(&dest.json, "json", false, "output machine-readable JSON (shorthand for --output-format json)")
	c.Flags().StringVar(&dest.outputFormat, "output-format", "", "output format: text (default), json, or sarif")
	c.Flags().StringVar(&dest.failOn, "fail-on", "", "exit with code 2 if risk is at or above this level (low, medium, high)")
	addOutputFlag(c, &dest.output)
}

// resolveFormat reconciles the legacy --json bool with --output-format,
// returning "text", "json", or "sarif". --json is kept as a working
// alias for --output-format json so existing scripts/CI gates built
// against --json aren't broken by the addition of --output-format.
func resolveFormat(f analysisFlags) (string, error) {
	format := f.outputFormat
	if f.json {
		if format != "" && format != "json" {
			return "", fmt.Errorf("--json conflicts with --output-format %s", format)
		}
		format = "json"
	}

	switch format {
	case "":
		return "text", nil
	case "text", "json", "sarif":
		return format, nil
	default:
		return "", fmt.Errorf("unknown --output-format %q (choices: text, json, sarif)", format)
	}
}

// addOutputFlag registers the shared --output/-o flag on c, writing into
// dest. Empty (the default) means stdout.
func addOutputFlag(c *cobra.Command, dest *string) {
	c.Flags().StringVarP(dest, "output", "o", "", "write the report to this file instead of stdout")
}

// openOutputTarget resolves the --output flag: an empty path means
// defaultW (typically the command's stdout), otherwise the named file is
// created (truncated if it exists). The returned close function must
// always be called; it is a no-op for stdout.
func openOutputTarget(defaultW io.Writer, path string) (io.Writer, func() error, error) {
	if path == "" {
		return defaultW, func() error { return nil }, nil
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create output file %q: %w", path, err)
	}
	return f, f.Close, nil
}
