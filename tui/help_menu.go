package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

// Detail busca o texto de ajuda completo de um MenuItem, pelo nome.
type Detail func(name string) string

// HelpMenu é a lista interativa de comandos que um `gtr` sem argumento
// mostra: as setas navegam, enter mostra o help de um comando no lugar da
// lista, esc volta. Run assume items não vazio; filtrar a árvore de
// comandos até sobrar nada não é um caso que quem chama consiga produzir.
type HelpMenu struct {
	input  io.Reader
	output io.Writer
}

func NewHelpMenu(input io.Reader, output io.Writer) HelpMenu {
	return HelpMenu{input: input, output: output}
}

func (menu HelpMenu) Run(items []MenuItem, detail Detail) error {
	program := tea.NewProgram(newMenuModel(items, detail), tea.WithInput(menu.input), tea.WithOutput(menu.output))

	_, err := program.Run()

	return err
}
