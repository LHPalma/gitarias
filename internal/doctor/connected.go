package doctor

import (
	"context"
	"errors"

	"github.com/LHPalma/gitarias/internal/forge"
)

// Connected pergunta ao GitHub quem somos, e é a única checagem que sai da
// máquina. Fica fora do Diagnose de propósito: o resto do doctor é local e
// termina sozinho, e é isso que o torna barato de rodar por curiosidade.
//
// O que ela afirma é estreito, e de propósito: que existe credencial e que o
// servidor a aceitou. Não afirma que o token é válido — um proxy que
// reautentica no caminho responde igual, e a RN-40 diz que o doctor não
// afirma o que não mediu.
func Connected(ctx context.Context, source forge.Source) Check {
	login, err := source.Viewer(ctx)

	switch {
	case errors.Is(err, forge.ErrUnavailable):
		return Check{
			Name:   "conexão",
			State:  Skipped,
			Detail: "depende do gh, que não está aqui",
		}
	case errors.Is(err, forge.ErrUnauthenticated):
		return Check{
			Name:   "conexão",
			State:  Failure,
			Detail: "sem credencial para o GitHub",
			Hint:   "entre com gh auth login, ou exporte GH_TOKEN com um token de acesso",
		}
	case err != nil:
		return Check{
			Name:   "conexão",
			State:  Warning,
			Detail: "o GitHub não respondeu como esperado",
			Hint:   err.Error(),
		}
	}

	return Check{Name: "conexão", State: Ok, Detail: login}
}
