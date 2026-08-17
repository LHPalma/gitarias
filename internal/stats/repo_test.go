package stats

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	verifyHead = "rev-parse --verify --quiet HEAD"
	logFormat  = "log --format=%an%x00%ae"
)

var errNotARepository = errors.New("fatal: not a git repository")

func withHistory(commits string) map[string]gittest.Response {
	return map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: commits},
	}
}

func TestByAuthorCountsAndSorts(t *testing.T) {
	runner := gittest.NewRunner(withHistory(
		"anteninha\x00anteninha@teste.com\n" +
			"NataLia\x00natalia@teste.com\n" +
			"anteninha\x00anteninha@teste.com\n" +
			"anteninha\x00anteninha@teste.com\n",
	))

	authors, err := NewRepo(runner).ByAuthor(t.Context(), nil)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(authors) != 2 {
		t.Fatalf("autores = %v, queria os dois", authors)
	}

	first := Author{Name: "anteninha", Email: "anteninha@teste.com", Commits: 3}
	if authors[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", authors[0], first)
	}

	second := Author{Name: "NataLia", Email: "natalia@teste.com", Commits: 1}
	if authors[1] != second {
		t.Errorf("segundo = %+v, queria %+v", authors[1], second)
	}
}

func TestByAuthorTiebreaksByName(t *testing.T) {
	runner := gittest.NewRunner(withHistory("anteninha\x00anteninha@teste.com\nNataLia\x00natalia@teste.com\n"))

	authors, err := NewRepo(runner).ByAuthor(t.Context(), nil)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(authors) != 2 || authors[0].Name != "NataLia" || authors[1].Name != "anteninha" {
		t.Errorf("autores = %v, empate em 1 commit tem de desempatar por nome — maiúscula pesa menos no ASCII", authors)
	}
}

func TestByAuthorPassesTheFiltersToGit(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat + " --author=anteninha --author=NataLia": {Output: "anteninha\x00anteninha@teste.com\n"},
	})

	if _, err := NewRepo(runner).ByAuthor(t.Context(), []string{"anteninha", "NataLia"}); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}

func TestByAuthorWithNoCommitsYet(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	authors, err := NewRepo(runner).ByAuthor(t.Context(), nil)
	if err != nil {
		t.Fatalf("repositório sem commit ainda não é erro, veio %v", err)
	}
	if authors != nil {
		t.Errorf("autores = %v, queria nenhum", authors)
	}

	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "log") {
			t.Fatalf("sem HEAD não há o que perguntar ao log, mas rodou %q", call)
		}
	}
}

func TestByAuthorPropagatesRevParseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).ByAuthor(t.Context(), nil); err == nil {
		t.Fatal("falha real do rev-parse não pode virar lista vazia")
	}
}

func TestByAuthorPropagatesLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).ByAuthor(t.Context(), nil); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestByAuthorIgnoresMalformedLines(t *testing.T) {
	runner := gittest.NewRunner(withHistory("anteninha\x00anteninha@teste.com\nlinha sem separador\n"))

	authors, err := NewRepo(runner).ByAuthor(t.Context(), nil)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(authors) != 1 {
		t.Fatalf("autores = %v, a linha quebrada não podia contar como autor", authors)
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
