package format

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSONWithoutItems(t *testing.T) {
	output := &bytes.Buffer{}

	if err := WriteJSON(output, []string{}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if output.String() != "[]\n" {
		t.Errorf("saida = %q, lista vazia sai como [] e nunca como null", output.String())
	}
}

func TestWriteJSONKeepsGlobCharactersLiteral(t *testing.T) {
	output := &bytes.Buffer{}

	if err := WriteJSON(output, []string{"a&b", "<dist>", "*.log"}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	for _, wanted := range []string{"a&b", "<dist>", "*.log"} {
		if !strings.Contains(output.String(), wanted) {
			t.Errorf("saida = %q, queria %q sem virar escape unicode", output.String(), wanted)
		}
	}
}

func TestWriteJSONPropagatesTheWriteError(t *testing.T) {
	if err := WriteJSON(brokenWriter{}, []string{"a"}); err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}
