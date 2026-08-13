package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var required = regexp.MustCompile(`(?m)^\s*(github\.com/[^\s]+|[a-z0-9.\-]+\.[a-z]{2,}/[^\s]+)\s+v[^\s]+`)

func modules(t *testing.T) []string {
	t.Helper()

	content, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("nao consegui ler o go.mod: %v", err)
	}

	var found []string
	for _, match := range required.FindAllStringSubmatch(string(content), -1) {
		found = append(found, match[1])
	}

	return found
}

func TestEveryDependencyIsInTheThirdPartyNotices(t *testing.T) {
	listed := modules(t)
	if len(listed) == 0 {
		t.Fatal("o go.mod nao declarou dependencia nenhuma; o teste passaria de graca")
	}

	if strings.Contains(notices, "PLACEHOLDER") {
		t.Fatal("o arquivo embutido e um marcador, nao o conteudo")
	}

	for _, module := range listed {
		if !strings.Contains(notices, module) {
			t.Errorf("%q esta no go.mod e nao no THIRD-PARTY-LICENSES; o binario redistribui o codigo dela sem a licenca", module)
		}
	}
}

func TestTheNoticesCarryTheLicenceTexts(t *testing.T) {
	for _, marker := range []string{"Apache License", "Redistribution and use in source and binary forms"} {
		if !strings.Contains(notices, marker) {
			t.Errorf("o texto %q nao esta no arquivo; resumir a licenca nao cumpre a exigencia de acompanhar a redistribuicao", marker)
		}
	}
}

func TestTheEmbeddedNoticesMatchTheFileOnDisk(t *testing.T) {
	content, err := os.ReadFile("THIRD-PARTY-LICENSES")
	if err != nil {
		t.Fatalf("nao consegui ler o arquivo: %v", err)
	}

	if string(content) != notices {
		t.Error("o embutido divergiu do arquivo; quem le o repositorio e quem recebe o binario veriam coisas diferentes")
	}
}

func TestTheProjectLicenceIsPresentAndNamesTheHolder(t *testing.T) {
	content, err := os.ReadFile("LICENSE")
	if err != nil {
		t.Fatalf("repositorio publico sem licenca e todos os direitos reservados: %v", err)
	}

	for _, marker := range []string{"MIT License", "Luiz Palma"} {
		if !strings.Contains(string(content), marker) {
			t.Errorf("o LICENSE nao contem %q", marker)
		}
	}
}
