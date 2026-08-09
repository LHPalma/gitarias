package git

import (
	"context"
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestEnsureRepoCarriesTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := EnsureRepo(ctx, gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
	}))

	if err == nil {
		t.Fatal("com o contexto cancelado nem o Ensure passa")
	}
}

func TestCommandRunnerRefusesAContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := (CommandRunner{}).Run(ctx, "version"); !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v, queria o cancelamento e nao a saida do git", err)
	}
}

func TestCommandRunnerWithInputRefusesAContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := (CommandRunner{}).RunWithInput(ctx, "entrada", "hash-object", "--stdin"); !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v, queria o cancelamento", err)
	}
}
