package cmd

import (
	"bytes"

	"github.com/LHPalma/gitarias/tui"
	"github.com/spf13/cobra"
)

// runInteractiveHelp é o RunE do comando raiz: acionado só quando o `gtr` é
// chamado sem subcomando nenhum. Sem terminal, cai para o help padrão do
// Cobra — a escolha não veio de flag explícita, então o silêncio é a
// resposta certa (ADR-002). Com terminal, abre o menu navegável.
func runInteractiveHelp(command *cobra.Command) error {
	if !isTerminal(command.OutOrStdout()) {
		return command.Help()
	}

	items, subcommands := menuItems(command)

	menu := tui.NewHelpMenu(command.InOrStdin(), command.OutOrStdout())

	return menu.Run(items, func(name string) string {
		return helpText(subcommands[name])
	})
}

// menuItems espelha o filtro que o próprio template padrão do Cobra usa
// para a seção "Available Commands": oculta os `Hidden` e `Deprecated`, mas
// mantém o `help` auto-gerado.
func menuItems(command *cobra.Command) ([]tui.MenuItem, map[string]*cobra.Command) {
	subcommands := map[string]*cobra.Command{}
	var items []tui.MenuItem

	for _, subcommand := range command.Commands() {
		if !subcommand.IsAvailableCommand() && subcommand.Name() != "help" {
			continue
		}

		items = append(items, tui.MenuItem{Name: subcommand.Name(), Short: subcommand.Short})
		subcommands[subcommand.Name()] = subcommand
	}

	return items, subcommands
}

// helpText captura o que `gtr <comando> --help` imprimiria, sem deixar essa
// escrita vazar direto pro terminal por baixo da TUI — o mesmo redirecionamento
// temporário que o próprio Cobra usa dentro de UsageString.
func helpText(subcommand *cobra.Command) string {
	output := &bytes.Buffer{}

	subcommand.SetOut(output)
	defer subcommand.SetOut(nil)

	subcommand.Help()

	return output.String()
}
