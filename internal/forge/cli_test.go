package forge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
)

const listed = `[
  {"number":7,"title":"muda o parser","headRefName":"feat-parser","baseRefName":"main",
   "state":"OPEN","isDraft":false,"url":"https://github.com/dono/repo/pull/7",
   "author":{"login":"luiz"}},
  {"number":8,"title":"rascunho","headRefName":"feat-outra","baseRefName":"feat-parser",
   "state":"OPEN","isDraft":true,"url":"https://github.com/dono/repo/pull/8",
   "author":{"login":"outra"}}
]`

func answering(output string) *exectest.Runner {
	return exectest.NewRunner(exectest.Response{Result: exec.Result{Output: output}})
}

func TestPullRequestsReadsWhatTheGhReturned(t *testing.T) {
	requests, err := NewCLI(answering(listed)).PullRequests(t.Context(), 30)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("pull requests = %+v, queria os dois", requests)
	}

	first := PullRequest{
		Number: 7, Title: "muda o parser", Head: "feat-parser", Base: "main",
		Author: "luiz", State: "open", URL: "https://github.com/dono/repo/pull/7",
	}
	if requests[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", requests[0], first)
	}
	if !requests[1].Draft {
		t.Error("o segundo e rascunho e tem de sair como tal")
	}
}

func TestPullRequestsLowercasesTheState(t *testing.T) {
	requests, _ := NewCLI(answering(listed)).PullRequests(t.Context(), 30)

	if requests[0].State != "open" {
		t.Errorf("estado = %q; no json o token e minusculo, e o gh grita em maiuscula", requests[0].State)
	}
}

func TestPullRequestsAsksForEveryFieldItReads(t *testing.T) {
	commands := answering(listed)

	if _, err := NewCLI(commands).PullRequests(t.Context(), 30); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	call := strings.Join(commands.Calls[0].Args, " ")
	for _, field := range fields {
		if !strings.Contains(call, field) {
			t.Errorf("chamada = %q, nao pediu o campo %q", call, field)
		}
	}
	if !strings.Contains(call, "--limit 30") {
		t.Errorf("chamada = %q, queria o limite", call)
	}
}

func TestPullRequestsRunsTheGh(t *testing.T) {
	commands := answering(listed)

	NewCLI(commands).PullRequests(t.Context(), 5)

	if commands.Calls[0].Name != "gh" {
		t.Errorf("chamou %q; quem fala com o GitHub e o gh, e e por isso que o gtr nao ve o token", commands.Calls[0].Name)
	}
	if !strings.Contains(strings.Join(commands.Calls[0].Args, " "), "--limit 5") {
		t.Errorf("chamada = %v, o limite pedido tem de chegar", commands.Calls[0].Args)
	}
}

func TestPullRequestsSaysWhenTheGhIsNotThere(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found in $PATH")})

	_, err := NewCLI(commands).PullRequests(t.Context(), 30)

	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("erro = %v; gh ausente se resolve instalando, e a falha de rede nao", err)
	}
}

func TestPullRequestsPropagatesWhatTheGhComplained(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{
		Result: exec.Result{Code: 1, Output: "gh: Not Found (HTTP 404)"},
	})

	_, err := NewCLI(commands).PullRequests(t.Context(), 30)

	if err == nil {
		t.Fatal("gh que roda e recusa tem de virar erro")
	}
	if errors.Is(err, ErrUnavailable) {
		t.Error("o gh esta la; o problema e outro, e confundir os dois manda instalar o que ja existe")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("erro = %v; o que o gh disse vale mais que um texto meu", err)
	}
}

func TestPullRequestsRefusesAnAnswerItCannotRead(t *testing.T) {
	_, err := NewCLI(answering("isto nao e json")).PullRequests(t.Context(), 30)

	if err == nil {
		t.Fatal("resposta ilegivel tem de virar erro, e nao lista vazia")
	}
}

func TestPullRequestsOfARepositoryWithoutAny(t *testing.T) {
	requests, err := NewCLI(answering("[]")).PullRequests(t.Context(), 30)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(requests) != 0 {
		t.Errorf("pull requests = %+v, queria nenhum", requests)
	}
}

func TestPullRequestsHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := NewCLI(answering(listed)).PullRequests(ctx, 30); err == nil {
		t.Fatal("contexto cancelado tem de parar a chamada")
	}
}

func TestViewerSaysWhoTheGitHubThinksWeAre(t *testing.T) {
	login, err := NewCLI(answering(`{"login":"LHPalma","id":83788360}`)).Viewer(t.Context())

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if login != "LHPalma" {
		t.Errorf("login = %q, queria o que veio no json", login)
	}
}

func TestViewerAsksTheRestEndpoint(t *testing.T) {
	commands := answering(`{"login":"x"}`)

	NewCLI(commands).Viewer(t.Context())

	if got := strings.Join(commands.Calls[0].Args, " "); got != "api user" {
		t.Errorf("chamada = %q; o api user e REST e devolve json contratual", got)
	}
}

func TestViewerSeparatesHavingNoCredentialFromBeingRefused(t *testing.T) {
	tests := []struct {
		name   string
		result exec.Result
		want   error
	}{
		{
			name:   "sem credencial nenhuma",
			result: exec.Result{Code: 4, Output: "To get started with GitHub CLI, please run: gh auth login"},
			want:   ErrUnauthenticated,
		},
		{
			name:   "servidor recusou",
			result: exec.Result{Code: 1, Output: "gh: Bad credentials (HTTP 401)"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewCLI(exectest.NewRunner(exectest.Response{Result: test.result})).Viewer(t.Context())

			if err == nil {
				t.Fatal("os dois casos sao erro")
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Errorf("erro = %v, queria %v", err, test.want)
			}
			if test.want == nil && errors.Is(err, ErrUnauthenticated) {
				t.Errorf("erro = %v; ha credencial, ela e que foi recusada, e mandar fazer login confundiria", err)
			}
		})
	}
}

func TestViewerSaysWhenTheGhIsNotThere(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found in $PATH")})

	if _, err := NewCLI(commands).Viewer(t.Context()); !errors.Is(err, ErrUnavailable) {
		t.Errorf("erro = %v, queria a ausencia do gh", err)
	}
}

func TestViewerRefusesAnAnswerItCannotRead(t *testing.T) {
	if _, err := NewCLI(answering("isto nao e json")).Viewer(t.Context()); err == nil {
		t.Fatal("resposta ilegivel tem de virar erro")
	}
}
