package cmd

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/aitrailers"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/forge"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

// scannablePullRequests tem os dois casos que separam atribuir de citar: o
// #4 carrega o rodapé como ele vazou, com CRLF e emoji; o #5 é o documento
// que fala sobre a própria detecção e cita a mesma frase no meio de uma
// linha.
const scannablePullRequests = `[
  {"number":4,"title":"docs(srs - worktrees): especifica a listagem","headRefName":"srs-worktrees",
   "baseRefName":"main","state":"OPEN","isDraft":false,"url":"https://github.com/dono/repo/pull/4",
   "author":{"login":"luiz"},
   "body":"O documento da listagem.\r\n\r\n🤖 Generated with [Claude Code](https://claude.com/claude-code)\r\n"},
  {"number":5,"title":"docs(srs - ai-trailers list): especifica a detecção","headRefName":"srs-ai-trailers-list",
   "baseRefName":"main","state":"OPEN","isDraft":false,"url":"https://github.com/dono/repo/pull/5",
   "author":{"login":"luiz"},
   "body":"Uma exclusão de graça: um rodapé como \"Generated with Claude Code\" não vem em %(trailers:only)."}
]`

const cleanPullRequests = `[
  {"number":9,"title":"docs(srs - overdub): especifica o conserto de um commit","headRefName":"srs-overdub",
   "baseRefName":"main","state":"OPEN","isDraft":false,"url":"https://github.com/dono/repo/pull/9",
   "author":{"login":"luiz"},"body":"Especifica o conserto de um commit no lugar."}
]`

func TestPullRequestScanFindsTheFooterThatLeaked(t *testing.T) {
	result := listing(t, answered(scannablePullRequests), "pr", "scan")

	for _, wanted := range []string{"#4", "Claude Code", "🤖 Generated with"} {
		if !strings.Contains(result.stdout, wanted) {
			t.Errorf("saída = %q, queria %q", result.stdout, wanted)
		}
	}
}

func TestPullRequestScanDoesNotFlagTheFooterQuotedInProse(t *testing.T) {
	result := listing(t, answered(scannablePullRequests), "pr", "scan")

	if strings.Contains(result.stdout, "#5") {
		t.Errorf("saída = %q; o #5 fala sobre a detecção, não é atribuído a ninguém", result.stdout)
	}
}

func TestPullRequestScanShowsTheLineThatMatched(t *testing.T) {
	result := listing(t, answered(scannablePullRequests), "pr", "scan")

	if !strings.Contains(result.stdout, "https://claude.com/claude-code") {
		t.Errorf("saída = %q; quem lê tem de poder conferir o casamento, ainda mais num comando que aceita falso positivo", result.stdout)
	}
}

func TestPullRequestScanFailsWhenItFindsSomething(t *testing.T) {
	result := listing(t, answered(scannablePullRequests), "pr", "scan")

	if result.err == nil {
		t.Fatal("o código de saída é o que faz o comando servir de portão")
	}
	if !strings.Contains(result.err.Error(), "1 pull request com atribuição") {
		t.Errorf("erro = %v, queria a contagem no singular", result.err)
	}
}

func TestPullRequestScanWithNothingToReport(t *testing.T) {
	result := listing(t, answered(cleanPullRequests), "pr", "scan")

	if result.err != nil {
		t.Fatalf("não achar nada é sucesso, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum pull request com atribuição") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestPullRequestScanJSON(t *testing.T) {
	result := listing(t, answered(scannablePullRequests), "pr", "scan", "--format", "json")

	var document pullRequestScanDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.PullRequests) != 1 {
		t.Fatalf("pull requests = %+v, queria só o #4", document.PullRequests)
	}
	if document.PullRequests[0].URL != "https://github.com/dono/repo/pull/4" {
		t.Errorf("url = %q; é por ela que quem lê o json abre o que casou", document.PullRequests[0].URL)
	}
	if document.PullRequests[0].Attributions[0].Tool != "Claude Code" {
		t.Errorf("atribuições = %+v, queria a ferramenta nomeada", document.PullRequests[0].Attributions)
	}
}

func TestPullRequestScanCSV(t *testing.T) {
	result := listing(t, answered(scannablePullRequests), "pr", "scan", "--format", "csv")

	if !strings.HasPrefix(result.stdout, "número,ferramenta,marca,título\n") {
		t.Errorf("saída = %q, queria o cabeçalho das colunas", result.stdout)
	}
	if !strings.Contains(result.stdout, "4,Claude Code,") {
		t.Errorf("saída = %q, queria a linha do #4", result.stdout)
	}
}

func TestPullRequestScanLooksAtTheOpenOnesByDefault(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}
	commands := exectest.NewRunner(answered(cleanPullRequests)...)

	command := NewRootCommand(gittest.NewRunner(responses), commands, noWeb(), noFinder(), noNotices)
	command.SetOut(&strings.Builder{})
	command.SetErr(&strings.Builder{})
	command.SetArgs([]string{"pr", "scan"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !strings.Contains(strings.Join(commands.Calls[0].Args, " "), "--state open") {
		t.Errorf("chamada = %v; o estado vai explícito, não herdado do padrão do gh", commands.Calls[0].Args)
	}
}

func TestPullRequestScanCarriesTheStateToTheGh(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}
	commands := exectest.NewRunner(answered(cleanPullRequests)...)

	command := NewRootCommand(gittest.NewRunner(responses), commands, noWeb(), noFinder(), noNotices)
	command.SetOut(&strings.Builder{})
	command.SetErr(&strings.Builder{})
	command.SetArgs([]string{"pr", "scan", "--state", "merged"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !strings.Contains(strings.Join(commands.Calls[0].Args, " "), "--state merged") {
		t.Errorf("chamada = %v; pull request mergeado ainda carrega o corpo que vazou", commands.Calls[0].Args)
	}
}

func TestPullRequestScanRefusesAStateTheGhWouldNotKnow(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}
	commands := exectest.NewRunner()

	command := NewRootCommand(gittest.NewRunner(responses), commands, noWeb(), noFinder(), noNotices)
	command.SetOut(&strings.Builder{})
	command.SetErr(&strings.Builder{})
	command.SetArgs([]string{"pr", "scan", "--state", "aberto"})

	if err := command.Execute(); err == nil {
		t.Fatal("estado que o gh não conhece tem de falhar antes da rede")
	}
	if len(commands.Calls) != 0 {
		t.Errorf("chamadas = %v; erro de digitação não paga uma ida ao servidor", commands.Calls)
	}
}

func TestPullRequestScanWithoutGh(t *testing.T) {
	outcomes := []exectest.Response{{Err: errors.New("executable file not found in $PATH")}}

	result := listing(t, outcomes, "pr", "scan")

	if result.err == nil {
		t.Fatal("sem gh não há corpo de pull request para ler")
	}
	if !strings.Contains(result.err.Error(), "gtr setup") {
		t.Errorf("erro = %v; a saída é acionável e o setup diz o comando da máquina", result.err)
	}
}

func TestPullRequestScanPropagatesWhatTheGhComplained(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 1, Output: "gh: Bad credentials (HTTP 401)"}}}

	result := listing(t, outcomes, "pr", "scan")

	if result.err == nil {
		t.Fatal("gh que roda e recusa tem de virar erro")
	}
	if strings.Contains(result.err.Error(), "gtr setup") {
		t.Errorf("erro = %v; o gh está instalado, e mandar instalá-lo confundiria", result.err)
	}
}

func TestPullRequestScanOutsideARepository(t *testing.T) {
	if executeWith(t, map[string]gittest.Response{}, answered(cleanPullRequests), "pr", "scan").err == nil {
		t.Fatal("fora de um repositório não há pull request a examinar")
	}
}

func TestPullRequestScanDeclaresTheNetworkCall(t *testing.T) {
	result := listing(t, nil, "pr", "scan", "--help")

	if !strings.Contains(result.stdout, "REDE") {
		t.Errorf("ajuda = %q; sai da máquina, e isso tem de estar dito", result.stdout)
	}
}

func TestPullRequestScanSaysWhyTheHistoryCannotAnswerThis(t *testing.T) {
	result := listing(t, nil, "pr", "scan", "--help")

	if !strings.Contains(result.stdout, "não está no git") {
		t.Errorf("ajuda = %q; o motivo de existir um comando de rede para isto é o corpo não estar no git", result.stdout)
	}
}

func TestPullRequestScanNeverEndsALineWithSpace(t *testing.T) {
	result := listing(t, answered(scannablePullRequests), "pr", "scan")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestPullRequestScanRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	if listing(t, answered(scannablePullRequests), "pr", "scan", "--format", "json", "--no-header").err == nil {
		t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
	}
}

func TestPullRequestScanPropagatesTheWriteFailure(t *testing.T) {
	empty := pullRequestScanTable{}
	if err := empty.text(brokenWriter{}); err == nil {
		t.Fatal("falha de escrita na mensagem de nada achado tem de virar erro")
	}

	populated := pullRequestScanTable{scans: []scannedPullRequest{{
		request: forge.PullRequest{Number: 4, Title: "documento"},
	}}}
	if err := populated.text(brokenWriter{}); err == nil {
		t.Fatal("falha ao esvaziar o alinhador tem de virar erro")
	}
}

func TestScanCountsInThePlural(t *testing.T) {
	data := pullRequestScanTable{scans: []scannedPullRequest{
		{request: forge.PullRequest{Number: 4}},
		{request: forge.PullRequest{Number: 5}},
	}}

	if err := scanned(data); err == nil || !strings.Contains(err.Error(), "2 pull requests") {
		t.Errorf("erro = %v, queria a contagem no plural", err)
	}
}

func TestPullRequestScanSuggestsItsOwnFileNameWhenRefusingADirectory(t *testing.T) {
	directory := t.TempDir() + string(filepath.Separator)

	result := listing(t, answered(scannablePullRequests), "pr", "scan", "--format", "csv", "--output", directory)

	if result.err == nil {
		t.Fatal("caminho terminado em separador nomeia um diretório")
	}
	if !strings.Contains(result.err.Error(), "atribuicoes.csv") {
		t.Errorf("erro = %v, o exemplo tem de ser do comando que rodou", result.err)
	}
}

func TestScannedToolsNamesEachToolOnce(t *testing.T) {
	scan := scannedPullRequest{
		request: forge.PullRequest{Number: 4},
		attributions: []aitrailers.Attribution{
			{Tool: "Claude Code", Line: "🤖 Generated with [Claude Code](https://claude.com/claude-code)"},
			{Tool: "Claude Code", Line: "https://claude.ai/code/session_01VW6gJjPdk4zCJMLTUJjQ6k"},
		},
	}

	if tools := scannedTools(scan); len(tools) != 1 {
		t.Errorf("ferramentas = %v; o rodapé e o link da sessão são a mesma ferramenta duas vezes", tools)
	}
	if lines := scannedLines(scan); len(lines) != 2 {
		t.Errorf("marcas = %v; a ferramenta se repete, a evidência não", lines)
	}
}
