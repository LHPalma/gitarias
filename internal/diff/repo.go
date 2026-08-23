package diff

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

type Repo struct {
	runner Runner
}

func NewRepo(runner Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

// Changes levanta os candidatos ao export: tracked modificados/apagados e
// untracked, na ordem que o git status devolve. Ignorados só entram com
// includeIgnored, colapsados por diretório — --ignored=matching, a mesma
// convenção de coleta do gtr ignore list.
func (repo *Repo) Changes(ctx context.Context, includeIgnored bool) ([]Change, error) {
	args := []string{"status", "--porcelain=v2", "-z", "--no-renames"}
	if includeIgnored {
		args = append(args, "--ignored=matching")
	}

	output, err := repo.runner.Run(ctx, args...)
	if err != nil {
		return nil, err
	}

	return parseChanges(output), nil
}

// parseChanges lê o formato --porcelain=v2 -z: registros separados por NUL,
// cada um começando por um caractere de tipo que nunca é espaço — "1" para
// entrada comum, "?" para untracked, "!" para ignorado (com --ignored). Isso
// importa porque Run devolve a saída com TrimSpace: o v1 abre uma entrada não
// staged com um espaço ("git status --porcelain -z"), e esse espaço, quando é
// o primeiro byte da saída inteira, é exatamente o que o TrimSpace comeria —
// cortando um caractere do primeiro caminho listado. O v2 nunca começa registro
// com espaço, então não tem essa borda para o Trim morder. --no-renames
// garante um caminho por registro — a v2 despacha renomeação/cópia (tipo "2")
// como dois caminhos no mesmo registro, separados por tab, que esta função não
// entende.
func parseChanges(output string) []Change {
	var changes []Change

	for _, record := range strings.Split(output, "\x00") {
		if record == "" {
			continue
		}

		switch record[0] {
		case '?', '!':
			changes = append(changes, Change{Path: record[2:], New: true})
		case '1':
			fields := strings.SplitN(record, " ", 9)
			if len(fields) != 9 {
				continue
			}
			changes = append(changes, Change{Path: fields[8], New: fields[1][0] == 'A'})
		}
	}

	return changes
}

// Export empacota changes num patch aplicável, com --binary e o SHA curto da
// base em comentário no cabeçalho. Roda inteiramente sobre um índice
// temporário — RN-17 — então nem o índice real nem a árvore são tocados:
// add -N -f nesse índice faz o diff HEAD enxergar cada caminho novo
// (untracked, ignorado ou só staged) como arquivo criado, com o conteúdo
// completo lido do disco — sem precisar do índice do usuário. Em cima de
// caminho já rastreado, add -N -f não muda nada: o diff sai igual.
func (repo *Repo) Export(ctx context.Context, changes []Change) (Patch, error) {
	if len(changes) == 0 {
		return Patch{}, nil
	}

	base, err := repo.runner.Run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return Patch{}, err
	}

	env, cleanup, err := repo.freshIndex(ctx, "gtr-diff-content-")
	if err != nil {
		return Patch{}, err
	}
	defer cleanup()

	paths := changePaths(changes)

	addArgs := append([]string{"add", "-N", "-f", "--"}, paths...)
	if _, err := repo.runner.RunWithEnv(ctx, env, addArgs...); err != nil {
		return Patch{}, err
	}

	diffArgs := append([]string{"diff", "--binary", "HEAD", "--"}, paths...)
	body, err := repo.runner.RunWithEnv(ctx, env, diffArgs...)
	if err != nil {
		return Patch{}, err
	}

	content := fmt.Sprintf("# gtr diff export\n# base: %s\n\n%s\n", base, body)

	return Patch{Content: content, Base: base}, nil
}

// Verify roda git apply --check --cached sobre um segundo índice temporário,
// também montado do zero a partir do HEAD — RF-16, RN-18. O patch vai pelo
// stdin, não por um arquivo: é a mesma garantia de RN-06 (sem shell, sem
// caminho interpolado) e o que faz esta chamada testável sem um caminho
// dinâmico no meio do argv. É a única garantia da feature que protege quem
// recebe o patch, não quem o gerou: git diff nunca responde sozinho "isto
// aplica?", e gerar sem erro não é prova disso.
func (repo *Repo) Verify(ctx context.Context, patch Patch) error {
	if patch.Content == "" {
		return nil
	}

	env, cleanup, err := repo.freshIndex(ctx, "gtr-diff-verify-")
	if err != nil {
		return err
	}
	defer cleanup()

	_, err = repo.runner.RunWithInputAndEnv(ctx, patch.Content, env, "apply", "--check", "--cached")

	return err
}

// freshIndex cria um diretório temporário e aponta GIT_INDEX_FILE pra dentro
// dele, populado com read-tree HEAD — um índice que começa idêntico ao HEAD,
// sem nenhum staged do índice real. O diretório volta junto do cleanup, então
// o chamador nunca precisa lembrar de apagar nada.
func (repo *Repo) freshIndex(ctx context.Context, prefix string) ([]string, func(), error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }

	env := []string{"GIT_INDEX_FILE=" + filepath.Join(dir, "index")}
	if _, err := repo.runner.RunWithEnv(ctx, env, "read-tree", "HEAD"); err != nil {
		cleanup()
		return nil, nil, err
	}

	return env, cleanup, nil
}

func changePaths(changes []Change) []string {
	paths := make([]string, len(changes))
	for i, change := range changes {
		paths[i] = change.Path
	}

	return paths
}
