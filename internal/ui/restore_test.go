package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/branch"
)

func TestDescribeRefusalSeparatesTheTwoReasons(t *testing.T) {
	ocupada := DescribeRefusal(branch.ErrAlreadyExists)
	podada := DescribeRefusal(branch.ErrGone)

	if !strings.Contains(ocupada, "sobrescreve") {
		t.Errorf("recusa = %q, queria dizer que nao sobrescreve", ocupada)
	}
	if !strings.Contains(podada, "podou") {
		t.Errorf("recusa = %q, queria dizer que o objeto sumiu", podada)
	}
	if ocupada == podada {
		t.Error("nome ocupado e commit podado sao situacoes diferentes e pedem conserto diferente")
	}
}

func TestDescribeRefusalKeepsWhatGitSaid(t *testing.T) {
	if got := DescribeRefusal(errors.New("fatal: bad object")); got != "fatal: bad object" {
		t.Errorf("recusa = %q; erro que nao e recusa conhecida sai como o git escreveu", got)
	}
}
