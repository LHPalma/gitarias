package format

import (
	"strings"
	"testing"
)

var offered = []string{"text", "csv"}

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
