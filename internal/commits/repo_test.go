package commits

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const insideWorkTree = "rev-parse --is-inside-work-tree"

func logOf(base string, lines string) map[string]gittest.Response {
	return map[string]gittest.Response{
		insideWorkTree: {Output: "true"},
		"log --reverse --format=%H%x00%s " + base + "..HEAD": {Output: lines},
	}
}

type recordingExtractor struct {
	extracted []string
	released  []string
	err       error
}

func (extractor *recordingExtractor) Extract(sha string, destination string) error {
	extractor.extracted = append(extractor.extracted, sha)
	if extractor.err != nil {
		return extractor.err
	}

	return os.MkdirAll(destination, 0o755)
}

func (extractor *recordingExtractor) Release(destination string) {
	extractor.released = append(extractor.released, destination)
}

func TestEnsure(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string]gittest.Response
		wantError bool
	}{
		{
			name:      "dentro de um repositorio",
			responses: map[string]gittest.Response{insideWorkTree: {Output: "true"}},
		},
		{
			name:      "fora de um repositorio",
			responses: map[string]gittest.Response{insideWorkTree: {Err: errors.New("fatal: not a git repository")}},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewRepo(gittest.NewRunner(test.responses), exectest.NewRunner()).Ensure()

			if test.wantError && err == nil {
				t.Fatal("esperava erro, veio nil")
			}
			if !test.wantError && err != nil {
				t.Fatalf("nao esperava erro, veio %v", err)
			}
		})
	}
}

func TestRange(t *testing.T) {
	responses := logOf("main", "aaa\x00primeiro assunto\nbbb\x00segundo assunto")

	found, err := NewRepo(gittest.NewRunner(responses), exectest.NewRunner()).Range("main")

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("commits = %+v, queria dois", found)
	}
	if found[0] != (Commit{SHA: "aaa", Subject: "primeiro assunto"}) {
		t.Errorf("primeiro = %+v", found[0])
	}
	if found[1].Subject != "segundo assunto" {
		t.Errorf("segundo = %+v", found[1])
	}
}

func TestRangeKeepsTheOrderTheGitGave(t *testing.T) {
	responses := logOf("main", "aaa\x00mais antigo\nbbb\x00do meio\nccc\x00mais novo")

	found, _ := NewRepo(gittest.NewRunner(responses), exectest.NewRunner()).Range("main")

	subjects := []string{found[0].Subject, found[1].Subject, found[2].Subject}
	want := []string{"mais antigo", "do meio", "mais novo"}

	for index := range want {
		if subjects[index] != want[index] {
			t.Fatalf("ordem = %v, queria %v; o --reverse e o que faz a varredura ir do antigo para o novo", subjects, want)
		}
	}
}

func TestRangeWithNothingInTheInterval(t *testing.T) {
	found, err := NewRepo(gittest.NewRunner(logOf("main", "")), exectest.NewRunner()).Range("main")

	if err != nil {
		t.Fatalf("intervalo vazio nao e falha, veio %v", err)
	}
	if len(found) != 0 {
		t.Errorf("commits = %+v, queria nenhum", found)
	}
}

func TestRangeDropsATruncatedRecord(t *testing.T) {
	responses := logOf("main", "aaa\x00completo\nsem-separador\n\x00sem-sha")

	found, err := NewRepo(gittest.NewRunner(responses), exectest.NewRunner()).Range("main")

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("commits = %+v, so o registro completo entra", found)
	}
}

func TestRangePropagatesTheGitFailure(t *testing.T) {
	responses := map[string]gittest.Response{insideWorkTree: {Output: "true"}}

	if _, err := NewRepo(gittest.NewRunner(responses), exectest.NewRunner()).Range("inexistente"); err == nil {
		t.Fatal("base desconhecida tem de virar erro")
	}
}

func TestCheckRunsTheCommandOnEveryCommit(t *testing.T) {
	list := []Commit{{SHA: "aaa", Subject: "um"}, {SHA: "bbb", Subject: "dois"}}
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Code: 0, Output: "ok"}},
		exectest.Response{Result: exec.Result{Code: 2, Output: "quebrou"}},
	)
	extractor := &recordingExtractor{}

	results, err := NewRepo(gittest.NewRunner(nil), commands).Check(list, extractor, "go", []string{"test", "./..."})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("resultados = %+v, queria os dois", results)
	}
	if !results[0].Passed() || results[1].Passed() {
		t.Errorf("resultados = %+v, queria o primeiro verde e o segundo vermelho", results)
	}
	if results[1].Code != 2 {
		t.Errorf("codigo = %d, queria o do comando", results[1].Code)
	}
	if results[1].Output != "quebrou" {
		t.Errorf("saida = %q, queria a capturada", results[1].Output)
	}
}

func TestCheckNeverStopsAtTheFirstFailure(t *testing.T) {
	list := []Commit{{SHA: "aaa"}, {SHA: "bbb"}, {SHA: "ccc"}}
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Code: 1}},
		exectest.Response{Result: exec.Result{Code: 1}},
		exectest.Response{Result: exec.Result{Code: 0}},
	)

	results, err := NewRepo(gittest.NewRunner(nil), commands).Check(list, &recordingExtractor{}, "make", nil)

	if err != nil {
		t.Fatalf("commit vermelho nao e falha da varredura, veio %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("resultados = %+v, queria os tres; parar no primeiro vermelho e o que o rebase --exec ja faz", results)
	}
	if !results[2].Passed() {
		t.Errorf("o commit depois dos vermelhos tem de ter rodado, veio %+v", results[2])
	}
}

func TestCheckPassesTheArgumentsThrough(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{})

	_, err := NewRepo(gittest.NewRunner(nil), commands).Check(
		[]Commit{{SHA: "aaa"}}, &recordingExtractor{}, "go", []string{"test", "-run", "Test A"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	call := commands.Calls[0]
	if call.Name != "go" {
		t.Errorf("nome = %q, queria go", call.Name)
	}
	if len(call.Args) != 3 || call.Args[2] != "Test A" {
		t.Errorf("argumentos = %q, o que tem espaco chega inteiro e nao dividido", call.Args)
	}
}

func TestCheckRunsInsideTheExtractedTreeAndCleansUp(t *testing.T) {
	extractor := &recordingExtractor{}
	commands := exectest.NewRunner(exectest.Response{})

	_, err := NewRepo(gittest.NewRunner(nil), commands).Check([]Commit{{SHA: "aaa"}}, extractor, "go", nil)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	directory := commands.Calls[0].Directory
	if !strings.Contains(directory, "aaa") {
		t.Errorf("diretorio = %q, queria o da arvore extraida do commit", directory)
	}
	if len(extractor.released) != 1 {
		t.Errorf("liberacoes = %v, o que foi extraido tem de ser solto", extractor.released)
	}
	if _, err := os.Stat(filepath.Dir(directory)); !os.IsNotExist(err) {
		t.Errorf("o diretorio temporario sobreviveu à varredura: %v", err)
	}
}

func TestCheckPropagatesTheExtractionFailure(t *testing.T) {
	extractor := &recordingExtractor{err: errors.New("sha nao existe")}

	_, err := NewRepo(gittest.NewRunner(nil), exectest.NewRunner()).Check([]Commit{{SHA: "aaa"}}, extractor, "go", nil)

	if err == nil {
		t.Fatal("falha na extracao tem de virar erro")
	}
}

func TestCheckPropagatesTheCommandThatCannotStart(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found")})

	_, err := NewRepo(gittest.NewRunner(nil), commands).Check([]Commit{{SHA: "aaa"}}, &recordingExtractor{}, "inexistente", nil)

	if err == nil {
		t.Fatal("comando que nem comeca e falha da varredura, nao commit vermelho")
	}
}

func TestCheckWithoutCommits(t *testing.T) {
	results, err := NewRepo(gittest.NewRunner(nil), exectest.NewRunner()).Check(nil, &recordingExtractor{}, "go", nil)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(results) != 0 {
		t.Errorf("resultados = %+v, queria nenhum", results)
	}
}

func TestCommitShort(t *testing.T) {
	tests := []struct {
		sha  string
		want string
	}{
		{sha: "0e3dcb1e9e6353173394deb2ec915029a1b2451c", want: "0e3dcb1"},
		{sha: "abc", want: "abc"},
		{sha: "", want: ""},
	}

	for _, test := range tests {
		t.Run(test.sha, func(t *testing.T) {
			if got := (Commit{SHA: test.sha}).Short(); got != test.want {
				t.Errorf("curto = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestCheckPropagatesTheWorkspaceItCannotCreate(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "nao-existe"))

	_, err := NewRepo(gittest.NewRunner(nil), exectest.NewRunner()).Check(
		[]Commit{{SHA: "aaa"}}, &recordingExtractor{}, "go", nil)

	if err == nil {
		t.Fatal("sem diretorio temporario nao ha onde extrair; tem de virar erro")
	}
}
