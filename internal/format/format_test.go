package format

import (
	"runtime"
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
		{name: "caminho absoluto", chosen: JSON, path: "/tmp/dados", want: "/tmp/dados.json"},
		{name: "caminho para fora do repositorio", chosen: CSV, path: "../ignorados", want: "../ignorados.csv"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.chosen.Path(test.path, "ignorados")
			if err != nil {
				t.Fatalf("nao esperava erro, veio %v", err)
			}
			if got != test.want {
				t.Errorf("caminho = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestPathRefusesWhatNamesADirectory(t *testing.T) {
	for _, path := range []string{"saida/", "saida/2026/", "/tmp/"} {
		t.Run("recusa "+path, func(t *testing.T) {
			got, err := CSV.Path(path, "ignorados")

			if err == nil {
				t.Fatalf("%q viraria um arquivo oculto chamado .csv dentro do diretorio", path)
			}
			if got != "" {
				t.Errorf("caminho = %q, o recusado nao pode sair utilizavel", got)
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("erro = %v, tem de nomear o caminho recusado", err)
			}
		})
	}
}

func TestPathSuggestsAFileNameWhenRefusing(t *testing.T) {
	_, err := CSV.Path("saida/", "ignorados")

	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "saida/ignorados.csv") {
		t.Errorf("erro = %v, queria um exemplo pronto de caminho valido", err)
	}
}

func TestPathSuggestsTheNameOfTheCommandThatAsked(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "ignorados", want: "saida/ignorados.csv"},
		{name: "branches", want: "saida/branches.csv"},
		{name: "worktrees", want: "saida/worktrees.csv"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := CSV.Path("saida/", test.name)

			if err == nil {
				t.Fatal("esperava erro, veio nil")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("erro = %v, o exemplo tem de ser do comando que rodou e nao de outro", err)
			}
		})
	}
}

func TestPathRefusesTheEmptyPath(t *testing.T) {
	if _, err := CSV.Path("", "ignorados"); err == nil {
		t.Fatal("caminho vazio nao nomeia arquivo nenhum")
	}
}

func TestPathNamesTheDirectoryWithLiteralQuotes(t *testing.T) {
	_, err := CSV.Path("saida/", "ignorados")
	if err == nil {
		t.Fatal("caminho terminado em separador nomeia diretorio, nao arquivo")
	}
	if !strings.Contains(err.Error(), `"saida/"`) {
		t.Errorf("erro = %v, queria o caminho entre aspas", err)
	}
}

// TestPathNeverDoublesTheWindowsSeparatorInTheMessage só roda no Windows
// porque só lá a barra invertida é separador — em outro SO o "\" nem
// dispara o ramo de "termina em separador". %q escaparia a barra como \\,
// e a mensagem passaria a citar um caminho que não existe.
func TestPathNeverDoublesTheWindowsSeparatorInTheMessage(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("barra invertida so e separador de caminho no Windows")
	}

	_, err := CSV.Path(`saida\`, "ignorados")
	if err == nil {
		t.Fatal("caminho terminado em separador nomeia diretorio, nao arquivo")
	}
	if strings.Contains(err.Error(), `\\`) {
		t.Errorf("erro = %v; %%q dobraria a barra invertida, e o caminho pareceria nao existir", err)
	}
	if !strings.Contains(err.Error(), `"saida\"`) {
		t.Errorf("erro = %v, queria o caminho intacto entre aspas", err)
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
