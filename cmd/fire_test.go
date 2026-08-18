package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
	"github.com/LHPalma/gitarias/internal/web/webtest"
)

const (
	fireStatus       = "status --porcelain"
	fireAddAll       = "add -A"
	fireCommit       = "commit -m Blaming regex."
	firePreviousHead = "rev-parse --short HEAD^"
	fireNewHead      = "rev-parse --short HEAD"
	firePush         = "push origin HEAD:refs/heads/fire/def5678"
)

func dirtyRepository() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		fireStatus:                        {Output: " M f.txt\n?? novo.txt\n"},
		fireAddAll:                        {Output: ""},
		fireCommit:                        {Output: ""},
		firePreviousHead:                  {Output: "abc1234"},
		fireNewHead:                       {Output: "def5678"},
		firePush:                          {Output: ""},
	}
}

// fired roda o gtr fire com git e internet roteirizados.
func fired(t *testing.T, responses map[string]gittest.Response, webResponses []webtest.Response, args ...string) execution {
	t.Helper()

	runner := gittest.NewRunner(responses)
	client := webtest.NewClient(webResponses...)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, noCommands(), client, noFinder(), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: runner.Calls}
}

func TestFireSavesEverythingWithTheRiffedMessage(t *testing.T) {
	result := fired(t, dirtyRepository(), []webtest.Response{{Output: "Blaming regex.\n"}}, "fire")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, `Salvo em fire/def5678: "Blaming regex."`) {
		t.Errorf("saída = %q, queria a branch e a mensagem usada", result.stdout)
	}
	if !strings.Contains(result.stdout, "git reset --hard abc1234") {
		t.Errorf("saída = %q, queria a instrução de recuperação", result.stdout)
	}

	var pushed bool
	for _, call := range result.calls {
		if call == firePush {
			pushed = true
		}
	}
	if !pushed {
		t.Errorf("chamadas = %v, o push tinha de ter rodado", result.calls)
	}
}

func TestFireFallsBackWhenTheRiffFails(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		fireStatus:                        {Output: " M f.txt\n"},
		fireAddAll:                        {Output: ""},
		"commit -m 🔥 fire":                {Output: ""},
		firePreviousHead:                  {Output: "abc1234"},
		fireNewHead:                       {Output: "def5678"},
		firePush:                          {Output: ""},
	}

	result := fired(t, responses, []webtest.Response{{Err: errors.New("fora do ar")}}, "fire")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "🔥 fire") {
		t.Errorf("saída = %q, queria a mensagem fixa quando a rede falha", result.stdout)
	}
}

func TestFireDoesNothingOnACleanTree(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		fireStatus:                        {Output: ""},
	}

	result := fired(t, responses, nil, "fire")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nada sujo para salvar.") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
	for _, call := range result.calls {
		if call == fireAddAll {
			t.Fatalf("árvore limpa não pode disparar add, mas rodou %q", call)
		}
	}
}

func TestFireOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := fired(t, responses, nil, "fire")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
}

func TestFirePropagatesTheDirtyCheckFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		fireStatus:                        {Err: errNotARepository},
	}

	if fired(t, responses, nil, "fire").err == nil {
		t.Fatal("falha do status tem de virar erro")
	}
}

func TestFirePropagatesTheSaveFailure(t *testing.T) {
	responses := dirtyRepository()
	responses[firePush] = gittest.Response{Err: errNotARepository}

	if fired(t, responses, []webtest.Response{{Output: "Blaming regex.\n"}}, "fire").err == nil {
		t.Fatal("falha do save (push) tem de virar erro")
	}
}

func TestFireTakesNoArguments(t *testing.T) {
	result := fired(t, dirtyRepository(), []webtest.Response{{Output: "Blaming regex.\n"}}, "fire", "sobrando")

	if result.err == nil {
		t.Fatal("o fire não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}

func TestFireDeclaresTheNetworkCall(t *testing.T) {
	result := fired(t, nil, nil, "fire", "--help")

	if !strings.Contains(result.stdout, "FAZ CHAMADA DE REDE") {
		t.Errorf("saída = %q, o --help tem de avisar que este comando fala com a internet", result.stdout)
	}
}

func TestJamIsTheSameCommand(t *testing.T) {
	responses := []webtest.Response{{Output: "Blaming regex.\n"}}

	byName := fired(t, dirtyRepository(), responses, "fire")
	byAlias := fired(t, dirtyRepository(), responses, "jam")

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de rodar o comando, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é o mesmo comando", byAlias.stdout, byName.stdout)
	}
}

func TestJamStaysOutOfTheRootHelp(t *testing.T) {
	result := fired(t, nil, nil, "--help")

	if strings.Contains(result.stdout, "jam") {
		t.Errorf("saída = %q; o apelido não entra na lista de comandos do --help raiz", result.stdout)
	}
	if !strings.Contains(result.stdout, "fire") {
		t.Errorf("saída = %q, o nome de verdade continua anunciado", result.stdout)
	}
}
