package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/doctor"
	"github.com/LHPalma/gitarias/internal/ui"
)

type diagnosisTable struct {
	checks []doctor.Check
}

func (data diagnosisTable) header() []string {
	return []string{"checagem", "estado", "detalhe", "como resolver"}
}

func (data diagnosisTable) rows() [][]string {
	rows := make([][]string, 0, len(data.checks))
	for _, check := range data.checks {
		rows = append(rows, []string{check.Name, ui.DescribeCheck(check), check.Detail, check.Hint})
	}

	return rows
}

func (data diagnosisTable) document() any {
	records := make([]checkRecord, 0, len(data.checks))
	for _, check := range data.checks {
		records = append(records, checkRecord{
			Check:  check.Name,
			State:  check.State.String(),
			Detail: check.Detail,
			Hint:   check.Hint,
		})
	}

	return diagnosisDocument{Checks: records}
}

func (data diagnosisTable) text(output io.Writer) error {
	writer := columns(&trimmingWriter{output: output})

	for _, check := range data.checks {
		fmt.Fprintf(writer, "  %s\t%s\t%s\n", ui.DescribeCheck(check), check.Name, check.Detail)
		if check.Hint != "" {
			fmt.Fprintf(writer, "  \t\t%s\n", check.Hint)
		}
	}

	return writer.Flush()
}

func (data diagnosisTable) failed(strict bool) int {
	failed := 0
	for _, check := range data.checks {
		if check.State == doctor.Failure || (strict && check.State == doctor.Warning) {
			failed++
		}
	}

	return failed
}
