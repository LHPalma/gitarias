package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/LHPalma/gitarias/internal/ignore"
)

type ignoredTable struct {
	entries []ignore.Entry
	expand  bool
}

func (data ignoredTable) header() []string {
	return []string{"origem", "linha", "padrão", "caminho"}
}

func (data ignoredTable) rows() [][]string {
	rows := make([][]string, 0, len(data.entries))
	for _, entry := range data.entries {
		rows = append(rows, []string{entry.Source, strconv.Itoa(entry.Line), entry.Pattern, entry.Path})
	}

	return rows
}

func (data ignoredTable) document() any {
	records := make([]ignoredRecord, 0, len(data.entries))
	for _, entry := range data.entries {
		records = append(records, ignoredRecord{
			Source:  entry.Source,
			Line:    entry.Line,
			Pattern: entry.Pattern,
			Path:    entry.Path,
		})
	}

	return ignoredDocument{Ignored: records}
}

func (data ignoredTable) text(output io.Writer) error {
	if len(data.entries) == 0 {
		_, err := fmt.Fprintln(output, "Nada está sendo ignorado aqui.")
		return err
	}

	if err := printIgnored(output, data.entries); err != nil {
		return err
	}

	if !data.expand && anyDirectory(data.entries) {
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
