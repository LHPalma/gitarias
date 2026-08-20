package cmd

import (
	"github.com/LHPalma/gitarias/internal/changelog"
	"github.com/spf13/cobra"
)

func newChangelogCommand(runner Runner) *cobra.Command {
	var options formatOptions

	command := &cobra.Command{
		Use:   "changelog",
		Short: "Gera o CHANGELOG.md a partir do histórico, agrupado por tipo do Conventional Commits",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runChangelog(command, changelog.NewRepo(runner), options)
		},
	}

	options.register(command)

	return command
}

func runChangelog(command *cobra.Command, repo *changelog.Repo, options formatOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	entries, err := repo.Entries(ctx)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "changelog", chosen, changelogTable{entries: entries})
}
