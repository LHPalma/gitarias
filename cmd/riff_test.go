package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
	"github.com/LHPalma/gitarias/internal/web/webtest"
)

// riffed roda o gtr riff sem precisar de repositório: o comando não toca no
// git, só na internet.
func riffed(t *testing.T, responses []webtest.Response, args ...string) execution {
	t.Helper()

	client := webtest.NewClient(responses...)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(gittest.NewRunner(nil), noCommands(), client, noFinder(), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: client.Calls}
}

func TestRiffPrintsTheMessage(t *testing.T) {
	result := riffed(t, []webtest.Response{{Output: "Blaming regex.\n"}}, "riff")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "Blaming regex.\n" {
		t.Errorf("saída = %q, queria a mensagem seguida de quebra de linha", result.stdout)
	}
}

func TestRiffPropagatesTheClientFailure(t *testing.T) {
	result := riffed(t, []webtest.Response{{Err: errors.New("fora do ar")}}, "riff")

	if result.err == nil {
		t.Fatal("falha do client tem de virar erro")
	}
}

func TestRiffTakesNoArguments(t *testing.T) {
	result := riffed(t, []webtest.Response{{Output: "oi"}}, "riff", "sobrando")

	if result.err == nil {
		t.Fatal("o riff não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}

func TestWhatTheCommitIsTheSameCommand(t *testing.T) {
	byName := riffed(t, []webtest.Response{{Output: "oi"}}, "riff")
	byAlias := riffed(t, []webtest.Response{{Output: "oi"}}, "whatthecommit")

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de rodar o comando, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é o mesmo comando", byAlias.stdout, byName.stdout)
	}
}

func TestRiffPropagatesTheWriteFailure(t *testing.T) {
	client := webtest.NewClient(webtest.Response{Output: "oi"})

	command := NewRootCommand(gittest.NewRunner(nil), noCommands(), client, noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"riff"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestRiffDeclaresTheNetworkCall(t *testing.T) {
	result := riffed(t, nil, "riff", "--help")

	if !strings.Contains(result.stdout, "FAZ CHAMADA DE REDE") {
		t.Errorf("saída = %q, o --help tem de avisar que este comando fala com a internet", result.stdout)
	}
}
