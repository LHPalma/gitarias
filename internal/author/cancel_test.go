package author

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestAuthorOperationsCarryTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		shortHead:                         {Output: "abc1234"},
		lastAuthor:                        {Output: "Real Person <real@real.com>"},
		amend:                             {Output: ""},
		rangeCount("deadbeef", "HEAD"):    {Output: "0"},
		dirtyCheck:                        {Output: ""},
		hardReset:                         {Output: ""},
	})
	repo := NewRepo(runner)

	if err := repo.Ensure(ctx); err == nil {
		t.Error("Ensure tem de recusar o contexto cancelado")
	}
	if _, err := repo.Plan(ctx, "", ""); err == nil {
		t.Error("Plan tem de recusar o contexto cancelado")
	}
	if err := repo.Rewrite(ctx, "", "", "Fake Name", "fake@fake.com"); err == nil {
		t.Error("Rewrite tem de recusar o contexto cancelado")
	}
	if _, err := repo.PlanReset(ctx, "deadbeef"); err == nil {
		t.Error("PlanReset tem de recusar o contexto cancelado")
	}
	if err := repo.Reset(ctx, "deadbeef"); err == nil {
		t.Error("Reset tem de recusar o contexto cancelado")
	}
}
