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
	noHeader  bool
	output    string
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
	command.Flags().StringVar(&options.format, "format", string(format.Text), "formato da saída: text, csv, tsv ou json")
	command.Flags().BoolVar(&options.noHeader, "no-header", false, "com csv ou tsv, omite a linha de nomes das colunas")
	command.Flags().StringVar(&options.output, "output", "", "grava num arquivo em vez do stdout; sem extensão, a do formato é acrescentada")
	command.Flags().StringVar(&options.separator, "separator", string(format.Comma), "separador do csv: , ; | ou \\t")

	return command
}

func runIgnoreList(command *cobra.Command, repo *ignore.Repo, options ignoreListOptions) error {
	chosen, err := format.Parse(options.format)
	if err != nil {
		return err
	}

	if command.Flags().Changed("no-header") && !chosen.Delimited() {
		return fmt.Errorf("--no-header só vale com --format csv ou tsv, e veio --format %s", chosen)
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

	rendering := ignoredRendering{format: chosen, separator: separator, expand: options.expand, header: !options.noHeader}

	if options.output == "" {
		return render(command.OutOrStdout(), rendering, entries)
	}

	path, err := chosen.Path(options.output)
	if err != nil {
		return err
	}

	return renderToFile(path, rendering, entries)
}

func renderToFile(path string, rendering ignoredRendering, entries []ignore.Entry) error {
	file, err := createFile(path)
	if err != nil {
		return err
	}

	return writeAndClose(file, func(output io.Writer) error {
		return renderForFile(output, rendering, entries)
	})
}

func renderForFile(output io.Writer, rendering ignoredRendering, entries []ignore.Entry) error {
	if rendering.format.Delimited() {
		if err := format.WriteByteOrderMark(output); err != nil {
			return err
		}
	}

	return render(output, rendering, entries)
}

func resolveSeparator(command *cobra.Command, chosen format.Format, options ignoreListOptions) (format.Separator, error) {
	if !command.Flags().Changed("separator") {
		if chosen == format.TSV {
			return format.Tab, nil
		}
		return format.Comma, nil
	}

	if chosen != format.CSV {
		return 0, fmt.Errorf("--separator só vale com --format csv, e veio --format %s", chosen)
	}

	separator, err := format.ParseSeparator(options.separator)
	if err != nil {
		return 0, err
	}

	if separator == format.Tab {
		fmt.Fprintln(command.ErrOrStderr(), "aviso: csv com tabulação é o que --format tsv já faz")
	}

	return separator, nil
}

func render(output io.Writer, rendering ignoredRendering, entries []ignore.Entry) error {
	switch {
	case rendering.format.Delimited():
		return format.WriteCSV(output, rendering.separator, ignoredRows(entries, rendering.header))
	case rendering.format == format.JSON:
		return format.WriteJSON(output, ignoredRecords(entries))
	default:
		return renderIgnoredText(output, rendering.expand, entries)
	}
}

func ignoredRows(entries []ignore.Entry, header bool) [][]string {
	rows := make([][]string, 0, len(entries)+1)
	if header {
		rows = append(rows, []string{"origem", "linha", "padrão", "caminho"})
	}
	for _, entry := range entries {
		rows = append(rows, []string{entry.Source, strconv.Itoa(entry.Line), entry.Pattern, entry.Path})
	}

	return rows
}

func ignoredRecords(entries []ignore.Entry) []ignoredRecord {
	records := make([]ignoredRecord, 0, len(entries))
	for _, entry := range entries {
		records = append(records, ignoredRecord{
			Source:  entry.Source,
			Line:    entry.Line,
			Pattern: entry.Pattern,
			Path:    entry.Path,
		})
	}

	return records
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
