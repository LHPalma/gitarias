package ignore

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	collapsedListing = "ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z"
	expandedListing  = "ls-files --others --ignored --exclude-standard -z"
	rules            = "check-ignore -z --stdin -v"
)

var errNotARepository = errors.New("fatal: not a git repository")

func inspected(candidates string, matches string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		collapsedListing:                  {Output: candidates},
		expandedListing:                   {Output: candidates},
		rules:                             {Output: matches},
	}
}

func TestList(t *testing.T) {
	runner := gittest.NewRunner(inspected(
		"app.log\x00node_modules/\x00",
		".gitignore\x002\x00*.log\x00app.log\x00"+
			".git/info/exclude\x001\x00local-only/\x00local-only/\x00",
	))

	entries, err := NewRepo(runner).List(false)
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("entradas = %v, queria as duas do check-ignore", entries)
	}

	first := Entry{Source: ".gitignore", Line: 2, Pattern: "*.log", Path: "app.log"}
	if entries[0] != first {
		t.Errorf("primeira = %+v, queria %+v", entries[0], first)
	}

	second := Entry{Source: ".git/info/exclude", Line: 1, Pattern: "local-only/", Path: "local-only/"}
	if entries[1] != second {
		t.Errorf("segunda = %+v, queria %+v", entries[1], second)
	}
}

func TestListDistinguishesWhoIgnored(t *testing.T) {
	runner := gittest.NewRunner(inspected(
		"a\x00b\x00",
		".gitignore\x001\x00a\x00a\x00"+
			"/home/luiz/.config/git/ignore\x007\x00b\x00b\x00",
	))

	entries, err := NewRepo(runner).List(false)
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if entries[0].Source == entries[1].Source {
		t.Fatal("a fonte da regra tem de separar o que o time ignorou do que a maquina ignorou")
	}
	if entries[1].Source != "/home/luiz/.config/git/ignore" {
		t.Errorf("fonte = %q, queria o core.excludesFile inteiro", entries[1].Source)
	}
}

func TestListCollapsesDirectoriesUnlessExpanded(t *testing.T) {
	tests := []struct {
		name    string
		expand  bool
		wanted  string
		refused string
	}{
		{name: "colapsado por padrao", wanted: collapsedListing, refused: expandedListing},
		{name: "expand lista arquivo a arquivo", expand: true, wanted: expandedListing, refused: collapsedListing},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := gittest.NewRunner(inspected("app.log\x00", ".gitignore\x002\x00*.log\x00app.log\x00"))

			if _, err := NewRepo(runner).List(test.expand); err != nil {
				t.Fatalf("nao esperava erro, veio %v", err)
			}

			var listed, refused bool
			for _, call := range runner.Calls {
				if call == test.wanted {
					listed = true
				}
				if call == test.refused {
					refused = true
				}
			}

			if !listed {
				t.Errorf("chamadas = %v, queria %q", runner.Calls, test.wanted)
			}
			if refused {
				t.Errorf("chamadas = %v, nao podia rodar %q", runner.Calls, test.refused)
			}
		})
	}
}

func TestListFeedsCheckIgnoreThroughStdin(t *testing.T) {
	candidates := "app.log\x00node_modules/\x00"
	runner := gittest.NewRunner(inspected(candidates, ".gitignore\x002\x00*.log\x00app.log\x00"))

	if _, err := NewRepo(runner).List(false); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if got := runner.Inputs[rules]; got != candidates {
		t.Errorf("stdin = %q, queria a lista inteira; caminho em argv estoura o argv em repositorio grande", got)
	}
	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "check-ignore") && strings.Contains(call, "app.log") {
			t.Fatalf("o caminho foi parar em argv: %q", call)
		}
	}
}

func TestListKeepsPathsThatWouldBeQuoted(t *testing.T) {
	runner := gittest.NewRunner(inspected(
		"relatório 2026.csv\x00",
		".gitignore\x004\x00relat*.csv\x00relatório 2026.csv\x00",
	))

	entries, err := NewRepo(runner).List(false)
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("entradas = %v, queria uma so", entries)
	}
	if entries[0].Path != "relatório 2026.csv" {
		t.Errorf("caminho = %q, queria o nome inteiro; sem -z o git citaria o acento", entries[0].Path)
	}
}

func TestListWithNothingIgnored(t *testing.T) {
	responses := inspected("inexistente.txt\x00", "")
	responses[rules] = gittest.Response{Err: &git.ExitError{Code: 1, Message: ""}}

	runner := gittest.NewRunner(responses)

	entries, err := NewRepo(runner).List(false)
	if err != nil {
		t.Fatalf("codigo 1 do check-ignore e resposta, nao falha; veio %v", err)
	}
	if entries != nil {
		t.Errorf("entradas = %v, queria nenhuma", entries)
	}
}

func TestListWithoutCandidatesNeverAsksForRules(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		collapsedListing:                  {Output: ""},
	})

	entries, err := NewRepo(runner).List(false)
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if entries != nil {
		t.Errorf("entradas = %v, queria nenhuma", entries)
	}

	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "check-ignore") {
			t.Fatalf("sem candidato nao ha o que perguntar, mas rodou %q", call)
		}
	}
}

func TestListPropagatesListingFailure(t *testing.T) {
	responses := inspected("", "")
	responses[collapsedListing] = gittest.Response{Err: errNotARepository}

	if _, err := NewRepo(gittest.NewRunner(responses)).List(false); err == nil {
		t.Fatal("falha do ls-files tem de virar erro")
	}
}

func TestListPropagatesRealCheckIgnoreFailure(t *testing.T) {
	responses := inspected("app.log\x00", "")
	responses[rules] = gittest.Response{Err: &git.ExitError{Code: 128, Message: "fatal: not a git repository"}}

	_, err := NewRepo(gittest.NewRunner(responses)).List(false)
	if err == nil {
		t.Fatal("codigo 128 e falha de verdade e nao pode virar lista vazia")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("erro = %v, queria a mensagem do git — RN-07", err)
	}
}

func TestListIgnoresMalformedRecords(t *testing.T) {
	runner := gittest.NewRunner(inspected(
		"a\x00b\x00",
		".gitignore\x00linha\x00a\x00a\x00"+
			".gitignore\x003\x00b\x00b\x00",
	))

	entries, err := NewRepo(runner).List(false)
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("entradas = %v, queria so a que tem numero de linha", entries)
	}
	if entries[0].Path != "b" {
		t.Errorf("caminho = %q, queria que o registro quebrado nao desalinhasse o resto", entries[0].Path)
	}
}

func TestListToleratesOutputWithoutTrailingSeparator(t *testing.T) {
	runner := gittest.NewRunner(inspected("app.log\x00", ".gitignore\x002\x00*.log\x00app.log"))

	entries, err := NewRepo(runner).List(false)
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entradas = %v, queria uma so", entries)
	}
}

func TestListDropsATruncatedRecord(t *testing.T) {
	runner := gittest.NewRunner(inspected(
		"app.log\x00tmp\x00",
		".gitignore\x002\x00*.log\x00app.log\x00"+
			".gitignore\x003\x00*.tmp",
	))

	entries, err := NewRepo(runner).List(false)
	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("entradas = %v, queria so o registro inteiro", entries)
	}
	if entries[0].Path != "app.log" {
		t.Errorf("caminho = %q, queria o do registro completo", entries[0].Path)
	}
}

func TestEnsureRejectsWhatIsNotARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	})

	err := NewRepo(runner).Ensure()
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

	if err := NewRepo(runner).Ensure(); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
}
