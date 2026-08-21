package cmd

import (
	"bufio"
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/changelog"
	"github.com/LHPalma/gitarias/internal/ui"
)

type changelogTable struct {
	entries []changelog.Entry
	since   string
	until   string
}

// textExtension: o --format text do changelog é Markdown de verdade, não
// texto de tela — --output sem extensão deve gerar CHANGELOG.md, como o
// --help do comando promete, não CHANGELOG.txt.
func (data changelogTable) textExtension() string {
	return ".md"
}

func (data changelogTable) header() []string {
	return []string{"hash", "tipo", "escopo", "breaking", "assunto"}
}

func (data changelogTable) rows() [][]string {
	rows := make([][]string, 0, len(data.entries))
	for _, entry := range data.entries {
		rows = append(rows, []string{entry.Hash, string(entry.Type), entry.Scope, yesNo(entry.Breaking), entry.Subject})
	}

	return rows
}

func (data changelogTable) document() any {
	records := make([]changelogRecord, 0, len(data.entries))
	for _, entry := range data.entries {
		records = append(records, changelogRecord{
			Hash:     entry.Hash,
			Type:     string(entry.Type),
			Scope:    entry.Scope,
			Breaking: entry.Breaking,
			Subject:  entry.Subject,
		})
	}

	return changelogDocument{Entries: records}
}

// text monta o CHANGELOG.md: uma seção "[Unreleased]" — o gtr ainda não
// tagueia release nenhuma —, com uma subseção por tipo do Conventional
// Commits, na ordem de changelog.Types, e só as que têm commit. O que caiu
// em Miscellaneous fecha a lista.
func (data changelogTable) text(output io.Writer) error {
	if len(data.entries) == 0 {
		_, err := fmt.Fprintln(output, data.emptyMessage())
		return err
	}

	grouped := map[changelog.Type][]changelog.Entry{}
	for _, entry := range data.entries {
		grouped[entry.Type] = append(grouped[entry.Type], entry)
	}

	writer := bufio.NewWriter(output)
	fmt.Fprintln(writer, "## [Unreleased]")

	for _, kind := range changelog.Types {
		section := grouped[kind]
		if len(section) == 0 {
			continue
		}

		fmt.Fprintf(writer, "\n### %s\n\n", ui.DescribeSection(kind))
		for _, entry := range section {
			fmt.Fprintln(writer, changelogLine(entry))
		}
	}

	return writer.Flush()
}

// emptyMessage distingue repositório sem commit nenhum de --since/--until
// que não casou com nada — as duas dão a mesma lista vazia do
// internal/changelog.Repo, mas dizer "nenhum commit no histórico deste
// repositório" quando o repositório tem centenas, só fora do período
// pedido, é enganoso: soa como repositório vazio, e não é isso que
// aconteceu.
func (data changelogTable) emptyMessage() string {
	switch {
	case data.since != "" && data.until != "":
		return fmt.Sprintf("Nenhum commit entre %s e %s.", data.since, data.until)
	case data.since != "":
		return fmt.Sprintf("Nenhum commit desde %s.", data.since)
	case data.until != "":
		return fmt.Sprintf("Nenhum commit até %s.", data.until)
	default:
		return "Nenhum commit no histórico deste repositório."
	}
}

func changelogLine(entry changelog.Entry) string {
	line := "- "
	if entry.Breaking {
		line += "⚠ BREAKING: "
	}
	if entry.Scope != "" {
		line += "**" + entry.Scope + ":** "
	}

	return line + entry.Subject + " (`" + shortHash(entry.Hash) + "`)"
}

func shortHash(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}

	return hash
}
