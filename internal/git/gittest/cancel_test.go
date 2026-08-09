package gittest

import (
	"context"
	"errors"
	"testing"
)

func TestRunnerHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	runner := NewRunner(map[string]Response{"status": {Output: "limpo"}})

	if _, err := runner.Run(ctx, "status"); !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v; o fake tem de cancelar como o de verdade, senao o teste do dominio mente", err)
	}
	if len(runner.Calls) != 0 {
		t.Errorf("chamadas = %v, nada pode ser executado com o contexto cancelado", runner.Calls)
	}
}
