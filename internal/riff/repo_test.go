package riff

import (
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/web/webtest"
)

func TestRandom(t *testing.T) {
	client := webtest.NewClient(webtest.Response{Output: "Blaming regex.\n"})

	message, err := NewRepo(client).Random(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if message != "Blaming regex." {
		t.Errorf("mensagem = %q, queria %q (sem a quebra de linha)", message, "Blaming regex.")
	}

	if len(client.Calls) != 1 || client.Calls[0] != source {
		t.Errorf("chamadas = %v, queria uma só, para %q", client.Calls, source)
	}
}

func TestRandomPropagatesTheClientFailure(t *testing.T) {
	client := webtest.NewClient(webtest.Response{Err: errors.New("fora do ar")})

	if _, err := NewRepo(client).Random(t.Context()); err == nil {
		t.Fatal("falha do client tem de virar erro")
	}
}

func TestRandomRefusesAnEmptyMessage(t *testing.T) {
	client := webtest.NewClient(webtest.Response{Output: "   \n"})

	if _, err := NewRepo(client).Random(t.Context()); err == nil {
		t.Fatal("corpo vazio (ou só espaço) tem de virar erro, não mensagem em branco")
	}
}
