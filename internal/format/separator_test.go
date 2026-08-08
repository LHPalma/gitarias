package format

import (
	"strings"
	"testing"
)

func TestParseSeparator(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  Separator
	}{
		{name: "virgula", value: ",", want: Comma},
		{name: "ponto e virgula", value: ";", want: Semicolon},
		{name: "barra vertical", value: "|", want: Pipe},
		{name: "tabulacao de verdade", value: "\t", want: Tab},
		{name: "tabulacao escrita como barra t", value: "\\t", want: Tab},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			separator, err := ParseSeparator(test.value)
			if err != nil {
				t.Fatalf("nao esperava erro, veio %v", err)
			}
			if separator != test.want {
				t.Errorf("separador = %q, queria %q", rune(separator), rune(test.want))
			}
		})
	}
}

func TestParseSeparatorKeepsTheSetClosed(t *testing.T) {
	for _, value := range []string{":", " ", "::", "", "\n", "\\n", "ç"} {
		t.Run("recusa "+value, func(t *testing.T) {
			if _, err := ParseSeparator(value); err == nil {
				t.Fatalf("%q esta fora do conjunto e tem de virar erro", value)
			}
		})
	}
}

func TestParseSeparatorRefusedLeavesNothingUsable(t *testing.T) {
	separator, err := ParseSeparator(":")

	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if separator != 0 {
		t.Errorf("separador = %q, o recusado tem de sair zerado; devolver a virgula faria quem ignorar o erro gravar um csv que ninguem pediu", rune(separator))
	}
}

func TestParseSeparatorNamesTheAccepted(t *testing.T) {
	_, err := ParseSeparator(":")
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}

	for _, accepted := range []string{",", ";", "|", "\\t"} {
		if !strings.Contains(err.Error(), accepted) {
			t.Errorf("erro = %q, tem de listar %q entre os aceitos", err, accepted)
		}
	}
}
