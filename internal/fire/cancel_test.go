package fire

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestFireOperationsCarryTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		status:                            {Output: " M f.txt\n"},
		addAll:                            {Output: ""},
		commit:                            {Output: ""},
		previousHead:                      {Output: "abc1234"},
		newHead:                           {Output: "def5678"},
	})
	repo := NewRepo(runner)

	if err := repo.Ensure(ctx); err == nil {
		t.Error("Ensure tem de recusar o contexto cancelado")
	}
	if _, err := repo.Dirty(ctx); err == nil {
		t.Error("Dirty tem de recusar o contexto cancelado")
	}
	if _, err := repo.Save(ctx, "m"); err == nil {
		t.Error("Save tem de recusar o contexto cancelado")
	}
}
