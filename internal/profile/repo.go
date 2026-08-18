package profile

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

type Repo struct {
	runner git.Runner
}

func NewRepo(runner git.Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

// Identity devolve o e-mail configurado como autor neste repositório — git
// config user.email —, ou o nome se só ele estiver configurado. Vazio quando
// nenhum dos dois está.
func (repo *Repo) Identity(ctx context.Context) (string, error) {
	email, err := repo.setting(ctx, "user.email")
	if err != nil {
		return "", err
	}
	if email != "" {
		return email, nil
	}

	return repo.setting(ctx, "user.name")
}

// setting devolve o valor configurado, vazio quando a chave não existe — o
// código de saída 1 do próprio git config --get, não qualquer erro: falha de
// verdade (cancelamento incluído) tem de subir, não virar "não configurado".
func (repo *Repo) setting(ctx context.Context, key string) (string, error) {
	output, err := repo.runner.Run(ctx, "config", "--get", key)
	if err == nil {
		return strings.TrimSpace(output), nil
	}

	var exitError *git.ExitError
	if errors.As(err, &exitError) && exitError.Code == 1 {
		return "", nil
	}

	return "", err
}

// CommitCount conta quantos commits de identity caem entre since e until —
// os dois em AAAA-MM-DD, os dois dias inteiros incluídos — no HEAD atual.
// Repositório sem nenhum commit ainda devolve zero, não erro.
//
// since/until viram meia-noite e 23:59:59 explícitas antes de chegar ao git,
// nunca a data nua: --since=2026-08-15 sozinho não vale meia-noite daquele
// dia, vale a hora corrente de agora, nesse dia — medido contra o git antes
// de decidir por isso. Sem a hora explícita, --since e --until iguais (um
// dia só) davam zero, mesmo com commit dentro do dia.
func (repo *Repo) CommitCount(ctx context.Context, identity string, since string, until string) (int, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return 0, err
	}
	if empty {
		return 0, nil
	}

	output, err := repo.runner.Run(ctx, "rev-list", "--count", "--author="+identity,
		"--since="+since+" 00:00:00", "--until="+until+" 23:59:59", "HEAD")
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, fmt.Errorf("rev-list devolveu uma contagem ilegível: %q", output)
	}

	return count, nil
}

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
