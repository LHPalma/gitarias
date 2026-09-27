package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/config"
	"github.com/LHPalma/gitarias/internal/ui"
)

type configTable struct {
	entries []config.Entry
}

func (data configTable) header() []string {
	return []string{"chave", "valor", "origem"}
}

func (data configTable) rows() [][]string {
	rows := make([][]string, 0, len(data.entries))
	for _, entry := range data.entries {
		rows = append(rows, []string{entry.Key, entry.Value, ui.DescribeConfigSource(entry.Source)})
	}

	return rows
}

func (data configTable) document() any {
	records := make([]configRecord, 0, len(data.entries))
	for _, entry := range data.entries {
		records = append(records, configRecord{Key: entry.Key, Value: entry.Value, Source: entry.Source.String()})
	}

	return configDocument{Config: records}
}

func (data configTable) text(output io.Writer) error {
	writer := columns(output)
	fmt.Fprintln(writer, "CHAVE\tVALOR\tORIGEM")
	for _, entry := range data.entries {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", entry.Key, entry.Value, ui.DescribeConfigSource(entry.Source))
	}

	return writer.Flush()
}
