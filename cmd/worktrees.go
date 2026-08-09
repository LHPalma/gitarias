package cmd

import (
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/worktree"
	"github.com/spf13/cobra"
)

func newWorktreesCommand(runner git.Runner) *cobra.Command {
	var options formatOptions

	command := &cobra.Command{
		Use:   "worktrees",
		Short: "Lista os working trees do repositório",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runWorktrees(command, worktree.NewRepo(runner), options)
		},
	}

	options.register(command)

	return command
}

func runWorktrees(command *cobra.Command, repo *worktree.Repo, options formatOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	if err := repo.Ensure(); err != nil {
		return err
	}

	worktrees, err := repo.List()
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "worktrees", chosen, worktreesTable{worktrees: worktrees})
}
