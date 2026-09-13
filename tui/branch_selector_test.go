package tui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/branch"
)

func TestNewBranchSelectorStoresTheStreams(t *testing.T) {
	input := strings.NewReader("")
	output := &bytes.Buffer{}

	selector := NewBranchSelector(input, output)

	if selector.input != input {
		t.Error("o input não foi guardado")
	}
	if selector.output != output {
		t.Error("o output não foi guardado")
	}
}

// erroredReader falha toda leitura, que é como o bubbletea descobre que o
// terminal sumiu no meio da sessão — a única forma de program.Run devolver
// erro sem um terminal de verdade envolvido.
type erroredReader struct{}

func (erroredReader) Read([]byte) (int, error) { return 0, errors.New("entrada quebrada") }

// As entradas destes testes trazem exatamente as teclas que encerram o
// programa, e nem um byte a mais: byte sobrando na entrada depois do quit
// deixa uma leitura pendente que segura o encerramento, e o teste trava em
// vez de falhar. Medido, não suposto.
func TestSelectStartsWithEveryCandidateChecked(t *testing.T) {
	candidates := []branch.Branch{{Name: "feat"}, {Name: "fix"}}

	selected, err := NewBranchSelector(strings.NewReader("\r"), &bytes.Buffer{}).Select(candidates)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(selected) != len(candidates) {
		t.Fatalf("selecionadas = %v, queria as duas: enter sozinho apaga a lista inteira, e desmarcar é a exceção", selected)
	}
	for index, entry := range selected {
		if entry.Name != candidates[index].Name {
			t.Errorf("posição %d = %q, queria %q", index, entry.Name, candidates[index].Name)
		}
	}
}

func TestSelectReturnsNothingAfterUncheckingEveryCandidate(t *testing.T) {
	selected, err := NewBranchSelector(strings.NewReader("n\r"), &bytes.Buffer{}).
		Select([]branch.Branch{{Name: "feat"}, {Name: "fix"}})

	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(selected) != 0 {
		t.Errorf("selecionadas = %v, queria nenhuma depois do n", selected)
	}
}

func TestSelectReturnsNothingWhenCancelled(t *testing.T) {
	selected, err := NewBranchSelector(strings.NewReader("q"), &bytes.Buffer{}).
		Select([]branch.Branch{{Name: "feat"}})

	if err != nil {
		t.Fatalf("cancelar não é erro, veio %v", err)
	}
	if len(selected) != 0 {
		t.Errorf("selecionadas = %v; quem cancelou não escolheu nada, e deletar o que ele não confirmou é o pior caminho possível", selected)
	}
}

func TestSelectPropagatesTheInputFailure(t *testing.T) {
	selected, err := NewBranchSelector(erroredReader{}, &bytes.Buffer{}).
		Select([]branch.Branch{{Name: "feat"}})

	if err == nil {
		t.Fatal("entrada que falha tem de virar erro, nunca uma seleção que ninguém fez")
	}
	if selected != nil {
		t.Errorf("selecionadas = %v, queria nil junto do erro", selected)
	}
}

func TestSelectDrawsOnTheOutputItWasGiven(t *testing.T) {
	output := &bytes.Buffer{}

	if _, err := NewBranchSelector(strings.NewReader("q"), output).Select([]branch.Branch{{Name: "feat"}}); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if output.Len() == 0 {
		t.Error("a tela saiu vazia; o programa tem de desenhar no output injetado, não no terminal do processo")
	}
}
