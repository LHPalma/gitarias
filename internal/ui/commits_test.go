package ui

import (
	"testing"

	"github.com/LHPalma/gitarias/internal/commits"
)

func TestDescribeOutcome(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{code: 0, want: "passou"},
		{code: 1, want: "falhou"},
		{code: 2, want: "falhou"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			if got := DescribeOutcome(commits.Result{Code: test.code}); got != test.want {
				t.Errorf("rotulo = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestHighlightOutcome(t *testing.T) {
	if got := HighlightOutcome(commits.Result{Code: 0}); got != "verde" {
		t.Errorf("rotulo = %q, queria verde", got)
	}
	if got := HighlightOutcome(commits.Result{Code: 1}); got != "VERMELHO" {
		t.Errorf("rotulo = %q, queria VERMELHO", got)
	}
}

func TestOutcomeVocabulariesStayApart(t *testing.T) {
	failed := commits.Result{Code: 1}

	if DescribeOutcome(failed) == HighlightOutcome(failed) {
		t.Error("a planilha e a tela pedem registros diferentes; a caixa alta e afordancia de tela")
	}
}
