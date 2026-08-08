package git

import (
	"errors"
	"testing"
)

func TestExitErrorReportsTheGitMessage(t *testing.T) {
	var err error = &ExitError{Code: 128, Message: "fatal: not a git repository"}

	if err.Error() != "fatal: not a git repository" {
		t.Errorf("mensagem = %q, queria a do git — RN-07", err.Error())
	}
}

func TestExitErrorExposesTheCode(t *testing.T) {
	var err error = &ExitError{Code: 1, Message: "nada casou"}

	var exitError *ExitError
	if !errors.As(err, &exitError) {
		t.Fatal("errors.As precisa alcancar o ExitError, senao o dominio nao distingue saida 1 de falha real")
	}
	if exitError.Code != 1 {
		t.Errorf("codigo = %d, queria 1", exitError.Code)
	}
}

func TestExitErrorIsNotConfusedWithOtherErrors(t *testing.T) {
	var exitError *ExitError

	if errors.As(errors.New("fatal: not a git repository"), &exitError) {
		t.Error("erro comum nao pode passar por ExitError, senao qualquer falha viraria codigo de saida")
	}
}
