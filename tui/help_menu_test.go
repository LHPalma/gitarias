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
