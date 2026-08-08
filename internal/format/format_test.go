package format

import (
	"strings"
	"testing"
)

var offered = []string{"text", "csv", "tsv", "json"}

func TestParse(t *testing.T) {
	for _, name := range offered {
		t.Run(name, func(t *testing.T) {
			chosen, err := Parse(name)
			if err != nil {
				t.Fatalf("nao esperava erro, veio %v", err)
			}
			if string(chosen) != name {
				t.Errorf("formato = %q, queria %q", chosen, name)
			}
		})
	}
}

func TestParseRejectsWhatIsNotOffered(t *testing.T) {
	for _, name := range []string{"yaml", "xml", "CSV", "", "text "} {
		t.Run("recusa "+name, func(t *testing.T) {
			if _, err := Parse(name); err == nil {
				t.Fatalf("%q nao esta no conjunto e tem de virar erro", name)
			}
		})
	}
}

func TestParseNamesTheAlternatives(t *testing.T) {
	_, err := Parse("yaml")
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}

	for _, name := range offered {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("erro = %q, tem de dizer que %q existe", err, name)
		}
	}
}

func TestExtension(t *testing.T) {
	tests := []struct {
		chosen Format
		want   string
	}{
		{chosen: Text, want: ".txt"},
		{chosen: CSV, want: ".csv"},
		{chosen: TSV, want: ".tsv"},
		{chosen: JSON, want: ".json"},
	}

	for _, test := range tests {
		t.Run(string(test.chosen), func(t *testing.T) {
			if got := test.chosen.Extension(); got != test.want {
				t.Errorf("extensao = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestPath(t *testing.T) {
	tests := []struct {
		name   string
		chosen Format
		path   string
		want   string
	}{
		{name: "sem extensao ganha a do formato", chosen: CSV, path: "ignorados", want: "ignorados.csv"},
		{name: "json tambem", chosen: JSON, path: "dados", want: "dados.json"},
		{name: "extensao presente fica como veio", chosen: CSV, path: "dados.txt", want: "dados.txt"},
		{name: "extensao errada nao e corrigida", chosen: JSON, path: "dados.csv", want: "dados.csv"},
		{name: "caminho com diretorio", chosen: TSV, path: "saida/ignorados", want: "saida/ignorados.tsv"},
		{name: "diretorio com ponto no meio", chosen: CSV, path: "v1.2/ignorados", want: "v1.2/ignorados.csv"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.chosen.Path(test.path); got != test.want {
				t.Errorf("caminho = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestDelimited(t *testing.T) {
	tests := []struct {
		chosen Format
		want   bool
	}{
		{chosen: CSV, want: true},
		{chosen: TSV, want: true},
		{chosen: Text},
		{chosen: JSON},
	}

	for _, test := range tests {
		t.Run(string(test.chosen), func(t *testing.T) {
			if got := test.chosen.Delimited(); got != test.want {
				t.Errorf("delimitado = %v, queria %v", got, test.want)
			}
		})
	}
}
