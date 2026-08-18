package cmd

import (
	"context"
	"fmt"

	"github.com/LHPalma/gitarias/internal/riff"
	"github.com/LHPalma/gitarias/internal/web"
	"github.com/spf13/cobra"
)

func newRiffCommand(client web.Client) *cobra.Command {
	return &cobra.Command{
		Use:     "riff",
		Aliases: []string{"whatthecommit"},
		Short:   "Uma mensagem de commit aleatória, do whatthecommit.com",
		Long: "Imprime uma mensagem de commit aleatória vinda do whatthecommit.com.\n\n" +
			"FAZ CHAMADA DE REDE — direto por HTTP, sem passar pelo gh. É o único\n" +
			"comando do gtr que fala com a internet sem ser através dele.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runRiff(command, riff.NewRepo(client))
		},
	}
}

func runRiff(command *cobra.Command, repo *riff.Repo) error {
	ctx, cancel := context.WithTimeout(command.Context(), networkDeadline)
	defer cancel()

	message, err := repo.Random(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(command.OutOrStdout(), message)
	return err
}
