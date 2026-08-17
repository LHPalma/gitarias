package stats

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestStatsOperationsCarryTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	repo := NewRepo(gittest.NewRunner(withHistory("anteninha\x00anteninha@teste.com\n")))

	if err := repo.Ensure(ctx); err == nil {
		t.Error("Ensure tem de recusar o contexto cancelado")
	}
	if _, err := repo.ByAuthor(ctx, nil); err == nil {
		t.Error("ByAuthor tem de recusar o contexto cancelado")
	}
}
