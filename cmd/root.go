package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/platform"
	"github.com/LHPalma/gitarias/internal/web"
	"github.com/spf13/cobra"
)

func NewRootCommand(runner Runner, commands exec.Runner, client web.Client, finder platform.Finder, notices string) *cobra.Command {
	command := &cobra.Command{
		Use:           "gtr",
		Short:         "Utilitários de git pro dia a dia",
		Long:          "gitarias (gtr) — utilitários para as tarefas repetitivas de git local.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	command.AddCommand(newAuthorCommand(runner))
	command.AddCommand(newBlameAICommand(runner))
	command.AddCommand(newBranchesCommand(runner))
	command.AddCommand(newChangelogCommand(runner))
	command.AddCommand(newChurnCommand(runner))
	command.AddCommand(newCommitsCommand(runner, commands))
	command.AddCommand(newDiffCommand(runner))
	command.AddCommand(newDoctorCommand(runner, commands))
	command.AddCommand(newFavoriteBandCommand())
	command.AddCommand(newFireCommand(runner, client))
	command.AddCommand(newIgnoreCommand(runner))
	command.AddCommand(newLicensesCommand(notices))
	command.AddCommand(newProfileCommand(runner))
	command.AddCommand(newPullRequestCommand(runner, commands))
	command.AddCommand(newRiffCommand(client))
	command.AddCommand(newSetupCommand(runner, commands, finder))
	command.AddCommand(newStatsCommand(runner))
	command.AddCommand(newAITrailersCommand(runner))
	command.AddCommand(newAITrailersStripStepCommand(runner))
	command.AddCommand(newUndoCommand(runner))
	command.AddCommand(newWeightCommand(runner))
	command.AddCommand(newWorktreesCommand(runner))

	return command
}

func newFavoriteBandCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "favorite-band",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(command.OutOrStdout(), "Alice In Chains")
			return err
		},
	}
}

func Run(command *cobra.Command) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := command.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(command.ErrOrStderr(), "erro:", describeCancellation(err))
		return 1
	}

	return 0
}
