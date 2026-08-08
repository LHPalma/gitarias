package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type closingWriter struct {
	written  *bytes.Buffer
	writeErr error
	closeErr error
	closed   bool
}

func (writer *closingWriter) Write(bytes []byte) (int, error) {
	if writer.writeErr != nil {
		return 0, writer.writeErr
	}

	return writer.written.Write(bytes)
}

func (writer *closingWriter) Close() error {
	writer.closed = true

	return writer.closeErr
}

func writing(text string) func(io.Writer) error {
	return func(output io.Writer) error {
		_, err := io.WriteString(output, text)
		return err
	}
}

func TestWriteAndCloseAlwaysCloses(t *testing.T) {
	destination := &closingWriter{written: &bytes.Buffer{}}

	if err := writeAndClose(destination, writing("conteúdo")); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !destination.closed {
		t.Error("o arquivo tem de ser fechado, senão o conteúdo pode não chegar ao disco")
	}
	if destination.written.String() != "conteúdo" {
		t.Errorf("escrito = %q, queria o conteúdo", destination.written)
	}
}

func TestWriteAndCloseReportsTheCloseFailure(t *testing.T) {
	destination := &closingWriter{written: &bytes.Buffer{}, closeErr: errors.New("disco cheio no flush")}

	err := writeAndClose(destination, writing("conteúdo"))

	if err == nil {
		t.Fatal("falha no close perde dado em silêncio e tem de virar erro")
	}
	if !strings.Contains(err.Error(), "disco cheio no flush") {
		t.Errorf("erro = %v, queria o do close", err)
	}
}

func TestWriteAndCloseKeepsTheWriteFailureAhead(t *testing.T) {
	destination := &closingWriter{
		written:  &bytes.Buffer{},
		writeErr: errors.New("falha na escrita"),
		closeErr: errors.New("falha no close"),
	}

	err := writeAndClose(destination, writing("conteúdo"))

	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "falha na escrita") {
		t.Errorf("erro = %v, queria a causa de origem e não a consequência", err)
	}
	if !destination.closed {
		t.Error("falha na escrita não dispensa fechar o arquivo")
	}
}
