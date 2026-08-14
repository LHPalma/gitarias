package doctor

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
)

func TestDiagnoseFindsGit(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0\n"}})

	checks := New(commands).Diagnose(t.Context())

	if len(checks) != 1 {
		t.Fatalf("checagens = %+v, queria a do git", checks)
	}
	if !checks[0].Passed() {
		t.Errorf("checagem = %+v, queria ok", checks[0])
	}
	if checks[0].Detail != "2.43.0" {
		t.Errorf("detalhe = %q, queria so a versao", checks[0].Detail)
	}
	if checks[0].Hint != "" {
		t.Errorf("dica = %q; so o que falhou precisa dizer como resolver", checks[0].Hint)
	}
}

func TestDiagnoseAsksTheGitOnThePath(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	New(commands).Diagnose(t.Context())

	call := commands.Calls[0]
	if call.Name != "git" || strings.Join(call.Args, " ") != "--version" {
		t.Errorf("chamada = %+v, queria git --version", call)
	}
}

func TestDiagnoseReportsGitMissing(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found in $PATH")})

	check := New(commands).Diagnose(t.Context())[0]

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

	check := New(commands).Diagnose(t.Context())[0]

	if check.State != Failure {
		t.Errorf("estado = %v; binario que existe e nao roda tambem e falha", check.State)
	}
	if !strings.Contains(check.Hint, "shared libraries") {
		t.Errorf("dica = %q; a saida do proprio git diz mais que qualquer texto meu", check.Hint)
	}
}

func TestVersionOfAnEmptyOutput(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "   \n"}})

	check := New(commands).Diagnose(t.Context())[0]

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
