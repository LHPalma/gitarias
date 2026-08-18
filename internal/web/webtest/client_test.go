package webtest

import (
	"context"
	"errors"
	"testing"
)

func TestClientAnswersInOrder(t *testing.T) {
	client := NewClient(
		Response{Output: "primeiro"},
		Response{Output: "segundo"},
	)

	first, _ := client.Get(context.Background(), "https://um.example")
	second, _ := client.Get(context.Background(), "https://dois.example")

	if first != "primeiro" || second != "segundo" {
		t.Errorf("respostas = %q e %q, queria na ordem", first, second)
	}
	if len(client.Calls) != 2 || client.Calls[0] != "https://um.example" || client.Calls[1] != "https://dois.example" {
		t.Errorf("chamadas = %v", client.Calls)
	}
}

func TestClientPropagatesTheScriptedError(t *testing.T) {
	client := NewClient(Response{Err: errors.New("fora do ar")})

	if _, err := client.Get(context.Background(), "https://um.example"); err == nil {
		t.Fatal("erro roteirizado tem de sair")
	}
}

func TestClientRefusesACallWithoutScript(t *testing.T) {
	client := NewClient(Response{})

	client.Get(context.Background(), "https://um.example")

	if _, err := client.Get(context.Background(), "https://dois.example"); err == nil {
		t.Fatal("chamada a mais tem de falhar alto, senão o teste passa de graça")
	}
}

func TestClientHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient(Response{Output: "não devia sair"})

	if _, err := client.Get(ctx, "https://um.example"); !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v; o fake tem de cancelar como o de verdade, senão o teste do domínio mente", err)
	}
	if len(client.Calls) != 0 {
		t.Errorf("chamadas = %v, nada roda com o contexto cancelado", client.Calls)
	}
}
