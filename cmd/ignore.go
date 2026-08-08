package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/ignore"
	"github.com/spf13/cobra"
)

func newIgnoreCommand(runner Runner) *cobra.Command {
	command := &cobra.Command{
		Use:   "ignore",
		Short: "Lê o que o repositório ignora",
	}

	command.AddCommand(newIgnoreListCommand(runner))

	return command
}

func newIgnoreListCommand(runner Runner) *cobra.Command {
	var expand bool

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista o que está sendo ignorado, com a regra que ignorou",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runIgnoreList(command, ignore.NewRepo(runner), expand)
		},
	}

	command.Flags().BoolVar(&expand, "expand", false, "lista arquivo a arquivo em vez de colapsar o diretório ignorado")

	return command
}

func runIgnoreList(command *cobra.Command, repo *ignore.Repo, expand bool) error {
	output := command.OutOrStdout()

	if err := repo.Ensure(); err != nil {
		return err
	}

	entries, err := repo.List(expand)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintln(output, "Nada está sendo ignorado aqui.")
		return nil
	}

	if err := printIgnored(output, entries); err != nil {
		return err
	}

	if !expand && anyDirectory(entries) {
		fmt.Fprintln(output, "\nDiretório ignorado conta como uma linha só. Use --expand para listar arquivo a arquivo.")
	}

	return nil
}

func anyDirectory(entries []ignore.Entry) bool {
	for _, entry := range entries {
		if entry.Directory {
			return true
		}
	}

	return false
}

func printIgnored(output io.Writer, entries []ignore.Entry) error {
	fmt.Fprintf(output, "Ignorados (%d):\n", len(entries))

	writer := columns(output)
	for _, entry := range entries {
		fmt.Fprintf(writer, "  %s\t%s:%d\t%s\n", entry.Path, entry.Source, entry.Line, entry.Pattern)
	}

	return writer.Flush()
}
