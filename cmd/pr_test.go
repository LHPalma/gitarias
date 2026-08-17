package cmd

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/forge"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const listedPullRequests = `[
  {"number":7,"title":"muda o parser","headRefName":"feat-parser","baseRefName":"main",
   "state":"OPEN","isDraft":false,"url":"https://github.com/dono/repo/pull/7",
   "author":{"login":"luiz"}},
  {"number":8,"title":"ainda cozinhando","headRefName":"feat-outra","baseRefName":"feat-parser",
   "state":"OPEN","isDraft":true,"url":"https://github.com/dono/repo/pull/8",
   "author":{"login":"outra"}}
]`

func listing(t *testing.T, outcomes []exectest.Response, args ...string) execution {
	t.Helper()

	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}

	return executeWith(t, responses, outcomes, args...)
}

func answered(output string) []exectest.Response {
	return []exectest.Response{{Result: exec.Result{Output: output}}}
}

func TestPullRequestListShowsWhatIsOpen(t *testing.T) {
	result := listing(t, answered(listedPullRequests), "pr", "list")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	for _, wanted := range []string{"#7", "feat-parser", "muda o parser"} {
		if !strings.Contains(result.stdout, wanted) {
			t.Errorf("saída = %q, queria %q", result.stdout, wanted)
		}
	}
}

func TestPullRequestListMarksTheDraft(t *testing.T) {
	result := listing(t, answered(listedPullRequests), "pr", "list")

	if !strings.Contains(result.stdout, "rascunho") {
		t.Errorf("saída = %q; rascunho não se revisa nem se mergeia, e isso muda o que dá para fazer", result.stdout)
	}
}

func TestPullRequestListJSON(t *testing.T) {
	result := listing(t, answered(listedPullRequests), "pr", "list", "--format", "json")

	var document pullRequestsDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.PullRequests) != 2 {
		t.Fatalf("pull requests = %+v, queria os dois", document.PullRequests)
	}

	first := pullRequestRecord{
		Number: 7, Title: "muda o parser", Head: "feat-parser", Base: "main",
		Author: "luiz", State: "open", URL: "https://github.com/dono/repo/pull/7",
	}
	if document.PullRequests[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", document.PullRequests[0], first)
	}
}

func TestPullRequestListCSV(t *testing.T) {
	result := listing(t, answered(listedPullRequests), "pr", "list", "--format", "csv")

	want := "número,estado,branch,base,autor,título\n" +
		"7,aberto,feat-parser,main,luiz,muda o parser\n" +
		"8,rascunho,feat-outra,feat-parser,outra,ainda cozinhando\n"

	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestPullRequestListWithoutGh(t *testing.T) {
	outcomes := []exectest.Response{{Err: errors.New("executable file not found in $PATH")}}

	result := listing(t, outcomes, "pr", "list")

	if result.err == nil {
		t.Fatal("sem gh o comando não tem como falar com o GitHub")
	}
	if !strings.Contains(result.err.Error(), "gtr setup") {
		t.Errorf("erro = %v; a saída é acionável e o setup diz o comando da máquina", result.err)
	}
}

func TestPullRequestListPropagatesWhatTheGhComplained(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 1, Output: "gh: Bad credentials (HTTP 401)"}}}

	result := listing(t, outcomes, "pr", "list")

	if result.err == nil {
		t.Fatal("gh que roda e recusa tem de virar erro")
	}
	if !strings.Contains(result.err.Error(), "401") {
		t.Errorf("erro = %v, queria o que o gh disse", result.err)
	}
	if strings.Contains(result.err.Error(), "gtr setup") {
		t.Errorf("erro = %v; o gh está instalado, e mandar instalá-lo confundiria", result.err)
	}
}

func TestPullRequestListOutsideARepository(t *testing.T) {
	if executeWith(t, map[string]gittest.Response{}, answered(listedPullRequests), "pr", "list").err == nil {
		t.Fatal("fora de um repositório não há pull request a listar")
	}
}

func TestPullRequestListRespectsTheLimit(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}
	commands := exectest.NewRunner(answered(listedPullRequests)...)

	command := NewRootCommand(gittest.NewRunner(responses), commands, noFinder(), noNotices)
	command.SetOut(&strings.Builder{})
	command.SetErr(&strings.Builder{})
	command.SetArgs([]string{"pr", "list", "--limit", "5"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !strings.Contains(strings.Join(commands.Calls[0].Args, " "), "--limit 5") {
		t.Errorf("chamada = %v, o limite tem de chegar ao gh", commands.Calls[0].Args)
	}
}

func TestPullRequestListWithNoneOpen(t *testing.T) {
	result := listing(t, answered("[]"), "pr", "list")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum pull request aberto") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestPullRequestListRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	if listing(t, answered(listedPullRequests), "pr", "list", "--format", "json", "--no-header").err == nil {
		t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
	}
}

func TestPullRequestListDeclaresTheNetworkCall(t *testing.T) {
	result := listing(t, nil, "pr", "list", "--help")

	if !strings.Contains(result.stdout, "REDE") {
		t.Errorf("ajuda = %q; é o primeiro comando do gtr que sai da máquina, e isso tem de estar dito", result.stdout)
	}
}

func TestPullRequestListNeverEndsALineWithSpace(t *testing.T) {
	result := listing(t, answered(listedPullRequests), "pr", "list")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestPullRequestListPropagatesTheWriteFailure(t *testing.T) {
	empty := pullRequestsTable{}
	if err := empty.text(brokenWriter{}); err == nil {
		t.Fatal("falha de escrita na mensagem de lista vazia tem de virar erro")
	}

	populated := pullRequestsTable{requests: []forge.PullRequest{{Number: 1, Head: "feat"}}}
	if err := populated.text(brokenWriter{}); err == nil {
		t.Fatal("falha ao esvaziar o alinhador tem de virar erro")
	}
}

func TestDescribePullRequestSeparatesTheStates(t *testing.T) {
	data := pullRequestsTable{requests: []forge.PullRequest{
		{Number: 1, State: "open"},
		{Number: 2, State: "merged"},
		{Number: 3, State: "closed"},
		{Number: 4, State: "open", Draft: true},
	}}

	rows := data.rows()
	wanted := []string{"aberto", "mergeado", "fechado", "rascunho"}

	for index, want := range wanted {
		if rows[index][1] != want {
			t.Errorf("estado %d = %q, queria %q", index, rows[index][1], want)
		}
	}
}
