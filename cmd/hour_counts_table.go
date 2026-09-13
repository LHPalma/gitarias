package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/LHPalma/gitarias/internal/profile"
)

type hourCountsTable struct {
	hours []profile.HourCount
}

func (data hourCountsTable) header() []string {
	return []string{"hora", "commits"}
}

// rows leva a hora crua, sem o "h" que o texto acrescenta — mesma disciplina
// do ui.Bytes, que mostra 2.9 MB na tela e 3000000 no csv: o que vai para
// planilha é o número que ordena e soma.
func (data hourCountsTable) rows() [][]string {
	rows := make([][]string, 0, len(data.hours))
	for _, hour := range data.hours {
		rows = append(rows, []string{strconv.Itoa(hour.Hour), strconv.Itoa(hour.Commits)})
	}

	return rows
}

func (data hourCountsTable) document() any {
	records := make([]hourCountRecord, 0, len(data.hours))
	for _, hour := range data.hours {
		records = append(records, hourCountRecord{Hour: hour.Hour, Commits: hour.Commits})
	}

	return hourCountsDocument{Hours: records}
}

func (data hourCountsTable) text(output io.Writer) error {
	writer := columns(&trimmingWriter{output: output})

	fmt.Fprintln(writer, "  HORA\tCOMMITS")
	for _, hour := range data.hours {
		fmt.Fprintf(writer, "  %02dh\t%d\n", hour.Hour, hour.Commits)
	}

	return writer.Flush()
}
