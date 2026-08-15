package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/format"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/LHPalma/gitarias/internal/weight"
	"github.com/spf13/cobra"
)

type weightOptions struct {
	formatOptions
	limit int
}

func newWeightCommand(runner Runner) *cobra.Command {
	var options weightOptions

	command := &cobra.Command{
		Use:     "weight",
		Aliases: []string{"roadie"},
		Short:   "Mostra o que mais pesa no histórico do repositório",
		Long: "Mostra o que mais pesa no histórico, somando todas as versões de cada caminho.\n\n" +
			"Apagar um arquivo não o tira do repositório: o commit em que ele existia continua\n" +
			"no histórico, e todo clone segue baixando aquele objeto. O que saiu da árvore\n" +
			"aparece aqui marcado, porque é onde costuma estar o peso que ninguém explica.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runWeight(command, weight.NewRepo(runner), options)
		},
	}

	command.Flags().IntVar(&options.limit, "limit", 10, "quantos caminhos mostrar; 0 traz todos")
	options.register(command)

	return command
}

func runWeight(command *cobra.Command, repo *weight.Repo, options weightOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	if err := repo.Ensure(command.Context()); err != nil {
		return err
	}

	report, err := repo.Heaviest(command.Context(), options.limit)
	if err != nil {
		return err
	}

	// O total só vai para o stdout no formato de tela. Nos outros ele sujaria a
	// saída contratual — no json ele já está no envelope, como total_bytes.
	headline := fmt.Sprintf("Um clone deste repositório baixa %s.\n", ui.DescribeBytes(report.Total))

	if chosen.format == format.Text {
		fmt.Fprintln(command.OutOrStdout(), headline)
	} else {
		fmt.Fprint(command.ErrOrStderr(), headline)
	}

	return emit(command.OutOrStdout(), options.output, "peso", chosen, weightTable{report: report})
}
