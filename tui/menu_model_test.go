package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func menuItems() []MenuItem {
	return []MenuItem{
		{Name: "branches", Short: "Lista branches mergeadas"},
		{Name: "worktrees", Short: "Lista working trees"},
		{Name: "doctor", Short: "Confere a máquina"},
	}
}

func recordingDetail(calls *[]string) Detail {
	return func(name string) string {
		*calls = append(*calls, name)
		return "ajuda de " + name
	}
}

func pressMenu(m menuModel, key tea.KeyMsg) menuModel {
	updated, _ := m.Update(key)
	return updated.(menuModel)
}

func TestMenuUpdateMovesCursor(t *testing.T) {
	var calls []string
	m := newMenuModel(menuItems(), recordingDetail(&calls))

	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, queria 1", m.cursor)
	}

	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyDown})
	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, queria travar em 2 (última linha)", m.cursor)
	}

	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, queria 1", m.cursor)
	}
}

func TestMenuUpdateCursorNeverGoesNegative(t *testing.T) {
	var calls []string
	m := pressMenu(newMenuModel(menuItems(), recordingDetail(&calls)), tea.KeyMsg{Type: tea.KeyUp})

	if m.cursor != 0 {
		t.Fatalf("cursor = %d, queria travar em 0", m.cursor)
	}
}

func TestMenuUpdateEnterFetchesDetailOfTheItemUnderTheCursor(t *testing.T) {
	var calls []string
	m := newMenuModel(menuItems(), recordingDetail(&calls))
	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyDown})

	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyEnter})

	if !m.showing {
		t.Fatal("enter deveria mostrar o detalhe")
	}
	if m.text != "ajuda de worktrees" {
		t.Fatalf("text = %q, queria o help de worktrees", m.text)
	}
	if len(calls) != 1 || calls[0] != "worktrees" {
		t.Fatalf("detail foi chamado com %v, queria só [worktrees]", calls)
	}
}

func TestMenuUpdateEscapeFromTheListQuitsWithoutShowing(t *testing.T) {
	var calls []string

	updated, command := newMenuModel(menuItems(), recordingDetail(&calls)).Update(tea.KeyMsg{Type: tea.KeyEsc})
	m := updated.(menuModel)

	if !m.quitting {
		t.Fatal("esc na lista deveria encerrar o programa")
	}
	if m.showing {
		t.Fatal("esc na lista não passa pelo detalhe")
	}
	if command == nil {
		t.Fatal("esperava tea.Quit")
	}
}

func TestMenuUpdateEscapeFromTheDetailGoesBackToTheList(t *testing.T) {
	var calls []string
	m := newMenuModel(menuItems(), recordingDetail(&calls))
	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyEnter})

	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.showing {
		t.Fatal("esc no detalhe deveria voltar pra lista")
	}
	if m.text != "" {
		t.Fatalf("text deveria limpar ao voltar, veio %q", m.text)
	}
	if m.quitting {
		t.Fatal("esc no detalhe não encerra o programa")
	}
}

func TestMenuUpdateQuitsFromTheDetailToo(t *testing.T) {
	var calls []string
	m := newMenuModel(menuItems(), recordingDetail(&calls))
	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyEnter})

	updated, command := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(menuModel)

	if !m.quitting {
		t.Fatal("ctrl+c no detalhe deveria encerrar o programa")
	}
	if command == nil {
		t.Fatal("esperava tea.Quit")
	}
}

func TestMenuUpdateIgnoresMessagesThatAreNotKeys(t *testing.T) {
	var calls []string
	m := newMenuModel(menuItems(), recordingDetail(&calls))

	updated, command := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if updated.(menuModel).cursor != m.cursor {
		t.Fatal("mensagem que não é tecla não pode mudar o modelo")
	}
	if command != nil {
		t.Fatal("mensagem que não é tecla não devolve comando")
	}
}

func TestMenuInitHasNoCommand(t *testing.T) {
	var calls []string
	if newMenuModel(menuItems(), recordingDetail(&calls)).Init() != nil {
		t.Fatal("Init não deveria agendar nenhum comando")
	}
}

func TestMenuViewListsEveryItemUnstyled(t *testing.T) {
	var calls []string
	view := newMenuModel(menuItems(), recordingDetail(&calls)).View()

	for _, item := range menuItems() {
		if !strings.Contains(view, item.Name) {
			t.Fatalf("view deveria conter %q, veio %q", item.Name, view)
		}
		if !strings.Contains(view, item.Short) {
			t.Fatalf("view deveria conter %q, veio %q", item.Short, view)
		}
	}
}

func TestMenuViewShowsTheDetailTextInsteadOfTheList(t *testing.T) {
	var calls []string
	m := newMenuModel(menuItems(), recordingDetail(&calls))
	m = pressMenu(m, tea.KeyMsg{Type: tea.KeyEnter})

	view := m.View()

	if !strings.Contains(view, "ajuda de branches") {
		t.Fatalf("view deveria mostrar o help capturado, veio %q", view)
	}
	if strings.Contains(view, "worktrees") {
		t.Fatalf("no detalhe a lista não deveria aparecer, veio %q", view)
	}
}

func TestMenuViewIsEmptyWhenQuitting(t *testing.T) {
	var calls []string
	m := newMenuModel(menuItems(), recordingDetail(&calls))
	m.quitting = true

	if view := m.View(); view != "" {
		t.Fatalf("view ao encerrar deveria ser vazia, veio %q", view)
	}
}
