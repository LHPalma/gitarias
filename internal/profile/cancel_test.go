package profile

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestProfileOperationsCarryTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		userEmail:                         {Output: "real@real.com"},
		verifyHead:                        {Output: "abc123"},
	})
	repo := NewRepo(runner)

	if err := repo.Ensure(ctx); err == nil {
		t.Error("Ensure tem de recusar o contexto cancelado")
	}
	if _, err := repo.Identity(ctx); err == nil {
		t.Error("Identity tem de recusar o contexto cancelado")
	}
	if _, err := repo.CommitCount(ctx, "real@real.com", "2026-08-15", "2026-08-15"); err == nil {
		t.Error("CommitCount tem de recusar o contexto cancelado")
	}
}
