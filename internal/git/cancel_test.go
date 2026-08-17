package git

import (
	"context"
	"errors"
	"strings"
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

func TestCommandRunnerWithEnvRefusesAContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := (CommandRunner{}).RunWithEnv(ctx, nil, "version"); !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v, queria o cancelamento", err)
	}
}

// TestCommandRunnerWithEnvPassesTheEnvironment prova o mecanismo que
// internal/author depende: a identidade viaja pelo ambiente do processo, não
// por um argumento que algum comando do git interpolaria numa string de
// shell. git var GIT_AUTHOR_IDENT devolve a identidade que uma escrita usaria
// agora, sem precisar de um repositório.
func TestCommandRunnerWithEnvPassesTheEnvironment(t *testing.T) {
	output, err := (CommandRunner{}).RunWithEnv(t.Context(),
		[]string{"GIT_AUTHOR_NAME=Fake Name", "GIT_AUTHOR_EMAIL=fake@fake.com"},
		"var", "GIT_AUTHOR_IDENT")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !strings.Contains(output, "Fake Name <fake@fake.com>") {
		t.Errorf("saída = %q, queria a identidade vinda do ambiente", output)
	}
}
