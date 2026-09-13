package tui

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewHelpMenuStoresTheStreams(t *testing.T) {
	input := strings.NewReader("")
	output := &bytes.Buffer{}

	menu := NewHelpMenu(input, output)

	if menu.input != input {
		t.Error("o input não foi guardado")
	}
	if menu.output != output {
		t.Error("o output não foi guardado")
	}
}

func TestHelpMenuRunsAndQuits(t *testing.T) {
	output := &bytes.Buffer{}

	err := NewHelpMenu(strings.NewReader("q"), output).
		Run([]MenuItem{{Name: "branches", Short: "as branches locais"}}, func(string) string { return "detalhe" })

	if err != nil {
		t.Fatalf("sair do menu não é erro, veio %v", err)
	}
	if output.Len() == 0 {
		t.Error("a tela saiu vazia; o menu tem de desenhar no output injetado")
	}
}

func TestHelpMenuPropagatesTheInputFailure(t *testing.T) {
	err := NewHelpMenu(erroredReader{}, &bytes.Buffer{}).
		Run([]MenuItem{{Name: "branches", Short: "as branches locais"}}, func(string) string { return "detalhe" })

	if err == nil {
		t.Fatal("entrada que falha tem de virar erro")
	}
}
