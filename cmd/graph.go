package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	graphJSON   bool
	graphOutput string
)

var graphCmd = &cobra.Command{
	Use:   "graph <path>",
	Short: "Show the dependency graph around a file",
	Long: `Show what a file imports (dependencies) and what imports it
(dependents), one level in each direction.`,
	Args: cobra.ExactArgs(1),
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().BoolVar(&graphJSON, "json", false, "output machine-readable JSON")
	addOutputFlag(graphCmd, &graphOutput)
	rootCmd.AddCommand(graphCmd)
}

// graphJSONResult is the JSON shape of `blast graph`.
type graphJSONResult struct {
	Target       string   `json:"target"`
	Dependencies []string `json:"dependencies"`
	Dependents   []string `json:"dependents"`
}

func runGraph(c *cobra.Command, args []string) error {
	target, root, err := resolveTarget(args[0])
	if err != nil {
		return err
	}

	w, closeOut, err := openOutputTarget(c.OutOrStdout(), graphOutput)
	if err != nil {
		return err
	}
	defer closeOut()

	g, err := buildGraph(root)
	if err != nil {
		return err
	}

	if !g.HasNode(target) {
		return fmt.Errorf("%q is not a recognized JS/TS module in this repository", args[0])
	}

	deps := g.Dependencies(target)
	dependents := g.Dependents(target)

	rel := func(p string) string {
		if r, err := filepath.Rel(root, p); err == nil {
			return r
		}
		return p
	}

	if graphJSON {
		relDeps := make([]string, len(deps))
		for i, d := range deps {
			relDeps[i] = rel(d)
		}
		relDependents := make([]string, len(dependents))
		for i, d := range dependents {
			relDependents[i] = rel(d)
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(graphJSONResult{
			Target:       rel(target),
			Dependencies: relDeps,
			Dependents:   relDependents,
		})
	}

	fmt.Fprintln(w, "Target")
	fmt.Fprintf(w, "  %s\n\n", rel(target))

	fmt.Fprintln(w, "Depends on")
	if len(deps) == 0 {
		fmt.Fprintln(w, "  (none found)")
	}
	for _, d := range deps {
		fmt.Fprintf(w, "  %s\n", rel(d))
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Depended on by")
	if len(dependents) == 0 {
		fmt.Fprintln(w, "  (none found)")
	}
	for _, d := range dependents {
		fmt.Fprintf(w, "  %s\n", rel(d))
	}

	return nil
}
