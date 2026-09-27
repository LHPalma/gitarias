package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestMergeList(t *testing.T) {
	tests := []struct {
		name       string
		repo       []string
		personal   []string
		want       []string
		wantSource Source
	}{
		{name: "nenhum define", wantSource: SourceDefault},
		{name: "só o repo define", repo: []string{"develop"}, want: []string{"develop"}, wantSource: SourceRepo},
		{name: "só o pessoal define", personal: []string{"develop"}, want: []string{"develop"}, wantSource: SourcePersonal},
		{name: "os dois definem, vence o repo", repo: []string{"develop"}, personal: []string{"outra"}, want: []string{"develop"}, wantSource: SourceRepo},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, source := mergeList(test.repo, test.personal)
			if source != test.wantSource {
				t.Errorf("origem = %v, queria %v", source, test.wantSource)
			}
			if len(got) != len(test.want) {
				t.Fatalf("valor = %v, queria %v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Errorf("valor = %v, queria %v", got, test.want)
				}
			}
		})
	}
}

func TestMergeString(t *testing.T) {
	tests := []struct {
		name       string
		repo       string
		personal   string
		want       string
		wantSource Source
	}{
		{name: "nenhum define", wantSource: SourceDefault},
		{name: "só o repo define", repo: "develop", want: "develop", wantSource: SourceRepo},
		{name: "só o pessoal define", personal: "develop", want: "develop", wantSource: SourcePersonal},
		{name: "os dois definem, vence o repo", repo: "develop", personal: "outra", want: "develop", wantSource: SourceRepo},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, source := mergeString(test.repo, test.personal)
			if got != test.want || source != test.wantSource {
				t.Errorf("valor/origem = %q/%v, queria %q/%v", got, source, test.want, test.wantSource)
			}
		})
	}
}

// writeRepo cria um diretório real que representa a raiz do repositório e
// devolve um Runner roteirizado para apontar rev-parse --show-toplevel ali.
func writeRepo(t *testing.T, content string) *gittest.Runner {
	t.Helper()

	root := t.TempDir()
	if content != "" {
		if err := os.WriteFile(filepath.Join(root, ".gtr.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("não consegui escrever o .gtr.yaml de teste: %v", err)
		}
	}

	return gittest.NewRunner(map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}})
}

func writePersonal(t *testing.T, content string) {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if content == "" {
		return
	}

	if err := os.MkdirAll(filepath.Join(dir, "gtr"), 0o755); err != nil {
		t.Fatalf("não consegui criar o diretório pessoal de teste: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gtr", "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("não consegui escrever o config.yaml pessoal de teste: %v", err)
	}
}

func TestLoad(t *testing.T) {
	t.Run("nenhum arquivo em nenhuma camada", func(t *testing.T) {
		writePersonal(t, "")
		runner := writeRepo(t, "")

		cfg, err := Load(t.Context(), runner)
		if err != nil {
			t.Fatalf("não esperava erro, veio %v", err)
		}
		if cfg.Branches.ProtectedSource != SourceDefault || cfg.Branches.BaseSource != SourceDefault {
			t.Errorf("origens = %v/%v, queria default/default", cfg.Branches.ProtectedSource, cfg.Branches.BaseSource)
		}
	})

	t.Run("só o pessoal define base", func(t *testing.T) {
		writePersonal(t, "branches:\n  base: develop\n")
		runner := writeRepo(t, "")

		cfg, err := Load(t.Context(), runner)
		if err != nil {
			t.Fatalf("não esperava erro, veio %v", err)
		}
		if cfg.Branches.Base != "develop" || cfg.Branches.BaseSource != SourcePersonal {
			t.Errorf("base/origem = %q/%v, queria develop/personal", cfg.Branches.Base, cfg.Branches.BaseSource)
		}
		if cfg.Branches.ProtectedSource != SourceDefault {
			t.Errorf("protected source = %v, queria default", cfg.Branches.ProtectedSource)
		}
	})

	t.Run("repo e pessoal definem base, vence o repo", func(t *testing.T) {
		writePersonal(t, "branches:\n  base: outra\n")
		runner := writeRepo(t, "branches:\n  base: develop\n")

		cfg, err := Load(t.Context(), runner)
		if err != nil {
			t.Fatalf("não esperava erro, veio %v", err)
		}
		if cfg.Branches.Base != "develop" || cfg.Branches.BaseSource != SourceRepo {
			t.Errorf("base/origem = %q/%v, queria develop/repo", cfg.Branches.Base, cfg.Branches.BaseSource)
		}
	})

	t.Run("repo define protected, pessoal define base, cada chave da sua camada", func(t *testing.T) {
		writePersonal(t, "branches:\n  base: develop\n")
		runner := writeRepo(t, "branches:\n  protected: [release/*]\n")

		cfg, err := Load(t.Context(), runner)
		if err != nil {
			t.Fatalf("não esperava erro, veio %v", err)
		}
		if cfg.Branches.Base != "develop" || cfg.Branches.BaseSource != SourcePersonal {
			t.Errorf("base/origem = %q/%v, queria develop/personal", cfg.Branches.Base, cfg.Branches.BaseSource)
		}
		if len(cfg.Branches.Protected) != 1 || cfg.Branches.Protected[0] != "release/*" || cfg.Branches.ProtectedSource != SourceRepo {
			t.Errorf("protected/origem = %v/%v, queria [release/*]/repo", cfg.Branches.Protected, cfg.Branches.ProtectedSource)
		}
	})

	t.Run("fora de um repositorio, camada de repo fica ausente sem erro", func(t *testing.T) {
		writePersonal(t, "branches:\n  base: develop\n")
		runner := gittest.NewRunner(map[string]gittest.Response{"rev-parse --show-toplevel": {Err: errors.New("fatal: not a git repository")}})

		cfg, err := Load(t.Context(), runner)
		if err != nil {
			t.Fatalf("não esperava erro, veio %v", err)
		}
		if cfg.Branches.Base != "develop" || cfg.Branches.BaseSource != SourcePersonal {
			t.Errorf("base/origem = %q/%v, queria develop/personal", cfg.Branches.Base, cfg.Branches.BaseSource)
		}
	})

	t.Run("lang nao suportada no arquivo do repo propaga erro", func(t *testing.T) {
		writePersonal(t, "")
		runner := writeRepo(t, "lang: pt-BR\nbranches:\n  base: develop\n")

		if _, err := Load(t.Context(), runner); err == nil {
			t.Fatal("esperava erro de lang não suportada")
		}
	})

	t.Run("yaml malformado no arquivo pessoal propaga erro", func(t *testing.T) {
		writePersonal(t, "branches: [")
		runner := writeRepo(t, "")

		if _, err := Load(t.Context(), runner); err == nil {
			t.Fatal("esperava erro de yaml malformado")
		}
	})

	t.Run("caminho de configuração que não é um arquivo comum propaga erro", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)
		if err := os.MkdirAll(filepath.Join(dir, "gtr", "config.yaml"), 0o755); err != nil {
			t.Fatalf("não consegui montar o caso de teste: %v", err)
		}
		runner := writeRepo(t, "")

		if _, err := Load(t.Context(), runner); err == nil {
			t.Fatal("esperava erro ao ler um diretório como se fosse o arquivo de configuração")
		}
	})
}

func TestConfigEntries(t *testing.T) {
	cfg := Config{Branches: Branches{
		Protected:       []string{"develop", "release/*"},
		ProtectedSource: SourceRepo,
		Base:            "develop",
		BaseSource:      SourceRepo,
	}}

	entries := cfg.Entries()
	if len(entries) != 2 {
		t.Fatalf("esperava 2 entradas, veio %d", len(entries))
	}

	protected, base := entries[0], entries[1]
	if protected.Key != "branches.protected" || protected.Value != "develop, release/*" || protected.Source != SourceRepo {
		t.Errorf("entrada protected = %+v", protected)
	}
	if base.Key != "branches.base" || base.Value != "develop" || base.Source != SourceRepo {
		t.Errorf("entrada base = %+v", base)
	}
}

func TestConfigEntriesWithNothingConfigured(t *testing.T) {
	entries := Config{}.Entries()

	for _, entry := range entries {
		if entry.Value != "" {
			t.Errorf("entrada %q = %q, queria vazio sem configuração nenhuma", entry.Key, entry.Value)
		}
		if entry.Source != SourceDefault {
			t.Errorf("entrada %q origem = %v, queria default", entry.Key, entry.Source)
		}
	}
}
