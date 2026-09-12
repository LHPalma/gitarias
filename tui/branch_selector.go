package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LHPalma/gitarias/internal/branch"
)

// BranchSelector implementa ui.Selector como uma lista de checkboxes rodada
// em tela cheia. Todas as candidatas começam marcadas — enter apaga a lista
// inteira sem exigir que o usuário marque uma a uma, e desmarcar é a exceção.
// Select assume candidates não vazio; filtrar antes é responsabilidade de
// quem chama.
type BranchSelector struct {
	input  io.Reader
	output io.Writer
}

func NewBranchSelector(input io.Reader, output io.Writer) BranchSelector {
	return BranchSelector{input: input, output: output}
}

func (selector BranchSelector) Select(candidates []branch.Branch) ([]branch.Branch, error) {
	program := tea.NewProgram(newModel(candidates), tea.WithInput(selector.input), tea.WithOutput(selector.output))

	final, err := program.Run()
	if err != nil {
		return nil, err
	}

	return final.(model).selected(), nil
}
