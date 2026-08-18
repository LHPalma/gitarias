package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClientRefusesAContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := (HTTPClient{}).Get(ctx, "http://example.invalid"); !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v, queria o cancelamento", err)
	}
}

func TestHTTPClientCancelsAWhileWaitingForTheServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := (HTTPClient{}).Get(ctx, server.URL)
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("prazo vencido tem de virar erro")
	}
	if elapsed > 4*time.Second {
		t.Errorf("levou %v; a requisição tinha de morrer com o prazo, não esperar o servidor", elapsed)
	}
}
