package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/ui"
)

// model é a lista de checkboxes exibida pelo BranchSelector. Update é função
// pura — (modelo, mensagem) → modelo novo — e por isso testável sem terminal
// nenhum, conforme a ADR-002.
type model struct {
	items     []item
	cursor    int
	confirmed bool
	quitting  bool
}

func newModel(candidates []branch.Branch) model {
	items := make([]item, len(candidates))
	for index, candidate := range candidates {
		items[index] = item{branch: candidate, checked: true}
	}

	return model{items: items}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	keyMessage, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMessage.String() {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(len(m.items)-1, m.cursor+1)
	case " ", "x":
		m.items[m.cursor].checked = !m.items[m.cursor].checked
	case "a":
		m.setAll(true)
	case "n":
		m.setAll(false)
	case "enter":
		m.confirmed = true
		m.quitting = true
		return m, tea.Quit
	case "esc", "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m *model) setAll(checked bool) {
	for index := range m.items {
		m.items[index].checked = checked
	}
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var builder strings.Builder

	fmt.Fprintf(&builder, "Selecione as branches para deletar (%d %s):\n\n",
		len(m.items), ui.Plural(len(m.items), "candidata", "candidatas"))

	for index, entry := range m.items {
		cursor := "  "
		if index == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		checkbox := "[ ]"
		if entry.checked {
			checkbox = checkStyle.Render("[x]")
		}

		fmt.Fprintf(&builder, "%s%s %s  %s\n", cursor, checkbox, entry.branch.Name, ui.DescribeMerge(entry.branch.Merge))
	}

	builder.WriteString("\n" + helpStyle.Render("↑/↓ move · espaço marca · a marca todas · n desmarca todas · enter confirma · esc cancela") + "\n")

	return builder.String()
}

// selected devolve as branches marcadas quando o usuário confirmou com
// enter, e nil quando ele cancelou — esc, q, ctrl+c ou fechar o terminal.
func (m model) selected() []branch.Branch {
	if !m.confirmed {
		return nil
	}

	var chosen []branch.Branch
	for _, entry := range m.items {
		if entry.checked {
			chosen = append(chosen, entry.branch)
		}
	}

	return chosen
}
