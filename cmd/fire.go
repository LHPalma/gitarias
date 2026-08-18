package cmd

import (
	"context"
	"fmt"

	"github.com/LHPalma/gitarias/internal/fire"
	"github.com/LHPalma/gitarias/internal/riff"
	"github.com/LHPalma/gitarias/internal/web"
	"github.com/spf13/cobra"
)

// fireFallbackMessage é usada quando o whatthecommit.com falha: o pânico não
// espera a internet.
const fireFallbackMessage = "🔥 fire"

func newFireCommand(runner Runner, client web.Client) *cobra.Command {
	return &cobra.Command{
		Use:     "fire",
		Aliases: []string{"jam"},
		Short:   "Comita e empurra tudo que está sujo, sem perguntar",
		Long: "Comita tudo que está sujo — rastreado ou não — e empurra para uma\n" +
			"branch nova no remoto, sem pedir confirmação: é o botão de pânico,\n" +
			"inspirado no projeto de piada git-fire. A branch local atual também\n" +
			"fica com o commit; só o destino remoto é novo, para não mexer no que\n" +
			"já estava rastreado lá.\n\n" +
			"FAZ CHAMADA DE REDE para a mensagem do commit (whatthecommit.com) — se\n" +
			"ela falhar, usa uma mensagem fixa e segue em frente.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runFire(command, fire.NewRepo(runner), riff.NewRepo(client))
		},
	}
}

func runFire(command *cobra.Command, repo *fire.Repo, messages *riff.Repo) error {
	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	dirty, err := repo.Dirty(ctx)
	if err != nil {
		return err
	}

	output := command.OutOrStdout()

	if !dirty {
		_, err := fmt.Fprintln(output, "Nada sujo para salvar.")
		return err
	}

	message := riffOrFallback(ctx, messages)

	save, err := repo.Save(ctx, message)
	if err != nil {
		return err
	}

	fmt.Fprintf(output, "Salvo em %s: %q\n", save.Branch, message)
	_, err = fmt.Fprintf(output, "Recuperável com: git reset --hard %s\n", save.Head)

	return err
}

// riffOrFallback busca a mensagem do whatthecommit.com; se a rede falhar, o
// pânico não espera — segue com a mensagem fixa.
func riffOrFallback(ctx context.Context, messages *riff.Repo) string {
	riffCtx, cancel := context.WithTimeout(ctx, networkDeadline)
	defer cancel()

	message, err := messages.Random(riffCtx)
	if err != nil {
		return fireFallbackMessage
	}

	return message
}
