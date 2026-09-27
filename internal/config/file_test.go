package config

import (
	"slices"
	"strings"
	"testing"
)

func TestParseFile(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantError     string
		wantLang      string
		wantBase      string
		wantProtected []string
	}{
		{
			name:    "arquivo vazio",
			content: "",
		},
		{
			name:          "branches completo",
			content:       "branches:\n  protected: [develop, \"release/*\"]\n  base: develop\n",
			wantBase:      "develop",
			wantProtected: []string{"develop", "release/*"},
		},
		{
			name:     "lang en explicito, permitido",
			content:  "lang: en\nbranches:\n  base: develop\n",
			wantLang: "en",
			wantBase: "develop",
		},
		{
			name:      "lang nao suportada",
			content:   "lang: pt-BR\nbranches:\n  protected: [protegidas]\n",
			wantError: `lang "pt-BR" não suportada`,
		},
		{
			name:      "yaml malformado",
			content:   "branches: [",
			wantError: "arquivo de configuração",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseFile("/caminho/config.yaml", []byte(test.content))

			if test.wantError != "" {
				if err == nil {
					t.Fatalf("esperava erro contendo %q, veio nil", test.wantError)
				}
				if !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("erro %q não contém %q", err, test.wantError)
				}
				return
			}

			if err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}
			if got.Lang != test.wantLang {
				t.Errorf("Lang = %q, queria %q", got.Lang, test.wantLang)
			}
			if got.Branches.Base != test.wantBase {
				t.Errorf("Branches.Base = %q, queria %q", got.Branches.Base, test.wantBase)
			}
			if !slices.Equal(got.Branches.Protected, test.wantProtected) {
				t.Errorf("Branches.Protected = %v, queria %v", got.Branches.Protected, test.wantProtected)
			}
		})
	}
}
