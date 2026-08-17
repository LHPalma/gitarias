package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	churnLog  = "log --format= --name-only --no-renames -z"
	churnTree = "ls-tree -r -z --name-only HEAD"
)

func churned() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --verify --quiet HEAD": {Output: "abc123"},
		churnLog:                          {Output: "main.go\x00README.md\x00main.go\x00"},
		churnTree:                         {Output: "main.go\x00README.md\x00"},
	}
}

func TestChurnListsTheBusiestFirst(t *testing.T) {
	result := execute(t, churned(), "", "churn")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	lines := strings.Split(strings.TrimRight(result.stdout, "\n"), "\n")
	if !strings.Contains(lines[0], "main.go") {
		t.Errorf("saída:\n%s\no mais tocado tem de vir primeiro", result.stdout)
	}
	if !strings.Contains(lines[0], "2 commits") {
		t.Errorf("saída = %q, queria a contagem no plural", result.stdout)
	}
	if !strings.Contains(lines[1], "1 commit") {
		t.Errorf("saída = %q, queria o singular para um só", result.stdout)
	}
}

func TestChurnMarksWhatLeftTheTree(t *testing.T) {
	responses := churned()
	responses[churnLog] = gittest.Response{Output: "build.zip\x00main.go\x00"}
	responses[churnTree] = gittest.Response{Output: "main.go\x00"}

	result := execute(t, responses, "", "churn")

	if !strings.Contains(result.stdout, "só no histórico") {
		t.Errorf("saída = %q; é esse o achado do comando", result.stdout)
	}
	if !strings.Contains(result.stdout, "na árvore") {
		t.Errorf("saída = %q, queria distinguir do que continua na árvore", result.stdout)
	}
}

func TestChurnRespectsTheLimit(t *testing.T) {
	responses := churned()
	responses[churnLog] = gittest.Response{Output: "main.go\x00main.go\x00README.md\x00"}

	result := execute(t, responses, "", "churn", "--limit", "1")

	if strings.Contains(result.stdout, "README.md") {
		t.Errorf("saída = %q, o limite tinha de cortar o menor", result.stdout)
	}
	if !strings.Contains(result.stdout, "main.go") {
		t.Errorf("saída = %q, o limite corta os menores e não os maiores", result.stdout)
	}
}

func TestChurnCSV(t *testing.T) {
	result := execute(t, churned(), "", "churn", "--format", "csv")

	want := "caminho,commits,onde\n" +
		"main.go,2,na árvore\n" +
		"README.md,1,na árvore\n"

	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestChurnJSON(t *testing.T) {
	result := execute(t, churned(), "", "churn", "--format", "json")

	var document churnDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Files) != 2 {
		t.Fatalf("arquivos = %+v, queria os dois", document.Files)
	}

	first := churnRecord{Path: "main.go", Commits: 2, InTree: true}
	if document.Files[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", document.Files[0], first)
	}
}

func TestChurnJSONWrapsTheListInTheCommandKey(t *testing.T) {
	result := execute(t, churned(), "", "churn", "--format", "json")

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.stdout), &envelope); err != nil {
		t.Fatalf("o json tem de ser um objeto e não um array, veio %q: %v", result.stdout, err)
	}

	if _, named := envelope["files"]; !named {
		t.Errorf("chaves = %v, a lista mora sob a chave do comando", envelope)
	}
}

func TestChurnOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "churn")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestChurnPropagatesTheFailure(t *testing.T) {
	responses := churned()
	responses[churnLog] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "churn").err == nil {
		t.Fatal("falha do git no log tem de virar erro")
	}
}

func TestChurnRefusesUnknownFormat(t *testing.T) {
	result := execute(t, churned(), "", "churn", "--format", "yaml")

	if result.err == nil {
		t.Fatal("formato desconhecido tem de virar erro")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação da flag vem antes de tocar no git", result.calls)
	}
}

func TestChurnRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	if execute(t, churned(), "", "churn", "--format", "json", "--no-header").err == nil {
		t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
	}
}

func TestChurnWithNoCommitsYet(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --verify --quiet HEAD": {Err: &git.ExitError{Code: 1, Message: ""}},
	}

	result := execute(t, responses, "", "churn")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum arquivo no histórico deste repositório.") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestChurnNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, churned(), "", "churn")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestChurnPropagatesWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(churned()), noCommands(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"churn"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestChurnTakesNoArguments(t *testing.T) {
	result := execute(t, churned(), "", "churn", "sobrando")

	if result.err == nil {
		t.Fatal("o churn não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}
