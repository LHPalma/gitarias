package commits

import "github.com/LHPalma/gitarias/internal/git"

type WorktreeExtractor struct {
	runner git.Runner
}

func NewWorktreeExtractor(runner git.Runner) *WorktreeExtractor {
	return &WorktreeExtractor{runner: runner}
}

func (extractor *WorktreeExtractor) Extract(sha string, destination string) error {
	_, err := extractor.runner.Run("worktree", "add", "--detach", "--quiet", destination, sha)

	return err
}

func (extractor *WorktreeExtractor) Release(destination string) {
	extractor.runner.Run("worktree", "remove", "--force", destination)
}
