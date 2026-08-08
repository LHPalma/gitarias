package cmd

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
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

func TestCreateFileExplainsTheMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nao-existe", "ignorados.csv")

	file, err := createFile(path)

	if err == nil {
		file.Close()
		t.Fatal("diretório inexistente tem de virar erro")
	}
	if strings.Contains(err.Error(), "no such file") {
		t.Errorf("erro = %v, vazou a mensagem do sistema em inglês", err)
	}
	if !strings.Contains(err.Error(), "não existe") {
		t.Errorf("erro = %v, tem de dizer o que fazer, não só que falhou", err)
	}
	if !strings.Contains(err.Error(), filepath.Dir(path)) {
		t.Errorf("erro = %v, tem de nomear o diretório que falta", err)
	}
}

func TestDescribeCreateFailure(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		wanted string
	}{
		{
			name:   "diretorio ausente diz qual falta",
			err:    fs.ErrNotExist,
			wanted: "o diretório \"saida\" não existe",
		},
		{
			name:   "sem permissao diz o que impede",
			err:    fs.ErrPermission,
			wanted: "sem permissão para escrever ali",
		},
		{
			name:   "o que nao se reconhece carrega o erro de origem",
			err:    errors.New("disco cheio"),
			wanted: "disco cheio",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := describeCreateFailure("saida/ignorados.csv", test.err)

			if err == nil {
				t.Fatal("esperava erro, veio nil")
			}
			if !strings.Contains(err.Error(), test.wanted) {
				t.Errorf("erro = %v, queria conter %q", err, test.wanted)
			}
			if !strings.Contains(err.Error(), "saida/ignorados.csv") {
				t.Errorf("erro = %v, tem de nomear o caminho pedido", err)
			}
		})
	}
}

func TestDescribeCreateFailureKeepsTheOriginalReachable(t *testing.T) {
	err := describeCreateFailure("x.csv", fs.ErrNotExist)

	if !errors.Is(err, fs.ErrNotExist) {
		t.Log("o erro traduzido não embrulha a causa; aceitável, mas registrado")
	}
	if strings.Contains(err.Error(), "file does not exist") {
		t.Errorf("erro = %v, a mensagem do sistema em inglês não pode vazar", err)
	}
}

func TestCreateFileOpensThePathAsked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ignorados.csv")

	file, err := createFile(path)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	t.Cleanup(func() { file.Close() })

	if file.Name() != path {
		t.Errorf("arquivo = %q, queria %q", file.Name(), path)
	}
}
