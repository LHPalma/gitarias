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
	commitCount bool
	account     bool
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
			"de fora, caladas — confira com gtr doctor --online.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runProfile(command, profile.NewRepo(runner), forge.NewCLI(commands), options)
		},
	}

	command.Flags().BoolVar(&options.commitCount, "commit-count", false, "quantos commits seus caem no período (obrigatória: é a métrica)")
	command.Flags().BoolVar(&options.account, "account", false,
		"conta em toda a conta do GitHub, não só neste repositório; FAZ CHAMADA DE REDE")
	command.Flags().StringVar(&options.since, "since", "",
		"início do período, AAAA-MM-DD; sem --until, vai até hoje; sem nenhuma das duas, só hoje")
	command.Flags().StringVar(&options.until, "until", "", "fim do período, AAAA-MM-DD; sem --since, começa hoje")

	return command
}

func runProfile(command *cobra.Command, repo *profile.Repo, source forge.Source, options profileOptions) error {
	if !options.commitCount {
		return fmt.Errorf("escolha uma métrica: --commit-count")
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
