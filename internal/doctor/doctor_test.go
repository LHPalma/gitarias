package doctor

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func insideRepo() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":               {Output: "true"},
		"symbolic-ref --short refs/remotes/origin/HEAD": {Output: "origin/main"},
		"rev-parse --verify --quiet refs/heads/main":    {Output: "abc"},
	}
}

func gitCheck(t *testing.T, checks []Check) Check {
	t.Helper()

	for _, check := range checks {
		if check.Name == "git" {
			return check
		}
	}

	t.Fatal("a checagem do git sumiu da lista")

	return Check{}
}

func TestDiagnoseFindsGit(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0\n"}})

	checks := New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context())

	check := gitCheck(t, checks)

	if !check.Passed() {
		t.Errorf("checagem = %+v, queria ok", check)
	}
	if check.Detail != "2.43.0" {
		t.Errorf("detalhe = %q, queria so a versao", check.Detail)
	}
	if check.Hint != "" {
		t.Errorf("dica = %q; so o que falhou precisa dizer como resolver", check.Hint)
	}
}

func TestDiagnoseAsksTheGitOnThePath(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context())

	call := commands.Calls[0]
	if call.Name != "git" || strings.Join(call.Args, " ") != "--version" {
		t.Errorf("chamada = %+v, queria git --version", call)
	}
}

func TestDiagnoseReportsGitMissing(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found in $PATH")})

	check := gitCheck(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()))

	if check.State != Failure {
		t.Errorf("estado = %v; sem git o gtr nao funciona, entao e falha e nao aviso", check.State)
	}
	if check.Hint == "" {
		t.Error("checagem que falha tem de dizer como resolver")
	}
	if !strings.Contains(check.Detail, "PATH") {
		t.Errorf("detalhe = %q, queria dizer onde procurou", check.Detail)
	}
}

func TestDiagnoseReportsGitThatDoesNotRun(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{
		Result: exec.Result{Code: 127, Output: "error while loading shared libraries"},
	})

	check := gitCheck(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()))

	if check.State != Failure {
		t.Errorf("estado = %v; binario que existe e nao roda tambem e falha", check.State)
	}
	if !strings.Contains(check.Hint, "shared libraries") {
		t.Errorf("dica = %q; a saida do proprio git diz mais que qualquer texto meu", check.Hint)
	}
}

func TestVersionOfAnEmptyOutput(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "   \n"}})

	check := gitCheck(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()))

	if !check.Passed() {
		t.Errorf("checagem = %+v; o git respondeu, entao passou mesmo sem versao legivel", check)
	}
	if check.Detail != "" {
		t.Errorf("detalhe = %q, queria vazio", check.Detail)
	}
}

func TestStateTokens(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{state: Ok, want: "ok"},
		{state: Warning, want: "warning"},
		{state: Failure, want: "failure"},
		{state: Skipped, want: "skipped"},
		{state: State(99), want: "ok"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			if got := test.state.String(); got != test.want {
				t.Errorf("token = %q, queria %q", got, test.want)
			}
		})
	}
}

func checkNamed(t *testing.T, checks []Check, name string) Check {
	t.Helper()

	for _, check := range checks {
		if check.Name == name {
			return check
		}
	}

	t.Fatalf("a checagem %q nao esta em %+v", name, checks)

	return Check{}
}

func TestDiagnoseSkipsWhatDependsOnARepository(t *testing.T) {
	responses := map[string]gittest.Response{}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	checks := New(gittest.NewRunner(responses), commands).Diagnose(t.Context())

	if got := checkNamed(t, checks, "repositório").State; got != Skipped {
		t.Errorf("estado = %v; fora de um repositorio nao ha falha, ha ausencia de contexto", got)
	}
	if got := checkNamed(t, checks, "base").State; got != Skipped {
		t.Errorf("estado = %v; sem repositorio nao ha base a resolver", got)
	}
}

func TestDiagnoseFindsTheBase(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	checks := New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context())

	base := checkNamed(t, checks, "base")
	if !base.Passed() {
		t.Fatalf("checagem = %+v, queria ok", base)
	}
	if base.Detail != "main" {
		t.Errorf("detalhe = %q, queria o nome da base", base.Detail)
	}
}

func TestDiagnoseWarnsWhenTheBaseIsNotDeterminable(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	base := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "base")

	if base.State != Warning {
		t.Errorf("estado = %v; base indeterminavel quebra so o branches, entao e aviso e nao falha", base.State)
	}
	if base.Hint == "" {
		t.Error("aviso tem de dizer a saida de emergencia")
	}
}
