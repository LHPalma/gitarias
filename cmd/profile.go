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
	streak      bool
	account     bool
	byRepo      bool
	byHour      bool
	byWeekday   bool
	author      string
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
			"--commit-count conta commits no período; --streak conta dias seguidos com\n" +
			"commit, a sequência em curso e a maior do histórico. Uma por vez.\n\n" +
			"--by-hour e --by-weekday quebram a contagem por hora do dia ou por dia da\n" +
			"semana, no fuso desta máquina. São recortes locais do --commit-count, e\n" +
			"herdam o período dele: sem --since, quebram o dia de hoje.\n\n" +
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

	command.Flags().BoolVar(&options.commitCount, "commit-count", false, "quantos commits seus caem no período (uma métrica é obrigatória)")
	command.Flags().BoolVar(&options.streak, "streak", false,
		"quantos dias seguidos com commit: a sequência em curso e a maior do histórico (uma métrica é obrigatória)")
	command.Flags().StringVar(&options.author, "author", "",
		"conta por este autor em vez da sua identidade, casando substring; só vale com --streak")
	command.Flags().BoolVar(&options.account, "account", false,
		"conta em toda a conta do GitHub, não só neste repositório; FAZ CHAMADA DE REDE")
	command.Flags().BoolVar(&options.byRepo, "by-repo", false,
		"quebra a soma de --account por repositório; só vale com --account")
	command.Flags().BoolVar(&options.byHour, "by-hour", false,
		"quebra a contagem por hora do dia, no fuso desta máquina; só vale com --commit-count")
	command.Flags().BoolVar(&options.byWeekday, "by-weekday", false,
		"quebra a contagem por dia da semana, no fuso desta máquina; só vale com --commit-count")
	command.Flags().StringVar(&options.since, "since", "",
		"início do período, AAAA-MM-DD; sem --until, vai até hoje; sem nenhuma das duas, só hoje")
	command.Flags().StringVar(&options.until, "until", "", "fim do período, AAAA-MM-DD; sem --since, começa hoje")
	options.register(command)

	return command
}

func runProfile(command *cobra.Command, repo *profile.Repo, source forge.Source, options profileOptions) error {
	if err := checkProfileFlags(command, options); err != nil {
		return err
	}

	var chosen rendering
	if options.broken() {
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

	if options.streak {
		return runStreak(command, repo, options)
	}

	if options.byHour || options.byWeekday {
		return runLocalBreakdown(command, repo, options, chosen, since, until)
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

// checkProfileFlags recusa toda combinação que não existe antes de qualquer
// chamada ao git — métrica ausente, as duas métricas juntas, e cada flag
// pedida fora da métrica que a usa. Flag setada de propósito e descartada
// calada é o pior modo de falha.
func checkProfileFlags(command *cobra.Command, options profileOptions) error {
	if !options.commitCount && !options.streak {
		return fmt.Errorf("escolha uma métrica: --commit-count ou --streak")
	}
	if options.commitCount && options.streak {
		return fmt.Errorf("escolha uma métrica só: --commit-count ou --streak")
	}
	if options.byRepo && !options.account {
		return fmt.Errorf("--by-repo só vale com --account")
	}
	if (options.byHour || options.byWeekday) && !options.commitCount {
		return fmt.Errorf("--by-hour e --by-weekday são recortes do --commit-count")
	}
	if options.byHour && options.byWeekday {
		return fmt.Errorf("escolha um recorte só: --by-hour ou --by-weekday")
	}
	if (options.byHour || options.byWeekday) && options.account {
		return fmt.Errorf("--by-hour e --by-weekday são recortes locais; a contagem da conta não traz hora nem dia da semana")
	}
	if !options.broken() && changedAnyFormatFlag(command) {
		return fmt.Errorf("--format, --no-header, --output e --separator só valem com --by-repo, --by-hour ou --by-weekday")
	}
	if options.streak && options.account {
		return fmt.Errorf("--account só vale com --commit-count")
	}
	if options.streak && (command.Flags().Changed("since") || command.Flags().Changed("until")) {
		return fmt.Errorf("--since e --until só valem com --commit-count; a sequência olha o histórico inteiro")
	}
	if command.Flags().Changed("author") && !options.streak {
		return fmt.Errorf("--author só vale com --streak")
	}

	return nil
}

// broken diz se alguma quebra está ligada — é o que decide se existe tabela
// para as flags de formato agirem sobre. As três quebram a mesma contagem em
// eixos diferentes: repositório, hora do dia, dia da semana.
func (options profileOptions) broken() bool {
	return options.byRepo || options.byHour || options.byWeekday
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

// runLocalBreakdown imprime a contagem local quebrada por hora ou por dia da
// semana. Herda o período do --commit-count inteiro, inclusive o padrão de
// hoje: o recorte muda o eixo da resposta, nunca a pergunta.
func runLocalBreakdown(command *cobra.Command, repo *profile.Repo, options profileOptions, chosen rendering, since string, until string) error {
	ctx := command.Context()

	identity, err := repo.Identity(ctx)
	if err != nil {
		return err
	}
	if identity == "" {
		return fmt.Errorf("configure git config user.name ou user.email para usar o gtr profile")
	}

	if options.byHour {
		hours, err := repo.CommitCountByHour(ctx, identity, since, until)
		if err != nil {
			return err
		}

		return emit(command.OutOrStdout(), options.output, "commits-por-hora", chosen, hourCountsTable{hours: hours})
	}

	weekdays, err := repo.CommitCountByWeekday(ctx, identity, since, until)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "commits-por-dia-da-semana", chosen, weekdayCountsTable{weekdays: weekdays})
}

// runStreak conta os dias seguidos com commit. Sem --author, o sujeito é a
// identidade configurada, como em toda métrica deste comando; com ela, é
// quem foi pedido — e aí a identidade local nem chega a ser lida, porque não
// é dela que se está falando.
//
// O relógio é lido uma vez só: duas leituras separadas podem cair em dias
// diferentes se a chamada atravessar a meia-noite, e a sequência passaria a
// ser apurada contra um dia e descrita contra outro.
func runStreak(command *cobra.Command, repo *profile.Repo, options profileOptions) error {
	ctx := command.Context()

	author := options.author
	if author == "" {
		identity, err := repo.Identity(ctx)
		if err != nil {
			return err
		}
		if identity == "" {
			return fmt.Errorf("configure git config user.name ou user.email para usar o gtr profile")
		}

		author = identity
	}

	now := time.Now()

	report, err := repo.Streaks(ctx, author, now)
	if err != nil {
		return err
	}

	return printStreaks(command.OutOrStdout(), report, now)
}

func printStreaks(output io.Writer, report profile.StreakReport, now time.Time) error {
	if report.Longest.Days == 0 {
		_, err := fmt.Fprintln(output, "Nenhum commit encontrado.")

		return err
	}

	if err := printCurrentStreak(output, report, now); err != nil {
		return err
	}

	_, err := fmt.Fprintf(output, "Maior sequência: %s.\n", describeStreak(report.Longest))

	return err
}

// printCurrentStreak nomeia o dia sem commit em vez de deixar a sequência
// parecer interrompida: ela termina ontem enquanto hoje ainda está correndo,
// e é justamente aí que a frase precisa lembrar que falta commitar.
func printCurrentStreak(output io.Writer, report profile.StreakReport, now time.Time) error {
	if report.Current.Days == 0 {
		_, err := fmt.Fprintf(output, "Sequência atual: nenhuma — o último commit foi em %s.\n",
			report.Last.Format(dateLayout))

		return err
	}

	if report.Current.End.Format(dateLayout) == now.Format(dateLayout) {
		_, err := fmt.Fprintf(output, "Sequência atual: %s.\n", describeStreak(report.Current))

		return err
	}

	_, err := fmt.Fprintf(output, "Sequência atual: %s — ainda sem commit hoje.\n", describeStreak(report.Current))

	return err
}

// describeStreak descreve uma sequência sem repetir a data quando ela dura
// um dia só: "1 dia, de X a X" diz duas vezes o que "1 dia, em X" diz uma.
func describeStreak(streak profile.Streak) string {
	days := fmt.Sprintf("%d %s", streak.Days, ui.Plural(streak.Days, "dia", "dias"))

	if streak.Days == 1 {
		return days + ", em " + streak.Start.Format(dateLayout)
	}

	return days + ", de " + streak.Start.Format(dateLayout) + " a " + streak.End.Format(dateLayout)
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
