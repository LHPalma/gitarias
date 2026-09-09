package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/LHPalma/gitarias/internal/aitrailers"
	"github.com/LHPalma/gitarias/internal/forge"
)

// scannedPullRequest é um pull request e o que casou nele. Guarda o request
// inteiro, e não só o número, porque a saída nomeia o que achou pelo título e
// pela URL — número sozinho obrigaria quem lê a abrir o navegador para saber
// do que se trata.
type scannedPullRequest struct {
	request      forge.PullRequest
	attributions []aitrailers.Attribution
}

type pullRequestScanTable struct {
	scans []scannedPullRequest
}

func (data pullRequestScanTable) header() []string {
	return []string{"número", "ferramenta", "marca", "título"}
}

func (data pullRequestScanTable) rows() [][]string {
	rows := make([][]string, 0, len(data.scans))
	for _, scan := range data.scans {
		rows = append(rows, []string{
			strconv.Itoa(scan.request.Number),
			strings.Join(scannedTools(scan), ", "),
			strings.Join(scannedLines(scan), "; "),
			scan.request.Title,
		})
	}

	return rows
}

func (data pullRequestScanTable) document() any {
	records := make([]scannedPullRequestRecord, 0, len(data.scans))
	for _, scan := range data.scans {
		marks := make([]attributionRecord, 0, len(scan.attributions))
		for _, attribution := range scan.attributions {
			marks = append(marks, attributionRecord{Tool: attribution.Tool, Line: attribution.Line})
		}

		records = append(records, scannedPullRequestRecord{
			Number:       scan.request.Number,
			Title:        scan.request.Title,
			URL:          scan.request.URL,
			Attributions: marks,
		})
	}

	return pullRequestScanDocument{PullRequests: records}
}

// text imprime a linha que casou embaixo de cada pull request. É a evidência,
// e sem ela a saída pediria confiança num casamento que quem lê não pode
// conferir — ainda mais num comando que aceita falso positivo de propósito.
func (data pullRequestScanTable) text(output io.Writer) error {
	if len(data.scans) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum pull request com atribuição de autoria de IA.")
		return err
	}

	fmt.Fprintf(output, "Pull requests com atribuição de autoria de IA (%d):\n", len(data.scans))

	writer := columns(&trimmingWriter{output: output})
	for _, scan := range data.scans {
		fmt.Fprintf(writer, "  #%d\t%s\t%s\n",
			scan.request.Number, strings.Join(scannedTools(scan), ", "), scan.request.Title)

		for _, attribution := range scan.attributions {
			fmt.Fprintf(writer, "      %s\n", attribution.Line)
		}
	}

	return writer.Flush()
}

func scannedTools(scan scannedPullRequest) []string {
	seen := map[string]bool{}
	var tools []string
	for _, attribution := range scan.attributions {
		if seen[attribution.Tool] {
			continue
		}

		seen[attribution.Tool] = true
		tools = append(tools, attribution.Tool)
	}

	return tools
}

func scannedLines(scan scannedPullRequest) []string {
	lines := make([]string, 0, len(scan.attributions))
	for _, attribution := range scan.attributions {
		lines = append(lines, attribution.Line)
	}

	return lines
}
