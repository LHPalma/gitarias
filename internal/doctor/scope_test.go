package doctor

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/forge"
)

func TestScopeApprovesWhenReadUserIsPresent(t *testing.T) {
	check := Scope(t.Context(), answers{scopes: []string{"repo", "read:user", "gist"}})

	if !check.Passed() {
		t.Fatalf("checagem = %+v, queria ok", check)
	}
	if !strings.Contains(check.Detail, "read:user") {
		t.Errorf("detalhe = %q, queria nomear o escopo presente", check.Detail)
	}
}

func TestScopeWarnsWhenReadUserIsMissing(t *testing.T) {
	check := Scope(t.Context(), answers{scopes: []string{"repo", "gist"}})

	if check.State != Warning {
		t.Errorf("estado = %v; falta um escopo que nenhum comando de hoje exige, e por isso é aviso, não falha", check.State)
	}
	if !strings.Contains(check.Hint, "read:user") {
		t.Errorf("dica = %q, queria o escopo que falta", check.Hint)
	}
	if !strings.Contains(check.Hint, "gh auth refresh") {
		t.Errorf("dica = %q, queria o comando que resolve", check.Hint)
	}
}

func TestScopeWarnsWhenThereIsNoScopeAtAll(t *testing.T) {
	check := Scope(t.Context(), answers{scopes: nil})

	if check.State != Warning {
		t.Errorf("estado = %v; token sem escopo nenhum também não tem read:user", check.State)
	}
}

func TestScopeSkipsWithoutTheGh(t *testing.T) {
	check := Scope(t.Context(), answers{err: forge.ErrUnavailable})

	if check.State != Skipped {
		t.Errorf("estado = %v; sem gh não há o que perguntar, e isso a checagem do gh já disse", check.State)
	}
}

func TestScopeSkipsWithoutCredential(t *testing.T) {
	check := Scope(t.Context(), answers{err: forge.ErrUnauthenticated})

	if check.State != Skipped {
		t.Errorf("estado = %v; sem credencial a checagem de conexão já reprova, repetir aqui só duplica o aviso", check.State)
	}
}

func TestScopeSkipsWhenTheServerRefused(t *testing.T) {
	check := Scope(t.Context(), answers{err: errors.New("o GitHub recusou: HTTP 403")})

	if check.State != Skipped {
		t.Errorf("estado = %v; a checagem de conexão já nomeia essa falha, escopo não tem o que somar", check.State)
	}
}

func TestScopeNeverClaimsAccessToWhatTheScopeAllows(t *testing.T) {
	check := Scope(t.Context(), answers{scopes: []string{"read:user"}})

	for _, forbidden := range []string{"acesso", "consegue"} {
		if strings.Contains(strings.ToLower(check.Detail+check.Hint), forbidden) {
			t.Errorf("a checagem afirma o que o token permite ler, não que a conta tem o que ler: %+v", check)
		}
	}
}
