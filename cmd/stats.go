package cmd

import (
	"github.com/LHPalma/gitarias/internal/stats"
	"github.com/spf13/cobra"
)

type statsOptions struct {
	formatOptions
	authors []string
}

func newStatsCommand(runner Runner) *cobra.Command {
	var options statsOptions

	command := &cobra.Command{
		Use:   "stats",
		Short: "Quantos commits cada autor tem no histórico",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runStats(command, stats.NewRepo(runner), options)
		},
	}

	command.Flags().StringArrayVar(&options.authors, "author", nil,
		"filtra por autor, nome ou e-mail, casando substring; repetível, soma quem casar com qualquer um")
	options.register(command)

	return command
}

func runStats(command *cobra.Command, repo *stats.Repo, options statsOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	authors, err := repo.ByAuthor(ctx, options.authors)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "autores", chosen, statsTable{authors: authors})
}
