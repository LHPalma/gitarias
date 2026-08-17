package churn

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	verifyHead = "rev-parse --verify --quiet HEAD"
	logFormat  = "log --format= --name-only --no-renames -z"
	treeFormat = "ls-tree -r -z --name-only HEAD"
)

var errNotARepository = errors.New("fatal: not a git repository")

func withHistory(touched string, tree string) map[string]gittest.Response {
	return map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: touched},
		treeFormat: {Output: tree},
	}
}

func TestBusiestCountsAndSorts(t *testing.T) {
	runner := gittest.NewRunner(withHistory(
		"main.go\x00README.md\x00main.go\x00main.go\x00",
		"main.go\x00README.md\x00",
	))

	files, err := NewRepo(runner).Busiest(t.Context(), 0)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("arquivos = %v, queria os dois", files)
	}

	first := File{Path: "main.go", Commits: 3, InTree: true}
	if files[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", files[0], first)
	}

	second := File{Path: "README.md", Commits: 1, InTree: true}
	if files[1] != second {
		t.Errorf("segundo = %+v, queria %+v", files[1], second)
	}
}

func TestBusiestTiebreaksByName(t *testing.T) {
	runner := gittest.NewRunner(withHistory("z.go\x00a.go\x00", ""))

	files, err := NewRepo(runner).Busiest(t.Context(), 0)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(files) != 2 || files[0].Path != "a.go" || files[1].Path != "z.go" {
		t.Errorf("arquivos = %v, empate em 1 commit tem de desempatar por nome", files)
	}
}

func TestBusiestMarksWhatLeftTheTree(t *testing.T) {
	runner := gittest.NewRunner(withHistory("build.zip\x00main.go\x00", "main.go\x00"))

	files, err := NewRepo(runner).Busiest(t.Context(), 0)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for _, file := range files {
		if file.Path == "build.zip" && file.InTree {
			t.Errorf("%q saiu da árvore e não podia estar marcado como presente", file.Path)
		}
		if file.Path == "main.go" && !file.InTree {
			t.Errorf("%q continua na árvore e tinha de estar marcado", file.Path)
		}
	}
}

func TestBusiestLimits(t *testing.T) {
	runner := gittest.NewRunner(withHistory("a.go\x00a.go\x00b.go\x00c.go\x00", ""))

	files, err := NewRepo(runner).Busiest(t.Context(), 1)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(files) != 1 || files[0].Path != "a.go" {
		t.Errorf("arquivos = %v, o limite tem de cortar os menores e manter o maior", files)
	}
}

func TestBusiestWithoutLimitBringsEverything(t *testing.T) {
	runner := gittest.NewRunner(withHistory("a.go\x00b.go\x00c.go\x00", ""))

	files, err := NewRepo(runner).Busiest(t.Context(), 0)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(files) != 3 {
		t.Errorf("arquivos = %v, limite zero tem de trazer todos", files)
	}
}

func TestBusiestWithNoCommitsYet(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	files, err := NewRepo(runner).Busiest(t.Context(), 0)
	if err != nil {
		t.Fatalf("repositório sem commit ainda não é erro, veio %v", err)
	}
	if files != nil {
		t.Errorf("arquivos = %v, queria nenhum", files)
	}

	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "log") {
			t.Fatalf("sem HEAD não há o que perguntar ao log, mas rodou %q", call)
		}
	}
}

func TestBusiestPropagatesRevParseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Busiest(t.Context(), 0); err == nil {
		t.Fatal("falha real do rev-parse não pode virar lista vazia")
	}
}

func TestBusiestPropagatesLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Busiest(t.Context(), 0); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestBusiestDegradesWhenTheTreeCannotBeListed(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: "main.go\x00"},
		treeFormat: {Err: errNotARepository},
	})

	files, err := NewRepo(runner).Busiest(t.Context(), 0)
	if err != nil {
		t.Fatalf("falha ao listar a árvore não pode derrubar o comando, veio %v", err)
	}
	if len(files) != 1 || files[0].InTree {
		t.Errorf("arquivos = %v, sem árvore legível nada pode ser marcado como presente", files)
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
