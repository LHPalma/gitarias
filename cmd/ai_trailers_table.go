package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/LHPalma/gitarias/internal/aitrailers"
)

type aiTrailersTable struct {
	findings []aitrailers.Finding
}

func (data aiTrailersTable) header() []string {
	return []string{"hash", "ferramenta", "trailer", "assunto"}
}

func (data aiTrailersTable) rows() [][]string {
	rows := make([][]string, 0, len(data.findings))
	for _, finding := range data.findings {
		rows = append(rows, []string{
			finding.Hash,
			strings.Join(toolNames(finding), ", "),
			strings.Join(aiRawTrailers(finding), "; "),
			finding.Subject,
		})
	}

	return rows
}

func (data aiTrailersTable) document() any {
	records := make([]aiFindingRecord, 0, len(data.findings))
	for _, finding := range data.findings {
		trailerRecords := make([]aiTrailerRecord, 0, len(finding.Trailers))
		for _, trailer := range finding.Trailers {
			trailerRecords = append(trailerRecords, aiTrailerRecord{Key: trailer.Key, Value: trailer.Value, Tool: trailer.Tool})
		}

		records = append(records, aiFindingRecord{Hash: finding.Hash, Subject: finding.Subject, Trailers: trailerRecords})
	}

	return aiTrailersDocument{Findings: records}
}

func (data aiTrailersTable) text(output io.Writer) error {
	if len(data.findings) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum commit com trailer de autoria de IA reconhecido.")
		return err
	}

	fmt.Fprintf(output, "Commits com trailer de autoria de IA (%d):\n", len(data.findings))

	writer := columns(&trimmingWriter{output: output})
	fmt.Fprintln(writer, "  HASH\tFERRAMENTA\tASSUNTO")
	for _, finding := range data.findings {
		fmt.Fprintf(writer, "  %s\t%s\t%s\n", shortHash(finding.Hash), strings.Join(toolNames(finding), ", "), finding.Subject)
	}

	return writer.Flush()
}

func toolNames(finding aitrailers.Finding) []string {
	seen := map[string]bool{}
	var names []string
	for _, trailer := range finding.Trailers {
		if seen[trailer.Tool] {
			continue
		}

		seen[trailer.Tool] = true
		names = append(names, trailer.Tool)
	}

	return names
}

func aiRawTrailers(finding aitrailers.Finding) []string {
	lines := make([]string, 0, len(finding.Trailers))
	for _, trailer := range finding.Trailers {
		lines = append(lines, trailer.Key+": "+trailer.Value)
	}

	return lines
}
