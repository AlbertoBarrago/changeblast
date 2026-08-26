package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/changeblast/internal/git"
	"github.com/AlbertoBarrago/changeblast/internal/output"
)

var historyJSON bool

var historyCmd = &cobra.Command{
	Use:   "history <path>",
	Short: "Show Git history signals for a file",
	Long: fmt.Sprintf(`Show churn and co-change frequency for a file, computed over the last
%d days or the last %d commits touching the file, whichever is smaller.`,
		git.HistoryWindowDays, git.HistoryWindowMaxCommits),
	Args: cobra.ExactArgs(1),
	RunE: runHistory,
}

func init() {
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "output machine-readable JSON")
	rootCmd.AddCommand(historyCmd)
}

func runHistory(c *cobra.Command, args []string) error {
	target, root, err := resolveTarget(args[0])
	if err != nil {
		return err
	}

	h, err := git.Analyze(root, target)
	if err != nil {
		return fmt.Errorf("failed to analyze git history: %w", err)
	}

	if historyJSON {
		enc := json.NewEncoder(c.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(output.ToHistoryJSON(root, h))
	}

	output.RenderHistoryText(c.OutOrStdout(), root, h)
	return nil
}
