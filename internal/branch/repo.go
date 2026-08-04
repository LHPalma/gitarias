package branch

import (
	"errors"
	"fmt"
	"strings"
)

type Repo struct {
	git Runner
}

func NewRepo(runner Runner) *Repo {
	return &Repo{git: runner}
}

func (repo *Repo) Ensure() error {
	if _, err := repo.git.Run("rev-parse", "--is-inside-work-tree"); err != nil {
		return errors.New("isso não é um repositório git")
	}
	return nil
}

func (repo *Repo) ResolveBase(requested string) (Base, error) {
	if requested != "" {
		if !repo.localExists(requested) {
			return Base{}, fmt.Errorf("a branch base %q não existe neste repositório", requested)
		}
		return Base{Name: requested, Source: BaseFromFlag}, nil
	}

	if originHead, err := repo.git.Run("symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		detected := strings.TrimPrefix(originHead, "origin/")
		if detected != "" && repo.localExists(detected) {
			return Base{Name: detected, Source: BaseFromOriginHead}, nil
		}
	}

	for _, candidate := range []string{"main", "master"} {
		if repo.localExists(candidate) {
			return Base{Name: candidate, Source: BaseFromLocal}, nil
		}
	}

	return Base{}, errors.New("não consegui determinar a branch base: nem main nem master existem aqui. Use --base <branch>")
}

func (repo *Repo) localExists(name string) bool {
	_, err := repo.git.Run("rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

func (repo *Repo) Merged(base Base) ([]Branch, error) {
	output, err := repo.git.Run("for-each-ref", "refs/heads/", "--merged", base.Name, "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	protected := map[string]bool{
		base.Name: true,
		"main":    true,
		"master":  true,
	}
	if currentBranch, err := repo.git.Run("branch", "--show-current"); err == nil && currentBranch != "" {
		protected[currentBranch] = true
	}

	var merged []Branch
	for _, line := range strings.Split(output, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || protected[name] {
			continue
		}
		merged = append(merged, Branch{Name: name})
	}

	return merged, nil
}

func (repo *Repo) Delete(branches []Branch) []DeleteResult {
	results := make([]DeleteResult, 0, len(branches))

	for _, target := range branches {
		_, err := repo.git.Run("branch", "-d", target.Name)
		results = append(results, DeleteResult{Branch: target, Err: err})
	}

	return results
}
