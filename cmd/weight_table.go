package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/LHPalma/gitarias/internal/weight"
)

type weightTable struct {
	report weight.Report
}

func (data weightTable) header() []string {
	return []string{"caminho", "bytes", "versões", "onde"}
}

func (data weightTable) rows() [][]string {
	rows := make([][]string, 0, len(data.report.Paths))
	for _, path := range data.report.Paths {
		rows = append(rows, []string{
			path.Name,
			fmt.Sprint(path.Bytes),
			fmt.Sprint(path.Versions),
			ui.DescribeResidence(path.InTree),
		})
	}

	return rows
}

func (data weightTable) document() any {
	records := make([]weightRecord, 0, len(data.report.Paths))
	for _, path := range data.report.Paths {
		records = append(records, weightRecord{
			Path:     path.Name,
			Bytes:    path.Bytes,
			Versions: path.Versions,
			InTree:   path.InTree,
		})
	}

	return weightDocument{Total: data.report.Total, Weight: records}
}

func (data weightTable) text(output io.Writer) error {
	if len(data.report.Paths) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum arquivo no histórico deste repositório.")
		return err
	}

	writer := columns(&trimmingWriter{output: output})
	for _, path := range data.report.Paths {
		fmt.Fprintf(writer, "  %s\t%s\t%s\t%s\n",
			ui.DescribeBytes(path.Bytes),
			ui.Plural(path.Versions, "1 versão", fmt.Sprintf("%d versões", path.Versions)),
			path.Name,
			ui.DescribeResidence(path.InTree))
	}

	return writer.Flush()
}
