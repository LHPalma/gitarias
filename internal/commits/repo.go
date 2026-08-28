package commits

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
)

type Repo struct {
	git      git.Runner
	commands exec.Runner
}

func NewRepo(runner git.Runner, commands exec.Runner) *Repo {
	return &Repo{git: runner, commands: commands}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.git)
}

func (repo *Repo) Range(ctx context.Context, base string) ([]Commit, error) {
	output, err := repo.git.Run(ctx, "log", "--reverse", "--format=%H%x00%s", base+"..HEAD")
	if err != nil {
		return nil, err
	}

	var found []Commit
	for _, line := range strings.Split(output, "\n") {
		sha, subject, complete := strings.Cut(line, "\x00")
		if !complete || sha == "" {
			continue
		}
		found = append(found, Commit{SHA: sha, Subject: subject})
	}

	return found, nil
}

func (repo *Repo) Check(ctx context.Context, list []Commit, extractor Extractor, name string, args []string) ([]Result, error) {
	workspace, err := os.MkdirTemp("", "gtr-check-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workspace)

	results := make([]Result, 0, len(list))
	for _, commit := range list {
		if cancelled := ctx.Err(); cancelled != nil {
			return nil, cancelled
		}

		result, err := repo.check(ctx, commit, extractor, workspace, name, args)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// BisectResult é o achado de uma busca binária sobre Bisect: Culprit é nil
// quando nenhum dos testados falhou, e Tested traz só os commits realmente
// rodados — normalmente log2(Total), não Total.
type BisectResult struct {
	Culprit *Result
	Tested  []Result
	Total   int
}

// Bisect assume o mesmo que o git bisect: list vem do mais antigo para o mais
// novo (como Range devolve), e existe no máximo uma transição de verde para
// vermelho nesse sentido. Acha o primeiro índice que falha testando
// log2(len(list)) commits, não todos — list[0..culprit) passa, list[culprit:]
// falha, e nenhum commit fora dessa fronteira é sequer extraído.
func (repo *Repo) Bisect(ctx context.Context, list []Commit, extractor Extractor, name string, args []string) (BisectResult, error) {
	workspace, err := os.MkdirTemp("", "gtr-bisect-")
	if err != nil {
		return BisectResult{}, err
	}
	defer os.RemoveAll(workspace)

	var tested []Result
	var culprit *Result

	low, high := 0, len(list)-1
	for low <= high {
		if cancelled := ctx.Err(); cancelled != nil {
			return BisectResult{}, cancelled
		}

		mid := low + (high-low)/2

		result, err := repo.check(ctx, list[mid], extractor, workspace, name, args)
		if err != nil {
			return BisectResult{}, err
		}
		tested = append(tested, result)

		if result.Passed() {
			low = mid + 1
			continue
		}

		found := result
		culprit = &found
		high = mid - 1
	}

	return BisectResult{Culprit: culprit, Tested: tested, Total: len(list)}, nil
}

func (repo *Repo) check(ctx context.Context, commit Commit, extractor Extractor, workspace string, name string, args []string) (Result, error) {
	destination := filepath.Join(workspace, commit.SHA)

	if err := extractor.Extract(ctx, commit.SHA, destination); err != nil {
		return Result{}, err
	}
	defer extractor.Release(ctx, destination)

	outcome, err := repo.commands.Run(ctx, destination, name, args...)
	if err != nil {
		return Result{}, err
	}

	return Result{Commit: commit, Code: outcome.Code, Output: outcome.Output}, nil
}
