package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	blameAIHeadLog           = "log -1 --format=%B"
	blameAIInterpretTrailers = "interpret-trailers --trailer Co-Authored-By: Claude <noreply@anthropic.com>"
)

func blameAIPlanLog(target string) string {
	return "log -1 --format=%h%x00%s " + target
}

func blameAIRebaseExec(sha string) string {
	return "rebase " + sha + "^ " + sha +
		` --exec git log -1 --format=%B | git interpret-trailers --trailer "Co-Authored-By: Claude <noreply@anthropic.com>" | git commit --amend --allow-empty -F -`
}

func TestBlameAIOnTheHeadAsksAndAdds(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		aiTrailersShortHead:               {Output: "abc123"},
		blameAIPlanLog("HEAD"):            {Output: "abc123\x00feat: algo"},
		blameAIHeadLog:                    {Output: "feat: algo\n\nbody\n"},
		blameAIInterpretTrailers:          {Output: "feat: algo\n\nbody\n\nCo-Authored-By: Claude <noreply@anthropic.com>\n"},
		aiTrailersAmend:                   {Output: ""},
	}

	result := execute(t, responses, "y\n", "blame-ai", "--tool", "claude")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Co-Authored-By: Claude <noreply@anthropic.com>") {
		t.Errorf("saída = %q, queria o trailer no preview", result.stdout)
	}
	if !strings.Contains(result.stdout, "feat: algo") {
		t.Errorf("saída = %q, queria o assunto do commit alvo", result.stdout)
	}
	if !strings.Contains(result.stdout, "Recuperável com: git reset --hard abc123") {
		t.Errorf("saída = %q, queria a linha de recuperação", result.stdout)
	}
	if !strings.Contains(result.stdout, "Pronto.") {
		t.Errorf("saída = %q, queria a confirmação de que rodou", result.stdout)
	}

	var amended bool
	for _, call := range result.calls {
		if call == aiTrailersAmend {
			amended = true
		}
	}
	if !amended {
		t.Errorf("chamadas = %v, o amend tinha de ter rodado", result.calls)
	}
}

func TestBlameAICancelledLeavesNothingChanged(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		aiTrailersShortHead:               {Output: "abc123"},
		blameAIPlanLog("HEAD"):            {Output: "abc123\x00feat: algo"},
	}

	result := execute(t, responses, "\n", "blame-ai", "--tool", "claude")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Cancelado, nada foi alterado.") {
		t.Errorf("saída = %q, queria a mensagem de cancelamento", result.stdout)
	}
	for _, call := range result.calls {
		if call == blameAIHeadLog || call == aiTrailersAmend {
			t.Fatalf("chamadas = %v, Enter vazio cancela, nada podia rodar depois do preview", result.calls)
		}
	}
}

func TestBlameAIRejectsAnUnknownTool(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
	}

	result := execute(t, responses, "", "blame-ai", "--tool", "chatgpt")

	if result.err == nil {
		t.Fatal("ferramenta desconhecida tem de virar erro")
	}
}

func TestBlameAIOnASpecificCommit(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		aiTrailersShortHead:               {Output: "abc123"},
		blameAIPlanLog("deadbeef"):        {Output: "dead123\x00feat: outro"},
		aiTrailersSymbolicRef:             {Output: "feature"},
		blameAIRebaseExec("deadbeef"):     {Output: ""},
		"rev-parse HEAD":                  {Output: "deadbeef-rewritten"},
		aiTrailersRebaseOnto("deadbeef-rewritten", "deadbeef", "feature"): {Output: ""},
	}

	result := execute(t, responses, "y\n", "blame-ai", "--commit", "deadbeef", "--tool", "claude")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "dead123") {
		t.Errorf("saída = %q, queria o hash abreviado do alvo", result.stdout)
	}
	if !strings.Contains(result.stdout, "Pronto.") {
		t.Errorf("saída = %q, queria a confirmação de que rodou", result.stdout)
	}

	var reattached bool
	for _, call := range result.calls {
		if call == aiTrailersRebaseOnto("deadbeef-rewritten", "deadbeef", "feature") {
			reattached = true
		}
	}
	if !reattached {
		t.Errorf("chamadas = %v, o reencaixe do rabo tinha de ter rodado", result.calls)
	}
}

func TestBlameAIOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "blame-ai", "--tool", "claude")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
}

func TestBlameAIPropagatesPlanFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		aiTrailersShortHead:               {Err: errNotARepository},
	}

	if execute(t, responses, "", "blame-ai", "--tool", "claude").err == nil {
		t.Fatal("falha do plan tem de virar erro")
	}
}

func TestBlameAIPropagatesBlameFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		aiTrailersShortHead:               {Output: "abc123"},
		blameAIPlanLog("HEAD"):            {Output: "abc123\x00feat: algo"},
		blameAIHeadLog:                    {Err: errNotARepository},
	}

	if execute(t, responses, "y\n", "blame-ai", "--tool", "claude").err == nil {
		t.Fatal("falha ao associar tem de virar erro")
	}
}

func TestBlameAIPropagatesTheReadFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		aiTrailersShortHead:               {Output: "abc123"},
		blameAIPlanLog("HEAD"):            {Output: "abc123\x00feat: algo"},
	}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(brokenReader{})
	command.SetArgs([]string{"blame-ai", "--tool", "claude"})

	if command.Execute() == nil {
		t.Fatal("falha na leitura da confirmação tem de virar erro")
	}
}

func TestBlameAIPropagatesWriteFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		aiTrailersShortHead:               {Output: "abc123"},
		blameAIPlanLog("HEAD"):            {Output: "abc123\x00feat: algo"},
	}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"blame-ai", "--tool", "claude"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestBlameAITakesNoArguments(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}

	result := execute(t, responses, "", "blame-ai", "--tool", "claude", "sobrando")

	if result.err == nil {
		t.Fatal("o blame-ai não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}
