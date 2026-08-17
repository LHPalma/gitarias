package ignore

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestIgnoreOperationsCarryTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	repo := NewRepo(gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {Output: ""},
	}))

	if err := repo.Ensure(ctx); err == nil {
		t.Error("Ensure tem de recusar o contexto cancelado")
	}
	if _, err := repo.List(ctx, false, nil); err == nil {
		t.Error("List tem de recusar o contexto cancelado")
	}
}
