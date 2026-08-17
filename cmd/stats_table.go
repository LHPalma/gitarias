package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/LHPalma/gitarias/internal/stats"
)

type statsTable struct {
	authors []stats.Author
}

func (data statsTable) header() []string {
	return []string{"nome", "email", "commits"}
}

func (data statsTable) rows() [][]string {
	rows := make([][]string, 0, len(data.authors))
	for _, author := range data.authors {
		rows = append(rows, []string{author.Name, author.Email, strconv.Itoa(author.Commits)})
	}

	return rows
}

func (data statsTable) document() any {
	records := make([]authorRecord, 0, len(data.authors))
	for _, author := range data.authors {
		records = append(records, authorRecord{Name: author.Name, Email: author.Email, Commits: author.Commits})
	}

	return statsDocument{Authors: records}
}

func (data statsTable) text(output io.Writer) error {
	if len(data.authors) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum commit encontrado.")
		return err
	}

	fmt.Fprintf(output, "Autores (%d):\n", len(data.authors))

	writer := columns(&trimmingWriter{output: output})
	fmt.Fprintln(writer, "  AUTOR\tCOMMITS")
	for _, author := range data.authors {
		fmt.Fprintf(writer, "  %s <%s>\t%d\n", author.Name, author.Email, author.Commits)
	}

	return writer.Flush()
}
