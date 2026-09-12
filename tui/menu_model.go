package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// menuModel é o modelo de Update puro por trás do HelpMenu. Tem duas faces:
// a lista de comandos, e — depois de um escolhido — o help daquele comando.
type menuModel struct {
	items    []MenuItem
	detail   Detail
	cursor   int
	showing  bool
	text     string
	quitting bool
}

func newMenuModel(items []MenuItem, detail Detail) menuModel {
	return menuModel{items: items, detail: detail}
}

func (m menuModel) Init() tea.Cmd {
	return nil
}

func (m menuModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	keyMessage, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.showing {
		return m.updateDetail(keyMessage)
	}

	return m.updateList(keyMessage)
}

func (m menuModel) updateList(keyMessage tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyMessage.String() {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(len(m.items)-1, m.cursor+1)
	case "enter":
		m.showing = true
		m.text = m.detail(m.items[m.cursor].Name)
	case "esc", "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m menuModel) updateDetail(keyMessage tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyMessage.String() {
	case "esc", "backspace":
		m.showing = false
		m.text = ""
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m menuModel) View() string {
	if m.quitting {
		return ""
	}
	if m.showing {
		return m.text + "\n" + helpStyle.Render("esc volta para a lista · q sai") + "\n"
	}

	return m.listView()
}

func (m menuModel) listView() string {
	width := 0
	for _, item := range m.items {
		width = max(width, len(item.Name))
	}

	var builder strings.Builder

	builder.WriteString("Comandos do gtr:\n\n")

	for index, item := range m.items {
		cursor := "  "
		if index == m.cursor {
			cursor = cursorStyle.Render("> ")
		}
		fmt.Fprintf(&builder, "%s%-*s  %s\n", cursor, width, item.Name, item.Short)
	}

	builder.WriteString("\n" + helpStyle.Render("↑/↓ move · enter mostra o help · esc/q sai") + "\n")

	return builder.String()
}
