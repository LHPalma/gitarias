package riff

import (
	"context"
	"testing"

	"github.com/LHPalma/gitarias/internal/web/webtest"
)

func TestRandomCarriesTheCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	client := webtest.NewClient(webtest.Response{Output: "não devia sair"})

	if _, err := NewRepo(client).Random(ctx); err == nil {
		t.Error("Random tem de recusar o contexto cancelado")
	}
}
