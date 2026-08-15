package ui

import (
	"errors"

	"github.com/LHPalma/gitarias/internal/branch"
)

// DescribeRefusal traduz a recusa de recriar uma branch. Erro que não é recusa
// conhecida sai como o git o escreveu.
func DescribeRefusal(err error) string {
	switch {
	case errors.Is(err, branch.ErrAlreadyExists):
		return "já existe uma branch com esse nome, e o gtr não sobrescreve"
	case errors.Is(err, branch.ErrGone):
		return "o commit não está mais no repositório; o git já podou o que ficou inalcançável"
	default:
		return err.Error()
	}
}
