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

func TestCommandRunnerWithInputAndEnvRefusesAContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := (CommandRunner{}).RunWithInputAndEnv(ctx, "entrada", nil, "hash-object", "--stdin"); !errors.Is(err, context.Canceled) {
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

// TestCommandRunnerWithInputAndEnvPassesBoth prova o mecanismo que
// internal/diff depende para o Verify: git apply --check --cached precisa do
// patch pelo stdin (RN-06, sem caminho de arquivo interpolado) e do índice
// temporário pelo ambiente (GIT_INDEX_FILE), na mesma chamada.
// hash-object --stdin prova o stdin (o SHA1 devolvido depende só do
// conteúdo lido); var GIT_AUTHOR_IDENT prova o ambiente, como o teste acima
// — juntos, na mesma chamada de RunWithInputAndEnv.
func TestCommandRunnerWithInputAndEnvPassesBoth(t *testing.T) {
	output, err := (CommandRunner{}).RunWithInputAndEnv(t.Context(),
		"conteudo de teste\n",
		[]string{"GIT_AUTHOR_NAME=Fake Name", "GIT_AUTHOR_EMAIL=fake@fake.com"},
		"hash-object", "--stdin")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want, err := (CommandRunner{}).RunWithInput(t.Context(), "conteudo de teste\n", "hash-object", "--stdin")
	if err != nil {
		t.Fatalf("não esperava erro ao calcular o esperado, veio %v", err)
	}
	if output != want {
		t.Errorf("hash = %q, queria %q — o stdin não chegou igual com o ambiente junto", output, want)
	}

	identity, err := (CommandRunner{}).RunWithInputAndEnv(t.Context(), "",
		[]string{"GIT_AUTHOR_NAME=Fake Name", "GIT_AUTHOR_EMAIL=fake@fake.com"},
		"var", "GIT_AUTHOR_IDENT")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !strings.Contains(identity, "Fake Name <fake@fake.com>") {
		t.Errorf("saída = %q, queria a identidade vinda do ambiente", identity)
	}
}
