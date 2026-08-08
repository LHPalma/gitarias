package format

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestWriteByteOrderMark(t *testing.T) {
	output := &bytes.Buffer{}

	if err := WriteByteOrderMark(output); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if !bytes.Equal(output.Bytes(), []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("bytes = %v, queria os tres do BOM; sem eles o Excel le relatÃ³rio", output.Bytes())
	}
}

func TestWriteByteOrderMarkPropagatesTheWriteError(t *testing.T) {
	if err := WriteByteOrderMark(brokenWriter{}); err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestWriteCSVLeavesTheByteOrderMarkOut(t *testing.T) {
	output := &bytes.Buffer{}

	if err := WriteCSV(output, Comma, [][]string{{"a"}}); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if output.String() != "a\n" {
		t.Errorf("saida = %q, o BOM e do destino e nao do formato; num pipe ele gruda no primeiro campo", output.String())
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

			if got := output.String(); got != test.want {
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

	got := output.String()
	if got != "\"relat;orio\";b\n" {
		t.Errorf("saida = %q, o separador dentro do campo tem de sair citado", got)
	}
}

func TestWriteCSVWithoutRows(t *testing.T) {
	output := &bytes.Buffer{}

	if err := WriteCSV(output, Comma, nil); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if output.String() != "" {
		t.Errorf("saida = %q, sem linha nada sai", output.String())
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
