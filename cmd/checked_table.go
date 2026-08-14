package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/LHPalma/gitarias/internal/commits"
	"github.com/LHPalma/gitarias/internal/ui"
)

type checkedTable struct {
	base    string
	command []string
	results []commits.Result
	verbose bool
}

func (data checkedTable) header() []string {
	return []string{"sha", "assunto", "código", "estado"}
}

func (data checkedTable) rows() [][]string {
	rows := make([][]string, 0, len(data.results))
	for _, result := range data.results {
		rows = append(rows, []string{
			result.Commit.SHA,
			result.Commit.Subject,
			strconv.Itoa(result.Code),
			ui.DescribeOutcome(result),
		})
	}

	return rows
}

func (data checkedTable) document() any {
	records := make([]commitRecord, 0, len(data.results))
	for _, result := range data.results {
		records = append(records, commitRecord{
			SHA:     result.Commit.SHA,
			Subject: result.Commit.Subject,
			Code:    result.Code,
			Outcome: outcomeToken(result),
		})
	}

	return checkedDocument{Base: data.base, Command: data.command, Commits: records}
}

func outcomeToken(result commits.Result) string {
	if result.Passed() {
		return "passed"
	}

	return "failed"
}

func (data checkedTable) text(output io.Writer) error {
	if len(data.results) == 0 {
		_, err := fmt.Fprintf(output, "Nenhum commit em %s..HEAD para verificar.\n", data.base)
		return err
	}

	header := fmt.Sprintf("Verificando %d %s sobre %s.\n\n",
		len(data.results), ui.Plural(len(data.results), "commit", "commits"), data.base)

	if _, err := fmt.Fprint(output, header); err != nil {
		return err
	}

	width := outcomeWidth(data.results)
	for _, result := range data.results {
		if err := data.printResult(output, result, width); err != nil {
			return err
		}
	}

	return data.printTally(output)
}

func (data checkedTable) printResult(output io.Writer, result commits.Result, width int) error {
	label := ui.HighlightOutcome(result)
	padding := strings.Repeat(" ", width-utf8.RuneCountInString(label))

	_, err := fmt.Fprintf(output, "  %s%s  %s  %s\n", label, padding, result.Commit.Short(), result.Commit.Subject)
	if err != nil {
		return err
	}

	if result.Passed() && !data.verbose {
		return nil
	}

	return printCaptured(output, result.Output)
}

func printCaptured(output io.Writer, captured string) error {
	for _, line := range strings.Split(strings.TrimRight(captured, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, err := fmt.Fprintf(output, "      %s\n", line); err != nil {
			return err
		}
	}

	return nil
}

func outcomeWidth(results []commits.Result) int {
	width := 0
	for _, result := range results {
		if length := utf8.RuneCountInString(ui.HighlightOutcome(result)); length > width {
			width = length
		}
	}

	return width
}

func (data checkedTable) printTally(output io.Writer) error {
	if data.failed() > 0 {
		return nil
	}

	if len(data.results) == 1 {
		_, err := fmt.Fprintln(output, "\nO commit se sustenta sozinho.")
		return err
	}

	_, err := fmt.Fprintf(output, "\nOs %d se sustentam sozinhos.\n", len(data.results))

	return err
}

func (data checkedTable) failed() int {
	failed := 0
	for _, result := range data.results {
		if !result.Passed() {
			failed++
		}
	}

	return failed
}
