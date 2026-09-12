package cmd

import (
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestBareCommandWithoutTerminalFallsBackToTheDefaultHelp(t *testing.T) {
	result := execute(t, nil, "")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Available Commands:") {
		t.Errorf("saída = %q, queria o help padrão do Cobra", result.stdout)
	}
	if !strings.Contains(result.stdout, "branches") {
		t.Errorf("saída = %q, queria listar os comandos, inclusive branches", result.stdout)
	}
}

func TestMenuItemsHidesHiddenCommandsButKeepsHelp(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(nil), noCommands(), noWeb(), noFinder(), noNotices)
	command.InitDefaultHelpCmd() // o que o Execute() já faz antes de rodar o RunE

	items, subcommands := menuItems(command)

	var names []string
	for _, item := range items {
		names = append(names, item.Name)
	}

	for _, hidden := range names {
		if hidden == "favorite-band" {
			t.Fatalf("favorite-band é Hidden e não podia aparecer, veio %v", names)
		}
	}

	if _, found := subcommands["branches"]; !found {
		t.Fatalf("esperava branches entre os subcomandos, veio %v", names)
	}
	if _, found := subcommands["help"]; !found {
		t.Fatalf("help é auto-gerado e deveria aparecer mesmo assim, veio %v", names)
	}
}

func TestMenuItemsCarryTheShortDescription(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(nil), noCommands(), noWeb(), noFinder(), noNotices)

	items, _ := menuItems(command)

	for _, item := range items {
		if item.Name == "branches" {
			if item.Short == "" {
				t.Fatal("branches deveria ter uma descrição curta")
			}
			return
		}
	}
	t.Fatal("branches não apareceu na lista")
}

func TestHelpTextMatchesTheCommandsOwnHelp(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(nil), noCommands(), noWeb(), noFinder(), noNotices)
	_, subcommands := menuItems(command)

	text := helpText(subcommands["branches"])

	if !strings.Contains(text, "Lista branches locais já mergeadas na branch base") {
		t.Errorf("help capturado = %q, queria a descrição do comando", text)
	}
	if !strings.Contains(text, "--clean") {
		t.Errorf("help capturado = %q, queria as flags do comando", text)
	}
}

func TestHelpTextResetsTheOutputAfterCapturing(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(nil), noCommands(), noWeb(), noFinder(), noNotices)
	_, subcommands := menuItems(command)

	helpText(subcommands["branches"])

	result := execute(t, nil, "", "branches", "--help")

	if !strings.Contains(result.stdout, "--clean") {
		t.Errorf("depois de capturar o help uma vez, a chamada normal parou de funcionar: %q", result.stdout)
	}
}
