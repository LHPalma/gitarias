package branch

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestBranchOperationsCarryTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	repo := NewRepo(gittest.NewRunner(map[string]gittest.Response{
		insideWorkTree: {Output: "true"},
		"symbolic-ref --short refs/remotes/origin/HEAD": {Output: "origin/main"},
	}))

	if err := repo.Ensure(ctx); err == nil {
		t.Error("Ensure tem de recusar o contexto cancelado")
	}
	if _, err := repo.ResolveBase(ctx, ""); err == nil {
		t.Error("ResolveBase tem de recusar o contexto cancelado")
	}
	if _, err := repo.Merged(ctx, Base{Name: "main"}); err == nil {
		t.Error("Merged tem de recusar o contexto cancelado")
	}
}
