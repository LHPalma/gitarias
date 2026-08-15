package undo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LHPalma/gitarias/internal/git"
)

const (
	directory = "gtr"
	file      = "deleted"
	separator = "\t"
)

// Journal é o registro do que o gtr deletou, num arquivo dentro do diretório
// do git. Cada linha traz o instante, a ponta e o nome, separados por tab —
// que é seguro porque nome de ref não aceita espaço em branco.
type Journal struct {
	runner git.Runner
}

func NewJournal(runner git.Runner) *Journal {
	return &Journal{runner: runner}
}

// Record acrescenta as entradas ao fim do diário, criando-o se preciso. Todas
// as entradas de uma chamada compartilham o instante, e é isso que as torna um
// lote para o Last.
func (journal *Journal) Record(ctx context.Context, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	path, err := journal.path(ctx)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var lines strings.Builder
	for _, entry := range entries {
		lines.WriteString(strings.Join([]string{
			entry.Deleted.UTC().Format(time.RFC3339),
			entry.SHA,
			entry.Name,
		}, separator))
		lines.WriteString("\n")
	}

	handle, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	// A escrita e o fechamento saem juntos: o Close de um arquivo aberto para
	// escrita pode falhar sozinho, e engoli-lo daria um diário truncado que se
	// diz completo.
	_, err = handle.WriteString(lines.String())

	return errors.Join(err, handle.Close())
}

// Last devolve as entradas do lote mais recente, na ordem em que foram
// gravadas, e nenhuma quando o diário não existe ou está vazio. Lote é o
// conjunto que compartilha o instante mais novo do arquivo.
func (journal *Journal) Last(ctx context.Context) ([]Entry, error) {
	entries, err := journal.all(ctx)
	if err != nil || len(entries) == 0 {
		return nil, err
	}

	newest := entries[0].Deleted
	for _, entry := range entries {
		if entry.Deleted.After(newest) {
			newest = entry.Deleted
		}
	}

	var batch []Entry
	for _, entry := range entries {
		if entry.Deleted.Equal(newest) {
			batch = append(batch, entry)
		}
	}

	return batch, nil
}

func (journal *Journal) all(ctx context.Context) ([]Entry, error) {
	path, err := journal.path(ctx)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, line := range strings.Split(string(content), "\n") {
		if entry, ok := parse(line); ok {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (journal *Journal) path(ctx context.Context) (string, error) {
	common, err := journal.runner.Run(ctx, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}

	return filepath.Join(common, directory, file), nil
}

// parse lê uma linha do diário e informa se ela serve. Linha truncada ou com
// instante ilegível é descartada em silêncio: um diário meio escrito não pode
// impedir de recuperar o que está bem escrito nas outras linhas.
func parse(line string) (Entry, bool) {
	fields := strings.SplitN(strings.TrimRight(line, "\r"), separator, 3)
	if len(fields) != 3 || fields[1] == "" || fields[2] == "" {
		return Entry{}, false
	}

	deleted, err := time.Parse(time.RFC3339, fields[0])
	if err != nil {
		return Entry{}, false
	}

	return Entry{Deleted: deleted, SHA: fields[1], Name: fields[2]}, true
}
