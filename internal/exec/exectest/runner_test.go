package exectest

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
)

func TestRunnerAnswersInOrder(t *testing.T) {
	runner := NewRunner(
		Response{Result: exec.Result{Code: 0, Output: "primeiro"}},
		Response{Result: exec.Result{Code: 1, Output: "segundo"}},
	)

	first, _ := runner.Run("/um", "go", "test")
	second, _ := runner.Run("/dois", "go", "test")

	if first.Output != "primeiro" || second.Output != "segundo" {
		t.Errorf("respostas = %q e %q, queria na ordem", first.Output, second.Output)
	}
	if len(runner.Calls) != 2 {
		t.Fatalf("chamadas = %v, queria as duas", runner.Calls)
	}
	if runner.Calls[0].Directory != "/um" || runner.Calls[1].Directory != "/dois" {
		t.Errorf("diretorios = %v", runner.Calls)
	}
}

func TestRunnerRecordsTheArguments(t *testing.T) {
	runner := NewRunner(Response{})

	runner.Run("/um", "go", "test", "./...")

	call := runner.Calls[0]
	if call.Name != "go" || strings.Join(call.Args, " ") != "test ./..." {
		t.Errorf("chamada = %+v", call)
	}
}

func TestRunnerPropagatesTheScriptedError(t *testing.T) {
	runner := NewRunner(Response{Err: errors.New("nao comecou")})

	if _, err := runner.Run("/um", "go"); err == nil {
		t.Fatal("erro roteirizado tem de sair")
	}
}

func TestRunnerRefusesACallWithoutScript(t *testing.T) {
	runner := NewRunner(Response{})

	runner.Run("/um", "go")

	if _, err := runner.Run("/dois", "go"); err == nil {
		t.Fatal("chamada a mais tem de falhar alto, senao o teste passa de graca")
	}
}
