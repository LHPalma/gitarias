package fire

import (
	"context"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

type Runner interface {
	git.Runner
}

type Repo struct {
	runner Runner
}

func NewRepo(runner Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

// Dirty diz se há qualquer mudança para salvar — rastreada ou não, staged ou
// não. Ao contrário do gtr author --reset, que só se importa com arquivo já
// rastreado, aqui é o oposto: o que interessa é tudo que git add -A pegaria.
func (repo *Repo) Dirty(ctx context.Context) (bool, error) {
	output, err := repo.runner.Run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(output) != "", nil
}

// Save é o que Save devolveu: o HEAD anterior ao commit de emergência — para
// quem quiser desfazer localmente — e a branch remota que recebeu a cópia.
type Save struct {
	Head   string
	Branch string
}

// Save comita tudo que está sujo — rastreado ou não — com message, na branch
// local atual, e empurra esse commit para uma branch nova no remoto. O nome
// da branch nova é fire/<SHA do commit>, não um carimbo de hora: não depende
// do relógio, não colide entre duas chamadas rápidas que gerem commits
// diferentes, e cada fire tem sempre um destino remoto próprio. O upstream de
// quem chamou não é tocado — quem recebe a cópia de segurança é sempre uma
// branch nova, nunca a que já existia lá.
//
// O HEAD anterior sai de HEAD^ depois do commit, não de HEAD antes dele: as
// duas leituras de HEAD, antes e depois, seriam o mesmo comando de git com
// respostas que têm de ser diferentes, e nada nesse texto distingue uma
// chamada da outra. HEAD^ depois do commit aponta pro mesmo commit que HEAD
// apontava antes, então a leitura única resolve as duas necessidades.
func (repo *Repo) Save(ctx context.Context, message string) (Save, error) {
	if _, err := repo.runner.Run(ctx, "add", "-A"); err != nil {
		return Save{}, err
	}
	if _, err := repo.runner.Run(ctx, "commit", "-m", message); err != nil {
		return Save{}, err
	}

	head, err := repo.runner.Run(ctx, "rev-parse", "--short", "HEAD^")
	if err != nil {
		return Save{}, err
	}

	newHead, err := repo.runner.Run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return Save{}, err
	}

	branch := "fire/" + newHead
	if _, err := repo.runner.Run(ctx, "push", "origin", "HEAD:refs/heads/"+branch); err != nil {
		return Save{}, err
	}

	return Save{Head: head, Branch: branch}, nil
}
