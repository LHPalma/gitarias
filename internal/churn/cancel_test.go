package churn

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestChurnOperationsCarryTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	repo := NewRepo(gittest.NewRunner(withHistory("main.go\x00", "main.go\x00")))

	if err := repo.Ensure(ctx); err == nil {
		t.Error("Ensure tem de recusar o contexto cancelado")
	}
	if _, err := repo.Busiest(ctx, 0); err == nil {
		t.Error("Busiest tem de recusar o contexto cancelado")
	}
}
