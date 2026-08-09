package exec

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCommandRunnerKillsTheChildOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	started := time.Now()
	_, err := CommandRunner{}.Run(ctx, t.TempDir(), "sleep", "30")
	elapsed := time.Since(started)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v, queria o cancelamento e nao um Result vermelho", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("levou %v; o filho tem de morrer junto, nao rodar os 30 segundos", elapsed)
	}
}

func TestCommandRunnerRefusesAContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := (CommandRunner{}).Run(ctx, t.TempDir(), "echo", "oi"); !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v, queria o cancelamento", err)
	}
}
