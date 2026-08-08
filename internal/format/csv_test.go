package format

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestWriteCSVStartsWithTheByteOrderMark(t *testing.T) {
	output := &bytes.Buffer{}

	if err := WriteCSV(output, Comma, [][]string{{"relatório 2026.csv"}}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if !bytes.HasPrefix(output.Bytes(), []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("bytes = %v, queria o BOM na frente; sem ele o Excel le relatÃ³rio", output.Bytes()[:3])
	}
}

func TestWriteCSVHonoursTheSeparator(t *testing.T) {
	tests := []struct {
		name      string
		separator Separator
		want      string
	}{
		{name: "virgula", separator: Comma, want: "a,b\n"},
		{name: "ponto e virgula", separator: Semicolon, want: "a;b\n"},
		{name: "barra vertical", separator: Pipe, want: "a|b\n"},
		{name: "tabulacao", separator: Tab, want: "a\tb\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := &bytes.Buffer{}

			if err := WriteCSV(output, test.separator, [][]string{{"a", "b"}}); err != nil {
				t.Fatalf("nao esperava erro, veio %v", err)
			}

			if got := strings.TrimPrefix(output.String(), byteOrderMark); got != test.want {
				t.Errorf("saida = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestWriteCSVQuotesTheSeparatorInsideAField(t *testing.T) {
	output := &bytes.Buffer{}

	if err := WriteCSV(output, Semicolon, [][]string{{"relat;orio", "b"}}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	got := strings.TrimPrefix(output.String(), byteOrderMark)
	if got != "\"relat;orio\";b\n" {
		t.Errorf("saida = %q, o separador dentro do campo tem de sair citado", got)
	}
}

func TestWriteCSVWithoutRows(t *testing.T) {
	output := &bytes.Buffer{}

	if err := WriteCSV(output, Comma, nil); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if output.String() != byteOrderMark {
		t.Errorf("saida = %q, sem linha so o BOM sai", output.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) {
	return 0, errors.New("disco cheio")
}

func TestWriteCSVPropagatesTheWriteError(t *testing.T) {
	err := WriteCSV(brokenWriter{}, Comma, [][]string{{"a"}})

	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "disco cheio") {
		t.Errorf("erro = %v, queria o da escrita", err)
	}
}
