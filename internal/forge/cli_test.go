package forge

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

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

const withScopes = "HTTP/2.0 200 OK\r\n" +
	"Content-Type: application/json; charset=utf-8\r\n" +
	"X-Oauth-Scopes: gist, read:org, read:user, repo, workflow\r\n" +
	"X-Accepted-Oauth-Scopes: \r\n" +
	"\r\n" +
	`{"login":"LHPalma"}`

func TestScopesReadsTheOauthScopesHeader(t *testing.T) {
	scopes, err := NewCLI(answering(withScopes)).Scopes(t.Context())

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	want := []string{"gist", "read:org", "read:user", "repo", "workflow"}
	if strings.Join(scopes, ",") != strings.Join(want, ",") {
		t.Errorf("escopos = %v, queria %v", scopes, want)
	}
}

func TestScopesAsksTheRestEndpointWithHeaders(t *testing.T) {
	commands := answering(withScopes)

	NewCLI(commands).Scopes(t.Context())

	if got := strings.Join(commands.Calls[0].Args, " "); got != "api user -i" {
		t.Errorf("chamada = %q; o -i traz os cabecalhos, e e onde o escopo mora", got)
	}
}

func TestScopesIgnoresTheAcceptedScopesHeader(t *testing.T) {
	scopes, err := NewCLI(answering(withScopes)).Scopes(t.Context())

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	for _, scope := range scopes {
		if scope == "" {
			t.Errorf("escopos = %v; o Accepted-Oauth-Scopes vazio nao pode virar escopo vazio", scopes)
		}
	}
}

func TestScopesIsEmptyWhenTheHeaderIsMissing(t *testing.T) {
	scopes, err := NewCLI(answering(`{"login":"LHPalma"}`)).Scopes(t.Context())

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(scopes) != 0 {
		t.Errorf("escopos = %v, queria nenhum: resposta sem o cabecalho nao e erro", scopes)
	}
}

func TestScopesIsEmptyWhenTheTokenHasNone(t *testing.T) {
	empty := "HTTP/2.0 200 OK\r\nX-Oauth-Scopes: \r\n\r\n" + `{"login":"LHPalma"}`

	scopes, err := NewCLI(answering(empty)).Scopes(t.Context())

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(scopes) != 0 {
		t.Errorf("escopos = %v, queria nenhum", scopes)
	}
}

func TestScopesSaysWhenTheGhIsNotThere(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found in $PATH")})

	if _, err := NewCLI(commands).Scopes(t.Context()); !errors.Is(err, ErrUnavailable) {
		t.Errorf("erro = %v, queria a ausencia do gh", err)
	}
}

func TestScopesSeparatesHavingNoCredentialFromBeingRefused(t *testing.T) {
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
			_, err := NewCLI(exectest.NewRunner(exectest.Response{Result: test.result})).Scopes(t.Context())

			if err == nil {
				t.Fatal("os dois casos sao erro")
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Errorf("erro = %v, queria %v", err, test.want)
			}
			if test.want == nil && errors.Is(err, ErrUnauthenticated) {
				t.Errorf("erro = %v; ha credencial, ela e que foi recusada", err)
			}
		})
	}
}

const contributed = `{"data":{"viewer":{"contributionsCollection":{"totalCommitContributions":42}}}}`

func aDay(t *testing.T, date string) time.Time {
	t.Helper()

	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatalf("data de teste invalida: %v", err)
	}

	return parsed
}

func TestAccountCommitCountReadsTheTotal(t *testing.T) {
	since, until := aDay(t, "2026-08-01"), aDay(t, "2026-08-23")

	count, err := NewCLI(answering(contributed)).AccountCommitCount(t.Context(), since, until)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if count != 42 {
		t.Errorf("contagem = %d, queria a do gh", count)
	}
}

func TestAccountCommitCountAsksGraphqlWithTheDates(t *testing.T) {
	since, until := aDay(t, "2026-08-01"), aDay(t, "2026-08-23")
	commands := answering(contributed)

	NewCLI(commands).AccountCommitCount(t.Context(), since, until)

	call := strings.Join(commands.Calls[0].Args, " ")
	if !strings.Contains(call, "api graphql") {
		t.Errorf("chamada = %q, queria a rota graphql", call)
	}
	if !strings.Contains(call, "from="+since.Format(time.RFC3339)) {
		t.Errorf("chamada = %q, queria o since formatado", call)
	}
	if !strings.Contains(call, "to="+until.Format(time.RFC3339)) {
		t.Errorf("chamada = %q, queria o until formatado", call)
	}
	if !strings.Contains(call, "totalCommitContributions") {
		t.Errorf("chamada = %q, queria a query pedindo o total", call)
	}
}

func TestAccountCommitCountAsksTheViewerNeverAnArbitraryLogin(t *testing.T) {
	commands := answering(contributed)

	NewCLI(commands).AccountCommitCount(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	call := strings.Join(commands.Calls[0].Args, " ")
	if !strings.Contains(call, "viewer") {
		t.Errorf("chamada = %q; quem autentica o gh decide quem somos, nao um login escolhido aqui", call)
	}
}

func TestAccountCommitCountSplitsAPeriodLongerThanAYear(t *testing.T) {
	since, until := aDay(t, "2024-01-01"), aDay(t, "2025-06-15")
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: `{"data":{"viewer":{"contributionsCollection":{"totalCommitContributions":10}}}}`}},
		exectest.Response{Result: exec.Result{Output: `{"data":{"viewer":{"contributionsCollection":{"totalCommitContributions":7}}}}`}},
	)

	count, err := NewCLI(commands).AccountCommitCount(t.Context(), since, until)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(commands.Calls) != 2 {
		t.Fatalf("chamadas = %d, o periodo passa de um ano e a api nao aceita isso numa consulta so", len(commands.Calls))
	}
	if count != 17 {
		t.Errorf("contagem = %d, queria a soma das duas janelas", count)
	}
}

func TestAccountCommitCountNeverAsksAWindowPastUntil(t *testing.T) {
	since, until := aDay(t, "2024-01-01"), aDay(t, "2025-06-15")
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: `{"data":{"viewer":{"contributionsCollection":{"totalCommitContributions":10}}}}`}},
		exectest.Response{Result: exec.Result{Output: `{"data":{"viewer":{"contributionsCollection":{"totalCommitContributions":7}}}}`}},
	)

	NewCLI(commands).AccountCommitCount(t.Context(), since, until)

	last := strings.Join(commands.Calls[len(commands.Calls)-1].Args, " ")
	if !strings.Contains(last, "to="+until.Format(time.RFC3339)) {
		t.Errorf("ultima chamada = %q; a ultima janela tem de parar em until, nunca passar dele", last)
	}
}

func TestAccountCommitCountSaysWhenTheGhIsNotThere(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found in $PATH")})

	_, err := NewCLI(commands).AccountCommitCount(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("erro = %v, queria a ausencia do gh", err)
	}
}

func TestAccountCommitCountSeparatesHavingNoCredentialFromBeingRefused(t *testing.T) {
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
			commands := exectest.NewRunner(exectest.Response{Result: test.result})

			_, err := NewCLI(commands).AccountCommitCount(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

			if err == nil {
				t.Fatal("os dois casos sao erro")
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Errorf("erro = %v, queria %v", err, test.want)
			}
			if test.want == nil && errors.Is(err, ErrUnauthenticated) {
				t.Errorf("erro = %v; ha credencial, ela e que foi recusada", err)
			}
		})
	}
}

func TestAccountCommitCountRefusesAnAnswerItCannotRead(t *testing.T) {
	_, err := NewCLI(answering("isto nao e json")).AccountCommitCount(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err == nil {
		t.Fatal("resposta ilegivel tem de virar erro")
	}
}

func TestAccountCommitCountHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := NewCLI(answering(contributed)).AccountCommitCount(ctx, aDay(t, "2026-08-01"), aDay(t, "2026-08-23")); err == nil {
		t.Fatal("contexto cancelado tem de parar a chamada")
	}
}

func byRepository(total int, entries string) string {
	return `{"data":{"viewer":{"contributionsCollection":{` +
		`"totalRepositoriesWithContributedCommits":` + strconv.Itoa(total) + `,` +
		`"commitContributionsByRepository":[` + entries + `]}}}}`
}

func repositoryEntry(name string, private bool, count int) string {
	return `{"repository":{"nameWithOwner":"` + name + `","isPrivate":` + strconv.FormatBool(private) + `},` +
		`"contributions":{"totalCount":` + strconv.Itoa(count) + `}}`
}

func TestAccountCommitCountByRepositoryReadsEachRepository(t *testing.T) {
	body := byRepository(2, repositoryEntry("LHPalma/gitarias", true, 156)+","+repositoryEntry("LHPalma/ticketerias", false, 5))

	counts, err := NewCLI(answering(body)).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(counts) != 2 {
		t.Fatalf("repositorios = %+v, queria os dois", counts)
	}

	first := RepositoryCommitCount{Repository: "LHPalma/gitarias", Private: true, Count: 156}
	if counts[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", counts[0], first)
	}
}

func TestAccountCommitCountByRepositorySortsByCountDescending(t *testing.T) {
	body := byRepository(3,
		repositoryEntry("LHPalma/pouco", false, 2)+","+
			repositoryEntry("LHPalma/muito", false, 100)+","+
			repositoryEntry("LHPalma/medio", false, 10))

	counts, err := NewCLI(answering(body)).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	want := []string{"LHPalma/muito", "LHPalma/medio", "LHPalma/pouco"}
	for index, name := range want {
		if counts[index].Repository != name {
			t.Errorf("posicao %d = %q, queria %q; a ordem tem de ser por contagem decrescente", index, counts[index].Repository, name)
		}
	}
}

func TestAccountCommitCountByRepositoryBreaksTiesByName(t *testing.T) {
	body := byRepository(2, repositoryEntry("LHPalma/zulu", false, 5)+","+repositoryEntry("LHPalma/alfa", false, 5))

	counts, err := NewCLI(answering(body)).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if counts[0].Repository != "LHPalma/alfa" || counts[1].Repository != "LHPalma/zulu" {
		t.Errorf("ordem = %+v, empate tem de desempatar por nome", counts)
	}
}

func TestAccountCommitCountByRepositoryAsksMaxRepositories(t *testing.T) {
	commands := answering(byRepository(0, ""))

	NewCLI(commands).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	call := strings.Join(commands.Calls[0].Args, " ")
	if !strings.Contains(call, "maxRepositories: 100") {
		t.Errorf("chamada = %q, queria o teto pedido explicitamente", call)
	}
}

func TestAccountCommitCountByRepositoryGivesUpWhenEvenTheNarrowestWindowIsTruncated(t *testing.T) {
	body := byRepository(150, repositoryEntry("LHPalma/gitarias", true, 156))
	since := aDay(t, "2026-08-01")
	until := since.Add(30 * time.Minute)

	_, err := NewCLI(answering(body)).AccountCommitCountByRepository(t.Context(), since, until)

	if err == nil {
		t.Fatal("resposta cortada calada tem de virar erro, nao lista incompleta")
	}
	if !strings.Contains(err.Error(), "150") {
		t.Errorf("erro = %v, queria o total real de repositorios", err)
	}
}

func TestAccountCommitCountByRepositoryBisectsATruncatedWindow(t *testing.T) {
	truncated := byRepository(150, repositoryEntry("LHPalma/gitarias", true, 100))
	firstHalf := byRepository(1, repositoryEntry("LHPalma/gitarias", true, 60))
	secondHalf := byRepository(1, repositoryEntry("LHPalma/ticketerias", false, 5))

	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: truncated}},
		exectest.Response{Result: exec.Result{Output: firstHalf}},
		exectest.Response{Result: exec.Result{Output: secondHalf}},
	)

	counts, err := NewCLI(commands).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err != nil {
		t.Fatalf("a janela cortada tinha de bisseccionar em vez de recusar, veio %v", err)
	}
	if len(commands.Calls) != 3 {
		t.Fatalf("chamadas = %d, queria a original truncada e as duas metades", len(commands.Calls))
	}

	want := []RepositoryCommitCount{
		{Repository: "LHPalma/gitarias", Private: true, Count: 60},
		{Repository: "LHPalma/ticketerias", Private: false, Count: 5},
	}
	if len(counts) != 2 || counts[0] != want[0] || counts[1] != want[1] {
		t.Errorf("contagens = %+v, queria %+v", counts, want)
	}
}

// flagValue lê o valor de uma flag -f nome=valor entre os argumentos de uma
// chamada, para inspecionar exatamente o que foi pedido ao gh.
func flagValue(args []string, name string) string {
	prefix := name + "="
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}

	return ""
}

func TestAccountCommitCountByRepositoryPropagatesAFailureInTheFirstHalf(t *testing.T) {
	truncated := byRepository(150, repositoryEntry("LHPalma/gitarias", true, 100))

	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: truncated}},
		exectest.Response{Result: exec.Result{Code: 1, Output: "gh: Bad credentials (HTTP 401)"}},
	)

	_, err := NewCLI(commands).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err == nil {
		t.Fatal("a primeira metade falhou, e isso tem de virar erro sem tentar a segunda")
	}
	if len(commands.Calls) != 2 {
		t.Errorf("chamadas = %d, a falha na primeira metade nao pode deixar tentar a segunda", len(commands.Calls))
	}
}

func TestAccountCommitCountByRepositoryPropagatesAFailureInTheSecondHalf(t *testing.T) {
	truncated := byRepository(150, repositoryEntry("LHPalma/gitarias", true, 100))

	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: truncated}},
		exectest.Response{Result: exec.Result{Output: byRepository(0, "")}},
		exectest.Response{Result: exec.Result{Code: 1, Output: "gh: Bad credentials (HTTP 401)"}},
	)

	_, err := NewCLI(commands).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err == nil {
		t.Fatal("a segunda metade falhou, e isso tem de virar erro mesmo com a primeira ok")
	}
	if len(commands.Calls) != 3 {
		t.Errorf("chamadas = %d, queria a truncada e as duas metades tentadas", len(commands.Calls))
	}
}

func TestAccountCommitCountByRepositoryBisectionNeverOverlapsTheHalves(t *testing.T) {
	truncated := byRepository(150, repositoryEntry("LHPalma/gitarias", true, 100))

	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: truncated}},
		exectest.Response{Result: exec.Result{Output: byRepository(0, "")}},
		exectest.Response{Result: exec.Result{Output: byRepository(0, "")}},
	)

	since, until := aDay(t, "2026-08-01"), aDay(t, "2026-08-23")
	NewCLI(commands).AccountCommitCountByRepository(t.Context(), since, until)

	firstHalfEnd, err := time.Parse(time.RFC3339, flagValue(commands.Calls[1].Args, "to"))
	if err != nil {
		t.Fatalf("nao consegui ler o to da primeira metade: %v", err)
	}
	secondHalfStart, err := time.Parse(time.RFC3339, flagValue(commands.Calls[2].Args, "from"))
	if err != nil {
		t.Fatalf("nao consegui ler o from da segunda metade: %v", err)
	}

	if !secondHalfStart.Equal(firstHalfEnd.Add(time.Second)) {
		t.Errorf("segunda metade comeca em %v, primeira termina em %v; tem de ser exatamente um segundo depois, nunca no mesmo instante", secondHalfStart, firstHalfEnd)
	}
}

func TestAccountCommitCountByRepositoryAcceptsAnUntruncatedAnswer(t *testing.T) {
	body := byRepository(1, repositoryEntry("LHPalma/gitarias", true, 156))

	_, err := NewCLI(answering(body)).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err != nil {
		t.Fatalf("total bate com o que veio, nao devia recusar: %v", err)
	}
}

func TestAccountCommitCountByRepositorySaysWhenTheGhIsNotThere(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found in $PATH")})

	_, err := NewCLI(commands).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("erro = %v, queria a ausencia do gh", err)
	}
}

func TestAccountCommitCountByRepositorySeparatesHavingNoCredentialFromBeingRefused(t *testing.T) {
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
			commands := exectest.NewRunner(exectest.Response{Result: test.result})

			_, err := NewCLI(commands).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

			if err == nil {
				t.Fatal("todos os casos sao erro")
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Errorf("erro = %v, queria %v", err, test.want)
			}
			if test.want == nil && errors.Is(err, ErrUnauthenticated) {
				t.Errorf("erro = %v; ha credencial, o problema e outro", err)
			}
		})
	}
}

func TestAccountCommitCountByRepositorySplitsAPeriodLongerThanAYear(t *testing.T) {
	since, until := aDay(t, "2024-01-01"), aDay(t, "2025-06-15")
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: byRepository(1, repositoryEntry("LHPalma/gitarias", true, 100))}},
		exectest.Response{Result: exec.Result{Output: byRepository(1, repositoryEntry("LHPalma/gitarias", true, 50))}},
	)

	counts, err := NewCLI(commands).AccountCommitCountByRepository(t.Context(), since, until)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(commands.Calls) != 2 {
		t.Fatalf("chamadas = %d, o periodo passa de um ano e a api nao aceita isso numa consulta so", len(commands.Calls))
	}
	if len(counts) != 1 || counts[0].Count != 150 {
		t.Errorf("contagens = %+v, queria as duas janelas somadas no mesmo repositorio", counts)
	}
}

func TestAccountCommitCountByRepositoryNeverAsksAWindowPastUntil(t *testing.T) {
	since, until := aDay(t, "2024-01-01"), aDay(t, "2025-06-15")
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: byRepository(0, "")}},
		exectest.Response{Result: exec.Result{Output: byRepository(0, "")}},
	)

	NewCLI(commands).AccountCommitCountByRepository(t.Context(), since, until)

	last := strings.Join(commands.Calls[len(commands.Calls)-1].Args, " ")
	if !strings.Contains(last, "to="+until.Format(time.RFC3339)) {
		t.Errorf("ultima chamada = %q; a ultima janela tem de parar em until, nunca passar dele", last)
	}
}

func TestAccountCommitCountByRepositoryMergesTheSameRepositoryAcrossWindows(t *testing.T) {
	since, until := aDay(t, "2024-01-01"), aDay(t, "2025-06-15")
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: byRepository(2,
			repositoryEntry("LHPalma/gitarias", true, 100)+","+repositoryEntry("LHPalma/ticketerias", false, 5))}},
		exectest.Response{Result: exec.Result{Output: byRepository(1, repositoryEntry("LHPalma/gitarias", true, 50))}},
	)

	counts, err := NewCLI(commands).AccountCommitCountByRepository(t.Context(), since, until)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(counts) != 2 {
		t.Fatalf("contagens = %+v; gitarias apareceu nas duas janelas e ticketerias so numa, mas sao dois repositorios so", counts)
	}

	first := RepositoryCommitCount{Repository: "LHPalma/gitarias", Private: true, Count: 150}
	if counts[0] != first {
		t.Errorf("primeiro = %+v, queria %+v; a soma das duas janelas no mesmo repositorio", counts[0], first)
	}
}

func TestAccountCommitCountByRepositoryStopsAtTheFirstWindowThatFails(t *testing.T) {
	since, until := aDay(t, "2024-01-01"), aDay(t, "2026-08-23")
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 1, Output: "gh: Bad credentials (HTTP 401)"}})

	_, err := NewCLI(commands).AccountCommitCountByRepository(t.Context(), since, until)

	if err == nil {
		t.Fatal("a primeira janela falhou, e isso e erro")
	}
	if len(commands.Calls) != 1 {
		t.Errorf("chamadas = %d, a falha na primeira janela nao pode deixar tentar as seguintes", len(commands.Calls))
	}
}

func TestAccountCommitCountByRepositoryRefusesAnAnswerItCannotRead(t *testing.T) {
	_, err := NewCLI(answering("isto nao e json")).AccountCommitCountByRepository(t.Context(), aDay(t, "2026-08-01"), aDay(t, "2026-08-23"))

	if err == nil {
		t.Fatal("resposta ilegivel tem de virar erro")
	}
}

func TestAccountCommitCountByRepositoryHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	body := byRepository(1, repositoryEntry("LHPalma/gitarias", true, 156))
	if _, err := NewCLI(answering(body)).AccountCommitCountByRepository(ctx, aDay(t, "2026-08-01"), aDay(t, "2026-08-23")); err == nil {
		t.Fatal("contexto cancelado tem de parar a chamada")
	}
}
