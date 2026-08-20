package cmd

import (
	"fmt"
	"time"

	"github.com/LHPalma/gitarias/internal/aitrailers"
	"github.com/spf13/cobra"
)

type aiTrailersPeriodOptions struct {
	since string
	until string
}

func (options *aiTrailersPeriodOptions) register(command *cobra.Command) {
	command.Flags().StringVar(&options.since, "since", "", "início do período, AAAA-MM-DD; sem ela, sem limite inferior")
	command.Flags().StringVar(&options.until, "until", "", "fim do período, AAAA-MM-DD; sem ela, sem limite superior")
}

// validate aceita since/until vazias — sem elas o list cobre o histórico
// inteiro, e o strip mexe só no HEAD. O formato AAAA-MM-DD é o mesmo do
// gtr profile e do gtr changelog.
func (options aiTrailersPeriodOptions) validate() error {
	if options.since != "" {
		if _, err := time.Parse(dateLayout, options.since); err != nil {
			return fmt.Errorf("--since inválida: %q, use AAAA-MM-DD", options.since)
		}
	}
	if options.until != "" {
		if _, err := time.Parse(dateLayout, options.until); err != nil {
			return fmt.Errorf("--until inválida: %q, use AAAA-MM-DD", options.until)
		}
	}

	return nil
}

func newAITrailersCommand(runner Runner) *cobra.Command {
	command := &cobra.Command{
		Use:     "ai-trailers",
		Aliases: []string{"synth"},
		Short:   "Encontra trailer de autoria de IA no histórico de commits",
	}

	command.AddCommand(newAITrailersListCommand(runner))
	command.AddCommand(newAITrailersStripCommand(runner))

	return command
}

func newAITrailersListCommand(runner Runner) *cobra.Command {
	var options struct {
		formatOptions
		aiTrailersPeriodOptions
	}

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista os commits com trailer de autoria de IA reconhecido",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runAITrailersList(command, aitrailers.NewRepo(runner), options.formatOptions, options.aiTrailersPeriodOptions)
		},
	}

	options.aiTrailersPeriodOptions.register(command)
	options.formatOptions.register(command)

	return command
}

func runAITrailersList(command *cobra.Command, repo *aitrailers.Repo, format formatOptions, period aiTrailersPeriodOptions) error {
	chosen, err := format.resolve(command)
	if err != nil {
		return err
	}

	if err := period.validate(); err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	findings, err := repo.List(ctx, period.since, period.until)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), format.output, "achados", chosen, aiTrailersTable{findings: findings})
}

func newAITrailersStripCommand(runner Runner) *cobra.Command {
	var options aiTrailersPeriodOptions

	command := &cobra.Command{
		Use:   "strip",
		Short: "Remove trailer de autoria de IA dos commits, com confirmação",
		Long: "Remove trailer de autoria de IA dos commits, com confirmação.\n\n" +
			"Sem --since nem --until, mexe só no commit mais recente. Com qualquer\n" +
			"um dos dois, reescreve toda a cadeia do período — cada commit dali\n" +
			"pra frente ganha hash novo, batendo trailer ou não, e o que vem\n" +
			"depois do período também, mesmo sem ter a mensagem tocada: o hash\n" +
			"de um commit inclui o hash do pai, e reescrever um pai cascateia. O\n" +
			"que fica intocado, hash incluso, é só o que já estava antes do\n" +
			"início do período.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runAITrailersStrip(command, aitrailers.NewRepo(runner), options)
		},
	}

	options.register(command)

	return command
}

func runAITrailersStrip(command *cobra.Command, repo *aitrailers.Repo, options aiTrailersPeriodOptions) error {
	if err := options.validate(); err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	plan, err := repo.PlanStrip(ctx, options.since, options.until)
	if err != nil {
		return err
	}

	output := command.OutOrStdout()

	if len(plan.Findings) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum commit com trailer de autoria de IA para remover.")
		return err
	}

	if err := (aiTrailersTable{findings: plan.Findings}).text(output); err != nil {
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
		_, err := fmt.Fprintln(output, "Cancelado, nada foi reescrito.")
		return err
	}

	if err := repo.Strip(ctx, options.since, options.until); err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Pronto.")

	return err
}

// newAITrailersStripStepCommand é o subcomando de encanamento que
// aitrailers.Repo.Strip invoca a cada passo da rebase, um por commit —
// nunca é para uso direto. Oculto pelo mesmo motivo do favorite-band.
func newAITrailersStripStepCommand(runner Runner) *cobra.Command {
	return &cobra.Command{
		Use:    aitrailers.StripStepCommand,
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			_, err := aitrailers.NewRepo(runner).StripHead(command.Context())
			return err
		},
	}
}
