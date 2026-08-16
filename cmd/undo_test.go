package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
	"github.com/LHPalma/gitarias/internal/undo"
)

// journalled monta um repositório com um diário já escrito, como o --clean o
// teria deixado, e devolve as respostas prontas para o undo funcionar.
func journalled(t *testing.T, lines string) map[string]gittest.Response {
	t.Helper()

	common := t.TempDir()
	if err := os.MkdirAll(filepath.Join(common, "gtr"), 0o755); err != nil {
		t.Fatalf("não consegui montar o cenário: %v", err)
	}
	if err := os.WriteFile(filepath.Join(common, "gtr", "deleted"), []byte(lines), 0o644); err != nil {
		t.Fatalf("não consegui montar o cenário: %v", err)
	}

	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --git-common-dir":      {Output: common},
		"cat-file -e aaa111^{commit}":     {Output: ""},
		"branch feat-a aaa111":            {Output: ""},
		"cat-file -e bbb222^{commit}":     {Output: ""},
		"branch feat-b bbb222":            {Output: ""},
	}
}

const twoDeleted = "2026-08-14T03:00:00Z\taaa111\tfeat-a\n2026-08-14T03:00:00Z\tbbb222\tfeat-b\n"

func TestUndoRecreatesTheLastBatch(t *testing.T) {
	result := execute(t, journalled(t, twoDeleted), "y\n", "undo")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	for _, wanted := range []string{"feat-a recriada", "feat-b recriada"} {
		if !strings.Contains(result.stdout, wanted) {
			t.Errorf("saída = %q, queria %q", result.stdout, wanted)
		}
	}
}

func TestUndoShowsTheTipsBeforeAsking(t *testing.T) {
	result := execute(t, journalled(t, twoDeleted), "y\n", "undo")

	if !strings.Contains(result.stdout, "aaa111") {
		t.Errorf("saída = %q; a ponta é como o autor reconhece o trabalho que vai voltar", result.stdout)
	}
	if !strings.Contains(result.stdout, "Recriar 2 branches?") {
		t.Errorf("saída = %q, queria a pergunta no plural", result.stdout)
	}
}

func TestUndoAsksBeforeTouchingAnything(t *testing.T) {
	result := execute(t, journalled(t, twoDeleted), "n\n", "undo")

	if result.err != nil {
		t.Fatalf("recusar não é erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Cancelado") {
		t.Errorf("saída = %q, queria dizer que nada foi feito", result.stdout)
	}
	for _, call := range result.calls {
		if strings.HasPrefix(call, "branch ") {
			t.Errorf("chamadas = %v; sem confirmação nenhuma ref é criada", result.calls)
		}
	}
}

func TestUndoWithAnEmptyJournal(t *testing.T) {
	result := execute(t, journalled(t, ""), "", "undo")

	if result.err != nil {
		t.Fatalf("não ter o que desfazer não é erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nada para desfazer") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestUndoOutsideARepository(t *testing.T) {
	if execute(t, map[string]gittest.Response{}, "", "undo").err == nil {
		t.Fatal("fora de um repositório não há branch a recriar")
	}
}

func TestUndoReportsWhatItRefused(t *testing.T) {
	responses := journalled(t, twoDeleted)
	responses["rev-parse --verify --quiet refs/heads/feat-a"] = gittest.Response{Output: "ocupada"}

	result := execute(t, responses, "y\n", "undo")

	if result.err == nil {
		t.Fatal("uma recusa tem de virar código de saída 1")
	}
	if !strings.Contains(result.stderr, "não sobrescreve") {
		t.Errorf("stderr = %q, queria dizer por que recusou", result.stderr)
	}
	if !strings.Contains(result.stdout, "feat-b recriada") {
		t.Errorf("saída = %q, a outra tinha de voltar mesmo assim", result.stdout)
	}
}

func TestUndoReportsThePrunedCommit(t *testing.T) {
	responses := journalled(t, twoDeleted)
	delete(responses, "cat-file -e aaa111^{commit}")

	result := execute(t, responses, "y\n", "undo")

	if !strings.Contains(result.stderr, "já podou") {
		t.Errorf("stderr = %q; sumir o objeto é diferente de já existir o nome", result.stderr)
	}
}

func TestUndoPropagatesAnUnexpectedGitFailure(t *testing.T) {
	responses := journalled(t, twoDeleted)
	responses["branch feat-a aaa111"] = gittest.Response{Err: errors.New("fatal: bad object")}

	result := execute(t, responses, "y\n", "undo")

	if result.err == nil {
		t.Fatal("falha do git tem de virar erro")
	}
	if !strings.Contains(result.stderr, "bad object") {
		t.Errorf("stderr = %q; o que o git disse vale mais que um texto meu", result.stderr)
	}
}

func TestUndoPropagatesTheJournalFailure(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}

	if execute(t, responses, "", "undo").err == nil {
		t.Fatal("sem saber onde fica o diário, o comando não pode seguir")
	}
}

func TestUndoNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, journalled(t, twoDeleted), "y\n", "undo")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestUndoPropagatesTheWriteFailure(t *testing.T) {
	moment := time.Date(2026, 8, 14, 3, 0, 0, 0, time.UTC)
	entries := []undo.Entry{{Deleted: moment, SHA: "aaa111", Name: "feat-a"}}

	if err := announce(brokenWriter{}, entries); err == nil {
		t.Fatal("falha ao escrever o cabeçalho tem de virar erro")
	}
	if err := announce(&countingWriter{allowed: 1}, entries); err == nil {
		t.Fatal("falha ao escrever a lista tem de subir")
	}
}

func TestShortShowsTheSevenCharactersGitWouldShow(t *testing.T) {
	if got := short("0504044a75c080561d677ada198d29ae14c3dbf"); got != "0504044" {
		t.Errorf("sha = %q, queria os sete que o git mostra", got)
	}
	if got := short("abc"); got != "abc" {
		t.Errorf("sha = %q; encurtar o que já é curto não pode cortar informação", got)
	}
}

// undoWith monta o comando à mão para poder quebrar a saída e a entrada, que o
// execute não expõe.
func undoWith(t *testing.T, output io.Writer, input io.Reader) error {
	t.Helper()

	command := NewRootCommand(gittest.NewRunner(journalled(t, twoDeleted)), exectest.NewRunner(), noFinder(), noNotices)
	command.SetOut(output)
	command.SetErr(&bytes.Buffer{})
	command.SetIn(input)
	command.SetArgs([]string{"undo"})

	return command.Execute()
}

func TestUndoPropagatesTheFailureToShowWhatItWouldRecreate(t *testing.T) {
	if err := undoWith(t, brokenWriter{}, strings.NewReader("y\n")); err == nil {
		t.Fatal("não conseguir mostrar o que vai voltar tem de parar antes de perguntar")
	}
}

func TestUndoPropagatesTheFailureToReadTheAnswer(t *testing.T) {
	if err := undoWith(t, &bytes.Buffer{}, brokenReader{}); err == nil {
		t.Fatal("RN-01: entrada ilegível não pode ser lida como um sim")
	}
}

func TestRewindIsTheSameCommand(t *testing.T) {
	byName := execute(t, journalled(t, twoDeleted), "n\n", "undo")
	byAlias := execute(t, journalled(t, twoDeleted), "n\n", "rewind")

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de rodar o comando, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é o mesmo comando", byAlias.stdout, byName.stdout)
	}
}

func TestRewindStaysOutOfTheRootHelp(t *testing.T) {
	result := execute(t, nil, "", "--help")

	if strings.Contains(result.stdout, "rewind") {
		t.Errorf("saída = %q; o apelido é um easter egg e não entra na lista de comandos", result.stdout)
	}
	if !strings.Contains(result.stdout, "undo") {
		t.Errorf("saída = %q, o nome de verdade continua anunciado", result.stdout)
	}
}
