package undo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func inRepo(t *testing.T) (*Journal, string) {
	t.Helper()

	common := t.TempDir()
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --git-common-dir": {Output: common},
	})

	return NewJournal(runner), common
}

func read(t *testing.T, common string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(common, directory, file))
	if err != nil {
		t.Fatalf("nao consegui ler o diario: %v", err)
	}

	return string(content)
}

func TestRecordThenLastBringsTheBatchBack(t *testing.T) {
	journal, _ := inRepo(t)
	moment := time.Date(2026, 8, 14, 3, 0, 0, 0, time.UTC)

	wanted := []Entry{
		{Deleted: moment, SHA: "aaa", Name: "feat-a"},
		{Deleted: moment, SHA: "bbb", Name: "feat-b"},
	}

	if err := journal.Record(t.Context(), wanted); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	got, err := journal.Last(t.Context())
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("entradas = %+v, queria as duas", got)
	}
	for index, entry := range wanted {
		if got[index].Name != entry.Name || got[index].SHA != entry.SHA {
			t.Errorf("entrada %d = %+v, queria %+v", index, got[index], entry)
		}
		if !got[index].Deleted.Equal(moment) {
			t.Errorf("instante %v, queria %v", got[index].Deleted, moment)
		}
	}
}

func TestLastBringsOnlyTheNewestBatch(t *testing.T) {
	journal, _ := inRepo(t)
	velho := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	novo := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	if err := journal.Record(t.Context(), []Entry{{Deleted: velho, SHA: "aaa", Name: "antiga"}}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if err := journal.Record(t.Context(), []Entry{
		{Deleted: novo, SHA: "bbb", Name: "recente-a"},
		{Deleted: novo, SHA: "ccc", Name: "recente-b"},
	}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	got, err := journal.Last(t.Context())
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("entradas = %+v; desfazer e desfazer a ultima limpeza, nao todas", got)
	}
	for _, entry := range got {
		if entry.Name == "antiga" {
			t.Error("a limpeza anterior nao entra no lote da ultima")
		}
	}
}

func TestLastKeepsTheOrderOfTheBatch(t *testing.T) {
	journal, _ := inRepo(t)
	moment := time.Now()

	if err := journal.Record(t.Context(), []Entry{
		{Deleted: moment, SHA: "aaa", Name: "primeira"},
		{Deleted: moment, SHA: "bbb", Name: "segunda"},
		{Deleted: moment, SHA: "ccc", Name: "terceira"},
	}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	got, _ := journal.Last(t.Context())

	order := make([]string, 0, len(got))
	for _, entry := range got {
		order = append(order, entry.Name)
	}
	if strings.Join(order, " ") != "primeira segunda terceira" {
		t.Errorf("ordem = %v, queria a mesma da delecao", order)
	}
}

func TestLastWithoutAJournal(t *testing.T) {
	journal, _ := inRepo(t)

	got, err := journal.Last(t.Context())

	if err != nil {
		t.Fatalf("diario que nunca existiu nao e erro, veio %v", err)
	}
	if got != nil {
		t.Errorf("entradas = %+v, queria nenhuma", got)
	}
}

func TestRecordOfNothingDoesNotCreateTheJournal(t *testing.T) {
	journal, common := inRepo(t)

	if err := journal.Record(t.Context(), nil); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if _, err := os.Stat(filepath.Join(common, directory)); !os.IsNotExist(err) {
		t.Error("sem nada a registrar, nada e criado")
	}
}

func TestRecordAppendsInsteadOfOverwriting(t *testing.T) {
	journal, common := inRepo(t)

	for _, name := range []string{"primeira", "segunda"} {
		if err := journal.Record(t.Context(), []Entry{{Deleted: time.Now(), SHA: "aaa", Name: name}}); err != nil {
			t.Fatalf("nao esperava erro, veio %v", err)
		}
	}

	content := read(t, common)
	for _, name := range []string{"primeira", "segunda"} {
		if !strings.Contains(content, name) {
			t.Errorf("diario = %q, perdeu %q: o registro anterior nao pode ser sobrescrito", content, name)
		}
	}
}

func TestJournalLivesUnderTheCommonDirectory(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --git-common-dir": {Output: t.TempDir()},
	})

	if err := NewJournal(runner).Record(t.Context(), []Entry{{Deleted: time.Now(), SHA: "a", Name: "b"}}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if runner.Calls[0] != "rev-parse --git-common-dir" {
		t.Errorf("chamadas = %v; dentro de um worktree o --git-dir aponta para outro lugar, e branch deletada e do repositorio inteiro", runner.Calls)
	}
}

func TestJournalPropagatesTheFailureToLocateTheRepository(t *testing.T) {
	runner := gittest.NewRunner(nil)
	journal := NewJournal(runner)

	if err := journal.Record(t.Context(), []Entry{{Deleted: time.Now(), SHA: "a", Name: "b"}}); err == nil {
		t.Error("sem saber onde fica o diario, gravar tem de falhar")
	}
	if _, err := journal.Last(t.Context()); err == nil {
		t.Error("sem saber onde fica o diario, ler tem de falhar")
	}
}

func TestRecordPropagatesTheWriteFailure(t *testing.T) {
	ocupado := filepath.Join(t.TempDir(), "arquivo")
	if err := os.WriteFile(ocupado, nil, 0o644); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}

	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --git-common-dir": {Output: ocupado},
	})

	if err := NewJournal(runner).Record(t.Context(), []Entry{{Deleted: time.Now(), SHA: "a", Name: "b"}}); err == nil {
		t.Error("diretorio que nao da para criar tem de virar erro")
	}
}

func TestRecordPropagatesTheFailureToOpenTheJournal(t *testing.T) {
	common := t.TempDir()
	if err := os.MkdirAll(filepath.Join(common, directory, file), 0o755); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}

	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --git-common-dir": {Output: common},
	})

	if err := NewJournal(runner).Record(t.Context(), []Entry{{Deleted: time.Now(), SHA: "a", Name: "b"}}); err == nil {
		t.Error("diario que nao da para abrir tem de virar erro")
	}
}

func TestLastPropagatesTheReadFailure(t *testing.T) {
	common := t.TempDir()
	if err := os.MkdirAll(filepath.Join(common, directory, file), 0o755); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}

	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --git-common-dir": {Output: common},
	})

	if _, err := NewJournal(runner).Last(t.Context()); err == nil {
		t.Error("diario que existe e nao da para ler tem de virar erro")
	}
}

func TestParseDiscardsTheLineThatDoesNotServe(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{name: "vazia", line: ""},
		{name: "so o instante", line: "2026-08-14T03:00:00Z"},
		{name: "sem o nome", line: "2026-08-14T03:00:00Z\taaa\t"},
		{name: "sem o sha", line: "2026-08-14T03:00:00Z\t\tfeat"},
		{name: "instante ilegivel", line: "ontem\taaa\tfeat"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, ok := parse(test.line); ok {
				t.Errorf("linha %q nao serve e nao pode ser aceita", test.line)
			}
		})
	}
}

func TestParseKeepsTheRestWhenOneLineIsBroken(t *testing.T) {
	journal, common := inRepo(t)

	if err := os.MkdirAll(filepath.Join(common, directory), 0o755); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}
	content := "linha quebrada\n2026-08-14T03:00:00Z\taaa\tsalva\n"
	if err := os.WriteFile(filepath.Join(common, directory, file), []byte(content), 0o644); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}

	got, err := journal.Last(t.Context())
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(got) != 1 || got[0].Name != "salva" {
		t.Errorf("entradas = %+v; diario meio escrito nao pode impedir de recuperar o que esta bem escrito", got)
	}
}

func TestJournalHonoursCancellation(t *testing.T) {
	journal, _ := inRepo(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := journal.Last(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("erro = %v, queria o cancelamento", err)
	}
}
