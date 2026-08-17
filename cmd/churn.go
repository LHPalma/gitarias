package cmd

import (
	"github.com/LHPalma/gitarias/internal/churn"
	"github.com/spf13/cobra"
)

type churnOptions struct {
	formatOptions
	limit int
}

func newChurnCommand(runner Runner) *cobra.Command {
	var options churnOptions

	command := &cobra.Command{
		Use:   "churn",
		Short: "Mostra quais arquivos mais mudam no histórico do repositório",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runChurn(command, churn.NewRepo(runner), options)
		},
	}

	command.Flags().IntVar(&options.limit, "limit", 10, "quantos caminhos mostrar; 0 traz todos")
	options.register(command)

	return command
}

func runChurn(command *cobra.Command, repo *churn.Repo, options churnOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	files, err := repo.Busiest(ctx, options.limit)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "arquivos", chosen, churnTable{files: files})
}
