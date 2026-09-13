package profile

import (
	"context"
	"testing"
	"time"

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
	if _, err := repo.Streaks(ctx, "real@real.com", time.Now()); err == nil {
		t.Error("Streaks tem de recusar o contexto cancelado")
	}
	if _, err := repo.CommitCountByHour(ctx, "real@real.com", "2026-09-01", "2026-09-13"); err == nil {
		t.Error("CommitCountByHour tem de recusar o contexto cancelado")
	}
	if _, err := repo.CommitCountByWeekday(ctx, "real@real.com", "2026-09-01", "2026-09-13"); err == nil {
		t.Error("CommitCountByWeekday tem de recusar o contexto cancelado")
	}
}
