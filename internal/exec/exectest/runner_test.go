package exectest

import (
	"context"
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

	first, _ := runner.Run(t.Context(), "/um", "go", "test")
	second, _ := runner.Run(t.Context(), "/dois", "go", "test")

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

	runner.Run(t.Context(), "/um", "go", "test", "./...")

	call := runner.Calls[0]
	if call.Name != "go" || strings.Join(call.Args, " ") != "test ./..." {
		t.Errorf("chamada = %+v", call)
	}
}

func TestRunnerPropagatesTheScriptedError(t *testing.T) {
	runner := NewRunner(Response{Err: errors.New("nao comecou")})

	if _, err := runner.Run(t.Context(), "/um", "go"); err == nil {
		t.Fatal("erro roteirizado tem de sair")
	}
}

func TestRunnerRefusesACallWithoutScript(t *testing.T) {
	runner := NewRunner(Response{})

	runner.Run(t.Context(), "/um", "go")

	if _, err := runner.Run(t.Context(), "/dois", "go"); err == nil {
		t.Fatal("chamada a mais tem de falhar alto, senao o teste passa de graca")
	}
}

func TestRunnerHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	runner := NewRunner(Response{Result: exec.Result{Output: "nao devia sair"}})

	if _, err := runner.Run(ctx, "/um", "go", "test"); !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v; o fake tem de cancelar como o de verdade, senao o teste do dominio mente", err)
	}
	if len(runner.Calls) != 0 {
		t.Errorf("chamadas = %v, nada roda com o contexto cancelado", runner.Calls)
	}
}
