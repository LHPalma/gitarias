package cmd

import (
	"fmt"
	"time"

	"github.com/LHPalma/gitarias/internal/redate"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
)

type redateOptions struct {
	date  string
	since string
	until string
}

func newRedateCommand(runner Runner) *cobra.Command {
	var options redateOptions

	command := &cobra.Command{
		Use:     "redate --date <AAAA-MM-DDTHH:MM:SS>",
		Aliases: []string{"reissue"},
		Short:   "Reescreve a data de commits, com confirmação",
		Long: "Reescreve a data de commits — a de autoria e a de committer —, com confirmação.\n\n" +
			"Sem --since nem --until, mexe só no commit mais recente: um\n" +
			"git commit --amend. Com qualquer um dos dois, reescreve todo o\n" +
			"período, e todos os commits dele recebem a mesma data. --since sozinha\n" +
			"vai até hoje; --until sozinha não tem piso.\n\n" +
			"Cada commit dali pra frente ganha hash novo, e o que vem depois do\n" +
			"período também. A hora é lida no fuso local. Com --until, recusa se\n" +
			"houver merge depois do período.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runRedate(command, redate.NewRepo(runner), options)
		},
	}

	command.Flags().StringVar(&options.date, "date", "", "a data a atribuir, AAAA-MM-DDTHH:MM:SS, no fuso local (obrigatório)")
	command.Flags().StringVar(&options.since, "since", "", "início do período, AAAA-MM-DD; sem ela, sem limite inferior")
	command.Flags().StringVar(&options.until, "until", "", "fim do período, AAAA-MM-DD; sem ela, vai até hoje")

	return command
}

func runRedate(command *cobra.Command, repo *redate.Repo, options redateOptions) error {
	if options.date == "" {
		return fmt.Errorf("--date é obrigatória")
	}

	date, err := redate.Parse(options.date)
	if err != nil {
		return err
	}

	if err := validateChangelogPeriod(options.since, options.until); err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	plan, err := repo.Plan(ctx, options.since, options.until)
	if err != nil {
		return err
	}

	output := command.OutOrStdout()

	if len(plan.Commits) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum commit no período para reescrever.")
		return err
	}

	count := len(plan.Commits)
	fmt.Fprintf(output, "Vai trocar a data de %d %s para %s:\n", count, ui.Plural(count, "commit", "commits"), date)
	for _, commit := range plan.Commits {
		fmt.Fprintf(output, "  %s  %s  %s\n", commit.Hash, formatRedateDate(commit.Date), commit.Subject)
	}
	if plan.Tail > 0 {
		fmt.Fprintf(output, "Mais %d %s depois do período serão preservados, só reencaixados em cima.\n",
			plan.Tail, ui.Plural(plan.Tail, "commit", "commits"))
	}
	fmt.Fprintf(output, "Recuperável com: git reset --hard %s\n", plan.Head)

	confirmed, err := confirm(command.InOrStdin(), output, "Confirma? [y/N] ")
	if err != nil {
		return err
	}
	if !confirmed {
		_, err := fmt.Fprintln(output, "Cancelado, nada foi reescrito.")
		return err
	}

	if err := repo.Rewrite(ctx, options.since, options.until, date); err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Pronto.")

	return err
}

// formatRedateDate mostra a data de autoria atual no mesmo formato de --date,
// para a prévia e o valor novo poderem ser lidos lado a lado. Uma data que o
// git devolveu num formato inesperado sai como veio, em vez de esconder o
// commit da prévia.
func formatRedateDate(iso string) string {
	parsed, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}

	return parsed.Format(redate.DateLayout)
}
