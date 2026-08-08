package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/LHPalma/gitarias/internal/format"
	"github.com/LHPalma/gitarias/internal/ignore"
	"github.com/spf13/cobra"
)

type ignoreListOptions struct {
	expand    bool
	format    string
	separator string
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
	command.Flags().StringVar(&options.format, "format", string(format.Text), "formato da saída: text ou csv")
	command.Flags().StringVar(&options.separator, "separator", string(format.Comma), "separador do csv: , ; | ou \\t")

	return command
}

func runIgnoreList(command *cobra.Command, repo *ignore.Repo, options ignoreListOptions) error {
	chosen, err := format.Parse(options.format)
	if err != nil {
		return err
	}

	separator, err := resolveSeparator(command, chosen, options)
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

	return render(command.OutOrStdout(), chosen, separator, options.expand, entries)
}

func resolveSeparator(command *cobra.Command, chosen format.Format, options ignoreListOptions) (format.Separator, error) {
	if !command.Flags().Changed("separator") {
		return format.Comma, nil
	}

	if chosen != format.CSV {
		return 0, fmt.Errorf("--separator só vale com --format csv, e veio --format %s", chosen)
	}

	return format.ParseSeparator(options.separator)
}

func render(output io.Writer, chosen format.Format, separator format.Separator, expand bool, entries []ignore.Entry) error {
	if chosen == format.CSV {
		return format.WriteCSV(output, separator, ignoredRows(entries))
	}

	return renderIgnoredText(output, expand, entries)
}

func ignoredRows(entries []ignore.Entry) [][]string {
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{entry.Source, strconv.Itoa(entry.Line), entry.Pattern, entry.Path})
	}

	return rows
}

func renderIgnoredText(output io.Writer, expand bool, entries []ignore.Entry) error {
	if len(entries) == 0 {
		_, err := fmt.Fprintln(output, "Nada está sendo ignorado aqui.")
		return err
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
	fmt.Fprintln(writer, "  CAMINHO\tORIGEM\tPADRÃO")
	for _, entry := range entries {
		fmt.Fprintf(writer, "  %s\t%s:%d\t%s\n", entry.Path, entry.Source, entry.Line, entry.Pattern)
	}

	return writer.Flush()
}
