package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

// configRepo monta as respostas mínimas para o gtr config: internal/config
// lê o .gtr.yaml do disco, não do fake de git, então rev-parse --show-toplevel
// tem de apontar para um diretório de verdade.
func configRepo(t *testing.T, gtrYAML string) map[string]gittest.Response {
	t.Helper()

	root := t.TempDir()
	if gtrYAML != "" {
		if err := os.WriteFile(filepath.Join(root, ".gtr.yaml"), []byte(gtrYAML), 0o644); err != nil {
			t.Fatalf("não consegui escrever o .gtr.yaml de teste: %v", err)
		}
	}

	return map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}}
}

func noPersonalConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestConfigCommandWithNothingConfigured(t *testing.T) {
	noPersonalConfig(t)
	result := execute(t, configRepo(t, ""), "", "config")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "CHAVE               VALOR  ORIGEM\n" +
		"branches.protected         padrão embutido\n" +
		"branches.base              padrão embutido\n"
	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestConfigCommandReadsTheRepoFile(t *testing.T) {
	noPersonalConfig(t)
	responses := configRepo(t, "branches:\n  protected: [develop, \"release/*\"]\n  base: develop\n")

	result := execute(t, responses, "", "config")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "CHAVE               VALOR               ORIGEM\n" +
		"branches.protected  develop, release/*  arquivo do repo\n" +
		"branches.base       develop             arquivo do repo\n"
	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestConfigCommandJSON(t *testing.T) {
	noPersonalConfig(t)
	responses := configRepo(t, "branches:\n  base: develop\n")

	result := execute(t, responses, "", "config", "--format", "json")

	var document configDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	want := configDocument{Config: []configRecord{
		{Key: "branches.protected", Value: "", Source: "default"},
		{Key: "branches.base", Value: "develop", Source: "repo"},
	}}
	if document.Config[0] != want.Config[0] || document.Config[1] != want.Config[1] {
		t.Errorf("documento = %+v, queria %+v", document, want)
	}
}

func TestConfigCommandCSV(t *testing.T) {
	noPersonalConfig(t)
	responses := configRepo(t, "branches:\n  protected: [develop, \"release/*\"]\n  base: develop\n")

	result := execute(t, responses, "", "config", "--format", "csv")

	want := "chave,valor,origem\n" +
		"branches.protected,\"develop, release/*\",arquivo do repo\n" +
		"branches.base,develop,arquivo do repo\n"
	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestConfigCommandWorksOutsideARepository(t *testing.T) {
	noPersonalConfig(t)
	responses := map[string]gittest.Response{}

	result := execute(t, responses, "", "config")

	if result.err != nil {
		t.Fatalf("gtr config não exige repositório, não esperava erro, veio %v", result.err)
	}
	if result.stdout == "" {
		t.Error("mesmo fora de um repositório, o padrão embutido tem de aparecer")
	}
}

func TestConfigCommandPropagatesUnsupportedLang(t *testing.T) {
	noPersonalConfig(t)
	responses := configRepo(t, "lang: pt-BR\nbranches:\n  base: develop\n")

	if execute(t, responses, "", "config").err == nil {
		t.Fatal("lang não suportada tem de virar erro, nunca ser ignorada em silêncio")
	}
}

func TestConfigCommandPropagatesMalformedYAML(t *testing.T) {
	noPersonalConfig(t)
	responses := configRepo(t, "branches: [")

	if execute(t, responses, "", "config").err == nil {
		t.Fatal("yaml malformado tem de virar erro")
	}
}

func TestConfigCommandRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	noPersonalConfig(t)
	responses := configRepo(t, "")

	if execute(t, responses, "", "config", "--format", "json", "--no-header").err == nil {
		t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
	}
}
