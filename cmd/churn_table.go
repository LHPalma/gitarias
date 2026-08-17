package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/LHPalma/gitarias/internal/churn"
	"github.com/LHPalma/gitarias/internal/ui"
)

type churnTable struct {
	files []churn.File
}

func (data churnTable) header() []string {
	return []string{"caminho", "commits", "onde"}
}

func (data churnTable) rows() [][]string {
	rows := make([][]string, 0, len(data.files))
	for _, file := range data.files {
		rows = append(rows, []string{
			file.Path,
			strconv.Itoa(file.Commits),
			ui.DescribeResidence(file.InTree),
		})
	}

	return rows
}

func (data churnTable) document() any {
	records := make([]churnRecord, 0, len(data.files))
	for _, file := range data.files {
		records = append(records, churnRecord{Path: file.Path, Commits: file.Commits, InTree: file.InTree})
	}

	return churnDocument{Files: records}
}

func (data churnTable) text(output io.Writer) error {
	if len(data.files) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum arquivo no histórico deste repositório.")
		return err
	}

	writer := columns(&trimmingWriter{output: output})
	for _, file := range data.files {
		fmt.Fprintf(writer, "  %s\t%s\t%s\n",
			ui.Plural(file.Commits, "1 commit", fmt.Sprintf("%d commits", file.Commits)),
			file.Path,
			ui.DescribeResidence(file.InTree))
	}

	return writer.Flush()
}
