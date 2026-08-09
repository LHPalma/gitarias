package commits

import (
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

func (repo *Repo) Ensure() error {
	return git.EnsureRepo(repo.git)
}

func (repo *Repo) Range(base string) ([]Commit, error) {
	output, err := repo.git.Run("log", "--reverse", "--format=%H%x00%s", base+"..HEAD")
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

func (repo *Repo) Check(list []Commit, extractor Extractor, name string, args []string) ([]Result, error) {
	workspace, err := os.MkdirTemp("", "gtr-check-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workspace)

	results := make([]Result, 0, len(list))
	for _, commit := range list {
		result, err := repo.check(commit, extractor, workspace, name, args)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func (repo *Repo) check(commit Commit, extractor Extractor, workspace string, name string, args []string) (Result, error) {
	destination := filepath.Join(workspace, commit.SHA)

	if err := extractor.Extract(commit.SHA, destination); err != nil {
		return Result{}, err
	}
	defer extractor.Release(destination)

	outcome, err := repo.commands.Run(destination, name, args...)
	if err != nil {
		return Result{}, err
	}

	return Result{Commit: commit, Code: outcome.Code, Output: outcome.Output}, nil
}
