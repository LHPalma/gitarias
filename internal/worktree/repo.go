package worktree

import (
	"context"
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

func (repo *Repo) List(ctx context.Context) ([]Worktree, error) {
	output, err := repo.runner.Run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	worktrees := parse(output)

	currentPath, err := repo.runner.Run(ctx, "rev-parse", "--show-toplevel")
	if err == nil && currentPath != "" {
		for index := range worktrees {
			worktrees[index].Current = worktrees[index].Path == currentPath
		}
	}

	return worktrees, nil
}

func parse(output string) []Worktree {
	var worktrees []Worktree
	var pending Worktree
	open := false

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")

		if line == "" {
			if open {
				worktrees = append(worktrees, pending)
				open = false
			}
			continue
		}

		keyword, value, _ := strings.Cut(line, " ")
		switch keyword {
		case "worktree":
			pending = Worktree{Path: value}
			open = true
		case "HEAD":
			pending.Head = value
		case "branch":
			pending.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "detached":
			pending.Detached = true
		case "bare":
			pending.Bare = true
		case "locked":
			pending.Locked = true
			pending.LockedReason = unquote(value)
		case "prunable":
			pending.Prunable = true
			pending.PrunableReason = unquote(value)
		}
	}

	if open {
		worktrees = append(worktrees, pending)
	}

	return worktrees
}

// IgnoredFiles lista o que está ignorado no working tree em path — o que
// git worktree remove apagaria junto com o diretório, sem contar como sujo
// para o git, e sem chance de recuperação depois. É a prova da cláusula 1
// da ADR-008 que esta operação consegue dar: mostrar o que se perde, não
// garantir que volta.
func (repo *Repo) IgnoredFiles(ctx context.Context, path string) ([]string, error) {
	output, err := repo.runner.Run(ctx, "-C", path,
		"ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "--no-empty-directory", "-z")
	if err != nil {
		return nil, err
	}

	return splitNUL(output), nil
}

// Remove executa git worktree remove em path, sem --force: se o git recusar
// por causa de arquivo versionado sujo ou não rastreado, o erro é
// propagado — ADR-007.
func (repo *Repo) Remove(ctx context.Context, path string) error {
	_, err := repo.runner.Run(ctx, "worktree", "remove", path)
	return err
}

// Release solta path de sua branch com git -C <path> checkout --detach — o
// único dos três caminhos que gtr branches ensina que é comprovadamente não
// destrutivo: verificado que ele preserva trabalho não commitado, movendo
// só o HEAD daquele working tree. ADR-007. Sem --force: o que o git
// recusar (rebase pela metade, por exemplo) continua recusado, propagado
// como erro.
func (repo *Repo) Release(ctx context.Context, path string) error {
	_, err := repo.runner.Run(ctx, "-C", path, "checkout", "--detach")
	return err
}

// Dirty diz se path tem trabalho não commitado — rastreado ou não, staged
// ou não. Release não perde esse trabalho, mas ele passa a viver num HEAD
// destacado depois, e quem não sabe disso se assusta; é esse aviso que
// Dirty alimenta.
func (repo *Repo) Dirty(ctx context.Context, path string) (bool, error) {
	output, err := repo.runner.Run(ctx, "-C", path, "status", "--porcelain")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(output) != "", nil
}

func splitNUL(output string) []string {
	if output == "" {
		return nil
	}

	fields := strings.Split(output, "\x00")
	if fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}

	return fields
}
