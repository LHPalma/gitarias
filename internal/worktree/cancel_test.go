package worktree

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestWorktreeOperationsCarryTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	repo := NewRepo(gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain":       {Output: "worktree /repo\n"},
	}))

	if err := repo.Ensure(ctx); err == nil {
		t.Error("Ensure tem de recusar o contexto cancelado")
	}
	if _, err := repo.List(ctx); err == nil {
		t.Error("List tem de recusar o contexto cancelado")
	}
}
