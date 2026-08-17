package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/forge"
)

// answers descreve um GitHub que responde o que o teste mandar.
type answers struct {
	login string
	err   error
}

func (source answers) PullRequests(context.Context, int) ([]forge.PullRequest, error) {
	return nil, nil
}

func (source answers) Viewer(context.Context) (string, error) {
	return source.login, source.err
}

func TestConnectedNamesWhoTheGitHubAccepted(t *testing.T) {
	check := Connected(t.Context(), answers{login: "LHPalma"})

	if !check.Passed() {
		t.Fatalf("checagem = %+v, queria ok", check)
	}
	if check.Detail != "LHPalma" {
		t.Errorf("detalhe = %q, queria o login", check.Detail)
	}
}

func TestConnectedFailsWithoutCredential(t *testing.T) {
	check := Connected(t.Context(), answers{err: forge.ErrUnauthenticated})

	if check.State != Failure {
		t.Errorf("estado = %v; sem credencial nenhum comando de PR funciona", check.State)
	}
	if !strings.Contains(check.Hint, "gh auth login") && !strings.Contains(check.Hint, "GH_TOKEN") {
		t.Errorf("dica = %q, queria os dois caminhos de entrar", check.Hint)
	}
}

func TestConnectedSkipsWithoutTheGh(t *testing.T) {
	check := Connected(t.Context(), answers{err: forge.ErrUnavailable})

	if check.State != Skipped {
		t.Errorf("estado = %v; sem gh nao ha o que perguntar, e isso a checagem do gh ja disse", check.State)
	}
}

func TestConnectedWarnsWhenTheServerRefused(t *testing.T) {
	check := Connected(t.Context(), answers{err: errors.New("o GitHub recusou: HTTP 403")})

	if check.State != Warning {
		t.Errorf("estado = %v; ha credencial, e o problema pode ser do outro lado", check.State)
	}
	if !strings.Contains(check.Hint, "403") {
		t.Errorf("dica = %q; o que o GitHub disse vale mais que um texto meu", check.Hint)
	}
}

func TestConnectedNeverClaimsTheTokenIsValid(t *testing.T) {
	check := Connected(t.Context(), answers{login: "LHPalma"})

	for _, forbidden := range []string{"válido", "valido"} {
		if strings.Contains(strings.ToLower(check.Detail+check.Hint), forbidden) {
			t.Errorf("RN-40: a checagem afirma que houve resposta aceita, nao que o token e valido: %+v", check)
		}
	}
}
