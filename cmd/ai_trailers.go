package cmd

import (
	"github.com/LHPalma/gitarias/internal/aitrailers"
	"github.com/spf13/cobra"
)

func newAITrailersCommand(runner Runner) *cobra.Command {
	command := &cobra.Command{
		Use:     "ai-trailers",
		Aliases: []string{"synth"},
		Short:   "Encontra trailer de autoria de IA no histórico de commits",
	}

	command.AddCommand(newAITrailersListCommand(runner))

	return command
}

func newAITrailersListCommand(runner Runner) *cobra.Command {
	var options formatOptions

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista os commits com trailer de autoria de IA reconhecido",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runAITrailersList(command, aitrailers.NewRepo(runner), options)
		},
	}

	options.register(command)

	return command
}

func runAITrailersList(command *cobra.Command, repo *aitrailers.Repo, options formatOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	findings, err := repo.List(ctx)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "achados", chosen, aiTrailersTable{findings: findings})
}
