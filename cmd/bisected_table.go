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

type bisectedTable struct {
	base    string
	command []string
	result  commits.BisectResult
	verbose bool
}

func (data bisectedTable) header() []string {
	return []string{"sha", "assunto", "código", "estado", "culpado"}
}

func (data bisectedTable) rows() [][]string {
	rows := make([][]string, 0, len(data.result.Tested))
	for _, tested := range data.result.Tested {
		rows = append(rows, []string{
			tested.Commit.SHA,
			tested.Commit.Subject,
			strconv.Itoa(tested.Code),
			ui.DescribeOutcome(tested),
			yesNo(data.isCulprit(tested)),
		})
	}

	return rows
}

func (data bisectedTable) document() any {
	records := make([]bisectedRecord, 0, len(data.result.Tested))
	for _, tested := range data.result.Tested {
		records = append(records, bisectedRecord{
			SHA:     tested.Commit.SHA,
			Subject: tested.Commit.Subject,
			Code:    tested.Code,
			Outcome: outcomeToken(tested),
			Culprit: data.isCulprit(tested),
		})
	}

	var culprit *commitRecord
	if data.result.Culprit != nil {
		culprit = &commitRecord{
			SHA:     data.result.Culprit.Commit.SHA,
			Subject: data.result.Culprit.Commit.Subject,
			Code:    data.result.Culprit.Code,
			Outcome: outcomeToken(*data.result.Culprit),
		}
	}

	return bisectedDocument{Base: data.base, Command: data.command, Total: data.result.Total, Tested: records, Culprit: culprit}
}

func (data bisectedTable) isCulprit(tested commits.Result) bool {
	return data.result.Culprit != nil && data.result.Culprit.Commit.SHA == tested.Commit.SHA
}

func (data bisectedTable) text(output io.Writer) error {
	if data.result.Total == 0 {
		_, err := fmt.Fprintf(output, "Nenhum commit em %s..HEAD para bisectar.\n", data.base)
		return err
	}

	header := fmt.Sprintf("Bisectando %d %s possíveis: %d %s.\n\n",
		data.result.Total, ui.Plural(data.result.Total, "commit", "commits"),
		len(data.result.Tested), ui.Plural(len(data.result.Tested), "testado", "testados"))

	if _, err := fmt.Fprint(output, header); err != nil {
		return err
	}

	width := outcomeWidth(data.result.Tested)
	for _, tested := range data.result.Tested {
		if err := data.printTested(output, tested, width); err != nil {
			return err
		}
	}

	return data.printConclusion(output)
}

func (data bisectedTable) printTested(output io.Writer, result commits.Result, width int) error {
	label := ui.HighlightOutcome(result)
	padding := strings.Repeat(" ", width-utf8.RuneCountInString(label))

	if _, err := fmt.Fprintf(output, "  %s%s  %s  %s\n", label, padding, result.Commit.Short(), result.Commit.Subject); err != nil {
		return err
	}

	if result.Passed() && !data.verbose {
		return nil
	}

	return printCaptured(output, result.Output)
}

func (data bisectedTable) printConclusion(output io.Writer) error {
	if data.result.Culprit == nil {
		_, err := fmt.Fprintf(output, "\nNenhum dos %d %s falhou.\n",
			len(data.result.Tested), ui.Plural(len(data.result.Tested), "testado", "testados"))
		return err
	}

	culprit := data.result.Culprit
	_, err := fmt.Fprintf(output, "\nPrimeiro commit ruim: %s  %s\n", culprit.Commit.Short(), culprit.Commit.Subject)

	return err
}
