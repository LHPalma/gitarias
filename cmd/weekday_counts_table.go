package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/LHPalma/gitarias/internal/profile"
	"github.com/LHPalma/gitarias/internal/ui"
)

type weekdayCountsTable struct {
	weekdays []profile.WeekdayCount
}

func (data weekdayCountsTable) header() []string {
	return []string{"dia", "commits"}
}

// rows leva o nome do dia em todos os formatos, não o número do enum: o dia
// da semana é rótulo, não grandeza — não ordena nem soma, e "3" obrigaria
// quem abre o csv a saber onde a semana começa.
func (data weekdayCountsTable) rows() [][]string {
	rows := make([][]string, 0, len(data.weekdays))
	for _, weekday := range data.weekdays {
		rows = append(rows, []string{ui.DescribeWeekday(weekday.Weekday), strconv.Itoa(weekday.Commits)})
	}

	return rows
}

func (data weekdayCountsTable) document() any {
	records := make([]weekdayCountRecord, 0, len(data.weekdays))
	for _, weekday := range data.weekdays {
		records = append(records, weekdayCountRecord{
			Weekday: ui.DescribeWeekday(weekday.Weekday),
			Commits: weekday.Commits,
		})
	}

	return weekdayCountsDocument{Weekdays: records}
}

func (data weekdayCountsTable) text(output io.Writer) error {
	writer := columns(&trimmingWriter{output: output})

	fmt.Fprintln(writer, "  DIA\tCOMMITS")
	for _, weekday := range data.weekdays {
		fmt.Fprintf(writer, "  %s\t%d\n", ui.DescribeWeekday(weekday.Weekday), weekday.Commits)
	}

	return writer.Flush()
}
