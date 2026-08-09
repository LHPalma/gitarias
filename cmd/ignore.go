package cmd

import (
	"github.com/LHPalma/gitarias/internal/ignore"
	"github.com/spf13/cobra"
)

type ignoreListOptions struct {
	formatOptions
	expand bool
}

func newIgnoreCommand(runner Runner) *cobra.Command {
	command := &cobra.Command{
		Use:   "ignore",
		Short: "Lê o que o repositório ignora",
	}

	command.AddCommand(newIgnoreListCommand(runner))

	return command
}

func newIgnoreListCommand(runner Runner) *cobra.Command {
	var options ignoreListOptions

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista o que está sendo ignorado, com a regra que ignorou",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runIgnoreList(command, ignore.NewRepo(runner), options)
		},
	}

	command.Flags().BoolVar(&options.expand, "expand", false, "lista arquivo a arquivo em vez de colapsar o diretório ignorado")
	options.register(command)

	return command
}

func runIgnoreList(command *cobra.Command, repo *ignore.Repo, options ignoreListOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	if err := repo.Ensure(); err != nil {
		return err
	}

	entries, err := repo.List(options.expand)
	if err != nil {
		return err
	}

	data := ignoredTable{entries: entries, expand: options.expand}

	return emit(command.OutOrStdout(), options.output, "ignorados", chosen, data)
}
