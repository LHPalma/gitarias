package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/LHPalma/gitarias/internal/worktree"
	"github.com/spf13/cobra"
)

func newWorktreesCommand(runner git.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "worktrees",
		Short: "Lista os working trees do repositório",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runWorktrees(command, worktree.NewRepo(runner))
		},
	}
}

func runWorktrees(command *cobra.Command, repo *worktree.Repo) error {
	output := command.OutOrStdout()

	if err := repo.Ensure(); err != nil {
		return err
	}

	worktrees, err := repo.List()
	if err != nil {
		return err
	}

	fmt.Fprintf(output, "Working trees (%d):\n", len(worktrees))

	writer := columns(output)
	for _, entry := range worktrees {
		marker := " "
		if entry.Current {
			marker = "*"
		}
		line := fmt.Sprintf("%s %s\t%s", marker, entry.Path, ui.DescribeCheckout(entry))
		if state := ui.DescribeState(entry); state != "" {
			line += "\t" + state
		}
		fmt.Fprintln(writer, line)
	}

	return writer.Flush()
}
