package commits

import (
	"context"

	"github.com/LHPalma/gitarias/internal/git"
)

type WorktreeExtractor struct {
	runner git.Runner
}

func NewWorktreeExtractor(runner git.Runner) *WorktreeExtractor {
	return &WorktreeExtractor{runner: runner}
}

func (extractor *WorktreeExtractor) Extract(ctx context.Context, sha string, destination string) error {
	_, err := extractor.runner.Run(ctx, "worktree", "add", "--detach", "--quiet", destination, sha)

	return err
}

func (extractor *WorktreeExtractor) Release(ctx context.Context, destination string) {
	extractor.runner.Run(ctx, "worktree", "remove", "--force", destination)
}
