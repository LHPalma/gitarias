package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/aitrailers"
	"github.com/spf13/cobra"
)

func newBlameAICommand(runner Runner) *cobra.Command {
	var commit string
	var tool string

	command := &cobra.Command{
		Use:   "blame-ai",
		Short: "Associa um trailer de IA a um commit, com confirmação",
		Long: "Associa um trailer de IA a um commit, com confirmação — o oposto do\n" +
			"gtr ai-trailers strip: em vez de tirar o crédito, bota.\n\n" +
			"Sem --commit, mexe no commit mais recente. O autor de verdade não\n" +
			"muda — isso só acrescenta um Co-Authored-By, o mesmo que a própria\n" +
			"ferramenta deixaria se tivesse ajudado de verdade, e que o gtr\n" +
			"ai-trailers list já reconheceria de volta.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runBlameAI(command, aitrailers.NewRepo(runner), commit, tool)
		},
	}

	command.Flags().StringVar(&commit, "commit", "",
		"o commit a marcar, por SHA; sem ela, o mais recente. Não funciona no commit raiz, que não tem pai")
	command.Flags().StringVar(&tool, "tool", "", "qual IA associar: claude ou copilot (obrigatória)")

	return command
}

func runBlameAI(command *cobra.Command, repo *aitrailers.Repo, commit string, tool string) error {
	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	plan, err := repo.PlanBlame(ctx, commit, tool)
	if err != nil {
		return err
	}

	output := command.OutOrStdout()

	if _, err := fmt.Fprintf(output, "Vai acrescentar %q ao commit %s (%s).\n",
		plan.Trailer.Key+": "+plan.Trailer.Value, plan.Target, plan.Subject); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Recuperável com: git reset --hard %s\n", plan.Head); err != nil {
		return err
	}

	confirmed, err := confirm(command.InOrStdin(), output, "Confirma? [y/N] ")
	if err != nil {
		return err
	}
	if !confirmed {
		_, err := fmt.Fprintln(output, "Cancelado, nada foi alterado.")
		return err
	}

	if err := repo.Blame(ctx, commit, tool); err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Pronto.")

	return err
}
