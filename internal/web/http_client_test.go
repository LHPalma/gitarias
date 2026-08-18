package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPClientGetsTheBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("oi"))
	}))
	defer server.Close()

	body, err := (HTTPClient{}).Get(t.Context(), server.URL)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if body != "oi" {
		t.Errorf("corpo = %q, queria %q", body, "oi")
	}
}

func TestHTTPClientFailsOnANonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	if _, err := (HTTPClient{}).Get(t.Context(), server.URL); err == nil {
		t.Fatal("status diferente de 200 tem de virar erro")
	}
}

func TestHTTPClientFailsWhenTheServerIsUnreachable(t *testing.T) {
	if _, err := (HTTPClient{}).Get(t.Context(), "http://127.0.0.1:1"); err == nil {
		t.Fatal("servidor inalcançável tem de virar erro")
	}
}

func TestHTTPClientFailsOnAnInvalidURL(t *testing.T) {
	if _, err := (HTTPClient{}).Get(t.Context(), ":// não é url"); err == nil {
		t.Fatal("URL inválida tem de virar erro antes de qualquer requisição")
	}
}

func TestHTTPClientFailsWhenTheBodyCannotBeRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "10")
		writer.Write([]byte("ab"))
	}))
	defer server.Close()

	if _, err := (HTTPClient{}).Get(t.Context(), server.URL); err == nil {
		t.Fatal("corpo cortado (menos bytes do que o Content-Length prometia) tem de virar erro na leitura")
	}
}

func TestHTTPClientCarriesTheStatusInTheMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	_, err := (HTTPClient{}).Get(t.Context(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "418") {
		t.Errorf("erro = %v, queria o código de status na mensagem", err)
	}
}
