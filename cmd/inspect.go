package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/changeblast/internal/impact"
	"github.com/AlbertoBarrago/changeblast/internal/output"
	"github.com/AlbertoBarrago/changeblast/internal/repository"
)

var inspectJSON bool

var inspectCmd = &cobra.Command{
	Use:   "inspect <path>",
	Short: "Analyze the blast radius of a file",
	Long: `Analyze a file's direct and indirect dependents within the JS/TS module
graph. See "blast --help" for the v0.1 module-resolution scope and
limitations.`,
	Args: cobra.ExactArgs(1),
	RunE: runInspect,
}

func init() {
	inspectCmd.Flags().BoolVar(&inspectJSON, "json", false, "output machine-readable JSON")
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(c *cobra.Command, args []string) error {
	target, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("target not found: %s", args[0])
	}

	root, err := repositoryRoot(target)
	if err != nil {
		return err
	}

	scanner, err := repository.NewScanner(root)
	if err != nil {
		return fmt.Errorf("failed to initialize scanner: %w", err)
	}

	g, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan repository: %w", err)
	}

	if !g.HasNode(target) {
		return fmt.Errorf("%s is not a recognized JS/TS module in this repository", args[0])
	}

	result := impact.Compute(g, target)

	if inspectJSON {
		enc := json.NewEncoder(c.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(output.ToInspectJSON(root, result))
	}

	output.RenderInspectText(c.OutOrStdout(), root, result)
	return nil
}

// repositoryRoot walks upward from path looking for a .git directory,
// falling back to the current working directory if none is found.
func repositoryRoot(path string) (string, error) {
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}

	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return os.Getwd()
		}
		dir = parent
	}
}
