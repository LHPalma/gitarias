package author

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

type Runner interface {
	git.Runner
	RunWithEnv(ctx context.Context, env []string, args ...string) (string, error)
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

// Plan devolve o que Rewrite afetaria, sem afetar nada. Base vazio mede só
// o HEAD; base preenchido mede base..until (until vazio significa HEAD), e o
// erro nomeia o intervalo quando ele não tem nenhum commit.
func (repo *Repo) Plan(ctx context.Context, base string, until string) (Plan, error) {
	head, err := repo.runner.Run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return Plan{}, err
	}

	if base == "" {
		author, err := repo.runner.Run(ctx, "log", "-1", "--format=%an <%ae>")
		if err != nil {
			return Plan{}, err
		}

		return Plan{Head: head, Count: 1, Author: author}, nil
	}

	resolvedUntil := until
	if resolvedUntil == "" {
		resolvedUntil = "HEAD"
	}

	count, err := repo.countRange(ctx, base, resolvedUntil)
	if err != nil {
		return Plan{}, err
	}
	if count == 0 {
		return Plan{}, fmt.Errorf("%s..%s não tem nenhum commit", base, resolvedUntil)
	}

	authors, err := repo.distinctAuthors(ctx, base, resolvedUntil)
	if err != nil {
		return Plan{}, err
	}

	tail, err := repo.countRange(ctx, resolvedUntil, "HEAD")
	if err != nil {
		return Plan{}, err
	}

	return Plan{Head: head, Count: count, Authors: authors, Tail: tail}, nil
}

func (repo *Repo) countRange(ctx context.Context, from string, to string) (int, error) {
	output, err := repo.runner.Run(ctx, "rev-list", "--count", from+".."+to)
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, fmt.Errorf("rev-list devolveu uma contagem ilegível: %q", output)
	}

	return count, nil
}

func (repo *Repo) distinctAuthors(ctx context.Context, base string, until string) ([]string, error) {
	output, err := repo.runner.Run(ctx, "log", base+".."+until, "--format=%an <%ae>")
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var authors []string
	for _, line := range strings.Split(output, "\n") {
		if line == "" || seen[line] {
			continue
		}

		seen[line] = true
		authors = append(authors, line)
	}

	sort.Strings(authors)

	return authors, nil
}

// Rewrite reatribui a autoria (nome e e-mail, autor e committer) do commit
// mais recente, ou de base..until quando base não é vazio (until vazio
// significa HEAD). O que vem depois de until é preservado: reescrever só
// base..until e deixar o resto intocado precisa de duas passadas — a
// primeira reescreve o alvo com HEAD destacado nele, a segunda reencaixa em
// cima o que ficou para trás (--onto), sem tocar no conteúdo delas. Quando
// until já é o HEAD, o reencaixe não tem nada para mover e só atualiza a
// ref — testado assim contra o git de verdade antes de decidir por um
// caminho só em vez de um atalho para esse caso.
//
// A identidade viaja pelo ambiente do processo — GIT_AUTHOR_NAME e as outras
// três —, nunca por dentro do argumento do --exec do rebase. O --exec do
// rebase roda a string por um shell, e o valor passado aqui é sempre um
// literal fixo: interpolar nome ou e-mail de quem chama ali dentro abriria
// injeção de shell pelo próprio dado que o comando existe para reatribuir.
// Medido contra um nome contendo `$(...)` e crase antes de decidir por essa
// forma: com o ambiente, o git tratou o texto como identidade e nada
// executou; com --author interpolado no --exec, teria executado.
func (repo *Repo) Rewrite(ctx context.Context, base string, until string, name string, email string) error {
	identity := []string{
		"GIT_AUTHOR_NAME=" + name,
		"GIT_AUTHOR_EMAIL=" + email,
		"GIT_COMMITTER_NAME=" + name,
		"GIT_COMMITTER_EMAIL=" + email,
	}

	if base == "" {
		_, err := repo.runner.RunWithEnv(ctx, identity, "commit", "--amend", "--no-edit", "--reset-author")
		return err
	}

	resolvedUntil := until
	if resolvedUntil == "" {
		resolvedUntil = "HEAD"
	}

	untilSHA, err := repo.runner.Run(ctx, "rev-parse", resolvedUntil)
	if err != nil {
		return err
	}

	ref, err := repo.runner.Run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		ref, err = repo.runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
	}

	if _, err := repo.runner.RunWithEnv(ctx, identity, "rebase", base, untilSHA, "--exec", "git commit --amend --no-edit --reset-author"); err != nil {
		return err
	}

	newUntilSHA, err := repo.runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return err
	}

	_, err = repo.runner.Run(ctx, "rebase", "--onto", newUntilSHA, untilSHA, ref)

	return err
}

// PlanReset devolve o que Reset afetaria, sem afetar nada: o HEAD atual
// (para recuperação), quantos commits deixam de estar alcançáveis a partir
// da branch, e se há mudança não commitada em arquivo rastreado — essas
// mudanças o --hard descarta sem deixar rastro.
func (repo *Repo) PlanReset(ctx context.Context, sha string) (ResetPlan, error) {
	head, err := repo.runner.Run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return ResetPlan{}, err
	}

	discarded, err := repo.countRange(ctx, sha, "HEAD")
	if err != nil {
		return ResetPlan{}, err
	}

	dirty := false
	if _, err := repo.runner.Run(ctx, "diff", "HEAD", "--quiet"); err != nil {
		var exitError *git.ExitError
		if !errors.As(err, &exitError) || exitError.Code != 1 {
			return ResetPlan{}, err
		}
		dirty = true
	}

	return ResetPlan{Head: head, Discarded: discarded, Dirty: dirty}, nil
}

// Reset move a branch para sha com git reset --hard, descartando tanto os
// commits que ficam pra trás quanto qualquer mudança não commitada em
// arquivo rastreado.
func (repo *Repo) Reset(ctx context.Context, sha string) error {
	_, err := repo.runner.Run(ctx, "reset", "--hard", sha)
	return err
}
