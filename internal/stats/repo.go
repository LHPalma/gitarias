package stats

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

// ByAuthor conta quantos commits cada autor tem no histórico do HEAD atual,
// do maior número de commits para o menor, desempatando por nome. Authors,
// se não vazio, restringe aos que casam com algum dos padrões — a mesma
// correspondência por substring que o --author do próprio git já faz, com
// vários valores combinados por OR. Um repositório sem nenhum commit ainda
// devolve lista vazia, não erro.
func (repo *Repo) ByAuthor(ctx context.Context, authors []string) ([]Author, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return nil, err
	}
	if empty {
		return nil, nil
	}

	args := []string{"log", "--format=%an%x00%ae"}
	for _, author := range authors {
		args = append(args, "--author="+author)
	}

	output, err := repo.runner.Run(ctx, args...)
	if err != nil {
		return nil, err
	}

	return aggregate(output), nil
}

type identity struct {
	name  string
	email string
}

func aggregate(output string) []Author {
	counts := map[identity]int{}

	for _, line := range strings.Split(output, "\n") {
		name, email, found := strings.Cut(line, separator)
		if !found {
			continue
		}

		counts[identity{name: name, email: email}]++
	}

	authors := make([]Author, 0, len(counts))
	for who, commits := range counts {
		authors = append(authors, Author{Name: who.name, Email: who.email, Commits: commits})
	}

	sort.Slice(authors, func(first int, second int) bool {
		if authors[first].Commits != authors[second].Commits {
			return authors[first].Commits > authors[second].Commits
		}

		return authors[first].Name < authors[second].Name
	})

	return authors
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
