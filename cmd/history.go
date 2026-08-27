package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/serval/internal/config"
	"github.com/AlbertoBarrago/serval/internal/git"
	"github.com/AlbertoBarrago/serval/internal/output"
)

var (
	historyJSON   bool
	historyOutput string
)

var historyCmd = &cobra.Command{
	Use:   "history <path>",
	Short: "Show Git history signals for a file",
	Long: fmt.Sprintf(`Show churn and co-change frequency for a file or directory, computed
over the last %d days or the last %d commits touching it, whichever is
smaller. <path> defaults to "." (the current directory) if omitted.`,
		git.HistoryWindowDays, git.HistoryWindowMaxCommits),
	Args: cobra.MaximumNArgs(1),
	RunE: runHistory,
}

func init() {
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "output machine-readable JSON")
	addOutputFlag(historyCmd, &historyOutput)
	rootCmd.AddCommand(historyCmd)
}

func runHistory(c *cobra.Command, args []string) error {
	target, root, err := resolveTarget(targetArg(args))
	if err != nil {
		return err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	window := git.Window{
		Days:       cfg.HistoryWindowDaysOr(git.HistoryWindowDays),
		MaxCommits: cfg.HistoryWindowMaxCommitsOr(git.HistoryWindowMaxCommits),
	}

	h, err := git.AnalyzeWithWindow(root, target, window)
	if err != nil {
		return fmt.Errorf("failed to analyze git history: %w", err)
	}

	w, closeOut, err := openOutputTarget(c.OutOrStdout(), historyOutput)
	if err != nil {
		return err
	}
	defer closeOut()

	if historyJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(output.ToHistoryJSON(root, h))
	}

	output.RenderHistoryText(w, root, h)
	return nil
}
