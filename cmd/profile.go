package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/forge"
	"github.com/LHPalma/gitarias/internal/profile"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
)

type profileOptions struct {
	formatOptions
	commitCount bool
	account     bool
	byRepo      bool
	since       string
	until       string
}

func newProfileCommand(runner Runner, commands exec.Runner) *cobra.Command {
	var options profileOptions

	command := &cobra.Command{
		Use:   "profile",
		Short: "Métricas sobre a sua identidade de git",
		Long: "Métricas sobre a sua identidade de git. Por padrão, local: só o que este\n" +
			"repositório viu, sem sair da máquina.\n\n" +
			"Com --account, --commit-count conta em toda a conta do GitHub, não só\n" +
			"aqui — FAZ CHAMADA DE REDE, pelo gh. A soma vale o que o token consegue\n" +
			"ler: sem o escopo read:user, contribuições de repositório privado ficam\n" +
			"de fora, caladas — confira com gtr doctor --online.\n\n" +
			"--by-repo quebra a soma da conta por repositório, em vez de somar tudo; só\n" +
			"vale com --account. Recusa em vez de trazer cortado quando nem dividindo a\n" +
			"janela até uma hora cabe a atividade da conta.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runProfile(command, profile.NewRepo(runner), forge.NewCLI(commands), options)
		},
	}

	command.Flags().BoolVar(&options.commitCount, "commit-count", false, "quantos commits seus caem no período (obrigatória: é a métrica)")
	command.Flags().BoolVar(&options.account, "account", false,
		"conta em toda a conta do GitHub, não só neste repositório; FAZ CHAMADA DE REDE")
	command.Flags().BoolVar(&options.byRepo, "by-repo", false,
		"quebra a soma de --account por repositório; só vale com --account")
	command.Flags().StringVar(&options.since, "since", "",
		"início do período, AAAA-MM-DD; sem --until, vai até hoje; sem nenhuma das duas, só hoje")
	command.Flags().StringVar(&options.until, "until", "", "fim do período, AAAA-MM-DD; sem --since, começa hoje")
	options.register(command)

	return command
}

func runProfile(command *cobra.Command, repo *profile.Repo, source forge.Source, options profileOptions) error {
	if !options.commitCount {
		return fmt.Errorf("escolha uma métrica: --commit-count")
	}
	if options.byRepo && !options.account {
		return fmt.Errorf("--by-repo só vale com --account")
	}
	if !options.byRepo && changedAnyFormatFlag(command) {
		return fmt.Errorf("--format, --no-header, --output e --separator só valem com --by-repo")
	}

	var chosen rendering
	if options.byRepo {
		resolved, err := options.resolve(command)
		if err != nil {
			return err
		}
		chosen = resolved
	}

	since, until, err := resolvePeriod(options.since, options.until)
	if err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	if options.account {
		if options.byRepo {
			return runAccountCommitCountByRepository(command, repo, source, options, chosen, since, until)
		}

		return runAccountCommitCount(command, repo, source, since, until)
	}

	identity, err := repo.Identity(ctx)
	if err != nil {
		return err
	}
	if identity == "" {
		return fmt.Errorf("configure git config user.name ou user.email para usar o gtr profile")
	}

	count, err := repo.CommitCount(ctx, identity, since, until)
	if err != nil {
		return err
	}

	return printCommitCount(command.OutOrStdout(), count, since, until)
}

// changedAnyFormatFlag existe para recusar --format, --no-header, --output e
// --separator fora de --by-repo em vez de descartá-los calados: --commit-count
// sozinho e --account sozinho imprimem uma linha só, e não há tabela para
// nenhum desses formatar.
func changedAnyFormatFlag(command *cobra.Command) bool {
	for _, name := range []string{"format", "no-header", "output", "separator"} {
		if command.Flags().Changed(name) {
			return true
		}
	}

	return false
}

func printCommitCount(output io.Writer, count int, since string, until string) error {
	var err error
	if since == until {
		_, err = fmt.Fprintf(output, "%d %s em %s.\n", count, ui.Plural(count, "commit", "commits"), since)
	} else {
		_, err = fmt.Fprintf(output, "%d %s entre %s e %s.\n", count, ui.Plural(count, "commit", "commits"), since, until)
	}

	return err
}

// runAccountCommitCount conta pela conta inteira no GitHub, e não pelo git
// local — RN-01 continua valendo: quem autentica o gh é sempre a identidade
// em vigor, nunca uma escolhida aqui. Termina avisando quantos commits deste
// repositório ainda não chegaram ao GitHub, porque esses não entram na soma
// que acabou de sair, e é fácil ler o número de cima como "tudo".
func runAccountCommitCount(command *cobra.Command, repo *profile.Repo, source forge.Source, since string, until string) error {
	start, end := periodBounds(since, until)

	ctx, cancel := context.WithTimeout(command.Context(), networkDeadline)
	defer cancel()

	count, err := source.AccountCommitCount(ctx, start, end)
	if errors.Is(err, forge.ErrUnavailable) {
		return fmt.Errorf("o gtr profile --account fala com o GitHub pelo gh, e ele não está no PATH; rode gtr setup para ver como instalar")
	}
	if err != nil {
		return err
	}

	output := command.OutOrStdout()
	if err := printCommitCount(output, count, since, until); err != nil {
		return err
	}

	return warnAboutUnpushedCommits(command, repo, output)
}

// runAccountCommitCountByRepository é o --by-repo: a mesma fonte do
// runAccountCommitCount, quebrada por repositório em vez de somada — período
// maior que um ano incluso, já que o domínio quebra em janelas, bissecciona
// quando uma janela vem cortada, e soma tudo por repositório sozinho.
func runAccountCommitCountByRepository(command *cobra.Command, repo *profile.Repo, source forge.Source, options profileOptions, chosen rendering, since string, until string) error {
	start, end := periodBounds(since, until)

	ctx, cancel := context.WithTimeout(command.Context(), networkDeadline)
	defer cancel()

	counts, err := source.AccountCommitCountByRepository(ctx, start, end)
	if errors.Is(err, forge.ErrUnavailable) {
		return fmt.Errorf("o gtr profile --account fala com o GitHub pelo gh, e ele não está no PATH; rode gtr setup para ver como instalar")
	}
	if err != nil {
		return err
	}

	output := command.OutOrStdout()
	if err := emit(output, options.output, "commits-por-repositorio", chosen, repositoryCommitCountsTable{counts: counts}); err != nil {
		return err
	}

	return warnAboutUnpushedCommits(command, repo, output)
}

// periodBounds vira since/until em meia-noite e 23:59:59 explícitas, no
// fuso local — mesma disciplina do CommitCount local (RN-02), agora contra
// uma API que exige timestamp e não aceita a data nua. Os dois erros de
// formato já foram descartados pelo resolvePeriod antes de chegar aqui.
func periodBounds(since string, until string) (time.Time, time.Time) {
	start, _ := time.ParseInLocation(dateLayout, since, time.Local)
	end, _ := time.ParseInLocation(dateLayout, until, time.Local)

	return start, end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
}

func warnAboutUnpushedCommits(command *cobra.Command, repo *profile.Repo, output io.Writer) error {
	unpushed, hasUpstream, err := repo.Unpushed(command.Context())
	if err != nil || !hasUpstream || unpushed == 0 {
		return err
	}

	_, err = fmt.Fprintf(output, "%d %s deste repositório ainda %s só %s, fora dessa contagem.\n",
		unpushed, ui.Plural(unpushed, "commit", "commits"), ui.Plural(unpushed, "é", "são"), ui.Plural(unpushed, "local", "locais"))

	return err
}

const dateLayout = "2006-01-02"

// resolvePeriod resolve since/until para AAAA-MM-DD, com hoje como padrão
// independente para cada uma: nenhuma das duas é hoje..hoje, só --since é
// <since>..hoje, só --until é hoje..<until>.
func resolvePeriod(since string, until string) (string, string, error) {
	today := time.Now().Format(dateLayout)

	if since == "" {
		since = today
	}
	if until == "" {
		until = today
	}

	if _, err := time.Parse(dateLayout, since); err != nil {
		return "", "", fmt.Errorf("--since inválida: %q, use AAAA-MM-DD", since)
	}
	if _, err := time.Parse(dateLayout, until); err != nil {
		return "", "", fmt.Errorf("--until inválida: %q, use AAAA-MM-DD", until)
	}

	return since, until, nil
}
