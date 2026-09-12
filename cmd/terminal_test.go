package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestIsTerminalWithABufferIsFalse(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Fatal("um *bytes.Buffer não é um *os.File, não pode passar por terminal")
	}
}

func TestIsTerminalWithAFileThatIsNotATTYIsFalse(t *testing.T) {
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	defer file.Close()

	if isTerminal(file) {
		t.Fatal(os.DevNull + " não é um terminal")
	}
}
