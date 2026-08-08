package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCommand(runner Runner) *cobra.Command {
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

func Run(command *cobra.Command) int {
	if err := command.Execute(); err != nil {
		fmt.Fprintln(command.ErrOrStderr(), "erro:", err)
		return 1
	}

	return 0
}
