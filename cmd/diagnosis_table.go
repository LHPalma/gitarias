package cmd

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

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
	width := labelWidth(data.checks)

	for _, check := range data.checks {
		if err := data.printCheck(output, check, width); err != nil {
			return err
		}
	}

	return nil
}

func (data diagnosisTable) printCheck(output io.Writer, check doctor.Check, width int) error {
	label := ui.DescribeCheck(check)
	padding := strings.Repeat(" ", width-utf8.RuneCountInString(label))

	line := fmt.Sprintf("  %s%s  %s", label, padding, check.Name)
	if check.Detail != "" {
		line += " " + check.Detail
	}

	if _, err := fmt.Fprintln(output, line); err != nil {
		return err
	}

	if check.Hint == "" {
		return nil
	}

	_, err := fmt.Fprintf(output, "  %s  %s\n", strings.Repeat(" ", width), check.Hint)

	return err
}

func labelWidth(checks []doctor.Check) int {
	width := 0
	for _, check := range checks {
		if length := utf8.RuneCountInString(ui.DescribeCheck(check)); length > width {
			width = length
		}
	}

	return width
}

func (data diagnosisTable) failed() int {
	failed := 0
	for _, check := range data.checks {
		if check.State == doctor.Failure {
			failed++
		}
	}

	return failed
}
