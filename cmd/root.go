package cmd

import (
	"fmt"
	"os"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/spf13/cobra"
)

func newRootCommand(runner git.Runner) *cobra.Command {
	command := &cobra.Command{
		Use:           "gtr",
		Short:         "Utilitários de git pro dia a dia",
		Long:          "gitarias (gtr) — utilitários para as tarefas repetitivas de git local.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	command.AddCommand(newBranchesCommand(runner))
	command.AddCommand(newWorktreesCommand(runner))

	return command
}

func Execute() {
	command := newRootCommand(git.CommandRunner{})

	if err := command.Execute(); err != nil {
		fmt.Fprintln(command.ErrOrStderr(), "erro:", err)
		os.Exit(1)
	}
}
