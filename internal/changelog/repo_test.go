package changelog

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	verifyHead = "rev-parse --verify --quiet HEAD"
	logFormat  = "log --format=%H%x00%s"
)

var errNotARepository = errors.New("fatal: not a git repository")

func withLog(output string) map[string]gittest.Response {
	return map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: output},
	}
}

func TestEntriesParsesEachOfTheElevenTypes(t *testing.T) {
	subjects := []struct {
		subject string
		kind    Type
	}{
		{"feat: adiciona x", Feat},
		{"fix: corrige y", Fix},
		{"perf: acelera z", Perf},
		{"refactor: reorganiza w", Refactor},
		{"docs: documenta v", Docs},
		{"test: cobre u", Test},
		{"build: ajusta t", Build},
		{"ci: ajusta s", CI},
		{"chore: arruma r", Chore},
		{"style: formata q", Style},
		{"revert: desfaz p", Revert},
	}

	var lines []string
	for index, item := range subjects {
		lines = append(lines, strings.Repeat("a", index+1)+separator+item.subject)
	}

	entries, err := NewRepo(gittest.NewRunner(withLog(strings.Join(lines, "\n")))).Entries(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(entries) != len(subjects) {
		t.Fatalf("entradas = %d, queria %d", len(entries), len(subjects))
	}

	for index, item := range subjects {
		if entries[index].Type != item.kind {
			t.Errorf("assunto %q classificado como %q, queria %q", item.subject, entries[index].Type, item.kind)
		}
	}
}

func TestEntriesParsesScope(t *testing.T) {
	entries, err := NewRepo(gittest.NewRunner(withLog("abc" + separator + "feat(cmd): novo comando"))).Entries(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("entradas = %v, queria uma", entries)
	}

	want := Entry{Hash: "abc", Type: Feat, Scope: "cmd", Subject: "novo comando"}
	if entries[0] != want {
		t.Errorf("entrada = %+v, queria %+v", entries[0], want)
	}
}

func TestEntriesParsesBreakingChange(t *testing.T) {
	entries, err := NewRepo(gittest.NewRunner(withLog("abc" + separator + "feat!: muda a api"))).Entries(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(entries) != 1 || !entries[0].Breaking {
		t.Errorf("entradas = %+v, queria breaking change marcado", entries)
	}
}

func TestEntriesParsesBreakingChangeWithScope(t *testing.T) {
	entries, err := NewRepo(gittest.NewRunner(withLog("abc" + separator + "feat(cmd)!: muda a api"))).Entries(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := Entry{Hash: "abc", Type: Feat, Scope: "cmd", Breaking: true, Subject: "muda a api"}
	if len(entries) != 1 || entries[0] != want {
		t.Errorf("entrada = %+v, queria %+v", entries, want)
	}
}

func TestEntriesFallsBackToMiscellaneous(t *testing.T) {
	subjects := []string{
		"WIP: ainda mexendo",
		"sem prefixo nenhum",
		"tipoinventado: não é dos onze",
	}

	var lines []string
	for index, subject := range subjects {
		lines = append(lines, strings.Repeat("a", index+1)+separator+subject)
	}

	entries, err := NewRepo(gittest.NewRunner(withLog(strings.Join(lines, "\n")))).Entries(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(entries) != len(subjects) {
		t.Fatalf("entradas = %d, queria %d", len(entries), len(subjects))
	}

	for index, subject := range subjects {
		if entries[index].Type != Miscellaneous {
			t.Errorf("assunto %q classificado como %q, queria Miscellaneous", subject, entries[index].Type)
		}
		if entries[index].Subject != subject {
			t.Errorf("subject = %q, o assunto cru não pode ser cortado", entries[index].Subject)
		}
	}
}

func TestEntriesKeepsNewestFirst(t *testing.T) {
	log := "b" + separator + "fix: segundo\n" + "a" + separator + "feat: primeiro"
	entries, err := NewRepo(gittest.NewRunner(withLog(log))).Entries(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(entries) != 2 || entries[0].Hash != "b" || entries[1].Hash != "a" {
		t.Errorf("entradas = %+v, queria a ordem do git log preservada", entries)
	}
}

func TestEntriesIgnoresMalformedLines(t *testing.T) {
	runner := gittest.NewRunner(withLog("a" + separator + "feat: ok\nlinha sem separador"))

	entries, err := NewRepo(runner).Entries(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entradas = %v, a linha quebrada não podia virar entrada", entries)
	}
}

func TestEntriesWithNoCommitsYet(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	entries, err := NewRepo(runner).Entries(t.Context())
	if err != nil {
		t.Fatalf("repositório sem commit ainda não é erro, veio %v", err)
	}
	if entries != nil {
		t.Errorf("entradas = %v, queria nenhuma", entries)
	}

	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "log") {
			t.Fatalf("sem HEAD não há o que perguntar ao log, mas rodou %q", call)
		}
	}
}

func TestEntriesPropagatesRevParseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Entries(t.Context()); err == nil {
		t.Fatal("falha real do rev-parse não pode virar lista vazia")
	}
}

func TestEntriesPropagatesLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Entries(t.Context()); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestEnsureRejectsWhatIsNotARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	})

	err := NewRepo(runner).Ensure(t.Context())
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "não é um repositório git") {
		t.Errorf("erro = %v, queria o do EnsureRepo", err)
	}
}

func TestEnsureAcceptsARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
	})

	if err := NewRepo(runner).Ensure(t.Context()); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}
