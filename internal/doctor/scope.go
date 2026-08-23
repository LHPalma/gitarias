package doctor

import (
	"context"
	"errors"

	"github.com/LHPalma/gitarias/internal/forge"
)

// readUser é o escopo que o GitHub exige para expor, pela API, as
// contribuições privadas do próprio usuário autenticado — repo e read:org
// não bastam para isso.
const readUser = "read:user"

// Scope pergunta ao GitHub quais permissões o token do gh carrega, e relata
// se read:user está entre elas. Fica fora do Diagnose pelo mesmo motivo que
// Connected: é rede, e o resto do doctor termina sozinho.
//
// O que ela afirma é estreito, como Connected: que o escopo está ou não
// entre os que o token declara. Não afirma que a conta em si tem acesso ao
// que esse escopo permite ler — só que o token carrega a permissão. Vira
// aviso, não falha, porque a ausência do escopo não impede nenhum comando
// que já roda hoje.
func Scope(ctx context.Context, source forge.Source) Check {
	scopes, err := source.Scopes(ctx)

	switch {
	case errors.Is(err, forge.ErrUnavailable):
		return Check{Name: "escopo", State: Skipped, Detail: "depende do gh, que não está aqui"}
	case errors.Is(err, forge.ErrUnauthenticated):
		return Check{Name: "escopo", State: Skipped, Detail: "depende de credencial, que não há"}
	case err != nil:
		return Check{Name: "escopo", State: Skipped, Detail: "depende da conexão, que falhou"}
	}

	for _, scope := range scopes {
		if scope == readUser {
			return Check{Name: "escopo", State: Ok, Detail: readUser + " presente"}
		}
	}

	return Check{
		Name:   "escopo",
		State:  Warning,
		Detail: "falta " + readUser,
		Hint:   "rode gh auth refresh -h github.com -s " + readUser + " para acrescentá-lo ao token",
	}
}
