package cmd

import (
	"fmt"
	"time"

	"github.com/LHPalma/gitarias/internal/changelog"
	"github.com/spf13/cobra"
)

type changelogOptions struct {
	formatOptions
	since string
	until string
}

func newChangelogCommand(runner Runner) *cobra.Command {
	var options changelogOptions

	command := &cobra.Command{
		Use:   "changelog",
		Short: "Gera o CHANGELOG.md a partir do histórico, agrupado por tipo do Conventional Commits",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runChangelog(command, changelog.NewRepo(runner), options)
		},
	}

	command.Flags().StringVar(&options.since, "since", "", "início do período, AAAA-MM-DD; sem ela, sem limite inferior")
	command.Flags().StringVar(&options.until, "until", "", "fim do período, AAAA-MM-DD; sem ela, sem limite superior")
	options.register(command)

	return command
}

func runChangelog(command *cobra.Command, repo *changelog.Repo, options changelogOptions) error {
	chosen, err := options.resolve(command)
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

	entries, err := repo.Entries(ctx, options.since, options.until)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "changelog", chosen, changelogTable{entries: entries})
}

// validateChangelogPeriod aceita since/until vazias — sem elas o changelog
// cobre o histórico inteiro, diferente do gtr profile, que assume hoje. O
// formato AAAA-MM-DD é o mesmo do profile; dateLayout é a constante que ele
// já declara, no mesmo pacote.
func validateChangelogPeriod(since string, until string) error {
	if since != "" {
		if _, err := time.Parse(dateLayout, since); err != nil {
			return fmt.Errorf("--since inválida: %q, use AAAA-MM-DD", since)
		}
	}
	if until != "" {
		if _, err := time.Parse(dateLayout, until); err != nil {
			return fmt.Errorf("--until inválida: %q, use AAAA-MM-DD", until)
		}
	}

	return nil
}
