package tui

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewBranchSelectorStoresTheStreams(t *testing.T) {
	input := strings.NewReader("")
	output := &bytes.Buffer{}

	selector := NewBranchSelector(input, output)

	if selector.input != input {
		t.Error("o input não foi guardado")
	}
	if selector.output != output {
		t.Error("o output não foi guardado")
	}
}
