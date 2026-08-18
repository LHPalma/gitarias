package cmd

import (
	"fmt"
	"time"

	"github.com/LHPalma/gitarias/internal/profile"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
)

type profileOptions struct {
	commitCount bool
	since       string
	until       string
}

func newProfileCommand(runner Runner) *cobra.Command {
	var options profileOptions

	command := &cobra.Command{
		Use:   "profile",
		Short: "Métricas sobre a sua identidade de git neste repositório",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runProfile(command, profile.NewRepo(runner), options)
		},
	}

	command.Flags().BoolVar(&options.commitCount, "commit-count", false, "quantos commits seus caem no período (obrigatória: é a métrica)")
	command.Flags().StringVar(&options.since, "since", "",
		"início do período, AAAA-MM-DD; sem --until, vai até hoje; sem nenhuma das duas, só hoje")
	command.Flags().StringVar(&options.until, "until", "", "fim do período, AAAA-MM-DD; sem --since, começa hoje")

	return command
}

func runProfile(command *cobra.Command, repo *profile.Repo, options profileOptions) error {
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

	output := command.OutOrStdout()

	if since == until {
		_, err = fmt.Fprintf(output, "%d %s em %s.\n", count, ui.Plural(count, "commit", "commits"), since)
	} else {
		_, err = fmt.Fprintf(output, "%d %s entre %s e %s.\n", count, ui.Plural(count, "commit", "commits"), since, until)
	}

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
