package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LHPalma/gitarias/internal/branch"
)

func candidates() []branch.Branch {
	return []branch.Branch{
		{Name: "feat-a", Merge: branch.MergedByAncestry},
		{Name: "feat-b", Merge: branch.MergedBySquash},
		{Name: "feat-c", Merge: branch.MergedByRebase},
	}
}

func press(m model, key tea.KeyMsg) model {
	updated, _ := m.Update(key)
	return updated.(model)
}

func runeKey(text string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}
}

func TestNewModelStartsWithEverythingChecked(t *testing.T) {
	m := newModel(candidates())

	for _, entry := range m.items {
		if !entry.checked {
			t.Fatalf("%s deveria começar marcada", entry.branch.Name)
		}
	}
}

func TestUpdateMovesCursor(t *testing.T) {
	m := newModel(candidates())

	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, queria 1", m.cursor)
	}

	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, queria travar em 2 (última linha)", m.cursor)
	}

	m = press(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, queria 1", m.cursor)
	}
}

func TestUpdateCursorNeverGoesNegative(t *testing.T) {
	m := press(newModel(candidates()), tea.KeyMsg{Type: tea.KeyUp})

	if m.cursor != 0 {
		t.Fatalf("cursor = %d, queria travar em 0", m.cursor)
	}
}

func TestUpdateTogglesTheItemUnderTheCursor(t *testing.T) {
	m := press(newModel(candidates()), tea.KeyMsg{Type: tea.KeySpace})

	if m.items[0].checked {
		t.Fatal("espaço deveria desmarcar a primeira candidata")
	}
	if !m.items[1].checked || !m.items[2].checked {
		t.Fatal("as outras não podiam mudar")
	}

	m = press(m, tea.KeyMsg{Type: tea.KeySpace})
	if !m.items[0].checked {
		t.Fatal("espaço de novo deveria remarcar")
	}
}

func TestUpdateSelectAllAndNone(t *testing.T) {
	m := press(newModel(candidates()), tea.KeyMsg{Type: tea.KeySpace})

	m = press(m, runeKey("n"))
	for _, entry := range m.items {
		if entry.checked {
			t.Fatalf("%s deveria estar desmarcada depois de 'n'", entry.branch.Name)
		}
	}

	m = press(m, runeKey("a"))
	for _, entry := range m.items {
		if !entry.checked {
			t.Fatalf("%s deveria estar marcada depois de 'a'", entry.branch.Name)
		}
	}
}

func TestUpdateEnterConfirmsAndQuits(t *testing.T) {
	updated, command := newModel(candidates()).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := updated.(model)

	if !m.confirmed || !m.quitting {
		t.Fatalf("enter deveria confirmar e encerrar, veio confirmed=%v quitting=%v", m.confirmed, m.quitting)
	}
	if command == nil {
		t.Fatal("enter deveria devolver tea.Quit")
	}
}

func TestUpdateEscapeCancelsWithoutConfirming(t *testing.T) {
	tests := []tea.KeyMsg{
		{Type: tea.KeyEsc},
		runeKey("q"),
		{Type: tea.KeyCtrlC},
	}

	for _, key := range tests {
		updated, command := newModel(candidates()).Update(key)
		m := updated.(model)

		if m.confirmed {
			t.Fatalf("%v não pode confirmar", key)
		}
		if !m.quitting {
			t.Fatalf("%v deveria encerrar o programa", key)
		}
		if command == nil {
			t.Fatalf("%v deveria devolver tea.Quit", key)
		}
	}
}

func TestUpdateIgnoresMessagesThatAreNotKeys(t *testing.T) {
	m := newModel(candidates())

	updated, command := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if updated.(model).cursor != m.cursor {
		t.Fatal("mensagem que não é tecla não pode mudar o modelo")
	}
	if command != nil {
		t.Fatal("mensagem que não é tecla não devolve comando")
	}
}

func TestSelectedWithoutConfirmationIsNil(t *testing.T) {
	if selected := newModel(candidates()).selected(); selected != nil {
		t.Fatalf("sem confirmar, esperava nil, veio %v", selected)
	}
}

func TestSelectedReturnsOnlyChecked(t *testing.T) {
	m := newModel(candidates())
	m.items[1].checked = false
	m.confirmed = true

	selected := m.selected()

	if len(selected) != 2 {
		t.Fatalf("selected = %v, queria 2 marcadas", selected)
	}
	for _, chosen := range selected {
		if chosen.Name == "feat-b" {
			t.Fatal("feat-b estava desmarcada e não podia voltar")
		}
	}
}

func TestInitHasNoCommand(t *testing.T) {
	if newModel(candidates()).Init() != nil {
		t.Fatal("Init não deveria agendar nenhum comando")
	}
}

func TestViewListsEveryCandidateUnstyled(t *testing.T) {
	view := newModel(candidates()).View()

	for _, entry := range candidates() {
		if !strings.Contains(view, entry.Name) {
			t.Fatalf("view deveria conter %q sem estilo, veio %q", entry.Name, view)
		}
	}
}

func TestViewIsEmptyWhenQuitting(t *testing.T) {
	m := newModel(candidates())
	m.quitting = true

	if view := m.View(); view != "" {
		t.Fatalf("view ao encerrar deveria ser vazia, veio %q", view)
	}
}
