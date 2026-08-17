package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/ignore"
	"github.com/spf13/cobra"
)

type ignoreListOptions struct {
	formatOptions
	expand     bool
	expandDirs []string
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
	command.Flags().StringArrayVar(&options.expandDirs, "expand-dir", nil,
		"expande só o(s) diretório(s) informado(s) em vez de colapsar; repetível, incompatível com --expand")
	options.register(command)

	if err := command.RegisterFlagCompletionFunc("expand-dir", completeExpandDir(runner)); err != nil {
		panic(err)
	}

	return command
}

func runIgnoreList(command *cobra.Command, repo *ignore.Repo, options ignoreListOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	if options.expand && len(options.expandDirs) > 0 {
		return fmt.Errorf("--expand já expande tudo; --expand-dir junto seria descartado em silêncio")
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	entries, err := repo.List(ctx, options.expand, options.expandDirs)
	if err != nil {
		return err
	}

	data := ignoredTable{entries: entries, expand: options.expand}

	return emit(command.OutOrStdout(), options.output, "ignorados", chosen, data)
}
