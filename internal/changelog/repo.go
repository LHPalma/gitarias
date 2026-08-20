package changelog

import (
	"context"
	"errors"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

const separator = "\x00"

type Repo struct {
	runner git.Runner
}

func NewRepo(runner git.Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

// Entries devolve os commits do histórico do HEAD atual, do mais recente
// para o mais antigo, já classificados pelo Conventional Commits. Um
// repositório sem nenhum commit ainda devolve lista vazia, não erro.
//
// since/until, os dois em AAAA-MM-DD, filtram o período — vazio não limita
// aquele lado. Igual ao internal/profile, viram meia-noite e 23:59:59
// explícitas antes de chegar ao git, nunca a data nua: --since sozinho não
// vale meia-noite daquele dia, vale a hora corrente de agora, nesse dia.
func (repo *Repo) Entries(ctx context.Context, since string, until string) ([]Entry, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return nil, err
	}
	if empty {
		return nil, nil
	}

	args := []string{"log", "--format=%H%x00%s"}
	if since != "" {
		args = append(args, "--since="+since+" 00:00:00")
	}
	if until != "" {
		args = append(args, "--until="+until+" 23:59:59")
	}

	output, err := repo.runner.Run(ctx, args...)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, line := range strings.Split(output, "\n") {
		hash, subject, found := strings.Cut(line, separator)
		if !found {
			continue
		}

		entries = append(entries, parse(hash, subject))
	}

	return entries, nil
}

// empty diz se o HEAD ainda não aponta para nenhum commit — repositório
// recém-criado, antes do primeiro. O --quiet evita a mensagem em stderr para
// o que é resposta esperada, não falha.
func (repo *Repo) empty(ctx context.Context) (bool, error) {
	_, err := repo.runner.Run(ctx, "rev-parse", "--verify", "--quiet", "HEAD")
	if err == nil {
		return false, nil
	}

	var exitError *git.ExitError
	if errors.As(err, &exitError) && exitError.Code == 1 {
		return true, nil
	}

	return false, err
}
