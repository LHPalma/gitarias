package churn

import (
	"context"
	"errors"
	"sort"
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

// Busiest devolve os caminhos mais tocados no histórico do HEAD atual, do
// maior número de commits para o menor, desempatando por nome. Um limite de
// zero ou menos traz todos. Repositório sem nenhum commit ainda devolve
// lista vazia, não erro.
//
// --no-renames trava a contagem contra a configuração de quem roda: sem ela,
// um renomeio conta como um caminho só ou como dois, dependendo do
// diff.renames de cada máquina. Aqui um renomeio sempre toca os dois
// caminhos, o antigo e o novo.
func (repo *Repo) Busiest(ctx context.Context, limit int) ([]File, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return nil, err
	}
	if empty {
		return nil, nil
	}

	output, err := repo.runner.Run(ctx, "log", "--format=", "--name-only", "--no-renames", "-z")
	if err != nil {
		return nil, err
	}

	present := repo.tracked(ctx)

	counts := map[string]int{}
	for _, path := range split(output) {
		counts[path]++
	}

	files := make([]File, 0, len(counts))
	for path, commits := range counts {
		files = append(files, File{Path: path, Commits: commits, InTree: present[path]})
	}

	sort.Slice(files, func(first int, second int) bool {
		if files[first].Commits != files[second].Commits {
			return files[first].Commits > files[second].Commits
		}

		return files[first].Path < files[second].Path
	})

	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	return files, nil
}

func split(output string) []string {
	if output == "" {
		return nil
	}

	fields := strings.Split(output, separator)
	if fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}

	return fields
}

// tracked lista o que ainda está na árvore, e não devolve erro de propósito:
// repositório sem HEAD não tem árvore, e nesse caso todo caminho do
// histórico está de fato fora dela — mesmo raciocínio do internal/weight.
func (repo *Repo) tracked(ctx context.Context) map[string]bool {
	listed, err := repo.runner.Run(ctx, "ls-tree", "-r", "-z", "--name-only", "HEAD")
	if err != nil {
		return map[string]bool{}
	}

	present := map[string]bool{}
	for _, name := range split(listed) {
		present[name] = true
	}

	return present
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
