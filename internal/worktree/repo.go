package worktree

import (
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

type Repo struct {
	runner git.Runner
}

func NewRepo(runner git.Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure() error {
	return git.EnsureRepo(repo.runner)
}

func (repo *Repo) List() ([]Worktree, error) {
	output, err := repo.runner.Run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	worktrees := parse(output)

	currentPath, err := repo.runner.Run("rev-parse", "--show-toplevel")
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
			pending.LockedReason = value
		case "prunable":
			pending.Prunable = true
			pending.PrunableReason = value
		}
	}

	if open {
		worktrees = append(worktrees, pending)
	}

	return worktrees
}
